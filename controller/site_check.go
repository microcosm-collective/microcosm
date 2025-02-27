package controller

import (
	"context"
	"fmt"
	"net/http"

	"github.com/golang/glog"
	"github.com/grafana/pyroscope-go"

	"github.com/microcosm-collective/microcosm/models"
)

// SiteCheckController is a web controller
type SiteCheckController struct{}

// SiteCheckHandler is a web handler
func SiteCheckHandler(w http.ResponseWriter, r *http.Request) {
	path := "/sites/{id}/status"
	pyroscope.TagWrapper(context.Background(), pyroscope.Labels("path", path), func(ctx context.Context) {

		c, status, err := models.MakeContext(r, w)
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}
		ctl := SiteCheckController{}

		method := c.GetHTTPMethod()
		switch method {
		case "OPTIONS":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				c.RespondWithOptions([]string{"OPTIONS", "HEAD", "GET"})
			})
			return
		case "GET":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				ctl.Read(c)
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

// Read handles GET
func (ctl *SiteCheckController) Read(c *models.Context) {
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
			"you must be the owner of the site to view its status",
			http.StatusForbidden,
		)
		return
	}

	siteHealth, status, err := models.CheckSiteHealth(m)
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("Error checking site status: %s", err.Error()),
			status,
		)
		return
	}

	glog.Infof("Got site health: %+v\n", siteHealth)

	c.RespondWithData(siteHealth)
}
