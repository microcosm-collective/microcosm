package controller

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"git.dee.kitchen/buro9/microcosm/audit"
	h "git.dee.kitchen/buro9/microcosm/helpers"
	"git.dee.kitchen/buro9/microcosm/models"
	"github.com/grafana/pyroscope-go"
)

// ProfilesHandler is a web handler
func ProfilesHandler(w http.ResponseWriter, r *http.Request) {
	path := "/profiles"
	pyroscope.TagWrapper(context.Background(), pyroscope.Labels("path", path), func(ctx context.Context) {
		c, status, err := models.MakeContext(r, w)
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}
		ctl := ProfilesController{}

		method := c.GetHTTPMethod()
		switch method {
		case "OPTIONS":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				c.RespondWithOptions([]string{"OPTIONS", "POST", "HEAD", "GET"})
			})
			return
		case "POST":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				ctl.Create(c)
			})
		case "HEAD":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				ctl.ReadMany(c)
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

// ProfilesController is a web controller
type ProfilesController struct{}

// Create handles POST
func (ctl *ProfilesController) Create(c *models.Context) {
	m := models.ProfileType{}
	m.Visible = true

	err := c.Fill(&m)
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("The post data is invalid: %v", err.Error()),
			http.StatusBadRequest,
		)
		return
	}

	// TODO: Auth rules on creation

	if m.SiteID != 0 {
		c.RespondWithErrorMessage(
			"You cannot supply a site ID when creating a profile",
			http.StatusBadRequest,
		)
		return
	}

	if m.UserID != 0 {
		c.RespondWithErrorMessage(
			"You cannot supply a user ID when creating a profile",
			http.StatusBadRequest,
		)
		return
	}

	// Populate site and user ID from goweb context
	m.SiteID = c.Site.ID
	m.UserID = c.Auth.UserID

	status, err := m.Insert()
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	audit.Create(
		c.Site.ID,
		h.ItemTypes[h.ItemTypeProfile],
		m.ID,
		c.Auth.ProfileID,
		time.Now(),
		c.IP,
	)

	c.RespondWithSeeOther(
		fmt.Sprintf(
			"%s/%d",
			h.APITypeProfile,
			m.ID,
		),
	)
}

// ReadMany handles GET for the collection
func (ctl *ProfilesController) ReadMany(c *models.Context) {
	// Start Authorisation
	perms := models.GetPermission(
		models.MakeAuthorisationContext(
			c, 0, h.ItemTypes[h.ItemTypeProfile], 0),
	)
	if !perms.CanRead {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}
	// End Authorisation

	// Fetch query string args if any exist
	limit, offset, status, err := h.GetLimitAndOffset(c.Request.URL.Query())
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	so := models.GetProfileSearchOptions(c.Request.URL.Query())
	so.ProfileID = c.Auth.ProfileID

	ems, total, pages, status, err := models.GetProfiles(
		c.Site.ID,
		so,
		limit,
		offset,
	)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	// Construct the response
	thisLink := h.GetLinkToThisPage(*c.Request.URL, offset, limit, total)

	m := models.ProfilesType{}
	m.Profiles = h.ConstructArray(ems, h.APITypeProfile, total, limit, offset, pages, c.Request.URL)
	m.Meta.Links = []h.LinkType{
		{Rel: "self", Href: thisLink.String()},
	}
	m.Meta.Permissions = perms

	if c.Auth.ProfileID > 0 {
		// Get watcher status
		watcherID, sendEmail, sendSms, ignored, status, err := models.GetWatcherAndIgnoreStatus(
			h.ItemTypes[h.ItemTypeProfile], 0, c.Auth.ProfileID,
		)
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}

		if ignored {
			m.Meta.Flags.Ignored = true
		}

		if watcherID > 0 {
			m.Meta.Flags.Watched = true
			m.Meta.Flags.SendEmail = sendEmail
			m.Meta.Flags.SendSMS = sendSms
		}
	}

	c.RespondWithData(m)
}
