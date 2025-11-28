package models

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/golang/glog"

	c "github.com/microcosm-collective/microcosm/cache"
	h "github.com/microcosm-collective/microcosm/helpers"
)

// ReportsType is a collection of reports
type ReportsType struct {
	Reports h.ArrayType    `json:"reports"`
	Meta    h.CoreMetaType `json:"meta"`
}

// ReportType encapsulates a report
type ReportType struct {
	ID                 int64     `json:"id"`
	ReportedByProfileID int64    `json:"reportedByProfileId"`
	ReportReasonID     int64     `json:"reportReasonId"`
	CommentID          int64     `json:"commentId"`
	ReportReasonExtra  string    `json:"reportReasonExtra"`
	Created            time.Time `json:"-"`

	// Populated fields
	ReportReason      ReportReasonType    `json:"reportReason,omitempty"`
	ReportedBy        ProfileSummaryType  `json:"reportedBy,omitempty"`
	Comment           CommentSummaryType  `json:"comment,omitempty"`

	Meta              ReportMetaType      `json:"meta"`
}

// ReportMetaType is the meta struct of a report
type ReportMetaType struct {
	h.CreatedType
	h.CoreMetaType
}

// Validate returns true if a report is valid
func (m *ReportType) Validate(siteID int64) (int, error) {
	if m.ReportedByProfileID <= 0 {
		return http.StatusBadRequest, fmt.Errorf("reportedByProfileId is required")
	}

	if m.ReportReasonID <= 0 {
		return http.StatusBadRequest, fmt.Errorf("reportReasonId is required")
	}

	if m.CommentID <= 0 {
		return http.StatusBadRequest, fmt.Errorf("commentId is required")
	}

	// Verify the comment exists
	_, status, err := GetCommentSummary(siteID, m.CommentID)
	if err != nil {
		return status, fmt.Errorf("invalid commentId: %v", err)
	}

	// Verify the report reason exists
	_, status, err = GetReportReason(m.ReportReasonID)
	if err != nil {
		return status, fmt.Errorf("invalid reportReasonId: %v", err)
	}

	return http.StatusOK, nil
}

// Insert saves a report
func (m *ReportType) Insert(siteID int64) (int, error) {
	status, err := m.Validate(siteID)
	if err != nil {
		return status, err
	}

	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	now := time.Now()
	m.Created = now
	m.Meta.Created = now

	var insertID int64
	err = tx.QueryRow(`
INSERT INTO reports (
    reported_by_profile_id, report_reason_id, report_reason_extra, created
) VALUES (
    $1, $2, $3, $4
) RETURNING report_id`,
		m.ReportedByProfileID,
		m.ReportReasonID,
		m.ReportReasonExtra,
		now,
	).Scan(&insertID)

	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error inserting data: %v", err)
	}

	m.ID = insertID

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("transaction failed: %v", err)
	}

	PurgeCache(h.ItemTypes[h.ItemTypeReport], m.ID)

	return http.StatusOK, nil
}

// Update saves changes to a report
func (m *ReportType) Update(siteID int64) (int, error) {
	status, err := m.Validate(siteID)
	if err != nil {
		return status, err
	}

	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
UPDATE reports
   SET reported_by_profile_id = $2,
       report_reason_id = $3,
       report_reason_extra = $4
 WHERE report_id = $1`,
		m.ID,
		m.ReportedByProfileID,
		m.ReportReasonID,
		m.ReportReasonExtra,
	)

	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error updating data: %v", err)
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("transaction failed: %v", err)
	}

	PurgeCache(h.ItemTypes[h.ItemTypeReport], m.ID)

	return http.StatusOK, nil
}

// Delete removes a report
func (m *ReportType) Delete() (int, error) {
	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
DELETE FROM reports
 WHERE report_id = $1`,
		m.ID,
	)

	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error deleting data: %v", err)
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("transaction failed: %v", err)
	}

	PurgeCache(h.ItemTypes[h.ItemTypeReport], m.ID)

	return http.StatusOK, nil
}

// Hydrate populates a partially populated struct
func (m *ReportType) Hydrate(siteID int64) (int, error) {
	// Get report reason
	reportReason, status, err := GetReportReason(m.ReportReasonID)
	if err != nil {
		return status, err
	}
	m.ReportReason = reportReason

	// Get profile
	profile, status, err := GetProfileSummary(siteID, m.ReportedByProfileID)
	if err != nil {
		return status, err
	}
	m.ReportedBy = profile
	m.Meta.CreatedBy = profile

	// Get comment
	comment, status, err := GetCommentSummary(siteID, m.CommentID)
	if err != nil {
		return status, err
	}
	m.Comment = comment

	return http.StatusOK, nil
}

// GetReport returns a report
func GetReport(siteID int64, reportID int64) (ReportType, int, error) {
	if reportID == 0 {
		return ReportType{}, http.StatusNotFound, fmt.Errorf("report not found")
	}

	// Try to get from cache
	mcKey := fmt.Sprintf("report_%d", reportID)
	if val, ok := c.Get(mcKey, ReportType{}); ok {
		m := val.(ReportType)
		status, err := m.Hydrate(siteID)
		if err != nil {
			return m, status, err
		}
		return m, http.StatusOK, nil
	}

	db, err := h.GetConnection()
	if err != nil {
		return ReportType{}, http.StatusInternalServerError, err
	}

	m := ReportType{}
	err = db.QueryRow(`
SELECT r.report_id, r.reported_by_profile_id, r.report_reason_id, 
       r.report_reason_extra, r.created, c.comment_id
  FROM reports r
  JOIN comments c ON r.comment_id = c.comment_id
 WHERE r.report_id = $1`,
		reportID,
	).Scan(
		&m.ID,
		&m.ReportedByProfileID,
		&m.ReportReasonID,
		&m.ReportReasonExtra,
		&m.Created,
		&m.CommentID,
	)

	if err == sql.ErrNoRows {
		return ReportType{}, http.StatusNotFound, fmt.Errorf("report with ID %d not found", reportID)
	} else if err != nil {
		return ReportType{}, http.StatusInternalServerError, fmt.Errorf("database query failed: %v", err)
	}

	m.Meta.Created = m.Created
	m.Meta.CreatedByID = m.ReportedByProfileID

	m.Meta.Links = []h.LinkType{
		h.GetLink("self", "", h.ItemTypeReport, m.ID),
		h.GetLink("comment", "", h.ItemTypeComment, m.CommentID),
	}

	// Update cache
	c.Set(mcKey, m, 60*60*24) // Cache for 24 hours

	// Hydrate with related data
	status, err := m.Hydrate(siteID)
	if err != nil {
		return m, status, err
	}

	return m, http.StatusOK, nil
}

// GetReports returns a collection of reports
func GetReports(siteID int64, reqURL *url.URL) (h.ArrayType, int, error) {
	query := reqURL.Query()
	limit, offset, status, err := h.GetLimitAndOffset(query)
	if err != nil {
		return h.ArrayType{}, status, err
	}

	db, err := h.GetConnection()
	if err != nil {
		return h.ArrayType{}, http.StatusInternalServerError, err
	}

	rows, err := db.Query(`
SELECT r.report_id, r.reported_by_profile_id, r.report_reason_id, 
       r.report_reason_extra, r.created, c.comment_id
  FROM reports r
  JOIN comments c ON r.comment_id = c.comment_id
 ORDER BY r.created DESC
 LIMIT $1 OFFSET $2`,
		limit,
		offset,
	)
	if err != nil {
		return h.ArrayType{}, http.StatusInternalServerError, fmt.Errorf("database query failed: %v", err)
	}
	defer rows.Close()

	var reports []interface{}
	for rows.Next() {
		m := ReportType{}
		err = rows.Scan(
			&m.ID,
			&m.ReportedByProfileID,
			&m.ReportReasonID,
			&m.ReportReasonExtra,
			&m.Created,
			&m.CommentID,
		)
		if err != nil {
			return h.ArrayType{}, http.StatusInternalServerError, fmt.Errorf("row parsing error: %v", err)
		}

		m.Meta.Created = m.Created
		m.Meta.CreatedByID = m.ReportedByProfileID

		m.Meta.Links = []h.LinkType{
			h.GetLink("self", "", h.ItemTypeReport, m.ID),
			h.GetLink("comment", "", h.ItemTypeComment, m.CommentID),
		}

		// Hydrate with related data
		_, err := m.Hydrate(siteID)
		if err != nil {
			glog.Warningf("Error hydrating report %d: %v", m.ID, err)
		}

		reports = append(reports, m)
	}
	err = rows.Err()
	if err != nil {
		return h.ArrayType{}, http.StatusInternalServerError, fmt.Errorf("error fetching rows: %v", err)
	}

	// Get total count
	var total int64
	err = db.QueryRow(`SELECT COUNT(*) FROM reports`).Scan(&total)
	if err != nil {
		return h.ArrayType{}, http.StatusInternalServerError, fmt.Errorf("count query failed: %v", err)
	}

	pages := h.GetPageCount(total, limit)

	reportsArray := h.ConstructArray(
		reports,
		h.APITypeReport,
		total,
		limit,
		offset,
		pages,
		reqURL,
	)

	return reportsArray, http.StatusOK, nil
}

// GetReportsByComment returns reports for a specific comment
func GetReportsByComment(siteID int64, commentID int64, reqURL *url.URL) (h.ArrayType, int, error) {
	query := reqURL.Query()
	limit, offset, status, err := h.GetLimitAndOffset(query)
	if err != nil {
		return h.ArrayType{}, status, err
	}

	db, err := h.GetConnection()
	if err != nil {
		return h.ArrayType{}, http.StatusInternalServerError, err
	}

	rows, err := db.Query(`
SELECT r.report_id, r.reported_by_profile_id, r.report_reason_id, 
       r.report_reason_extra, r.created
  FROM reports r
 WHERE r.comment_id = $1
 ORDER BY r.created DESC
 LIMIT $2 OFFSET $3`,
		commentID,
		limit,
		offset,
	)
	if err != nil {
		return h.ArrayType{}, http.StatusInternalServerError, fmt.Errorf("database query failed: %v", err)
	}
	defer rows.Close()

	var reports []interface{}
	for rows.Next() {
		m := ReportType{}
		m.CommentID = commentID
		err = rows.Scan(
			&m.ID,
			&m.ReportedByProfileID,
			&m.ReportReasonID,
			&m.ReportReasonExtra,
			&m.Created,
		)
		if err != nil {
			return h.ArrayType{}, http.StatusInternalServerError, fmt.Errorf("row parsing error: %v", err)
		}

		m.Meta.Created = m.Created
		m.Meta.CreatedByID = m.ReportedByProfileID

		m.Meta.Links = []h.LinkType{
			h.GetLink("self", "", h.ItemTypeReport, m.ID),
			h.GetLink("comment", "", h.ItemTypeComment, m.CommentID),
		}

		// Hydrate with related data
		_, err := m.Hydrate(siteID)
		if err != nil {
			glog.Warningf("Error hydrating report %d: %v", m.ID, err)
		}

		reports = append(reports, m)
	}
	err = rows.Err()
	if err != nil {
		return h.ArrayType{}, http.StatusInternalServerError, fmt.Errorf("error fetching rows: %v", err)
	}

	// Get total count
	var total int64
	err = db.QueryRow(`
SELECT COUNT(*) 
  FROM reports 
 WHERE comment_id = $1`,
		commentID,
	).Scan(&total)
	if err != nil {
		return h.ArrayType{}, http.StatusInternalServerError, fmt.Errorf("count query failed: %v", err)
	}

	pages := h.GetPageCount(total, limit)

	reportsArray := h.ConstructArray(
		reports,
		h.APITypeReport,
		total,
		limit,
		offset,
		pages,
		reqURL,
	)

	return reportsArray, http.StatusOK, nil
}
