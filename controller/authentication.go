package controller

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/golang/glog"

	"github.com/microcosm-cc/microcosm/audit"
	conf "github.com/microcosm-cc/microcosm/config"
	h "github.com/microcosm-cc/microcosm/helpers"
	"github.com/microcosm-cc/microcosm/models"
)

// AuthController is a web controller
type AuthController struct{}

// AuthAccessTokenController is a web controller
type AuthAccessTokenController struct{}

// AuthHandler is a web handler
func AuthHandler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	ctl := AuthController{}

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

// AuthAccessTokenHandler is a web handler
func AuthAccessTokenHandler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	ctl := AuthAccessTokenController{}

	switch c.GetHTTPMethod() {
	case "OPTIONS":
		c.RespondWithOptions([]string{"OPTIONS", "HEAD", "GET", "DELETE"})
		return
	case "HEAD":
		ctl.Read(c)
	case "GET":
		ctl.Read(c)
	case "DELETE":
		ctl.Delete(c)
	default:
		c.RespondWithStatus(http.StatusMethodNotAllowed)
		return
	}
}

// Create handles POST
func (ctl *AuthController) Create(c *models.Context) {
	accessTokenRequest := models.AccessTokenRequestType{}
	err := c.Fill(&accessTokenRequest)
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("The post data is invalid: %v", err.Error()),
			http.StatusBadRequest,
		)
		return
	}

	// Audience is the host that Persona authenticates the user for
	var audience string
	if c.Site.Domain != "" {
		// i.e. www.lfgss.com for CNAME
		audience = c.Site.Domain
	} else if c.Site.SubdomainKey == "root" {
		// i.e. microcosm.app for root
		audience = conf.ConfigStrings[conf.MicrocosmDomain]
	} else {
		// i.e. lfgss.microcosm.app for subdomain
		audience = fmt.Sprintf("%s.%s", c.Site.SubdomainKey, conf.ConfigStrings[conf.MicrocosmDomain])
	}

	// Verify persona assertion
	personaRequest := models.PersonaRequestType{
		Assertion: accessTokenRequest.Assertion,
		Audience:  audience,
	}

	jsonData, err := json.Marshal(personaRequest)
	if err != nil {
		glog.Errorf("Could not marshal Persona req: %s", err.Error())
		c.RespondWithErrorMessage(
			fmt.Sprintf("Bad persona request format: %v", err.Error()),
			http.StatusBadRequest,
		)
		return
	}

	resp, err := http.Post(
		conf.ConfigStrings[conf.PersonaVerifierURL],
		"application/json",
		bytes.NewReader(jsonData),
	)
	if err != nil {
		glog.Errorln(err.Error())
		c.RespondWithErrorMessage(
			fmt.Sprintf("Persona verification error: %v", err.Error()),
			http.StatusInternalServerError,
		)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		glog.Errorf("Couldn't read Persona response: %s", err.Error())
		c.RespondWithErrorMessage(
			fmt.Sprintf("Error unmarshalling persona response: %v", err.Error()),
			http.StatusInternalServerError,
		)
		return
	}
	resp.Body.Close()
	var personaResponse = models.PersonaResponseType{}
	json.Unmarshal(body, &personaResponse)

	if personaResponse.Status != "okay" {
		// Split and decode the assertion to log the user's email address.
		var decoded bool
		if personaRequest.Assertion != "" {
			parts := strings.Split(personaRequest.Assertion, "~")
			moreParts := strings.Split(parts[0], ".")
			if len(moreParts) > 1 {
				data, err := base64.StdEncoding.DecodeString(moreParts[1])
				if err != nil {
					glog.Errorf("Could not base64 decode %s: %s", moreParts[1], err)
				} else {
					decoded = true
					glog.Errorf("Bad Persona response: %+v with decoded assertion: %+v", personaResponse, data)
				}
			}
		}
		if !decoded {
			glog.Errorf("Bad Persona response: %+v with assertion: %+v", personaResponse, personaRequest)
		}
		c.RespondWithErrorMessage(
			fmt.Sprintf("Persona login error: %v", personaResponse.Status),
			http.StatusInternalServerError,
		)
		return
	}

	if personaResponse.Email == "" {
		glog.Errorf("No persona email address")
		c.RespondWithErrorMessage(
			"Persona error: no email address received",
			http.StatusInternalServerError,
		)
		return
	}

	// Retrieve user details by email address
	user, status, err := models.GetUserByEmailAddress(personaResponse.Email)
	if status == http.StatusNotFound {
		// Check whether this email is a spammer before we attempt to create
		// an account
		if models.IsSpammer(personaResponse.Email) {
			glog.Errorf("Spammer: %s", personaResponse.Email)
			c.RespondWithErrorMessage("Spammer", http.StatusInternalServerError)
			return
		}

		user, _, err = models.CreateUserByEmailAddress(personaResponse.Email)
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
	profile, status, err := models.GetOrCreateProfile(c.Site, user)
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("Failed to create profile with ID %d: %v", profile.ID, err.Error()),
			status,
		)
		return
	}

	// Fetch API client details by secret
	client, err := models.RetrieveClientBySecret(accessTokenRequest.ClientSecret)
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
	m.ClientID = client.ClientID

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

// Read handles GET
func (ctl *AuthAccessTokenController) Read(c *models.Context) {
	// Extract access token from request and retrieve its metadata
	m, status, err := models.GetAccessToken(c.RouteVars["access_token"])
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("Error retrieving access token: %v", err.Error()),
			status,
		)
		return
	}
	c.RespondWithData(m)
}

// Delete handles DELETE
func (ctl *AuthAccessTokenController) Delete(c *models.Context) {
	auth_access_token := c.Auth.AccessToken.TokenValue
	path_access_token := c.RouteVars["access_token"]

	if auth_access_token == `` {
		c.RespondWithErrorMessage(
			`?access_token=${access_token} expected in query string as the access_token that authenticates the current request`,
			http.StatusBadRequest,
		)
		return
	}

	if path_access_token == `` {
		c.RespondWithErrorMessage(
			`/api/v1/auth/${access_token} expected in the URI as the access_token to delete`,
			http.StatusBadRequest,
		)
		return
	}

	if !strings.EqualFold(auth_access_token, path_access_token) {
		c.RespondWithErrorMessage(
			`/api/v1/auth/${access_token} and ?access_token=${access_token} must match as you can only delete the access_token for the currently authenticated session`,
			http.StatusBadRequest,
		)
		return
	}

	// Extract access token from request and delete its record
	m, status, err := models.GetAccessToken(path_access_token)
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("error retrieving access token: %v", err.Error()),
			status,
		)
		return
	}

	status, err = m.Delete()
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("error deleting access token: %v", err.Error()),
			status,
		)
		return
	}

	audit.Delete(
		c.Site.ID,
		h.ItemTypes[h.ItemTypeAuth],
		m.UserID,
		c.Auth.ProfileID,
		time.Now(),
		c.IP,
	)

	c.RespondWithOK()
}
