package controller

import (
	"fmt"
	"net/http"
	"time"

	"github.com/microcosm-cc/microcosm/audit"
	h "github.com/microcosm-cc/microcosm/helpers"
	"github.com/microcosm-cc/microcosm/models"
)

func ProfileOptionsHandler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	ctl := ProfileOptionsController{}

	switch c.GetHttpMethod() {
	case "OPTIONS":
		c.RespondWithOptions([]string{"OPTIONS", "HEAD", "GET", "PUT"})
		return
	case "HEAD":
		ctl.Read(c)
	case "GET":
		ctl.Read(c)
	case "PUT":
		ctl.Update(c)
	default:
		c.RespondWithStatus(http.StatusMethodNotAllowed)
		return
	}
}

type ProfileOptionsController struct{}

func (ctl *ProfileOptionsController) Read(c *models.Context) {

	if c.Auth.ProfileId < 1 {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}

	m, status, err := models.GetProfileOptions(c.Auth.ProfileId)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}
	c.ResponseWriter.Header().Set("Cache-Control", "no-cache, max-age=0")
	c.RespondWithData(m)
}

func (ctl *ProfileOptionsController) Update(c *models.Context) {

	var m = models.ProfileOptionType{}
	err := c.Fill(&m)
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("The post data is invalid: %v", err.Error()),
			http.StatusBadRequest,
		)
		return
	}

	// Profile ID cannot be changed
	m.ProfileId = c.Auth.ProfileId

	status, err := m.Update()
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	audit.Update(
		c.Site.Id,
		h.ItemTypes[h.ItemTypeProfile],
		m.ProfileId,
		c.Auth.ProfileId,
		time.Now(),
		c.IP,
	)

	// Respond
	c.RespondWithSeeOther(
		fmt.Sprintf("/api/v1/profiles/options"),
	)
}
