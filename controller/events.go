package controller

import (
	"fmt"
	"net/http"
	"time"

	"github.com/microcosm-cc/microcosm/audit"
	h "github.com/microcosm-cc/microcosm/helpers"
	"github.com/microcosm-cc/microcosm/models"
)

func EventsHandler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	ctl := EventsController{}

	switch c.GetHttpMethod() {
	case "OPTIONS":
		c.RespondWithOptions([]string{"OPTIONS", "POST", "HEAD", "GET"})
		return
	case "POST":
		ctl.Create(c)
	case "HEAD":
		ctl.ReadMany(c)
	case "GET":
		ctl.ReadMany(c)
	default:
		c.RespondWithStatus(http.StatusMethodNotAllowed)
		return
	}
}

type EventsController struct{}

func (ctl *EventsController) Create(c *models.Context) {

	m := models.EventType{}
	m.Meta.Flags.Open = true

	err := c.Fill(&m)
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("The post data is invalid: %v", err.Error()),
			http.StatusBadRequest,
		)
		return
	}

	// Start : Authorisation
	perms := models.GetPermission(
		models.MakeAuthorisationContext(
			c, 0, h.ItemTypes[h.ItemTypeMicrocosm], m.MicrocosmId),
	)
	if !perms.CanCreate {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}
	// End : Authorisation

	// Populate where applicable from auth and context
	m.Meta.CreatedById = c.Auth.ProfileId
	m.Meta.Created = time.Now()

	status, err := m.Insert(c.Site.Id, c.Auth.ProfileId)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	audit.Create(
		c.Site.Id,
		h.ItemTypes[h.ItemTypeEvent],
		m.Id,
		c.Auth.ProfileId,
		time.Now(),
		c.IP,
	)

	go models.SendUpdatesForNewItemInAMicrocosm(c.Site.Id, m)

	go models.RegisterWatcher(
		c.Auth.ProfileId,
		h.UpdateTypes[h.UpdateTypeNewComment],
		m.Id,
		h.ItemTypes[h.ItemTypeEvent],
		c.Site.Id,
	)

	c.RespondWithSeeOther(
		fmt.Sprintf(
			"%s/%d",
			h.ApiTypeEvent,
			m.Id,
		),
	)
}

func (ctl *EventsController) ReadMany(c *models.Context) {

	// Start Authorisation
	perms := models.GetPermission(
		models.MakeAuthorisationContext(
			c, 0, h.ItemTypes[h.ItemTypeEvent], 0),
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

	attending, status, err := h.GetAttending(query)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	ems, total, pages, status, err := models.GetEvents(c.Site.Id, c.Auth.ProfileId, attending, limit, offset)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	// Construct the response
	thisLink := h.GetLinkToThisPage(*c.Request.URL, offset, limit, total)

	m := models.EventsType{}
	m.Events = h.ConstructArray(
		ems,
		h.ApiTypeEvent,
		total,
		limit,
		offset,
		pages,
		c.Request.URL,
	)
	m.Meta.Links =
		[]h.LinkType{
			h.LinkType{Rel: "self", Href: thisLink.String()},
		}
	m.Meta.Permissions = perms

	c.ResponseWriter.Header().Set("Cache-Control", `no-cache, max-age=0`)

	c.RespondWithData(m)
}
