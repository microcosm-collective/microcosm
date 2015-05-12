package controller

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/lib/pq"

	"github.com/microcosm-cc/microcosm/audit"
	h "github.com/microcosm-cc/microcosm/helpers"
	"github.com/microcosm-cc/microcosm/models"
)

func MicrocosmHandler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	ctl := MicrocosmController{}

	switch c.GetHttpMethod() {
	case "OPTIONS":
		c.RespondWithOptions([]string{"OPTIONS", "HEAD", "GET", "PUT", "PATCH", "DELETE"})
		return
	case "HEAD":
		ctl.Read(c)
	case "GET":
		ctl.Read(c)
	case "PUT":
		ctl.Update(c)
	case "PATCH":
		ctl.Patch(c)
	case "DELETE":
		ctl.Delete(c)
	default:
		c.RespondWithStatus(http.StatusMethodNotAllowed)
		return
	}
}

type MicrocosmController struct{}

func (ctl *MicrocosmController) Read(c *models.Context) {
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

	// Get Microcosm
	m, status, err := models.GetMicrocosm(c.Site.ID, itemId, c.Auth.ProfileId)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	// Get Items
	m.Items, status, err = models.GetItems(c.Site.ID, m.ID, c.Auth.ProfileId, c.Request.URL)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}
	m.Meta.Permissions = perms

	if c.Auth.ProfileId > 0 {
		// Get watcher status
		watcherId, sendEmail, sendSms, ignored, status, err := models.GetWatcherAndIgnoreStatus(
			h.ItemTypes[h.ItemTypeMicrocosm], m.ID, c.Auth.ProfileId,
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
			m.Meta.Flags.SendSms = sendSms
		}
	}

	c.RespondWithData(m)
}

func (ctl *MicrocosmController) Update(c *models.Context) {
	_, itemTypeId, itemId, status, err := c.GetItemTypeAndItemId()
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	// Validate inputs
	m, status, err := models.GetMicrocosm(c.Site.ID, itemId, c.Auth.ProfileId)
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

	// Populate where applicable from auth and context
	m.SiteID = c.Site.ID
	m.Meta.EditedByNullable = sql.NullInt64{Int64: c.Auth.ProfileId, Valid: true}
	m.Meta.EditedNullable = pq.NullTime{Time: time.Now(), Valid: true}

	status, err = m.Update()
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	audit.Replace(
		c.Site.ID,
		h.ItemTypes[h.ItemTypeMicrocosm],
		m.ID,
		c.Auth.ProfileId,
		time.Now(),
		c.IP,
	)

	c.RespondWithSeeOther(
		fmt.Sprintf(
			"%s/%d",
			h.ApiTypeMicrocosm,
			m.ID,
		),
	)
}

func (ctl *MicrocosmController) Patch(c *models.Context) {
	_, itemTypeId, itemId, status, err := c.GetItemTypeAndItemId()
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
	ac := models.MakeAuthorisationContext(c, 0, itemTypeId, itemId)
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
				c.RespondWithErrorMessage("/meta/flags/sticky requires a bool value", http.StatusBadRequest)
				return
			}
			if (patch.Bool.Bool == false && !(perms.IsModerator || perms.IsOwner)) || !perms.IsModerator {
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

	// Populate where applicable from auth and context
	m, status, err := models.GetMicrocosm(c.Site.ID, itemId, c.Auth.ProfileId)
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
		h.ItemTypes[h.ItemTypeMicrocosm],
		m.ID,
		c.Auth.ProfileId,
		time.Now(),
		c.IP,
	)

	c.RespondWithOK()
}

func (ctl *MicrocosmController) Delete(c *models.Context) {
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

	m, status, err := models.GetMicrocosm(c.Site.ID, itemId, c.Auth.ProfileId)
	if err != nil {
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
		h.ItemTypes[h.ItemTypeMicrocosm],
		m.ID,
		c.Auth.ProfileId,
		time.Now(),
		c.IP,
	)

	c.RespondWithOK()
}
