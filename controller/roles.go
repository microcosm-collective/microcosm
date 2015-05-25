package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/microcosm-cc/microcosm/audit"
	h "github.com/microcosm-cc/microcosm/helpers"
	"github.com/microcosm-cc/microcosm/models"
)

// RolesController is a web controller
type RolesController struct{}

// RolesHandler is a web handler
func RolesHandler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	ctl := RolesController{}

	switch c.GetHTTPMethod() {
	case "OPTIONS":
		c.RespondWithOptions([]string{"OPTIONS", "GET", "HEAD", "POST"})
		return
	case "GET":
		ctl.ReadMany(c)
	case "HEAD":
		ctl.ReadMany(c)
	case "POST":
		ctl.Create(c)
	default:
		c.RespondWithStatus(http.StatusMethodNotAllowed)
		return
	}
}

// ReadMany handles GET
// If microcosm_id is provided in request args then these are the roles for this
// microcosm, otherwise this is a list of the default roles on this site
func (ctl *RolesController) ReadMany(c *models.Context) {
	var microcosmID int64
	if sid, exists := c.RouteVars["microcosm_id"]; exists {
		id, err := strconv.ParseInt(sid, 10, 64)
		if err != nil {
			c.RespondWithErrorMessage("microcosm_id in URL is not a number", http.StatusBadRequest)
			return
		}

		microcosmID = id
	}

	perms := models.GetPermission(
		models.MakeAuthorisationContext(c, 0, h.ItemTypes[h.ItemTypeMicrocosm], microcosmID),
	)
	if microcosmID > 0 {
		// Related to a Microcosm
		if !perms.CanRead {
			c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
			return
		}
	} else {
		// Default role for the site
		if !c.Auth.IsSiteOwner {
			c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
			return
		}
	}

	// Fetch query string args if any exist
	limit, offset, status, err := h.GetLimitAndOffset(c.Request.URL.Query())
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	ems, total, pages, status, err := models.GetRoles(c.Site.ID, microcosmID, c.Auth.ProfileID, limit, offset)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	// Construct the response
	m := models.RolesType{}
	m.Roles = h.ConstructArray(
		ems,
		h.APITypeRole,
		total,
		limit,
		offset,
		pages,
		c.Request.URL,
	)

	c.RespondWithData(m)
}

// Create handles POST
// If microcosm_id is provided in request args then this is a role on a microcosm
// otherwise this is a default role on this site
func (ctl *RolesController) Create(c *models.Context) {

	var microcosmID int64
	if sid, exists := c.RouteVars["microcosm_id"]; exists {
		id, err := strconv.ParseInt(sid, 10, 64)
		if err != nil {
			c.RespondWithErrorMessage("microcosm_id in URL is not a number", http.StatusBadRequest)
			return
		}

		microcosmID = id
	}

	// Validate inputs
	m := models.RoleType{}
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
		models.MakeAuthorisationContext(c, microcosmID, h.ItemTypes[h.ItemTypeMicrocosm], microcosmID),
	)
	if microcosmID > 0 {
		// Related to a Microcosm
		if !perms.IsModerator && !c.Auth.IsSiteOwner {
			c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
			return
		}
	} else {
		// Default role for the site
		if !c.Auth.IsSiteOwner {
			c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
			return
		}
	}
	// End : Authorisation

	// Populate where applicable from auth and context
	m.SiteID = c.Site.ID
	m.MicrocosmID = microcosmID
	m.Meta.CreatedByID = c.Auth.ProfileID
	m.Meta.Created = time.Now()

	status, err := m.Insert(c.Site.ID, c.Auth.ProfileID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	audit.Create(
		c.Site.ID,
		h.ItemTypes[h.ItemTypeRole],
		m.ID,
		c.Auth.ProfileID,
		time.Now(),
		c.IP,
	)

	c.RespondWithSeeOther(m.GetLink())
}
