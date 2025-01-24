package controller

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/grafana/pyroscope-go"
	h "github.com/microcosm-collective/microcosm/helpers"
	"github.com/microcosm-collective/microcosm/models"
)

// RoleProfilesController is a web controller
type RoleProfilesController struct{}

// RoleProfilesHandler is a web handler
func RoleProfilesHandler(w http.ResponseWriter, r *http.Request) {
	path := "/roles/{id}/profiles"
	pyroscope.TagWrapper(context.Background(), pyroscope.Labels("path", path), func(ctx context.Context) {

		c, status, err := models.MakeContext(r, w)
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}
		ctl := RoleProfilesController{}

		method := c.GetHTTPMethod()
		switch method {
		case "OPTIONS":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				c.RespondWithOptions([]string{"OPTIONS", "GET", "HEAD", "POST", "PUT", "DELETE"})
			})
			return
		case "GET":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				ctl.ReadMany(c)
			})
		case "HEAD":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				ctl.ReadMany(c)
			})
		case "PUT":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				ctl.UpdateMany(c)
			})
		case "DELETE":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				ctl.DeleteMany(c)
			})
		default:
			c.RespondWithStatus(http.StatusMethodNotAllowed)
			return
		}
	})
}

// ReadMany handles GET for the collection
// Returns an array of all of the profiles explicitly assigned to this role
func (ctl *RoleProfilesController) ReadMany(c *models.Context) {
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
		c.RespondWithErrorMessage("role_id in URL is not a number", http.StatusBadRequest)
		return
	}

	r, status, err := models.GetRole(c.Site.ID, microcosmID, roleID, c.Auth.ProfileID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
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

	limit, offset, status, err := h.GetLimitAndOffset(c.Request.URL.Query())
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	ems, total, pages, status, err := models.GetRoleProfiles(c.Site.ID, roleID, limit, offset)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	// Construct the response
	m := models.RoleProfilesType{}
	m.RoleProfiles = h.ConstructArray(
		ems,
		fmt.Sprintf("%s/profiles", r.GetLink()),
		total,
		limit,
		offset,
		pages,
		c.Request.URL,
	)

	c.RespondWithData(m)
}

// UpdateMany handles PUT on the collection
// Assigns one or more profiles explicitly to this role
func (ctl *RoleProfilesController) UpdateMany(c *models.Context) {
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
		c.RespondWithErrorMessage("role_id in URL is not a number", http.StatusBadRequest)
		return
	}

	_, status, err := models.GetRole(c.Site.ID, microcosmID, roleID, c.Auth.ProfileID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	ems := []models.RoleProfileType{}

	err = c.Fill(&ems)
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

	status, err = models.UpdateManyRoleProfiles(c.Site.ID, roleID, ems)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	c.RespondWithOK()
}

// DeleteMany handles DELETE
// Removes one or more profiles (that were explicitly assigned) from this role
func (ctl *RoleProfilesController) DeleteMany(c *models.Context) {
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
		c.RespondWithErrorMessage("role_id in URL is not a number", http.StatusBadRequest)
		return
	}

	_, status, err := models.GetRole(c.Site.ID, microcosmID, roleID, c.Auth.ProfileID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	ems := []models.RoleProfileType{}

	err = c.Fill(&ems)
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

	status, err = models.DeleteManyRoleProfiles(roleID, ems)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	c.RespondWithOK()
}
