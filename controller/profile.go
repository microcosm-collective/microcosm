package controller

import (
	"fmt"
	"net/http"
	"time"

	"github.com/microcosm-cc/microcosm/audit"
	h "github.com/microcosm-cc/microcosm/helpers"
	"github.com/microcosm-cc/microcosm/models"
)

func ProfileHandler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	ctl := ProfileController{}

	switch c.GetHTTPMethod() {
	case "OPTIONS":
		c.RespondWithOptions([]string{"OPTIONS", "HEAD", "GET", "PUT", "DELETE"})
		return
	case "HEAD":
		ctl.Read(c)
	case "GET":
		ctl.Read(c)
	case "PUT":
		ctl.Update(c)
	case "DELETE":
		ctl.Delete(c)
	default:
		c.RespondWithStatus(http.StatusMethodNotAllowed)
		return
	}
}

type ProfileController struct{}

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

func (ctl *ProfileController) Read(c *models.Context) {
	_, itemTypeId, itemId, status, err := c.GetItemTypeAndItemID()
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	// Start Authorisation
	perms := models.GetPermission(
		models.MakeAuthorisationContext(
			c, 0, itemTypeId, itemId),
	)
	if c.Site.ID == 1 {
		if c.Auth.ProfileID != itemId {
			perms.CanRead = false
		}
	}
	if !perms.CanRead {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}
	// End Authorisation

	m, status, err := models.GetProfile(c.Site.ID, itemId)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}
	m.Meta.Permissions = perms

	if c.Auth.ProfileID > 0 {
		// Get watcher status
		watcherId, sendEmail, sendSms, ignored, status, err := models.GetWatcherAndIgnoreStatus(
			h.ItemTypes[h.ItemTypeProfile], m.ID, c.Auth.ProfileID,
		)
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}

		if ignored {
			m.Meta.Flags.Ignored = true
		}

		if watcherId > 0 {
			m.Meta.Flags.Watched = true
			m.Meta.Flags.SendEmail = sendEmail
			m.Meta.Flags.SendSMS = sendSms
		}

		if c.Auth.ProfileID == m.ID {
			// Get counts of things
			m.GetUnreadHuddleCount()
		}

		if perms.IsOwner {
			user, status, err := models.GetUser(c.Auth.UserID)
			if err != nil {
				c.RespondWithErrorDetail(err, status)
				return
			}
			m.Email = user.Email
		}
	}

	c.RespondWithData(m)
}

func (ctl *ProfileController) Update(c *models.Context) {
	_, itemTypeId, itemId, status, err := c.GetItemTypeAndItemID()
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	m, status, err := models.GetProfile(c.Site.ID, itemId)
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
			c, 0, itemTypeId, itemId),
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

func (ctl *ProfileController) Delete(c *models.Context) {

	// Right now no-one can delete as it would break attribution
	// of things like Comments
	c.RespondWithNotImplemented()
	return

	/*
		_, itemTypeId, itemId, status, err := c.GetItemTypeAndItemID()
		if err != nil {
			c.RespondWithErrorDetail(err, status)
		}

		m := models.ProfileType{}
		m.Id = itemId

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
