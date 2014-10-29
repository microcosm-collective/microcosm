package controller

import (
	"fmt"
	"net/http"
	"time"

	"github.com/microcosm-cc/microcosm/audit"
	e "github.com/microcosm-cc/microcosm/errors"
	h "github.com/microcosm-cc/microcosm/helpers"
	"github.com/microcosm-cc/microcosm/models"
)

type CommentsController struct{}

func CommentsHandler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	ctl := CommentsController{}

	switch c.GetHttpMethod() {
	case "OPTIONS":
		c.RespondWithOptions([]string{"OPTIONS", "POST"})
		return
	case "POST":
		ctl.Create(c)
	default:
		c.RespondWithStatus(http.StatusMethodNotAllowed)
		return
	}
}

// Creates a single comment
func (ctl *CommentsController) Create(c *models.Context) {

	// Initialise (non-zero defaults must be set)
	m := models.CommentSummaryType{}

	// Fill from POST data
	err := c.Fill(&m)
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("The post data is invalid: %v", err.Error()),
			http.StatusBadRequest,
		)
		return
	}

	// Populate where applicable from auth and context
	m.Meta.CreatedById = c.Auth.ProfileId
	m.Meta.Created = time.Now()

	status, err := m.Validate(c.Site.Id, false)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	// Start : Authorisation
	perms := models.GetPermission(
		models.MakeAuthorisationContext(
			c, 0, m.ItemTypeId, m.ItemId),
	)
	if !perms.CanCreate {
		c.RespondWithErrorDetail(
			e.New(c.Site.Id, c.Auth.ProfileId, "comments.go::Create", e.NoCreate, "Not authorized to create comment: CanCreate false"),
			http.StatusForbidden,
		)
		return
	}
	// End : Authorisation

	// Create
	status, err = m.Insert(c.Site.Id)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	go audit.Create(
		c.Site.Id,
		h.ItemTypes[h.ItemTypeComment],
		m.Id,
		c.Auth.ProfileId,
		time.Now(),
		c.IP,
	)

	// Send updates and register watcher
	if m.ItemTypeId == h.ItemTypes[h.ItemTypeHuddle] {
		models.RegisterWatcher(
			c.Auth.ProfileId,
			h.UpdateTypes[h.UpdateTypeNewCommentInHuddle],
			m.ItemId,
			m.ItemTypeId,
			c.Site.Id,
		)

		go models.SendUpdatesForNewCommentInHuddle(c.Site.Id, m)
		models.MarkAsRead(h.ItemTypes[h.ItemTypeHuddle], m.ItemId, c.Auth.ProfileId, time.Now())
		models.UpdateUnreadHuddleCount(c.Auth.ProfileId)
	} else {
		models.RegisterWatcher(
			c.Auth.ProfileId,
			h.UpdateTypes[h.UpdateTypeNewComment],
			m.ItemId,
			m.ItemTypeId,
			c.Site.Id,
		)

		go models.SendUpdatesForNewCommentInItem(c.Site.Id, m)
	}

	if m.InReplyTo > 0 {
		go models.SendUpdatesForNewReplyToYourComment(c.Site.Id, m)
	}

	// Respond
	c.RespondWithSeeOther(
		fmt.Sprintf(
			"%s/%d",
			h.ApiTypeComment,
			m.Id,
		),
	)
}
