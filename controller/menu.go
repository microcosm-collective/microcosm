package controller

import (
	"fmt"
	"net/http"
	"strconv"

	h "github.com/microcosm-cc/microcosm/helpers"
	"github.com/microcosm-cc/microcosm/models"
)

type MenuController struct{}

func MenuHandler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	var siteId int64

	if id, exists := c.RouteVars["site_id"]; exists {
		siteId, err = strconv.ParseInt(id, 10, 64)
		if err != nil {
			c.RespondWithErrorMessage(
				fmt.Sprintf("The supplied site_id ('%s') is not a number.", id),
				http.StatusBadRequest,
			)
			return
		}
	}

	if siteId == 0 {
		siteId = c.Site.ID
	}

	ctl := MenuController{}

	switch c.GetHTTPMethod() {
	case "OPTIONS":
		c.RespondWithOptions([]string{"OPTIONS", "GET", "HEAD", "POST", "PUT", "DELETE"})
		return
	case "GET":
		ctl.Read(c, siteId)
	case "HEAD":
		ctl.Read(c, siteId)
	case "PUT":
		ctl.Update(c, siteId)
	case "DELETE":
		ctl.Delete(c, siteId)
	default:
		c.RespondWithStatus(http.StatusMethodNotAllowed)
		return
	}
}

// Returns an array of attributes associated with the given entity
func (ctl *MenuController) Read(c *models.Context, siteId int64) {

	ems, status, err := models.GetMenu(siteId)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	c.RespondWithData(ems)
}

// Updates one or more attributes on the given entity
func (ctl *MenuController) Update(c *models.Context, siteId int64) {
	ems := []h.LinkType{}

	// Start :: Auth
	site, status, err := models.GetSite(siteId)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	// Use the user ID to check, since the current context is a different site (the root site)
	// than the site the owner profile is associated with.
	owner, status, err := models.GetProfileSummary(site.ID, site.OwnedByID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	if owner.UserID != c.Auth.UserID {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}
	// End :: Auth

	err = c.Fill(&ems)
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("The post data is invalid: %v", err.Error()),
			http.StatusBadRequest,
		)
		return
	}

	status, err = models.UpdateMenu(siteId, ems)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	c.RespondWithOK()
}

func (ctl *MenuController) Delete(c *models.Context, siteId int64) {

	// Start :: Auth
	site, status, err := models.GetSite(siteId)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	// Use the user ID to check, since the current context is a different site (the root site)
	// than the site the owner profile is associated with.
	owner, status, err := models.GetProfileSummary(site.ID, site.OwnedByID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	if owner.UserID != c.Auth.UserID {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}
	// End :: Auth

	status, err = models.DeleteMenu(siteId)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	c.RespondWithOK()
}
