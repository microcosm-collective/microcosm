package controller

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/grafana/pyroscope-go"
	"github.com/microcosm-cc/microcosm/audit"
	h "github.com/microcosm-cc/microcosm/helpers"
	"github.com/microcosm-cc/microcosm/models"
)

// ProfileHandler is a web handler
func ProfileHandler(w http.ResponseWriter, r *http.Request) {
	path := "/profiles/{id}"
	pyroscope.TagWrapper(context.Background(), pyroscope.Labels("path", path), func(context.Context) {
		c, status, err := models.MakeContext(r, w)
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}

		ctl := ProfileController{}

		method := c.GetHTTPMethod()
		switch method {
		case "OPTIONS":
			pyroscope.TagWrapper(context.Background(), pyroscope.Labels("method", method), func(context.Context) {
				c.RespondWithOptions([]string{"OPTIONS", "HEAD", "GET", "PUT", "PATCH", "DELETE"})
			})
			return
		case "HEAD":
			pyroscope.TagWrapper(context.Background(), pyroscope.Labels("method", method), func(context.Context) {
				ctl.Read(c)
			})
		case "GET":
			pyroscope.TagWrapper(context.Background(), pyroscope.Labels("method", method), func(context.Context) {
				ctl.Read(c)
			})
		case "PUT":
			pyroscope.TagWrapper(context.Background(), pyroscope.Labels("method", method), func(context.Context) {
				ctl.Update(c)
			})
		case "PATCH":
			pyroscope.TagWrapper(context.Background(), pyroscope.Labels("method", method), func(context.Context) {
				ctl.Patch(c)
			})
		case "DELETE":
			pyroscope.TagWrapper(context.Background(), pyroscope.Labels("method", method), func(context.Context) {
				ctl.Delete(c)
			})
		default:
			c.RespondWithStatus(http.StatusMethodNotAllowed)
			return
		}
	})
}

// ProfileController is a web controller
type ProfileController struct{}

// Create handles POST
func (ctl *ProfileController) Create(c *models.Context) {
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

// Read handles GET
func (ctl *ProfileController) Read(c *models.Context) {
	_, itemTypeID, itemID, status, err := c.GetItemTypeAndItemID()
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	// Start Authorisation
	perms := models.GetPermission(
		models.MakeAuthorisationContext(
			c, 0, itemTypeID, itemID),
	)
	if c.Site.ID == 1 {
		if c.Auth.ProfileID != itemID {
			perms.CanRead = false
		}
	}
	if !perms.CanRead {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}
	// End Authorisation

	m, status, err := models.GetProfile(c.Site.ID, itemID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}
	m.Meta.Permissions = perms

	if c.Auth.ProfileID > 0 {
		watcherID, sendEmail, sendSms, ignored, status, err :=
			models.GetWatcherAndIgnoreStatus(
				h.ItemTypes[h.ItemTypeProfile], m.ID, c.Auth.ProfileID,
			)
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}

		if watcherID > 0 {
			m.Meta.Flags.Watched = true
			m.Meta.Flags.SendEmail = sendEmail
			m.Meta.Flags.SendSMS = sendSms
		}

		if ignored {
			m.Meta.Flags.Ignored = true
		}

		if c.Auth.ProfileID == m.ID {
			m.GetUnreadHuddleCount()
		}

		if perms.IsOwner || perms.IsSiteOwner {
			m.Email = models.GetProfileEmail(c.Site.ID, m.ID)

			// Get member status
			attrs, _, _, status, err := models.GetAttributes(
				h.ItemTypes[h.ItemTypeProfile],
				m.ID,
				h.DefaultQueryLimit,
				h.DefaultQueryOffset,
			)
			if err != nil {
				c.RespondWithErrorDetail(err, status)
			}
			for _, attr := range attrs {
				if attr.Key == "is_member" && attr.Boolean.Valid {
					m.Member = attr.Boolean.Bool
				}
			}
		}
	}

	c.RespondWithData(m)
}

// Update handles PUT
func (ctl *ProfileController) Update(c *models.Context) {
	_, itemTypeID, itemID, status, err := c.GetItemTypeAndItemID()
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	m, status, err := models.GetProfile(c.Site.ID, itemID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
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

	// Start Authorisation
	perms := models.GetPermission(
		models.MakeAuthorisationContext(
			c, 0, itemTypeID, itemID),
	)
	if !perms.CanUpdate {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}
	// End Authorisation

	// Populate site and user ID from goweb context
	m.SiteID = c.Site.ID

	status, err = m.Update()
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	audit.Replace(
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

// Patch handles PATCH
func (ctl *ProfileController) Patch(c *models.Context) {
	_, itemTypeID, itemID, status, err := c.GetItemTypeAndItemID()
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	patches := []h.PatchType{}
	err = c.Fill(&patches)
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("The post data is invalid: %v", err.Error()),
			http.StatusBadRequest,
		)
		return
	}

	status, err = h.TestPatch(patches)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	// Start Authorisation
	ac := models.MakeAuthorisationContext(c, 0, itemTypeID, itemID)
	perms := models.GetPermission(ac)
	if !perms.CanUpdate {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}

	// All patches are 'replace'
	for _, patch := range patches {
		status, err := patch.ScanRawValue()
		if !patch.Bool.Valid {
			c.RespondWithErrorDetail(err, status)
			return
		}

		switch patch.Path {
		case "/member":
			// Only super users' can sticky and unsticky
			if !patch.Bool.Valid {
				c.RespondWithErrorMessage("/member requires a bool value", http.StatusBadRequest)
				return
			}
		default:
			c.RespondWithErrorMessage("Invalid patch operation path", http.StatusBadRequest)
			return
		}
	}
	// End Authorisation

	// Populate where applicable from auth and context
	m, status, err := models.GetProfile(c.Site.ID, itemID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	status, err = m.Patch(ac, patches)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	audit.Update(
		c.Site.ID,
		h.ItemTypes[h.ItemTypeProfile],
		m.ID,
		c.Auth.ProfileID,
		time.Now(),
		c.IP,
	)

	c.RespondWithOK()
}

// Delete handles DELETE
func (ctl *ProfileController) Delete(c *models.Context) {
	// Right now no-one can delete as it would break attribution
	// of things like Comments
	c.RespondWithNotImplemented()

	/*
		_, itemTypeID, itemID, status, err := c.GetItemTypeAndItemID()
		if err != nil {
			c.RespondWithErrorDetail(err, status)
		}

		m := models.ProfileType{}
		m.Id = itemID

		status, err := m.Delete()
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}

		audit.Replace(
			c.Site.ID,
			h.ItemTypes[h.ItemTypeProfile],
			m.Id,
			c.Auth.ProfileID,
			time.Now(),
			c.IP,
		)

		c.RespondWithOK()
	*/
}
