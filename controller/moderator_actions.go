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

// ModeratorActionsController is a web controller
type ModeratorActionsController struct{}

// ModeratorActionsHandler is a web handler
func ModeratorActionsHandler(w http.ResponseWriter, r *http.Request) {
	path := "/moderator-actions"
	pyroscope.TagWrapper(context.Background(), pyroscope.Labels("path", path), func(ctx context.Context) {
		c, status, err := models.MakeContext(r, w)
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}
		ctl := ModeratorActionsController{}

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

// ReadMany handles GET for the collection
func (ctl *ModeratorActionsController) ReadMany(c *models.Context) {
	// Check if we're filtering by comment ID
	commentID := c.Request.URL.Query().Get("commentId")
	if commentID != "" {
		// Convert to int64
		var id int64
		_, err := fmt.Sscanf(commentID, "%d", &id)
		if err != nil {
			c.RespondWithErrorMessage(
				fmt.Sprintf("Invalid commentId: %v", err.Error()),
				http.StatusBadRequest,
			)
			return
		}

		actions, status, err := models.GetModeratorActionsByComment(c.Site.ID, id, c.Request.URL)
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}

		c.RespondWithData(actions)
		return
	}

	// Get all moderator actions
	actions, status, err := models.GetModeratorActions(c.Site.ID, c.Request.URL)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	c.RespondWithData(actions)
}

// Create handles POST
func (ctl *ModeratorActionsController) Create(c *models.Context) {
	// Parse the input
	m := models.ModeratorActionType{}
	err := c.Fill(&m)
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("The post data is invalid: %v", err.Error()),
			http.StatusBadRequest,
		)
		return
	}

	// Set the moderator profile ID
	m.ModeratorProfileID = c.Auth.ProfileID
	m.Meta.CreatedByID = c.Auth.ProfileID

	// Insert
	status, err := m.Insert(c.Site.ID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	audit.Create(
		c.Site.ID,
		h.ItemTypes[h.ItemTypeModeratorAction],
		m.ID,
		c.Auth.ProfileID,
		time.Now(),
		c.IP,
	)

	// Respond
	c.RespondWithSeeOther(
		fmt.Sprintf(
			"%s/%d",
			h.APITypeModeratorAction,
			m.ID,
		),
	)
}
