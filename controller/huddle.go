package controller

import (
	"net/http"
	"time"

	"github.com/microcosm-cc/microcosm/audit"
	h "github.com/microcosm-cc/microcosm/helpers"
	"github.com/microcosm-cc/microcosm/models"
)

type HuddleController struct{}

func HuddleHandler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	ctl := HuddleController{}

	switch c.GetHttpMethod() {
	case "OPTIONS":
		c.RespondWithOptions([]string{"OPTIONS", "GET", "HEAD", "DELETE"})
		return
	case "GET":
		ctl.Read(c)
	case "HEAD":
		ctl.Read(c)
	case "DELETE":
		ctl.Delete(c)
	default:
		c.RespondWithStatus(http.StatusMethodNotAllowed)
		return
	}
}

// Returns a single huddle
func (ctl *HuddleController) Read(c *models.Context) {
	_, itemTypeId, itemId, status, err := c.GetItemTypeAndItemId()
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	// Start Authorisation
	perms := models.GetPermission(
		models.MakeAuthorisationContext(
			c, 0, itemTypeId, itemId),
	)
	if !perms.CanRead {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}
	// End Authorisation

	// Get Huddle
	m, status, err := models.GetHuddle(c.Site.ID, c.Auth.ProfileId, itemId)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	// Get Comments
	m.Comments, status, err = models.GetComments(c.Site.ID, h.ItemTypeHuddle, m.Id, c.Request.URL, c.Auth.ProfileId, m.Meta.Created)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}
	m.Meta.Permissions = perms

	if c.Auth.ProfileId > 0 {
		// Mark as read (to the last comment on this page if applicable)
		read := m.Meta.Created

		switch m.Comments.Items.(type) {
		case []models.CommentSummaryType:
			comments := m.Comments.Items.([]models.CommentSummaryType)

			if len(comments) > 0 {
				read = comments[len(comments)-1].Meta.Created
			}

			if m.Comments.Page >= m.Comments.Pages {
				read = time.Now()
			}
		default:
		}

		models.MarkAsRead(h.ItemTypes[h.ItemTypeHuddle], m.Id, c.Auth.ProfileId, read)
		models.UpdateUnreadHuddleCount(c.Auth.ProfileId)

		// Get watcher status
		watcherId, sendEmail, sendSms, _, status, err := models.GetWatcherAndIgnoreStatus(
			h.ItemTypes[h.ItemTypeHuddle], m.Id, c.Auth.ProfileId,
		)
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}

		if watcherId > 0 {
			m.Meta.Flags.Watched = true
			m.Meta.Flags.SendEmail = sendEmail
			m.Meta.Flags.SendSms = sendSms
		}
	}

	c.RespondWithData(m)
}

// Deletes a single huddle
func (ctl *HuddleController) Delete(c *models.Context) {
	_, itemTypeId, itemId, status, err := c.GetItemTypeAndItemId()
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	// Start Authorisation
	perms := models.GetPermission(
		models.MakeAuthorisationContext(
			c, 0, itemTypeId, itemId),
	)
	if !perms.CanDelete {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}
	// End Authorisation

	m, status, err := models.GetHuddle(c.Site.ID, c.Auth.ProfileId, itemId)
	if err != nil {
		if status == http.StatusNotFound {
			c.RespondWithOK()
			return
		}

		c.RespondWithErrorDetail(err, status)
		return
	}

	status, err = m.Delete(c.Site.ID, c.Auth.ProfileId)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	audit.Delete(
		c.Site.ID,
		h.ItemTypes[h.ItemTypeHuddle],
		m.Id,
		c.Auth.ProfileId,
		time.Now(),
		c.IP,
	)

	c.RespondWithOK()
}
