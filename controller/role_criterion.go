package controller

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	h "git.dee.kitchen/buro9/microcosm/helpers"
	"git.dee.kitchen/buro9/microcosm/models"
	"github.com/grafana/pyroscope-go"
)

// RoleCriterionController is a web controller
type RoleCriterionController struct{}

// RoleCriterionHandler is a web handler
func RoleCriterionHandler(w http.ResponseWriter, r *http.Request) {
	path := "/roles/{id}/criteria/{id}"
	pyroscope.TagWrapper(context.Background(), pyroscope.Labels("path", path), func(ctx context.Context) {

		c, status, err := models.MakeContext(r, w)
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}
		ctl := RoleCriterionController{}

		method := c.GetHTTPMethod()
		switch method {
		case "OPTIONS":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				c.RespondWithOptions([]string{"OPTIONS", "GET", "HEAD", "PUT", "DELETE"})
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
// Returns a single criterion that is used as part of the filter for this role
func (ctl *RoleCriterionController) Read(c *models.Context) {
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

	criterionID, err := strconv.ParseInt(c.RouteVars["criterion_id"], 10, 64)
	if err != nil {
		c.RespondWithErrorMessage("criterion_id in URL is not a number", http.StatusBadRequest)
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

	m, status, err := models.GetRoleCriterion(criterionID, roleID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	c.RespondWithData(m)
}

// Update handles PUT
func (ctl *RoleCriterionController) Update(c *models.Context) {
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

	criterionID, err := strconv.ParseInt(c.RouteVars["criterion_id"], 10, 64)
	if err != nil {
		c.RespondWithErrorMessage("criterion_id in URL is not a number", http.StatusBadRequest)
		return
	}

	m, status, err := models.GetRoleCriterion(criterionID, roleID)
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

	status, err = m.Update(roleID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	c.RespondWithSeeOther(m.GetLink(r.GetLink()))
}

// Delete handles DELETE
func (ctl *RoleCriterionController) Delete(c *models.Context) {
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

	criterionID, err := strconv.ParseInt(c.RouteVars["criterion_id"], 10, 64)
	if err != nil {
		c.RespondWithErrorMessage("criterion_id in URL is not a number", http.StatusBadRequest)
		return
	}

	m, status, err := models.GetRoleCriterion(criterionID, roleID)
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

	status, err = m.Delete(roleID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	c.RespondWithOK()
}
