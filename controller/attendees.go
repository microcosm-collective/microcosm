package controller

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/golang/glog"
	"github.com/lib/pq"

	"github.com/microcosm-cc/microcosm/audit"
	e "github.com/microcosm-cc/microcosm/errors"
	h "github.com/microcosm-cc/microcosm/helpers"
	"github.com/microcosm-cc/microcosm/models"
)

// AttendeesController is a web controller
type AttendeesController struct{}

// AttendeesHandler is a web handler
func AttendeesHandler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	ctl := AttendeesController{}

	switch c.GetHTTPMethod() {
	case "OPTIONS":
		c.RespondWithOptions([]string{"OPTIONS", "PUT", "HEAD", "GET"})
		return
	case "PUT":
		ctl.UpdateMany(c)
	case "HEAD":
		ctl.ReadMany(c)
	case "GET":
		ctl.ReadMany(c)
	default:
		c.RespondWithStatus(http.StatusMethodNotAllowed)
		return
	}
}

// UpdateMany handles PUT on the collection
func (ctl *AttendeesController) UpdateMany(c *models.Context) {
	// Verify event_id is a positive integer
	eventID, err := strconv.ParseInt(c.RouteVars["event_id"], 10, 64)
	if err != nil {
		glog.Errorln(err.Error())
		c.RespondWithErrorMessage(
			fmt.Sprintf("The supplied event ID ('%s') is not a number.", c.RouteVars["event_id"]),
			http.StatusBadRequest,
		)
		return
	}

	ems := []models.AttendeeType{}

	err = c.Fill(&ems)
	if err != nil {
		glog.Errorln(err.Error())
		c.RespondWithErrorMessage(
			fmt.Sprintf("The post data is invalid: %v", err.Error()),
			http.StatusBadRequest,
		)
		return
	}

	// Start : Authorisation
	perms := models.GetPermission(
		models.MakeAuthorisationContext(
			c, 0, h.ItemTypes[h.ItemTypeEvent], eventID),
	)

	if !perms.CanCreate {
		c.RespondWithErrorDetail(
			e.New(c.Site.ID, c.Auth.ProfileID, "attendees.go::UpdateMany", e.NoCreate, "Not authorized to create attendee: CanCreate false"),
			http.StatusForbidden,
		)
		return
	}
	// Everyone can set self to any status.  Event/site owners can set people to any status apart from 'attending'.
	// Also check that profile exists on site.
	if perms.IsOwner || perms.IsModerator || perms.IsSiteOwner {
		for _, m := range ems {
			if m.ProfileID != c.Auth.ProfileID && m.RSVP == "yes" {
				c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
				return
			}
			_, status, err := models.GetProfileSummary(c.Site.ID, m.ProfileID)
			if err != nil {
				c.RespondWithErrorMessage(h.NoAuthMessage, status)
				return
			}
		}
	} else {
		for _, m := range ems {
			if m.ProfileID != c.Auth.ProfileID {
				c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
				return
			}
			_, status, err := models.GetProfileSummary(c.Site.ID, m.ProfileID)
			if err != nil {
				c.RespondWithErrorMessage(h.NoAuthMessage, status)
				return
			}
		}
	}
	// End : Authorisation

	t := time.Now()
	// Populate where applicable from auth and context
	for i := range ems {
		ems[i].EventID = eventID
		ems[i].Meta.CreatedByID = c.Auth.ProfileID
		ems[i].Meta.Created = t
		ems[i].Meta.EditedNullable = pq.NullTime{Time: t, Valid: true}
		ems[i].Meta.EditedByNullable = sql.NullInt64{Int64: c.Auth.ProfileID, Valid: true}
	}

	status, err := models.UpdateManyAttendees(c.Site.ID, ems)
	if err != nil {
		glog.Error(err)
		c.RespondWithErrorDetail(err, status)
		return
	}
	for _, m := range ems {
		if m.RSVP == "yes" {
			go models.SendUpdatesForNewAttendeeInAnEvent(c.Site.ID, m)

			// The new attendee should be following the event now
			go models.RegisterWatcher(
				m.ProfileID,
				h.UpdateTypes[h.UpdateTypeEventReminder],
				m.EventID,
				h.ItemTypes[h.ItemTypeEvent],
				c.Site.ID,
			)
		}

		audit.Replace(
			c.Site.ID,
			h.ItemTypes[h.ItemTypeAttendee],
			m.ID,
			c.Auth.ProfileID,
			time.Now(),
			c.IP,
		)
	}

	c.RespondWithOK()
}

// ReadMany handles GET for the collection
func (ctl *AttendeesController) ReadMany(c *models.Context) {

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
	// End Authorisation

	// Fetch query string args if any exist
	query := c.Request.URL.Query()

	limit, offset, status, err := h.GetLimitAndOffset(query)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	attending, status, err := h.AttendanceStatus(query)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	ems, total, pages, status, err := models.GetAttendees(c.Site.ID, eventID, limit, offset, attending == "attending")
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	// Construct the response
	thisLink := h.GetLinkToThisPage(*c.Request.URL, offset, limit, total)

	m := models.AttendeesType{}
	m.Attendees = h.ConstructArray(
		ems,
		fmt.Sprintf(h.APITypeAttendee, 0),
		total,
		limit,
		offset,
		pages,
		c.Request.URL,
	)
	m.Meta.Links =
		[]h.LinkType{
			{Rel: "self", Href: thisLink.String()},
		}
	m.Meta.Permissions = perms

	c.RespondWithData(m)
}
