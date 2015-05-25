package controller

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/microcosm-cc/microcosm/models"
)

// GeoCodeController is a web handler
type GeoCodeController struct{}

// GeoCodeHandler is a web handler
func GeoCodeHandler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	if c.Request.Method != "GET" {
		c.RespondWithNotImplemented()
		return
	}

	ctl := GeoCodeController{}
	ctl.Read(c)
}

// Error is a generic error handler for the Geo controller
func (ctl *GeoCodeController) Error(c *models.Context, message string, status int) {
	errorJSON := `{"error":["` + message + `"]}`

	contentLength := len(errorJSON)
	c.ResponseWriter.Header().Set("Content-Length", strconv.Itoa(contentLength))

	dur := time.Now().Sub(c.StartTime)
	go models.SendUsage(c, status, contentLength, dur, []string{"message"})

	c.WriteResponse([]byte(errorJSON), status)
	return
}

// Read handles GET
func (ctl *GeoCodeController) Read(c *models.Context) {
	c.ResponseWriter.Header().Set("Content-Type", "application/json")

	// Debugging info
	dur := time.Now().Sub(c.StartTime)

	place := strings.Trim(c.Request.URL.Query().Get("q"), " ")

	if strings.Trim(c.Request.URL.Query().Get("q"), " ") == "" {
		ctl.Error(c, "query needed", http.StatusBadRequest)
		return
	}
	if c.Auth.ProfileID <= 0 {
		ctl.Error(c, "no auth", http.StatusForbidden)
		return
	}

	u, _ := url.Parse("http://open.mapquestapi.com/nominatim/v1/search.php")
	q := u.Query()
	q.Set("format", "json")
	// We are not interested in the array returned, just the best match which is the first response
	q.Set("limit", "1")
	q.Set("q", place)
	u.RawQuery = q.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		ctl.Error(c, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		ctl.Error(c, err.Error(), http.StatusInternalServerError)
		return
	}

	// Again, not interested in the outer array [], so we substring that out
	t := string(body)
	t = t[1 : len(t)-1]

	// But if we now have nothing (no matches were found, we need to return an empty object)
	if strings.Trim(t, ` `) == `` {
		t = `{}`
	}
	body = []byte(t)

	// Return our JavaScript object
	contentLength := len(body)
	c.ResponseWriter.Header().Set("Content-Length", strconv.Itoa(contentLength))
	go models.SendUsage(c, http.StatusOK, contentLength, dur, []string{})

	c.WriteResponse([]byte(body), http.StatusOK)
}
