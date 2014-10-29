package controller

import (
	"net/http"

	h "github.com/microcosm-cc/microcosm/helpers"
	"github.com/microcosm-cc/microcosm/models"
)

func UpdateOptionsHandler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	ctl := UpdateOptionsController{}

	switch c.GetHttpMethod() {
	case "OPTIONS":
		c.RespondWithOptions([]string{"OPTIONS", "HEAD", "GET"})
		return
	case "HEAD":
		ctl.ReadMany(c)
	case "GET":
		ctl.ReadMany(c)
	default:
		c.RespondWithStatus(http.StatusMethodNotAllowed)
		return
	}
}

type UpdateOptionsController struct{}

// ReadMany fetches alert preferences for a user based on their Profile ID. As such
// they are site-specific and the AlertTypeId acts as the resource ID.
func (ctl *UpdateOptionsController) ReadMany(c *models.Context) {

	if c.Auth.ProfileId < 1 {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}

	prefs, status, err := models.GetUpdateOptions(c.Site.Id, c.Auth.ProfileId)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	c.ResponseWriter.Header().Set("Cache-Control", "no-cache, max-age=0")
	c.RespondWithData(prefs)
}
