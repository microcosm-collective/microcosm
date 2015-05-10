package controller

import (
	"net/http"
	"strconv"

	h "github.com/microcosm-cc/microcosm/helpers"
	"github.com/microcosm-cc/microcosm/models"
)

type HuddleParticipantController struct{}

func HuddleParticipantHandler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	ctl := HuddleParticipantController{}

	switch c.GetHttpMethod() {
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

// Returns information on a profile assigned to this huddle
func (ctl *HuddleParticipantController) Read(c *models.Context) {
	huddleId, err := strconv.ParseInt(c.RouteVars["huddle_id"], 10, 64)
	if err != nil {
		c.RespondWithErrorMessage("huddle_id in URL is not a number", http.StatusBadRequest)
		return
	}

	_, status, err := models.GetHuddle(c.Site.ID, c.Auth.ProfileId, huddleId)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	// Start Authorisation
	perms := models.GetPermission(
		models.MakeAuthorisationContext(
			c, 0, h.ItemTypes[h.ItemTypeHuddle], huddleId),
	)
	if !perms.CanRead {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}
	// End Authorisation

	profileId, err := strconv.ParseInt(c.RouteVars["profile_id"], 10, 64)
	if err != nil {
		c.RespondWithErrorMessage("profile_id in URL is not a number", http.StatusBadRequest)
		return
	}

	m, status, err := models.GetHuddleParticipant(c.Site.ID, huddleId, profileId)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	c.RespondWithData(m)
}

// Explicitly associates a profile to this huddle
func (ctl *HuddleParticipantController) Update(c *models.Context) {
	huddleId, err := strconv.ParseInt(c.RouteVars["huddle_id"], 10, 64)
	if err != nil {
		c.RespondWithErrorMessage("huddle_id in URL is not a number", http.StatusBadRequest)
		return
	}

	r, status, err := models.GetHuddle(c.Site.ID, c.Auth.ProfileId, huddleId)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	profileId, err := strconv.ParseInt(c.RouteVars["profile_id"], 10, 64)
	if err != nil {
		c.RespondWithErrorMessage("profile_id in URL is not a number", http.StatusBadRequest)
		return
	}

	// Start Authorisation
	perms := models.GetPermission(
		models.MakeAuthorisationContext(
			c, 0, h.ItemTypes[h.ItemTypeHuddle], huddleId),
	)
	if !perms.CanUpdate {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}

	if r.IsConfidential == false {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}
	// End Authorisation

	m := models.HuddleParticipantType{}
	m.Id = profileId

	status, err = m.Update(c.Site.ID, huddleId)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	c.RespondWithSeeOther(m.GetLink(r.GetLink()))
}

// Deletes a profile from the huddle
func (ctl *HuddleParticipantController) Delete(c *models.Context) {

	huddleId, err := strconv.ParseInt(c.RouteVars["huddle_id"], 10, 64)
	if err != nil {
		c.RespondWithErrorMessage("huddle_id in URL is not a number", http.StatusBadRequest)
		return
	}

	_, status, err := models.GetHuddle(c.Site.ID, c.Auth.ProfileId, huddleId)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	profileId, err := strconv.ParseInt(c.RouteVars["profile_id"], 10, 64)
	if err != nil {
		c.RespondWithErrorMessage("profile_id in URL is not a number", http.StatusBadRequest)
		return
	}

	// Start Authorisation
	perms := models.GetPermission(
		models.MakeAuthorisationContext(
			c, 0, h.ItemTypes[h.ItemTypeHuddle], huddleId),
	)
	if !perms.CanDelete {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}
	// End Authorisation

	if profileId != c.Auth.ProfileId {
		c.RespondWithErrorMessage("Only the participant in question can remove a participant from a huddle", http.StatusBadRequest)
		return
	}

	m := models.HuddleParticipantType{}
	m.Id = profileId

	status, err = m.Delete(huddleId)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	c.RespondWithOK()
}
