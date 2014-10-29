package controller

import (
	"net/http"

	"github.com/microcosm-cc/microcosm/models"
)

type EffectivePermissionsController struct{}

func EffectivePermissionsHandler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	ctl := EffectivePermissionsController{}

	switch c.GetHttpMethod() {
	case "OPTIONS":
		c.RespondWithOptions([]string{"OPTIONS", "GET", "HEAD"})
		return
	case "GET":
		ctl.ReadMany(c)
	case "HEAD":
		ctl.ReadMany(c)
	default:
		c.RespondWithStatus(http.StatusMethodNotAllowed)
		return
	}
}

// Returns the effective permissions a profile has for a given microcosm
func (ctl *EffectivePermissionsController) ReadMany(c *models.Context) {
}
