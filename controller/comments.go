package controller

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/grafana/pyroscope-go"
	"github.com/microcosm-collective/microcosm/audit"
	e "github.com/microcosm-collective/microcosm/errors"
	h "github.com/microcosm-collective/microcosm/helpers"
	"github.com/microcosm-collective/microcosm/models"
)

// CommentsController is a web controller
type CommentsController struct{}

// CommentsHandler is a web handler
func CommentsHandler(w http.ResponseWriter, r *http.Request) {
	path := "/comments"
	pyroscope.TagWrapper(context.Background(), pyroscope.Labels("path", path), func(ctx context.Context) {
		c, status, err := models.MakeContext(r, w)
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}
		ctl := CommentsController{}

		method := c.GetHTTPMethod()
		switch method {
		case "OPTIONS":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				c.RespondWithOptions([]string{"OPTIONS", "POST"})
			})
			return
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

// Create handles POST
func (ctl *CommentsController) Create(c *models.Context) {
	m := models.CommentSummaryType{}
	err := c.Fill(&m)
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("The post data is invalid: %v", err.Error()),
			http.StatusBadRequest,
		)
		return
	}

	// Populate where applicable from auth and context
	m.Meta.CreatedByID = c.Auth.ProfileID
	m.Meta.Created = time.Now()

	status, err := m.Validate(c.Site.ID, false)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	// Start : Authorisation
	perms := models.GetPermission(
		models.MakeAuthorisationContext(
			c, 0, m.ItemTypeID, m.ItemID),
	)
	if !perms.CanCreate {
		c.RespondWithErrorDetail(
			e.New(
				c.Site.ID,
				c.Auth.ProfileID,
				"comments.go::Create",
				e.NoCreate,
				"Not authorized to create comment: CanCreate false",
			),
			http.StatusForbidden,
		)
		return
	}
	// End : Authorisation

	// Create
	status, err = m.Insert(c.Site.ID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	go audit.Create(
		c.Site.ID,
		h.ItemTypes[h.ItemTypeComment],
		m.ID,
		c.Auth.ProfileID,
		time.Now(),
		c.IP,
	)

	// Send updates and register watcher
	if m.ItemTypeID == h.ItemTypes[h.ItemTypeHuddle] {
		models.RegisterWatcher(
			c.Auth.ProfileID,
			h.UpdateTypes[h.UpdateTypeNewCommentInHuddle],
			m.ItemID,
			m.ItemTypeID,
			c.Site.ID,
		)

		go models.SendUpdatesForNewCommentInHuddle(c.Site.ID, m)
		models.MarkAsRead(h.ItemTypes[h.ItemTypeHuddle], m.ItemID, c.Auth.ProfileID, time.Now())
		models.UpdateUnreadHuddleCount(c.Auth.ProfileID)
	} else {
		models.RegisterWatcher(
			c.Auth.ProfileID,
			h.UpdateTypes[h.UpdateTypeNewComment],
			m.ItemID,
			m.ItemTypeID,
			c.Site.ID,
		)

		go models.SendUpdatesForNewCommentInItem(c.Site.ID, m)
	}

	if m.InReplyTo > 0 {
		go models.SendUpdatesForNewReplyToYourComment(c.Site.ID, m)
	}

	// Respond
	c.RespondWithSeeOther(
		fmt.Sprintf(
			"%s/%d",
			h.APITypeComment,
			m.ID,
		),
	)
}
