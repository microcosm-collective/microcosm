package controller

import (
	"fmt"
	"net/http"
	"time"

	"git.dee.kitchen/buro9/microcosm/audit"
	h "git.dee.kitchen/buro9/microcosm/helpers"
	"git.dee.kitchen/buro9/microcosm/models"
)

// ProfileOptionsHandler is a web handler
func ProfileOptionsHandler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}
	ctl := ProfileOptionsController{}

	method := c.GetHTTPMethod()
	switch method {
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

// ProfileOptionsController is a web controller
type ProfileOptionsController struct{}

// Read handles GET
func (ctl *ProfileOptionsController) Read(c *models.Context) {
	if c.Auth.ProfileID < 1 {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}

	m, status, err := models.GetProfileOptions(c.Auth.ProfileID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	c.RespondWithData(m)
}

// Update handles PUT
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
	m.ProfileID = c.Auth.ProfileID

	status, err := m.Update()
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	audit.Update(
		c.Site.ID,
		h.ItemTypes[h.ItemTypeProfile],
		m.ProfileID,
		c.Auth.ProfileID,
		time.Now(),
		c.IP,
	)

	// Respond
	c.RespondWithSeeOther("/api/v1/profiles/options")
}
