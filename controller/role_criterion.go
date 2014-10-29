package controller

import (
	"fmt"
	"net/http"
	"strconv"

	h "github.com/microcosm-cc/microcosm/helpers"
	"github.com/microcosm-cc/microcosm/models"
)

type RoleCriterionController struct{}

func RoleCriterionHandler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	ctl := RoleCriterionController{}

	switch c.GetHttpMethod() {
	case "OPTIONS":
		c.RespondWithOptions([]string{"OPTIONS", "GET", "HEAD", "PUT", "DELETE"})
		return
	case "GET":
		ctl.Read(c)
	case "HEAD":
		ctl.Read(c)
	case "PUT":
		ctl.Update(c)
	case "DELETE":
		ctl.Delete(c)
	default:
		c.RespondWithStatus(http.StatusMethodNotAllowed)
		return
	}
}

// Returns a single criterion that is used as part of the filter for this role
func (ctl *RoleCriterionController) Read(c *models.Context) {
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
		c.RespondWithErrorMessage("role_id in URL is not a number", http.StatusBadRequest)
		return
	}

	_, status, err := models.GetRole(c.Site.Id, microcosmId, roleId, c.Auth.ProfileId)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	criterionId, err := strconv.ParseInt(c.RouteVars["criterion_id"], 10, 64)
	if err != nil {
		c.RespondWithErrorMessage("criterion_id in URL is not a number", http.StatusBadRequest)
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

	m, status, err := models.GetRoleCriterion(criterionId, roleId)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	c.RespondWithData(m)
}

// Updates (replaces) the criterion
func (ctl *RoleCriterionController) Update(c *models.Context) {
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
		c.RespondWithErrorMessage("role_id in URL is not a number", http.StatusBadRequest)
		return
	}

	r, status, err := models.GetRole(c.Site.Id, microcosmId, roleId, c.Auth.ProfileId)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	criterionId, err := strconv.ParseInt(c.RouteVars["criterion_id"], 10, 64)
	if err != nil {
		c.RespondWithErrorMessage("criterion_id in URL is not a number", http.StatusBadRequest)
		return
	}

	m, status, err := models.GetRoleCriterion(criterionId, roleId)
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

	status, err = m.Update(roleId)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	c.RespondWithSeeOther(m.GetLink(r.GetLink()))
}

// Deletes the criterion
func (ctl *RoleCriterionController) Delete(c *models.Context) {
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
		c.RespondWithErrorMessage("role_id in URL is not a number", http.StatusBadRequest)
		return
	}

	_, status, err := models.GetRole(c.Site.Id, microcosmId, roleId, c.Auth.ProfileId)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	criterionId, err := strconv.ParseInt(c.RouteVars["criterion_id"], 10, 64)
	if err != nil {
		c.RespondWithErrorMessage("criterion_id in URL is not a number", http.StatusBadRequest)
		return
	}

	m, status, err := models.GetRoleCriterion(criterionId, roleId)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
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

	status, err = m.Delete(roleId)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	c.RespondWithOK()
}
