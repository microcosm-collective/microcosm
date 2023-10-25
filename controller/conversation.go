package controller

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/grafana/pyroscope-go"
	"github.com/lib/pq"

	"github.com/microcosm-cc/microcosm/audit"
	h "github.com/microcosm-cc/microcosm/helpers"
	"github.com/microcosm-cc/microcosm/models"
)

// ConversationController is a web controller
type ConversationController struct{}

// ConversationHandler is a web handler
func ConversationHandler(w http.ResponseWriter, r *http.Request) {
	path := "/conversations/{id}"
	pyroscope.TagWrapper(context.Background(), pyroscope.Labels("path", path), func(context.Context) {
		c, status, err := models.MakeContext(r, w)
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}

		ctl := ConversationController{}

		method := c.GetHTTPMethod()
		switch method {
		case "OPTIONS":
			pyroscope.TagWrapper(context.Background(), pyroscope.Labels("method", method), func(context.Context) {
				c.RespondWithOptions([]string{"OPTIONS", "GET", "HEAD", "PUT", "PATCH", "DELETE"})
			})
			return
		case "GET":
			pyroscope.TagWrapper(context.Background(), pyroscope.Labels("method", method), func(context.Context) {
				ctl.Read(c)
			})
		case "HEAD":
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

// Read handles GET
func (ctl *ConversationController) Read(c *models.Context) {
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
	if !perms.CanRead {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}
	// End Authorisation

	// Get Conversation
	m, status, err := models.GetConversation(c.Site.ID, itemID, c.Auth.ProfileID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	// Get Comments
	m.Comments, status, err = models.GetComments(
		c.Site.ID,
		h.ItemTypeConversation,
		m.ID,
		c.Request.URL,
		c.Auth.ProfileID,
		m.Meta.Created,
	)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}
	m.Meta.Permissions = perms

	if c.Auth.ProfileID > 0 {
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

		go models.MarkAsRead(h.ItemTypes[h.ItemTypeConversation], m.ID, c.Auth.ProfileID, read)

		// Get watcher status
		watcherID, sendEmail, sendSms, ignored, status, err := models.GetWatcherAndIgnoreStatus(
			h.ItemTypes[h.ItemTypeConversation], m.ID, c.Auth.ProfileID,
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
	go models.IncrementViewCount(h.ItemTypes[h.ItemTypeConversation], m.ID)

	c.RespondWithData(m)
}

// Update handles PUT
func (ctl *ConversationController) Update(c *models.Context) {
	_, itemTypeID, itemID, status, err := c.GetItemTypeAndItemID()
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	// Validate inputs
	m, status, err := models.GetConversation(c.Site.ID, itemID, c.Auth.ProfileID)
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

	// Populate where applicable from auth and context
	m.Meta.EditedByNullable = sql.NullInt64{Int64: c.Auth.ProfileID, Valid: true}
	m.Meta.EditedNullable = pq.NullTime{Time: time.Now(), Valid: true}

	status, err = m.Update(c.Site.ID, c.Auth.ProfileID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	audit.Replace(
		c.Site.ID,
		h.ItemTypes[h.ItemTypeConversation],
		m.ID,
		c.Auth.ProfileID,
		time.Now(),
		c.IP,
	)

	c.RespondWithSeeOther(
		fmt.Sprintf(
			"%s/%d",
			h.APITypeConversation,
			m.ID,
		),
	)
}

// Patch handles PATCH
func (ctl *ConversationController) Patch(c *models.Context) {
	_, itemTypeID, itemID, status, err := c.GetItemTypeAndItemID()
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	// Validate inputs
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
		case "/meta/flags/sticky":
			// Only super users' can sticky and unsticky
			if !perms.IsModerator {
				c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
				return
			}
			if !patch.Bool.Valid {
				c.RespondWithErrorMessage("/meta/flags/sticky requires a bool value", http.StatusBadRequest)
				return
			}
		case "/meta/flags/open":
			// Only super users' and item owners can open and close
			if !(perms.IsModerator || perms.IsOwner) {
				c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
				return
			}
			if !patch.Bool.Valid {
				c.RespondWithErrorMessage("/meta/flags/open requires a bool value", http.StatusBadRequest)
				return
			}
		case "/meta/flags/deleted":
			// Only super users' can undelete, but super users' and owners can delete
			if !patch.Bool.Valid {
				c.RespondWithErrorMessage("/meta/flags/deleted requires a bool value", http.StatusBadRequest)
				return
			}
			if (!patch.Bool.Bool && !(perms.IsModerator || perms.IsOwner)) || !perms.IsModerator {
				c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
				return
			}
		case "/meta/flags/moderated":
			if !perms.IsModerator {
				c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
				return
			}
		default:
			c.RespondWithErrorMessage("Invalid patch operation path", http.StatusBadRequest)
			return
		}
	}
	// End Authorisation

	m, status, err := models.GetConversation(c.Site.ID, itemID, c.Auth.ProfileID)
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
		h.ItemTypes[h.ItemTypeConversation],
		m.ID,
		c.Auth.ProfileID,
		time.Now(),
		c.IP,
	)

	c.RespondWithOK()
}

// Delete handles DELETE
func (ctl *ConversationController) Delete(c *models.Context) {
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
	if !perms.CanDelete {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}
	// End Authorisation

	m, status, err := models.GetConversation(c.Site.ID, itemID, c.Auth.ProfileID)
	if err != nil {
		if status == http.StatusNotFound {
			c.RespondWithOK()
			return
		}

		c.RespondWithErrorDetail(err, status)
		return
	}

	status, err = m.Delete()
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	audit.Delete(
		c.Site.ID,
		h.ItemTypes[h.ItemTypeConversation],
		m.ID,
		c.Auth.ProfileID,
		time.Now(),
		c.IP,
	)

	c.RespondWithOK()
}
