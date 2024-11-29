package controller

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	h "git.dee.kitchen/buro9/microcosm/helpers"
	"git.dee.kitchen/buro9/microcosm/models"
	"github.com/grafana/pyroscope-go"
)

// MenuController is a web controller
type MenuController struct{}

// MenuHandler is a web handler
func MenuHandler(w http.ResponseWriter, r *http.Request) {
	path := "/site/menu"
	pyroscope.TagWrapper(context.Background(), pyroscope.Labels("path", path), func(ctx context.Context) {
		c, status, err := models.MakeContext(r, w)
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}

		var siteID int64

		if id, exists := c.RouteVars["site_id"]; exists {
			siteID, err = strconv.ParseInt(id, 10, 64)
			if err != nil {
				c.RespondWithErrorMessage(
					fmt.Sprintf("The supplied site_id ('%s') is not a number.", id),
					http.StatusBadRequest,
				)
				return
			}
		}

		if siteID == 0 {
			siteID = c.Site.ID
		}

		ctl := MenuController{}

		method := c.GetHTTPMethod()
		switch method {
		case "OPTIONS":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				c.RespondWithOptions([]string{"OPTIONS", "GET", "HEAD", "POST", "PUT", "DELETE"})
			})
			return
		case "GET":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				ctl.Read(c, siteID)
			})
		case "HEAD":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				ctl.Read(c, siteID)
			})
		case "PUT":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				ctl.Update(c, siteID)
			})
		case "DELETE":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				ctl.Delete(c, siteID)
			})
		default:
			c.RespondWithStatus(http.StatusMethodNotAllowed)
			return
		}
	})
}

// Read handles GET
func (ctl *MenuController) Read(c *models.Context, siteID int64) {

	ems, status, err := models.GetMenu(siteID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	c.RespondWithData(ems)
}

// Update handles PUT
func (ctl *MenuController) Update(c *models.Context, siteID int64) {
	ems := []h.LinkType{}

	// Start :: Auth
	site, status, err := models.GetSite(siteID)
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

	status, err = models.UpdateMenu(siteID, ems)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	c.RespondWithOK()
}

// Delete handles DELETE
func (ctl *MenuController) Delete(c *models.Context, siteID int64) {
	// Start :: Auth
	site, status, err := models.GetSite(siteID)
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

	status, err = models.DeleteMenu(siteID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	c.RespondWithOK()
}
