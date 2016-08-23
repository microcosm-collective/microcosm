package controller

import (
	// This is required by auth0
	_ "crypto/sha512"

	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"golang.org/x/oauth2"

	"github.com/golang/glog"

	"github.com/microcosm-cc/microcosm/audit"
	conf "github.com/microcosm-cc/microcosm/config"
	h "github.com/microcosm-cc/microcosm/helpers"
	"github.com/microcosm-cc/microcosm/models"
)

// Auth0Controller is a web controller
type Auth0Controller struct{}

// Auth0Handler is a web handler
func Auth0Handler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	ctl := Auth0Controller{}

	switch c.GetHTTPMethod() {
	case "OPTIONS":
		c.RespondWithOptions([]string{"OPTIONS", "POST"})
		return
	case "POST":
		ctl.Create(c)
	default:
		c.RespondWithStatus(http.StatusMethodNotAllowed)
		return
	}
}

// Create handles POST
func (ctl *Auth0Controller) Create(c *models.Context) {
	/////////////////////////////////////
	// Set up and validation of inputs //
	/////////////////////////////////////

	// Need auth0 config
	if c.Site.Auth0Domain == "" ||
		c.Site.Auth0ClientID == "" ||
		c.Site.Auth0ClientSecret == "" {

		glog.Errorf("auth0 is not configured for this site")
		c.RespondWithErrorMessage(
			fmt.Sprintf("auth0 is not configured for this site"),
			http.StatusBadRequest,
		)
		return
	}

	// The info the client needs to send us
	type auth0Callback struct {
		// the auth0 code returned from a callback
		Code string
		// the microcosm oauth client secret
		ClientSecret string
	}

	callback := auth0Callback{}
	err := c.Fill(&callback)
	if err != nil {
		glog.Errorf("The post data is invalid: %v", err.Error())
		c.RespondWithErrorMessage(
			fmt.Sprintf("The post data is invalid: %v", err.Error()),
			http.StatusBadRequest,
		)
		return
	}

	if callback.Code == "" {
		glog.Errorf("code is a required POST parameter and is the auth0 code")
		c.RespondWithErrorMessage(
			fmt.Sprintf("code is a required POST parameter and is the auth0 code"),
			http.StatusBadRequest,
		)
		return
	}
	if callback.ClientSecret == "" {
		glog.Errorf("clientsecret is a required POST parameter and is the microcosm client secret")
		c.RespondWithErrorMessage(
			fmt.Sprintf("clientsecret is a required POST parameter and is the microcosm client secret"),
			http.StatusBadRequest,
		)
		return
	}

	var callbackURL string
	if c.Site.Domain != "" {
		// i.e. www.lfgss.com for CNAME
		if c.Site.ForceSSL {
			callbackURL = "https://" + c.Site.Domain + "/auth0login"
		} else {
			callbackURL = "http://" + c.Site.Domain + "/auth0login"
		}
	} else if c.Site.SubdomainKey == "root" {
		// i.e. microco.sm for root
		callbackURL = "https://" + conf.ConfigStrings[conf.MicrocosmDomain] + "/auth0login"
	} else {
		// i.e. lfgss.microco.sm for subdomain
		callbackURL = "https://" + c.Site.SubdomainKey + "." + conf.ConfigStrings[conf.MicrocosmDomain] + "/auth0login"
	}

	/////////////////////////////
	// Exchange code for token //
	/////////////////////////////

	oauth2Config := &oauth2.Config{
		ClientID:     c.Site.Auth0ClientID,
		ClientSecret: c.Site.Auth0ClientSecret,
		RedirectURL:  callbackURL,
		Scopes:       []string{"openid", "name", "email", "nickname"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://" + c.Site.Auth0Domain + "/authorize",
			TokenURL: "https://" + c.Site.Auth0Domain + "/oauth/token",
		},
	}

	// Exchanging the code for a token
	token, err := oauth2Config.Exchange(oauth2.NoContext, callback.Code)
	if err != nil {
		glog.Errorf(err.Error())
		c.RespondWithErrorMessage(
			fmt.Sprintf(err.Error()),
			http.StatusInternalServerError,
		)
		return
	}

	//////////////////////////////////
	// Exchange token for user info //
	//////////////////////////////////

	client := oauth2Config.Client(oauth2.NoContext, token)
	resp, err := client.Get("https://" + c.Site.Auth0Domain + "/userinfo")
	if err != nil {
		glog.Errorf(err.Error())
		c.RespondWithErrorMessage(
			fmt.Sprintf(err.Error()),
			http.StatusInternalServerError,
		)
		return
	}

	// Reading the body
	raw, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		glog.Errorf(err.Error())
		c.RespondWithErrorMessage(
			fmt.Sprintf(err.Error()),
			http.StatusInternalServerError,
		)
		return
	}

	type Auth0UserInfo struct {
		UserID   string `json:"user_id"`
		Email    string `json:"email"`
		Name     string `json:"name"`
		Nickname string `json:"nickname"`
		Picture  string `json:"picture"`
	}

	var userInfo Auth0UserInfo
	if err := json.Unmarshal(raw, &userInfo); err != nil {
		glog.Errorf(err.Error())
		c.RespondWithErrorMessage(
			fmt.Sprintf(err.Error()),
			http.StatusInternalServerError,
		)
		return
	}

	if userInfo.Email == "" {
		glog.Errorf("auth0 error: no email address received. userinfo = %+v", userInfo)
		c.RespondWithErrorMessage(
			"auth0 error: no email address received",
			http.StatusInternalServerError,
		)
		return
	}

	////////////////////////////////////////////////
	// Create or get a Microcosm user and profile //
	////////////////////////////////////////////////

	// Retrieve user details by email address
	user, status, err := models.GetUserByEmailAddress(userInfo.Email)
	if status == http.StatusNotFound {
		// Check whether this email is a spammer before we attempt to create
		// an account
		if models.IsSpammer(userInfo.Email) {
			glog.Errorf("Spammer: %s", userInfo.Email)
			c.RespondWithErrorMessage("Spammer", http.StatusInternalServerError)
			return
		}

		user, status, err = models.CreateUserByEmailAddress(userInfo.Email)
		if err != nil {
			c.RespondWithErrorMessage(
				fmt.Sprintf("Couldn't create user: %v", err.Error()),
				http.StatusInternalServerError,
			)
			return
		}
	} else if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("Error retrieving user: %v", err.Error()),
			http.StatusInternalServerError,
		)
		return
	}

	// Create a corresponding profile for this user
	// TODO(buro9:2016-08-23): We could use the nickname and picture here and do
	// a better job of creating the profile contextually
	profile, status, err := models.GetOrCreateProfile(c.Site, user)
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("Failed to create profile with ID %d: %v", profile.ID, err.Error()),
			status,
		)
		return
	}

	//////////////////////////////////////////////////
	// Return a Microcosm access token for the user //
	//////////////////////////////////////////////////

	// Fetch API client details by secret
	microcosmOAuthClient, err := models.RetrieveClientBySecret(callback.ClientSecret)
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("Error processing client secret: %v", err.Error()),
			http.StatusInternalServerError,
		)
		return
	}

	// Create and store access token
	tokenValue, err := h.RandString(128)
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("Could not generate a random string: %v", err.Error()),
			http.StatusInternalServerError,
		)
		return
	}

	m := models.AccessTokenType{}
	m.TokenValue = tokenValue
	m.UserID = user.ID
	m.ClientID = microcosmOAuthClient.ClientID

	status, err = m.Insert()
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("Could not create an access token: %v", err.Error()),
			status,
		)
		return
	}

	audit.Create(
		c.Site.ID,
		h.ItemTypes[h.ItemTypeAuth],
		profile.ID,
		profile.ID,
		time.Now(),
		c.IP,
	)

	c.RespondWithData(tokenValue)
}
