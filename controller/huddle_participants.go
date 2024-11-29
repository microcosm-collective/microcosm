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

// HuddleParticipantsController is a web controller
type HuddleParticipantsController struct{}

// HuddleParticipantsHandler is a web handler
func HuddleParticipantsHandler(w http.ResponseWriter, r *http.Request) {
	path := "/huddles/{id}/participants"
	pyroscope.TagWrapper(context.Background(), pyroscope.Labels("path", path), func(ctx context.Context) {
		c, status, err := models.MakeContext(r, w)
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}
		ctl := HuddleParticipantsController{}

		method := c.GetHTTPMethod()
		switch method {
		case "OPTIONS":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				c.RespondWithOptions([]string{"OPTIONS", "GET", "HEAD", "POST", "PUT"})
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
		default:
			c.RespondWithStatus(http.StatusMethodNotAllowed)
			return
		}
	})
}

// ReadMany handles GET for the collection
func (ctl *HuddleParticipantsController) ReadMany(c *models.Context) {
	// Validate inputs
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

	limit, offset, status, err := h.GetLimitAndOffset(c.Request.URL.Query())
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	ems, total, pages, status, err := models.GetHuddleParticipants(c.Site.ID, huddleID, limit, offset)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	// Construct the response
	m := models.HuddleParticipantsType{}
	m.HuddleParticipants = h.ConstructArray(
		ems,
		fmt.Sprintf("%s/participants", r.GetLink()),
		total,
		limit,
		offset,
		pages,
		c.Request.URL,
	)

	c.RespondWithData(m)
}

// UpdateMany handles PUT for the collection
func (ctl *HuddleParticipantsController) UpdateMany(c *models.Context) {
	// Validate inputs
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

	// Start Authorisation
	perms := models.GetPermission(
		models.MakeAuthorisationContext(
			c, 0, h.ItemTypes[h.ItemTypeHuddle], huddleID),
	)
	if !perms.CanUpdate {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}

	if r.IsConfidential {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}
	// End Authorisation

	ems := []models.HuddleParticipantType{}

	err = c.Fill(&ems)
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("The post data is invalid: %v", err.Error()),
			http.StatusBadRequest,
		)
		return
	}

	status, err = models.UpdateManyHuddleParticipants(c.Site.ID, huddleID, ems)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	c.RespondWithSeeOther(
		fmt.Sprintf(
			"%s/%d",
			h.APITypeHuddle,
			huddleID,
		),
	)
}
