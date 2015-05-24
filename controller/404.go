package controller

import (
	"net/http"

	"github.com/microcosm-cc/microcosm/models"
	"github.com/microcosm-cc/microcosm/resolver"
)

type Redirect404Controller struct{}

func Redirect404Handler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	ctl := Redirect404Controller{}

	switch c.GetHTTPMethod() {
	case "OPTIONS":
		c.RespondWithOptions([]string{"OPTIONS", "GET"})
		return
	case "GET":
		ctl.Read(c)
	default:
		c.RespondWithStatus(http.StatusMethodNotAllowed)
		return
	}
}

func (ctl *Redirect404Controller) Read(c *models.Context) {

	// Just call this with authentication (if you have it) and pass in the
	// unknown URL as a GET param:
	//
	//   https://dev1.microco.sm/api/v1/resolve?url=http%3A%2F%2Fwww.lfgss.com%2Fforumdisplay.php%3Ff%3D1
	//
	// The URL in the GET param should be URL encoded to ensure that if it has
	// any querystring, nothing goes awry. An example querystring URL might be:
	//
	//   http://www.lfgss.com/private.php?t=7865&page=32
	//
	// If you don't get a page/offset back from this, then you haven't URL encoded
	// it.

	u := c.Request.URL
	q := u.Query()
	inUrl := q.Get("url")
	if inUrl == "" {
		c.RespondWithError(http.StatusBadRequest)
		return
	}

	c.RespondWithData(resolver.Resolve(c.Site.ID, inUrl, c.Auth.ProfileID))
}
