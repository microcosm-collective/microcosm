package controller

import (
	"fmt"
	"net/http"
	"time"

	"github.com/microcosm-cc/microcosm/audit"
	h "github.com/microcosm-cc/microcosm/helpers"
	"github.com/microcosm-cc/microcosm/models"
)

// MicrocosmsHandler is a web handler
func MicrocosmsHandler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	ctl := MicrocosmsController{}

	switch c.GetHTTPMethod() {
	case "OPTIONS":
		c.RespondWithOptions([]string{"OPTIONS", "POST", "HEAD", "GET"})
		return
	case "POST":
		ctl.Create(c)
	case "HEAD":
		ctl.ReadMany(c)
	case "GET":
		ctl.ReadMany(c)
	default:
		c.RespondWithStatus(http.StatusMethodNotAllowed)
		return
	}
}

// MicrocosmsController is a web controller
type MicrocosmsController struct{}

// Create handles GET
func (ctl *MicrocosmsController) Create(c *models.Context) {
	// Validate inputs
	m := models.MicrocosmType{}
	err := c.Fill(&m)
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("The post data is invalid: %v", err.Error()),
			http.StatusBadRequest,
		)
		return
	}

	// Start : Authorisation
	perms := models.GetPermission(
		models.MakeAuthorisationContext(c, 0, h.ItemTypes[h.ItemTypeSite], c.Site.ID),
	)
	if !perms.CanCreate {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}
	// End : Authorisation

	// Populate where applicable from auth and context
	m.SiteID = c.Site.ID
	m.Meta.CreatedByID = c.Auth.ProfileID
	m.Meta.Created = time.Now()
	m.OwnedByID = c.Auth.ProfileID

	status, err := m.Insert()
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	audit.Create(
		c.Site.ID,
		h.ItemTypes[h.ItemTypeMicrocosm],
		m.ID,
		c.Auth.ProfileID,
		time.Now(),
		c.IP,
	)

	c.RespondWithSeeOther(
		fmt.Sprintf(
			"%s/%d",
			h.APITypeMicrocosm,
			m.ID,
		),
	)
}

// ReadMany handles GET
func (ctl *MicrocosmsController) ReadMany(c *models.Context) {
	rootMicrocosmID := models.GetRootMicrocosmID(c.Site.ID)

	perms := models.GetPermission(
		models.MakeAuthorisationContext(
			c,
			rootMicrocosmID,
			h.ItemTypes[h.ItemTypeMicrocosm],
			rootMicrocosmID,
		),
	)
	if !perms.CanRead {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}

	// Get Microcosm
	m, status, err := models.GetMicrocosm(
		c.Site.ID,
		rootMicrocosmID,
		c.Auth.ProfileID,
	)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	// Get Items
	m.Items, status, err = models.GetItems(
		c.Site.ID,
		m.ID,
		c.Auth.ProfileID,
		c.Request.URL,
	)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}
	m.Meta.Permissions = perms

	if c.Auth.ProfileID > 0 {
		// Get watcher status
		watcherID, sendEmail, sendSms, ignored, status, err :=
			models.GetWatcherAndIgnoreStatus(
				h.ItemTypes[h.ItemTypeMicrocosm], m.ID, c.Auth.ProfileID,
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
