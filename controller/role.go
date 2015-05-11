package controller

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/lib/pq"

	"github.com/microcosm-cc/microcosm/audit"
	h "github.com/microcosm-cc/microcosm/helpers"
	"github.com/microcosm-cc/microcosm/models"
)

type RoleController struct{}

func RoleHandler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	ctl := RoleController{}

	switch c.GetHttpMethod() {
	case "OPTIONS":
		c.RespondWithOptions([]string{"OPTIONS", "GET", "HEAD", "PUT", "PATCH", "DELETE"})
		return
	case "GET":
		ctl.Read(c)
	case "HEAD":
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

// Returns the information about a single role
func (ctl *RoleController) Read(c *models.Context) {
	// Validate inputs
	var microcosmId int64
	if sid, exists := c.RouteVars["microcosm_id"]; exists {
		id, err := strconv.ParseInt(sid, 10, 64)
		if err != nil {
			c.RespondWithErrorMessage("microcosm_id in URL is not a number", http.StatusBadRequest)
			return
		}

		microcosmId = id
	}

	roleId, err := strconv.ParseInt(c.RouteVars["role_id"], 10, 64)
	if err != nil {
		c.RespondWithErrorMessage("microcosm_id in URL is not a number", http.StatusBadRequest)
		return
	}

	// Authorisation
	perms := models.GetPermission(
		models.MakeAuthorisationContext(c, microcosmId, h.ItemTypes[h.ItemTypeMicrocosm], microcosmId),
	)
	if microcosmId > 0 {
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

	// Get Role
	m, status, err := models.GetRole(c.Site.ID, microcosmId, roleId, c.Auth.ProfileId)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	m.Meta.Permissions = perms

	c.RespondWithData(m)
}

// Updates (replaces) a single role
func (ctl *RoleController) Update(c *models.Context) {
	// Validate inputs
	var microcosmId int64
	if sid, exists := c.RouteVars["microcosm_id"]; exists {
		id, err := strconv.ParseInt(sid, 10, 64)
		if err != nil {
			c.RespondWithErrorMessage("microcosm_id in URL is not a number", http.StatusBadRequest)
			return
		}

		microcosmId = id
	}

	roleId, err := strconv.ParseInt(c.RouteVars["role_id"], 10, 64)
	if err != nil {
		c.RespondWithErrorMessage("microcosm_id in URL is not a number", http.StatusBadRequest)
		return
	}

	m, status, err := models.GetRole(c.Site.ID, microcosmId, roleId, c.Auth.ProfileId)
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

	// Authorisation
	perms := models.GetPermission(
		models.MakeAuthorisationContext(c, microcosmId, h.ItemTypes[h.ItemTypeMicrocosm], microcosmId),
	)
	if microcosmId > 0 {
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

	// Populate where applicable from auth and context
	m.Meta.EditedByNullable = sql.NullInt64{Int64: c.Auth.ProfileId, Valid: true}
	m.Meta.EditedNullable = pq.NullTime{Time: time.Now(), Valid: true}

	status, err = m.Update(c.Site.ID, c.Auth.ProfileId)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	audit.Replace(
		c.Site.ID,
		h.ItemTypes[h.ItemTypeRole],
		m.ID,
		c.Auth.ProfileId,
		time.Now(),
		c.IP,
	)

	c.RespondWithSeeOther(m.GetLink())
}

// Allows the boolean properties of the role to be altered without updating the
// entire role
func (ctl *RoleController) Patch(c *models.Context) {
	// Validate inputs
	var microcosmId int64
	if sid, exists := c.RouteVars["microcosm_id"]; exists {
		id, err := strconv.ParseInt(sid, 10, 64)
		if err != nil {
			c.RespondWithErrorMessage("microcosm_id in URL is not a number", http.StatusBadRequest)
			return
		}

		microcosmId = id
	}

	roleId, err := strconv.ParseInt(c.RouteVars["role_id"], 10, 64)
	if err != nil {
		c.RespondWithErrorMessage("microcosm_id in URL is not a number", http.StatusBadRequest)
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

	status, err := h.TestPatch(patches)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	// Start Authorisation
	ac := models.MakeAuthorisationContext(c, microcosmId, h.ItemTypes[h.ItemTypeMicrocosm], microcosmId)
	perms := models.GetPermission(ac)
	if microcosmId > 0 {
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

	// All patches are 'replace'
	for _, patch := range patches {
		status, err := patch.ScanRawValue()
		if !patch.Bool.Valid {
			c.RespondWithErrorDetail(err, status)
			return
		}

		switch patch.Path {
		case "/moderator":
			if !patch.Bool.Valid {
				c.RespondWithErrorMessage("/moderator requires a bool value", http.StatusBadRequest)
				return
			}
		case "/banned":
			if !patch.Bool.Valid {
				c.RespondWithErrorMessage("/banned requires a bool value", http.StatusBadRequest)
				return
			}
		case "/read":
			if !patch.Bool.Valid {
				c.RespondWithErrorMessage("/read requires a bool value", http.StatusBadRequest)
				return
			}
		case "/create":
			if !patch.Bool.Valid {
				c.RespondWithErrorMessage("/create requires a bool value", http.StatusBadRequest)
				return
			}
		case "/update":
			if !patch.Bool.Valid {
				c.RespondWithErrorMessage("/update requires a bool value", http.StatusBadRequest)
				return
			}
		case "/delete":
			if !patch.Bool.Valid {
				c.RespondWithErrorMessage("/delete requires a bool value", http.StatusBadRequest)
				return
			}
		case "/closeOwn":
			if !patch.Bool.Valid {
				c.RespondWithErrorMessage("/closeOwn requires a bool value", http.StatusBadRequest)
				return
			}
		case "/openOwn":
			if !patch.Bool.Valid {
				c.RespondWithErrorMessage("/openOwn requires a bool value", http.StatusBadRequest)
				return
			}
		case "/readOthers":
			if !patch.Bool.Valid {
				c.RespondWithErrorMessage("/readOthers requires a bool value", http.StatusBadRequest)
				return
			}
		default:
			c.RespondWithErrorMessage("Invalid patch operation path", http.StatusBadRequest)
			return
		}
	}
	// End Authorisation

	m, status, err := models.GetRole(c.Site.ID, microcosmId, roleId, c.Auth.ProfileId)
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
		h.ItemTypes[h.ItemTypeRole],
		m.ID,
		c.Auth.ProfileId,
		time.Now(),
		c.IP,
	)

	c.RespondWithOK()
}

// Deletes this role
func (ctl *RoleController) Delete(c *models.Context) {
	// Validate inputs
	var microcosmId int64
	if sid, exists := c.RouteVars["microcosm_id"]; exists {
		id, err := strconv.ParseInt(sid, 10, 64)
		if err != nil {
			c.RespondWithErrorMessage("microcosm_id in URL is not a number", http.StatusBadRequest)
			return
		}

		microcosmId = id
	}

	roleId, err := strconv.ParseInt(c.RouteVars["role_id"], 10, 64)
	if err != nil {
		c.RespondWithErrorMessage("microcosm_id in URL is not a number", http.StatusBadRequest)
		return
	}

	// Start Authorisation
	perms := models.GetPermission(
		models.MakeAuthorisationContext(c, microcosmId, h.ItemTypes[h.ItemTypeMicrocosm], microcosmId),
	)
	if microcosmId > 0 {
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

	m, status, err := models.GetRole(c.Site.ID, microcosmId, roleId, c.Auth.ProfileId)
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
		h.ItemTypes[h.ItemTypeRole],
		m.ID,
		c.Auth.ProfileId,
		time.Now(),
		c.IP,
	)

	c.RespondWithOK()
}
