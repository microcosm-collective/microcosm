package controller

import (
	"fmt"
	"net/http"
	"time"

	"github.com/microcosm-cc/microcosm/audit"
	h "github.com/microcosm-cc/microcosm/helpers"
	"github.com/microcosm-cc/microcosm/models"
)

func PollsHandler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	ctl := PollsController{}

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

type PollsController struct{}

func (ctl *PollsController) Create(c *models.Context) {

	// Validate inputs
	m := models.PollType{}
	m.PollOpen = true

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
			c, m.MicrocosmID, 0, 0),
	)
	if !perms.CanCreate {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}
	// End : Authorisation

	// Populate where applicable from auth and context
	m.Meta.CreatedById = c.Auth.ProfileId
	m.Meta.Created = time.Now()

	status, err := m.Insert(c.Site.ID, c.Auth.ProfileId)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	audit.Create(
		c.Site.ID,
		h.ItemTypes[h.ItemTypePoll],
		m.ID,
		c.Auth.ProfileId,
		time.Now(),
		c.IP,
	)

	go models.SendUpdatesForNewItemInAMicrocosm(c.Site.ID, m)

	go models.RegisterWatcher(
		c.Auth.ProfileId,
		h.UpdateTypes[h.UpdateTypeNewComment],
		m.ID,
		h.ItemTypes[h.ItemTypePoll],
		c.Site.ID,
	)

	c.RespondWithSeeOther(
		fmt.Sprintf(
			"%s/%d",
			h.ApiTypePoll,
			m.ID,
		),
	)
}

func (ctl *PollsController) ReadMany(c *models.Context) {

	// Start Authorisation
	perms := models.GetPermission(
		models.MakeAuthorisationContext(
			c, 0, h.ItemTypes[h.ItemTypePoll], 0),
	)
	if !perms.CanRead {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}
	// End Authorisation

	// Fetch query string args if any exist
	limit, offset, status, err := h.GetLimitAndOffset(c.Request.URL.Query())
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	ems, total, pages, status, err := models.GetPolls(c.Site.ID, c.Auth.ProfileId, limit, offset)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	// Construct the response
	thisLink := h.GetLinkToThisPage(*c.Request.URL, offset, limit, total)

	m := models.PollsType{}
	m.Polls = h.ConstructArray(
		ems,
		h.ApiTypePoll,
		total,
		limit,
		offset,
		pages,
		c.Request.URL,
	)
	m.Meta.Links = []h.LinkType{
		h.LinkType{Rel: "self", Href: thisLink.String()},
	}
	m.Meta.Permissions = perms

	c.RespondWithData(m)
}
