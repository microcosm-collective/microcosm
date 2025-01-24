package controller

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/grafana/pyroscope-go"
	"github.com/microcosm-collective/microcosm/audit"
	h "github.com/microcosm-collective/microcosm/helpers"
	"github.com/microcosm-collective/microcosm/models"
)

// SiteController is a web controller
type SiteController struct{}

// SiteHandler is a web handler
func SiteHandler(w http.ResponseWriter, r *http.Request) {
	path := "/sites/{id}"
	pyroscope.TagWrapper(context.Background(), pyroscope.Labels("path", path), func(ctx context.Context) {

		c, status, err := models.MakeContext(r, w)
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}
		ctl := SiteController{}

		method := c.GetHTTPMethod()
		switch method {
		case "OPTIONS":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				c.RespondWithOptions([]string{"OPTIONS", "HEAD", "GET", "PUT"})
			})
			return
		case "GET":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				ctl.Read(c)
			})
		case "PUT":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				ctl.Update(c)
			})
		case "HEAD":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				ctl.Read(c)
			})
		default:
			c.RespondWithStatus(http.StatusMethodNotAllowed)
			return
		}
	})
}

// Read handles GET either by /site or /sites/{site_id}
func (ctl *SiteController) Read(c *models.Context) {
	// Check whether this site is being accessed by ID
	siteQuery, exists := c.RouteVars["site_id"]
	if exists {
		siteID, err := strconv.ParseInt(siteQuery, 10, 64)
		if err != nil {
			c.RespondWithErrorMessage(
				fmt.Sprintf("The supplied site ID ('%s') is not a number.", siteQuery),
				http.StatusBadRequest,
			)
			return
		}

		site, status, err := models.GetSite(siteID)
		if err != nil {
			c.RespondWithErrorMessage(
				fmt.Sprintf("No site with ID %d found", siteID),
				status,
			)
			return
		}

		c.RespondWithData(site)
		return

	}

	// Site already populated in context, so only fetch permissions
	c.Site.Meta.Permissions = models.GetPermission(
		models.MakeAuthorisationContext(
			c, 0, h.ItemTypes[h.ItemTypeSite], c.Site.ID),
	)
	c.RespondWithData(c.Site)
}

// Update handles PUT
func (ctl *SiteController) Update(c *models.Context) {
	_, _, itemID, status, err := c.GetItemTypeAndItemID()
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	m, status, err := models.GetSite(itemID)
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
			"you must be the owner of the site to update it",
			http.StatusForbidden,
		)
		return
	}

	err = c.Fill(&m)
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("the post data is invalid: %v", err.Error()),
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

	c.RespondWithSeeOther(fmt.Sprintf("%s/%d", h.APITypeSite, m.ID))
}
