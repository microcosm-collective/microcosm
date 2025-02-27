package controller

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/golang/glog"
	"github.com/grafana/pyroscope-go"

	"github.com/microcosm-collective/microcosm/audit"
	h "github.com/microcosm-collective/microcosm/helpers"
	"github.com/microcosm-collective/microcosm/models"
)

// SitesController is a web controller
type SitesController struct{}

// SitesHandler is a web handler
func SitesHandler(w http.ResponseWriter, r *http.Request) {
	path := "/sites"
	pyroscope.TagWrapper(context.Background(), pyroscope.Labels("path", path), func(ctx context.Context) {

		c, status, err := models.MakeContext(r, w)
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}
		ctl := SitesController{}

		method := c.GetHTTPMethod()
		switch method {
		case "OPTIONS":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				c.RespondWithOptions([]string{"OPTIONS", "POST"})
			})
			return
		case "POST":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				ctl.Create(c)
			})
		case "GET":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				ctl.ReadMany(c)
			})
		default:
			c.RespondWithStatus(http.StatusMethodNotAllowed)
			return
		}
	})
}

// ReadMany handles GET for a collection
func (ctl *SitesController) ReadMany(c *models.Context) {
	limit, offset, status, err := h.GetLimitAndOffset(c.Request.URL.Query())
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	var userID int64

	// Passing ?filter=owned returns sites owned by logged-in user.
	if c.Request.FormValue("filter") == "owned" {
		if c.Auth.UserID == 0 {
			c.RespondWithErrorMessage(
				"you must be logged in to list your own sites",
				http.StatusForbidden,
			)
			return
		}

		userID = c.Auth.UserID
	}

	// Passing 0 as userID means return all sites.
	ems, total, pages, status, err := models.GetSites(userID, limit, offset)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	thisLink := h.GetLinkToThisPage(*c.Request.URL, offset, limit, total)

	m := models.SitesType{}
	m.Sites = h.ConstructArray(
		ems,
		h.APITypeSite,
		total,
		limit,
		offset,
		pages,
		c.Request.URL,
	)
	m.Meta.Links =
		[]h.LinkType{
			{Rel: "self", Href: thisLink.String()},
		}

	c.RespondWithData(m)
}

// Create handles POST
func (ctl *SitesController) Create(c *models.Context) {
	if c.Auth.ProfileID < 1 {
		c.RespondWithErrorMessage(
			"you must be logged in to the root site create a site",
			http.StatusForbidden,
		)
		return
	}

	if c.Site.ID != 1 {
		c.RespondWithErrorMessage(
			"sites can only be created from the root site",
			http.StatusBadRequest,
		)
		return
	}

	bytes, _ := httputil.DumpRequest(c.Request, true)
	glog.Infoln(string(bytes))

	m := models.SiteType{}
	// Default theme
	m.ThemeID = 1
	err := c.Fill(&m)
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("The post data is invalid: %v", err.Error()),
			http.StatusBadRequest,
		)
		return
	}

	// Fetch full user info from context user ID
	user, status, err := models.GetUser(c.Auth.UserID)
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("Could not retrieve use associated with ID %d", c.Auth.UserID),
			status,
		)
		return
	}

	site, _, _, err := models.CreateOwnedSite(m, user)
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("Error creating site: %s", err.Error()),
			http.StatusInternalServerError,
		)
		return
	}

	audit.Create(
		c.Site.ID,
		h.ItemTypes[h.ItemTypeSite],
		site.ID,
		c.Auth.ProfileID,
		time.Now(),
		c.IP,
	)

	c.RespondWithSeeOther(fmt.Sprintf("%s/%d", h.APITypeSite, site.ID))
}
