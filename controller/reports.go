package controller

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/grafana/pyroscope-go"
	h "github.com/microcosm-collective/microcosm/helpers"
	"github.com/microcosm-collective/microcosm/models"
)

// ReportsHandler accepts user submissions and serves the moderator report queue.
func ReportsHandler(w http.ResponseWriter, r *http.Request) {
	pyroscope.TagWrapper(context.Background(), pyroscope.Labels("path", "/reports"), func(ctx context.Context) {
		c, status, err := models.MakeContext(r, w)
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}
		switch c.GetHTTPMethod() {
		case "OPTIONS":
			c.RespondWithOptions([]string{"OPTIONS", "GET", "POST"})
		case "GET":
			listReports(c)
		case "POST":
			submitReport(c)
		default:
			c.RespondWithStatus(http.StatusMethodNotAllowed)
		}
	})
}

// ReportHandler reads a report or changes its state.
func ReportHandler(w http.ResponseWriter, r *http.Request) {
	pyroscope.TagWrapper(context.Background(), pyroscope.Labels("path", "/reports/{report_id}"), func(ctx context.Context) {
		c, status, err := models.MakeContext(r, w)
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}
		switch c.GetHTTPMethod() {
		case "OPTIONS":
			c.RespondWithOptions([]string{"OPTIONS", "GET", "PATCH"})
		case "GET":
			readReport(c)
		case "PATCH":
			changeReportState(c)
		default:
			c.RespondWithStatus(http.StatusMethodNotAllowed)
		}
	})
}

// ReportTypesHandler returns the active categories callers can submit.
func ReportTypesHandler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}
	if c.GetHTTPMethod() == "OPTIONS" {
		c.RespondWithOptions([]string{"OPTIONS", "GET"})
		return
	}
	if c.GetHTTPMethod() != "GET" {
		c.RespondWithStatus(http.StatusMethodNotAllowed)
		return
	}
	types, status, err := models.GetReportTypes()
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}
	c.RespondWithData(map[string]interface{}{"reportTypes": types})
}

type reportSubmission struct {
	CommentID int64  `json:"commentId"`
	Type      string `json:"type"`
}

func submitReport(c *models.Context) {
	if c.Auth.ProfileID < 1 {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}
	var request reportSubmission
	if err := c.Fill(&request); err != nil {
		c.RespondWithErrorMessage(fmt.Sprintf("The post data is invalid: %v", err), http.StatusBadRequest)
		return
	}
	report, status, err := models.SubmitReport(c.Site.ID, c.Auth.ProfileID, request.CommentID, request.Type)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}
	c.RespondWithData(report)
}

func listReports(c *models.Context) {
	if !canManageReports(c) {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}
	query := c.Request.URL.Query()
	limit, offset, status, err := h.GetLimitAndOffset(query)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}
	state := query.Get("state")
	reports, total, pages, status, err := models.GetReports(c.Site.ID, state, limit, offset, 0)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}
	page := models.ReportsPage{}
	page.Reports = h.ConstructArray(reports, "/api/v1/reports", total, limit, offset, pages, c.Request.URL)
	c.RespondWithData(page)
}

func readReport(c *models.Context) {
	if !canManageReports(c) {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}
	id, err := strconv.ParseInt(c.RouteVars["report_id"], 10, 64)
	if err != nil || id < 1 {
		c.RespondWithErrorMessage("report_id must be a positive number", http.StatusBadRequest)
		return
	}
	reports, _, _, status, err := models.GetReports(c.Site.ID, "", 1, 0, id)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}
	if len(reports) == 0 {
		c.RespondWithStatus(http.StatusNotFound)
		return
	}
	c.RespondWithData(reports[0])
}

type reportStateRequest struct {
	State string `json:"state"`
	Note  string `json:"note"`
}

func changeReportState(c *models.Context) {
	if !canManageReports(c) {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}
	id, err := strconv.ParseInt(c.RouteVars["report_id"], 10, 64)
	if err != nil || id < 1 {
		c.RespondWithErrorMessage("report_id must be a positive number", http.StatusBadRequest)
		return
	}
	var request reportStateRequest
	if err := c.Fill(&request); err != nil {
		c.RespondWithErrorMessage(fmt.Sprintf("The patch data is invalid: %v", err), http.StatusBadRequest)
		return
	}
	status, err := models.UpdateReportState(c.Site.ID, id, c.Auth.ProfileID, request.State, request.Note)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}
	c.RespondWithOK()
}

func canManageReports(c *models.Context) bool {
	rootMicrocosmID := models.GetRootMicrocosmID(c.Site.ID)
	if rootMicrocosmID == 0 {
		return false
	}
	perms := models.GetPermission(models.MakeAuthorisationContext(c, rootMicrocosmID, h.ItemTypes[h.ItemTypeMicrocosm], rootMicrocosmID))
	return c.Auth.IsSiteOwner || perms.IsModerator
}
