package controller

import (
	"net/http"
	"strconv"

	h "git.dee.kitchen/buro9/microcosm/helpers"
	"git.dee.kitchen/buro9/microcosm/models"
)

// RoleProfileController is a web controller
type RoleProfileController struct{}

// RoleProfileHandler is a web handler
func RoleProfileHandler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}
	ctl := RoleProfileController{}

	method := c.GetHTTPMethod()
	switch method {
	case "OPTIONS":
		c.RespondWithOptions([]string{"OPTIONS", "GET", "HEAD", "POST", "PUT", "DELETE"})
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

// Read handles GET
// Returns information on a profile assigned to this role
func (ctl *RoleProfileController) Read(c *models.Context) {
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

	profileID, err := strconv.ParseInt(c.RouteVars["profile_id"], 10, 64)
	if err != nil {
		c.RespondWithErrorMessage("profile_id in URL is not a number", http.StatusBadRequest)
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

	m, status, err := models.GetRoleProfile(c.Site.ID, roleID, profileID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	c.RespondWithData(m)
}

// Update handles PUT
// Explicitly associates a profile to this role
func (ctl *RoleProfileController) Update(c *models.Context) {
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

	profileID, err := strconv.ParseInt(c.RouteVars["profile_id"], 10, 64)
	if err != nil {
		c.RespondWithErrorMessage("profile_id in URL is not a number", http.StatusBadRequest)
		return
	}

	m := models.RoleProfileType{}
	m.ID = profileID

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

	status, err = m.Update(c.Site.ID, roleID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	c.RespondWithSeeOther(m.GetLink(r.GetLink()))
}

// Delete handles DELETE
// Note: This only affects explicitly assigned roles and not roles implicitly
// included by criteria
func (ctl *RoleProfileController) Delete(c *models.Context) {
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

	profileID, err := strconv.ParseInt(c.RouteVars["profile_id"], 10, 64)
	if err != nil {
		c.RespondWithErrorMessage("profile_id in URL is not a number", http.StatusBadRequest)
		return
	}

	m := models.RoleProfileType{}
	m.ID = profileID

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
