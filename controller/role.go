package controller

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/grafana/pyroscope-go"
	"github.com/lib/pq"

	"git.dee.kitchen/buro9/microcosm/audit"
	h "git.dee.kitchen/buro9/microcosm/helpers"
	"git.dee.kitchen/buro9/microcosm/models"
)

// RoleController is a web controller
type RoleController struct{}

// RoleHandler is a web handler
func RoleHandler(w http.ResponseWriter, r *http.Request) {
	path := "/roles/{id}"
	pyroscope.TagWrapper(context.Background(), pyroscope.Labels("path", path), func(ctx context.Context) {
		c, status, err := models.MakeContext(r, w)
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}
		ctl := RoleController{}

		method := c.GetHTTPMethod()
		switch method {
		case "OPTIONS":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				c.RespondWithOptions([]string{"OPTIONS", "GET", "HEAD", "PUT", "PATCH", "DELETE"})
			})
			return
		case "GET":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				ctl.Read(c)
			})
		case "HEAD":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				ctl.Read(c)
			})
		case "PUT":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				ctl.Update(c)
			})
		case "PATCH":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				ctl.Patch(c)
			})
		case "DELETE":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				ctl.Delete(c)
			})
		default:
			c.RespondWithStatus(http.StatusMethodNotAllowed)
			return
		}
	})
}

// Read handles GET
func (ctl *RoleController) Read(c *models.Context) {
	// Validate inputs
	var microcosmID int64
	if sid, exists := c.RouteVars["microcosm_id"]; exists {
		id, err := strconv.ParseInt(sid, 10, 64)
		if err != nil {
			c.RespondWithErrorMessage("microcosm_id in URL is not a number", http.StatusBadRequest)
			return
		}

		microcosmID = id
	}

	roleID, err := strconv.ParseInt(c.RouteVars["role_id"], 10, 64)
	if err != nil {
		c.RespondWithErrorMessage("microcosm_id in URL is not a number", http.StatusBadRequest)
		return
	}

	// Authorisation
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

	// Get Role
	m, status, err := models.GetRole(c.Site.ID, microcosmID, roleID, c.Auth.ProfileID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	m.Meta.Permissions = perms

	c.RespondWithData(m)
}

// Update handles PUT
func (ctl *RoleController) Update(c *models.Context) {
	// Validate inputs
	var microcosmID int64
	if sid, exists := c.RouteVars["microcosm_id"]; exists {
		id, err := strconv.ParseInt(sid, 10, 64)
		if err != nil {
			c.RespondWithErrorMessage("microcosm_id in URL is not a number", http.StatusBadRequest)
			return
		}

		microcosmID = id
	}

	roleID, err := strconv.ParseInt(c.RouteVars["role_id"], 10, 64)
	if err != nil {
		c.RespondWithErrorMessage("microcosm_id in URL is not a number", http.StatusBadRequest)
		return
	}

	m, status, err := models.GetRole(c.Site.ID, microcosmID, roleID, c.Auth.ProfileID)
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
		h.ItemTypes[h.ItemTypeRole],
		m.ID,
		c.Auth.ProfileID,
		time.Now(),
		c.IP,
	)

	c.RespondWithSeeOther(m.GetLink())
}

// Patch handles PATCH
func (ctl *RoleController) Patch(c *models.Context) {
	// Validate inputs
	var microcosmID int64
	if sid, exists := c.RouteVars["microcosm_id"]; exists {
		id, err := strconv.ParseInt(sid, 10, 64)
		if err != nil {
			c.RespondWithErrorMessage("microcosm_id in URL is not a number", http.StatusBadRequest)
			return
		}

		microcosmID = id
	}

	roleID, err := strconv.ParseInt(c.RouteVars["role_id"], 10, 64)
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
	ac := models.MakeAuthorisationContext(c, microcosmID, h.ItemTypes[h.ItemTypeMicrocosm], microcosmID)
	perms := models.GetPermission(ac)
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

	m, status, err := models.GetRole(c.Site.ID, microcosmID, roleID, c.Auth.ProfileID)
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
		c.Auth.ProfileID,
		time.Now(),
		c.IP,
	)

	c.RespondWithOK()
}

// Delete handles DELETE
func (ctl *RoleController) Delete(c *models.Context) {
	// Validate inputs
	var microcosmID int64
	if sid, exists := c.RouteVars["microcosm_id"]; exists {
		id, err := strconv.ParseInt(sid, 10, 64)
		if err != nil {
			c.RespondWithErrorMessage("microcosm_id in URL is not a number", http.StatusBadRequest)
			return
		}

		microcosmID = id
	}

	roleID, err := strconv.ParseInt(c.RouteVars["role_id"], 10, 64)
	if err != nil {
		c.RespondWithErrorMessage("microcosm_id in URL is not a number", http.StatusBadRequest)
		return
	}

	// Start Authorisation
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

	m, status, err := models.GetRole(c.Site.ID, microcosmID, roleID, c.Auth.ProfileID)
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
		c.Auth.ProfileID,
		time.Now(),
		c.IP,
	)

	c.RespondWithOK()
}
