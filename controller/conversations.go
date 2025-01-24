package controller

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/grafana/pyroscope-go"
	"github.com/microcosm-collective/microcosm/audit"
	h "github.com/microcosm-collective/microcosm/helpers"
	"github.com/microcosm-collective/microcosm/models"
)

// ConversationsController is a web controller
type ConversationsController struct{}

// ConversationsHandler is a web handler
func ConversationsHandler(w http.ResponseWriter, r *http.Request) {
	path := "/conversations/"
	pyroscope.TagWrapper(context.Background(), pyroscope.Labels("path", path), func(ctx context.Context) {
		c, status, err := models.MakeContext(r, w)
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}
		ctl := ConversationsController{}

		method := c.GetHTTPMethod()
		switch method {
		case "OPTIONS":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				c.RespondWithOptions([]string{"OPTIONS", "GET", "HEAD", "POST"})
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
		case "POST":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				ctl.Create(c)
			})
		default:
			c.RespondWithStatus(http.StatusMethodNotAllowed)
			return
		}
	})
}

// ReadMany handles GET
func (ctl *ConversationsController) ReadMany(c *models.Context) {
	// Start Authorisation
	perms := models.GetPermission(
		models.MakeAuthorisationContext(
			c, 0, h.ItemTypes[h.ItemTypeConversation], 0),
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

	ems, total, pages, status, err := models.GetConversations(c.Site.ID, c.Auth.ProfileID, limit, offset)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	// Construct the response
	thisLink := h.GetLinkToThisPage(*c.Request.URL, offset, limit, total)

	m := models.ConversationsType{}
	m.Conversations = h.ConstructArray(
		ems,
		h.APITypeConversation,
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

// Create handles POST
func (ctl *ConversationsController) Create(c *models.Context) {
	// Validate inputs
	m := models.ConversationType{}
	m.Meta.Flags.Deleted = false
	m.Meta.Flags.Moderated = false
	m.Meta.Flags.Open = true
	m.Meta.Flags.Sticky = false

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
			c, 0, h.ItemTypes[h.ItemTypeMicrocosm], m.MicrocosmID),
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
		h.ItemTypes[h.ItemTypeConversation],
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
		h.ItemTypes[h.ItemTypeConversation],
		c.Site.ID,
	)

	c.RespondWithSeeOther(
		fmt.Sprintf(
			"%s/%d",
			h.APITypeConversation,
			m.ID,
		),
	)
}
