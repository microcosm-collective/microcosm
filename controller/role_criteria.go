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

// RoleCriteriaController is a web controller
type RoleCriteriaController struct{}

// RoleCriteriaHandler is a web handler
func RoleCriteriaHandler(w http.ResponseWriter, r *http.Request) {
	path := "/roles/{id}/criteria"
	pyroscope.TagWrapper(context.Background(), pyroscope.Labels("path", path), func(ctx context.Context) {
		c, status, err := models.MakeContext(r, w)
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}
		ctl := RoleCriteriaController{}

		method := c.GetHTTPMethod()
		switch method {
		case "OPTIONS":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				c.RespondWithOptions([]string{"OPTIONS", "GET", "HEAD", "POST", "DELETE"})
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
		case "POST":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				ctl.Create(c)
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
func (ctl *RoleCriteriaController) ReadMany(c *models.Context) {
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

	ems, total, pages, status, err := models.GetRoleCriteria(roleID, limit, offset)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	// Construct the response
	m := models.RoleCriteriaType{}
	m.RoleCriteria = h.ConstructArray(
		ems,
		fmt.Sprintf("%s/criteria", r.GetLink()),
		total,
		limit,
		offset,
		pages,
		c.Request.URL,
	)

	c.RespondWithData(m)
}

// Create handles POST
func (ctl *RoleCriteriaController) Create(c *models.Context) {
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

	m := models.RoleCriterionType{}

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

	status, err = m.Insert(roleID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	c.RespondWithSeeOther(m.GetLink(r.GetLink()))
}

// DeleteMany handles DELETE for the collection
func (ctl *RoleCriteriaController) DeleteMany(c *models.Context) {
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

	ems := []models.RoleCriterionType{}

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

	status, err = models.DeleteManyRoleCriteria(roleID, ems)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	c.RespondWithOK()
}
