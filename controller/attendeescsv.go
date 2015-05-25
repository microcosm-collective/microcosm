package controller

import (
	"fmt"
	"net/http"
	"strconv"

	h "github.com/microcosm-cc/microcosm/helpers"
	"github.com/microcosm-cc/microcosm/models"
)

// AttendeesCSVController is a web controller
type AttendeesCSVController struct{}

// AttendeesCSVHandler is a web handler
func AttendeesCSVHandler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	ctl := AttendeesCSVController{}

	switch c.GetHTTPMethod() {
	case "OPTIONS":
		c.RespondWithOptions([]string{"OPTIONS", "HEAD", "GET"})
		return
	case "HEAD":
		ctl.ReadMany(c)
	case "GET":
		ctl.ReadMany(c)
	default:
		c.RespondWithStatus(http.StatusMethodNotAllowed)
		return
	}
}

// ReadMany handles GET
func (ctl *AttendeesCSVController) ReadMany(c *models.Context) {
	eventID, err := strconv.ParseInt(c.RouteVars["event_id"], 10, 64)
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("The supplied event_id ('%s') is not a number.", c.RouteVars["event_id"]),
			http.StatusBadRequest,
		)
		return
	}

	// Start Authorisation
	perms := models.GetPermission(
		models.MakeAuthorisationContext(
			c, 0, h.ItemTypes[h.ItemTypeEvent], eventID),
	)
	if !perms.CanRead {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}
	if !(perms.IsOwner || perms.IsModerator || perms.IsSiteOwner) {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}
	// End Authorisation

	attendees, status, err := models.GetAttendeesCSV(c.Site.ID, eventID, c.Auth.ProfileID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	c.ResponseWriter.Header().Set(`Cache-Control`, `no-cache, max-age=0`)
	c.ResponseWriter.Header().Set(`Vary`, `Authorization`)
	c.ResponseWriter.Header().Set(`Content-Type`, `text/csv`)
	c.ResponseWriter.Header().Set(`Content-Disposition`, fmt.Sprintf(`attachment; filename=event%d.csv`, eventID))
	c.WriteResponse([]byte(attendees), http.StatusOK)
}
