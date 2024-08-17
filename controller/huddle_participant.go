package controller

import (
	"net/http"
	"strconv"

	h "git.dee.kitchen/buro9/microcosm/helpers"
	"git.dee.kitchen/buro9/microcosm/models"
)

// HuddleParticipantController is a web controller
type HuddleParticipantController struct{}

// HuddleParticipantHandler is a web handler
func HuddleParticipantHandler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}
	ctl := HuddleParticipantController{}

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
func (ctl *HuddleParticipantController) Read(c *models.Context) {
	huddleID, err := strconv.ParseInt(c.RouteVars["huddle_id"], 10, 64)
	if err != nil {
		c.RespondWithErrorMessage("huddle_id in URL is not a number", http.StatusBadRequest)
		return
	}

	_, status, err := models.GetHuddle(c.Site.ID, c.Auth.ProfileID, huddleID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	// Start Authorisation
	perms := models.GetPermission(
		models.MakeAuthorisationContext(
			c, 0, h.ItemTypes[h.ItemTypeHuddle], huddleID),
	)
	if !perms.CanRead {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}
	// End Authorisation

	profileID, err := strconv.ParseInt(c.RouteVars["profile_id"], 10, 64)
	if err != nil {
		c.RespondWithErrorMessage("profile_id in URL is not a number", http.StatusBadRequest)
		return
	}

	m, status, err := models.GetHuddleParticipant(c.Site.ID, huddleID, profileID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	c.RespondWithData(m)
}

// Update handles PUT
func (ctl *HuddleParticipantController) Update(c *models.Context) {
	huddleID, err := strconv.ParseInt(c.RouteVars["huddle_id"], 10, 64)
	if err != nil {
		c.RespondWithErrorMessage("huddle_id in URL is not a number", http.StatusBadRequest)
		return
	}

	r, status, err := models.GetHuddle(c.Site.ID, c.Auth.ProfileID, huddleID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	profileID, err := strconv.ParseInt(c.RouteVars["profile_id"], 10, 64)
	if err != nil {
		c.RespondWithErrorMessage("profile_id in URL is not a number", http.StatusBadRequest)
		return
	}

	// Start Authorisation
	perms := models.GetPermission(
		models.MakeAuthorisationContext(
			c, 0, h.ItemTypes[h.ItemTypeHuddle], huddleID),
	)
	if !perms.CanUpdate {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}

	if !r.IsConfidential {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}
	// End Authorisation

	m := models.HuddleParticipantType{}
	m.ID = profileID

	status, err = m.Update(c.Site.ID, huddleID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	c.RespondWithSeeOther(m.GetLink(r.GetLink()))
}

// Delete handles DELETE
func (ctl *HuddleParticipantController) Delete(c *models.Context) {
	huddleID, err := strconv.ParseInt(c.RouteVars["huddle_id"], 10, 64)
	if err != nil {
		c.RespondWithErrorMessage("huddle_id in URL is not a number", http.StatusBadRequest)
		return
	}

	_, status, err := models.GetHuddle(c.Site.ID, c.Auth.ProfileID, huddleID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	profileID, err := strconv.ParseInt(c.RouteVars["profile_id"], 10, 64)
	if err != nil {
		c.RespondWithErrorMessage("profile_id in URL is not a number", http.StatusBadRequest)
		return
	}

	// Start Authorisation
	perms := models.GetPermission(
		models.MakeAuthorisationContext(
			c, 0, h.ItemTypes[h.ItemTypeHuddle], huddleID),
	)
	if !perms.CanDelete {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}
	// End Authorisation

	if profileID != c.Auth.ProfileID {
		c.RespondWithErrorMessage("Only the participant in question can remove a participant from a huddle", http.StatusBadRequest)
		return
	}

	m := models.HuddleParticipantType{}
	m.ID = profileID

	status, err = m.Delete(huddleID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	c.RespondWithOK()
}
