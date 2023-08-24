package controller

import (
	"fmt"
	"net/http"
	"time"

	"github.com/microcosm-cc/microcosm/audit"
	h "github.com/microcosm-cc/microcosm/helpers"
	"github.com/microcosm-cc/microcosm/models"
)

// PollsHandler is a web handler
func PollsHandler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	ctl := PollsController{}

	switch c.GetHTTPMethod() {
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

// PollsController is a web controller
type PollsController struct{}

// Create handles POST
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
	m.Meta.CreatedByID = c.Auth.ProfileID
	m.Meta.Created = time.Now()

	status, err := m.Insert(c.Site.ID, c.Auth.ProfileID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	audit.Create(
		c.Site.ID,
		h.ItemTypes[h.ItemTypePoll],
		m.ID,
		c.Auth.ProfileID,
		time.Now(),
		c.IP,
	)

	go models.SendUpdatesForNewItemInAMicrocosm(c.Site.ID, m)

	go models.RegisterWatcher(
		c.Auth.ProfileID,
		h.UpdateTypes[h.UpdateTypeNewComment],
		m.ID,
		h.ItemTypes[h.ItemTypePoll],
		c.Site.ID,
	)

	c.RespondWithSeeOther(
		fmt.Sprintf(
			"%s/%d",
			h.APITypePoll,
			m.ID,
		),
	)
}

// ReadMany handles GET for the collection
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

	ems, total, pages, status, err := models.GetPolls(c.Site.ID, c.Auth.ProfileID, limit, offset)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	// Construct the response
	thisLink := h.GetLinkToThisPage(*c.Request.URL, offset, limit, total)

	m := models.PollsType{}
	m.Polls = h.ConstructArray(
		ems,
		h.APITypePoll,
		total,
		limit,
		offset,
		pages,
		c.Request.URL,
	)
	m.Meta.Links = []h.LinkType{
		{Rel: "self", Href: thisLink.String()},
	}
	m.Meta.Permissions = perms

	c.RespondWithData(m)
}
