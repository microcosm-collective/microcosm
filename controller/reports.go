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

// ReportsController is a web controller
type ReportsController struct{}

// ReportsHandler is a web handler
func ReportsHandler(w http.ResponseWriter, r *http.Request) {
	path := "/reports"
	pyroscope.TagWrapper(context.Background(), pyroscope.Labels("path", path), func(ctx context.Context) {
		c, status, err := models.MakeContext(r, w)
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}
		ctl := ReportsController{}

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
func (ctl *ReportsController) ReadMany(c *models.Context) {
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

		reports, status, err := models.GetReportsByComment(c.Site.ID, id, c.Request.URL)
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}

		c.RespondWithData(reports)
		return
	}

	// Get all reports
	reports, status, err := models.GetReports(c.Site.ID, c.Request.URL)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	c.RespondWithData(reports)
}

// Create handles POST
func (ctl *ReportsController) Create(c *models.Context) {
	// Parse the input
	m := models.ReportType{}
	err := c.Fill(&m)
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("The post data is invalid: %v", err.Error()),
			http.StatusBadRequest,
		)
		return
	}

	// Set the reporter profile ID
	m.ReportedByProfileID = c.Auth.ProfileID
	m.Meta.CreatedByID = c.Auth.ProfileID

	// Insert
	status, err := m.Insert(c.Site.ID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	audit.Create(
		c.Site.ID,
		h.ItemTypes[h.ItemTypeReport],
		m.ID,
		c.Auth.ProfileID,
		time.Now(),
		c.IP,
	)

	// Respond
	c.RespondWithSeeOther(
		fmt.Sprintf(
			"%s/%d",
			h.APITypeReport,
			m.ID,
		),
	)
}
