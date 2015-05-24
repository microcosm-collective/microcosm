package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/microcosm-cc/microcosm/audit"
	h "github.com/microcosm-cc/microcosm/helpers"
	"github.com/microcosm-cc/microcosm/models"
)

type SiteController struct{}

func SiteHandler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}
	ctl := SiteController{}

	switch c.GetHTTPMethod() {
	case "OPTIONS":
		c.RespondWithOptions([]string{"OPTIONS", "HEAD", "GET", "PUT"})
		return
	case "GET":
		ctl.Read(c)
	case "PUT":
		ctl.Update(c)
	case "HEAD":
		ctl.Read(c)
	default:
		c.RespondWithStatus(http.StatusMethodNotAllowed)
		return
	}
}

//Read can be accessed either by /site or /sites/{site_id} since it is registered
//as handler for both routes.
func (ctl *SiteController) Read(c *models.Context) {

	// Check whether this site is being accessed by ID
	siteQuery, exists := c.RouteVars["site_id"]
	if exists {

		siteId, err := strconv.ParseInt(siteQuery, 10, 64)
		if err != nil {
			c.RespondWithErrorMessage(
				fmt.Sprintf("The supplied site ID ('%s') is not a number.", siteQuery),
				http.StatusBadRequest,
			)
			return
		}

		site, status, err := models.GetSite(siteId)
		if err != nil {
			c.RespondWithErrorMessage(
				fmt.Sprintf("No site with ID %d found", siteId),
				status,
			)
			return
		}

		c.RespondWithData(site)
		return

	} else {

		// Site already populated in context, so only fetch permissions
		c.Site.Meta.Permissions = models.GetPermission(
			models.MakeAuthorisationContext(
				c, 0, h.ItemTypes[h.ItemTypeSite], c.Site.ID),
		)
		c.RespondWithData(c.Site)
		return

	}
}

func (ctl *SiteController) Update(c *models.Context) {

	_, _, itemId, status, err := c.GetItemTypeAndItemID()
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	m, status, err := models.GetSite(itemId)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	// Use the user ID to check, since the current context is a different site (the root site)
	// than the site the owner profile is associated with.
	owner, status, err := models.GetProfileSummary(m.ID, m.OwnedByID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	if owner.UserID != c.Auth.UserID {
		c.RespondWithErrorMessage(
			fmt.Sprintf("You must be the owner of the site to update it"),
			http.StatusForbidden,
		)
		return
	}

	err = c.Fill(&m)
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("The post data is invalid: %v", err.Error()),
			http.StatusBadRequest,
		)
		return
	}

	status, err = m.Update()
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	audit.Replace(
		c.Site.ID,
		h.ItemTypes[h.ItemTypeSite],
		m.ID,
		c.Auth.ProfileID,
		time.Now(),
		c.IP,
	)

	c.RespondWithSeeOther(fmt.Sprintf("%s/%d", h.ApiTypeSite, m.ID))
}
