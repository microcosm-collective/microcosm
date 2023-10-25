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

	"github.com/microcosm-cc/microcosm/audit"
	h "github.com/microcosm-cc/microcosm/helpers"
	"github.com/microcosm-cc/microcosm/models"
)

// AttendeeController is a web controller
type AttendeeController struct{}

// AttendeeHandler is a web handler
func AttendeeHandler(w http.ResponseWriter, r *http.Request) {
	path := "/events/{id}/attendees/{id}"
	pyroscope.TagWrapper(context.Background(), pyroscope.Labels("path", path), func(context.Context) {
		c, status, err := models.MakeContext(r, w)
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}

		ctl := AttendeeController{}

		method := c.GetHTTPMethod()
		switch method {
		case "OPTIONS":
			pyroscope.TagWrapper(context.Background(), pyroscope.Labels("method", method), func(context.Context) {
				c.RespondWithOptions([]string{"OPTIONS", "HEAD", "GET", "PUT", "DELETE"})
			})
			return
		case "HEAD":
			pyroscope.TagWrapper(context.Background(), pyroscope.Labels("method", method), func(context.Context) {
				ctl.Read(c)
			})
		case "GET":
			pyroscope.TagWrapper(context.Background(), pyroscope.Labels("method", method), func(context.Context) {
				ctl.Read(c)
			})
		case "PUT":
			pyroscope.TagWrapper(context.Background(), pyroscope.Labels("method", method), func(context.Context) {
				ctl.Update(c)
			})
		case "DELETE":
			pyroscope.TagWrapper(context.Background(), pyroscope.Labels("method", method), func(context.Context) {
				ctl.Delete(c)
			})
		default:
			c.RespondWithStatus(http.StatusMethodNotAllowed)
			return
		}
	})
}

// Read handles GET
func (ctl *AttendeeController) Read(c *models.Context) {
	// Verify ID is a positive integer
	eventID, err := strconv.ParseInt(c.RouteVars["event_id"], 10, 64)
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("The supplied event_id ('%s') is not a number.", c.RouteVars["event_id"]),
			http.StatusBadRequest,
		)
		return
	}

	profileID, err := strconv.ParseInt(c.RouteVars["profile_id"], 10, 64)
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("The supplied profile_id ('%s') is not a number.", c.RouteVars["profile_id"]),
			http.StatusBadRequest,
		)
		return
	}

	attendeeID, status, err := models.GetAttendeeID(eventID, profileID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	// Start Authorisation
	perms := models.GetPermission(
		models.MakeAuthorisationContext(
			c, 0, h.ItemTypes[h.ItemTypeAttendee], attendeeID),
	)
	if !perms.CanRead {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}
	// End Authorisation

	// Read Event
	m, status, err := models.GetAttendee(c.Site.ID, attendeeID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	m.Meta.Permissions = perms

	c.RespondWithData(m)
}

// Update handles PUT
func (ctl *AttendeeController) Update(c *models.Context) {
	// Verify ID is a positive integer
	eventID, err := strconv.ParseInt(c.RouteVars["event_id"], 10, 64)
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("The supplied event_id ('%s') is not a number.", c.RouteVars["event_id"]),
			http.StatusBadRequest,
		)
		return
	}

	m := models.AttendeeType{}

	err = c.Fill(&m)
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("The post data is invalid: %v", err.Error()),
			http.StatusBadRequest,
		)
		return
	}

	// Start Authorisation
	perms := models.GetPermission(
		models.MakeAuthorisationContext(
			c, 0, h.ItemTypes[h.ItemTypeEvent], eventID),
	)
	if !perms.CanUpdate {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}

	if perms.IsOwner || perms.IsModerator || perms.IsSiteOwner {
		if m.ProfileID != c.Auth.ProfileID && m.RSVP == "yes" {
			c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
			return
		}
	} else {
		if m.ProfileID != c.Auth.ProfileID {
			c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
			return
		}
	}
	_, status, err := models.GetProfileSummary(c.Site.ID, m.ProfileID)
	if err != nil {
		c.RespondWithErrorMessage(h.NoAuthMessage, status)
		return
	}
	// End Authorisation

	// Populate where applicable from auth and context
	t := time.Now()
	m.EventID = eventID
	m.Meta.CreatedByID = c.Auth.ProfileID
	m.Meta.Created = t
	m.Meta.EditedByNullable = sql.NullInt64{Int64: c.Auth.ProfileID, Valid: true}
	m.Meta.EditedNullable = pq.NullTime{Time: t, Valid: true}

	status, err = m.Update(c.Site.ID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	if m.RSVP == "yes" {
		go models.SendUpdatesForNewAttendeeInAnEvent(c.Site.ID, m)
	}

	audit.Replace(
		c.Site.ID,
		h.ItemTypes[h.ItemTypeAttendee],
		m.ID,
		c.Auth.ProfileID,
		time.Now(),
		c.IP,
	)

	c.RespondWithSeeOther(
		fmt.Sprintf("%s/%d", fmt.Sprintf(h.APITypeAttendee, m.EventID), m.ProfileID),
	)
}

// Delete handles DELETE
func (ctl *AttendeeController) Delete(c *models.Context) {
	// Validate inputs
	eventID, err := strconv.ParseInt(c.RouteVars["event_id"], 10, 64)
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("The supplied event_id ('%s') is not a number.", c.RouteVars["event_id"]),
			http.StatusBadRequest,
		)
		return
	}

	profileID, err := strconv.ParseInt(c.RouteVars["profile_id"], 10, 64)
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("The supplied profile_id ('%s') is not a number.", c.RouteVars["profile_id"]),
			http.StatusBadRequest,
		)
		return
	}

	attendeeID, _, err := models.GetAttendeeID(eventID, profileID)
	if err != nil {
		c.RespondWithOK()
		return
	}

	// Start Authorisation
	perms := models.GetPermission(
		models.MakeAuthorisationContext(
			c, 0, h.ItemTypes[h.ItemTypeAttendee], attendeeID),
	)
	if !perms.CanDelete {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}
	// End Authorisation

	m, status, err := models.GetAttendee(c.Site.ID, attendeeID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	// Delete resource
	status, err = m.Delete(c.Site.ID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	audit.Replace(
		c.Site.ID,
		h.ItemTypes[h.ItemTypeAttendee],
		attendeeID,
		c.Auth.ProfileID,
		time.Now(),
		c.IP,
	)

	c.RespondWithOK()
}
