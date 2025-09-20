package models

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/url"

	c "github.com/microcosm-collective/microcosm/cache"
	h "github.com/microcosm-collective/microcosm/helpers"
)

// ReportReasonsType is a collection of report reasons
type ReportReasonsType struct {
	ReportReasons h.ArrayType    `json:"reportReasons"`
	Meta          h.CoreMetaType `json:"meta"`
}

// ReportReasonType encapsulates a report reason
type ReportReasonType struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`

	Meta h.CoreMetaType `json:"meta"`
}

// Validate returns true if a report reason is valid
func (m *ReportReasonType) Validate() (int, error) {
	if m.Title == "" {
		return http.StatusBadRequest, fmt.Errorf("title is required")
	}

	return http.StatusOK, nil
}

// Insert saves a report reason
func (m *ReportReasonType) Insert() (int, error) {
	status, err := m.Validate()
	if err != nil {
		return status, err
	}

	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	var insertID int64
	err = tx.QueryRow(`
INSERT INTO report_reasons (
    title, description
) VALUES (
    $1, $2
) RETURNING report_reason_id`,
		m.Title,
		m.Description,
	).Scan(&insertID)

	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error inserting data: %v", err)
	}

	m.ID = insertID

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("transaction failed: %v", err)
	}

	PurgeCache(h.ItemTypes[h.ItemTypeReportReason], m.ID)

	return http.StatusOK, nil
}

// Update saves changes to a report reason
func (m *ReportReasonType) Update() (int, error) {
	status, err := m.Validate()
	if err != nil {
		return status, err
	}

	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
UPDATE report_reasons
   SET title = $2,
       description = $3
 WHERE report_reason_id = $1`,
		m.ID,
		m.Title,
		m.Description,
	)

	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error updating data: %v", err)
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("transaction failed: %v", err)
	}

	PurgeCache(h.ItemTypes[h.ItemTypeReportReason], m.ID)

	return http.StatusOK, nil
}

// Delete removes a report reason
func (m *ReportReasonType) Delete() (int, error) {
	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
DELETE FROM report_reasons
 WHERE report_reason_id = $1`,
		m.ID,
	)

	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error deleting data: %v", err)
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("transaction failed: %v", err)
	}

	PurgeCache(h.ItemTypes[h.ItemTypeReportReason], m.ID)

	return http.StatusOK, nil
}

// GetReportReason returns a report reason
func GetReportReason(reportReasonID int64) (ReportReasonType, int, error) {
	if reportReasonID == 0 {
		return ReportReasonType{}, http.StatusNotFound, fmt.Errorf("report reason not found")
	}

	// Try to get from cache
	mcKey := fmt.Sprintf("reportreason_%d", reportReasonID)
	if val, ok := c.Get(mcKey, ReportReasonType{}); ok {
		m := val.(ReportReasonType)
		return m, http.StatusOK, nil
	}

	db, err := h.GetConnection()
	if err != nil {
		return ReportReasonType{}, http.StatusInternalServerError, err
	}

	m := ReportReasonType{}
	err = db.QueryRow(`
SELECT report_reason_id, title, description
  FROM report_reasons
 WHERE report_reason_id = $1`,
		reportReasonID,
	).Scan(
		&m.ID,
		&m.Title,
		&m.Description,
	)

	if err == sql.ErrNoRows {
		return ReportReasonType{}, http.StatusNotFound, fmt.Errorf("report reason with ID %d not found", reportReasonID)
	} else if err != nil {
		return ReportReasonType{}, http.StatusInternalServerError, fmt.Errorf("database query failed: %v", err)
	}

	m.Meta.Links = []h.LinkType{
		h.GetLink("self", "", h.ItemTypeReportReason, m.ID),
	}

	// Update cache
	c.Set(mcKey, m, 60*60*24) // Cache for 24 hours

	return m, http.StatusOK, nil
}

// GetReportReasons returns a collection of report reasons
func GetReportReasons(reqURL *url.URL) (h.ArrayType, int, error) {
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
SELECT report_reason_id, title, description
  FROM report_reasons
 ORDER BY title
 LIMIT $1 OFFSET $2`,
		limit,
		offset,
	)
	if err != nil {
		return h.ArrayType{}, http.StatusInternalServerError, fmt.Errorf("database query failed: %v", err)
	}
	defer rows.Close()

	var reportReasons []interface{}
	for rows.Next() {
		m := ReportReasonType{}
		err = rows.Scan(
			&m.ID,
			&m.Title,
			&m.Description,
		)
		if err != nil {
			return h.ArrayType{}, http.StatusInternalServerError, fmt.Errorf("row parsing error: %v", err)
		}

		m.Meta.Links = []h.LinkType{
			h.GetLink("self", "", h.ItemTypeReportReason, m.ID),
		}

		reportReasons = append(reportReasons, m)
	}
	err = rows.Err()
	if err != nil {
		return h.ArrayType{}, http.StatusInternalServerError, fmt.Errorf("error fetching rows: %v", err)
	}

	// Get total count
	var total int64
	err = db.QueryRow(`SELECT COUNT(*) FROM report_reasons`).Scan(&total)
	if err != nil {
		return h.ArrayType{}, http.StatusInternalServerError, fmt.Errorf("count query failed: %v", err)
	}

	pages := h.GetPageCount(total, limit)

	reportReasonsArray := h.ConstructArray(
		reportReasons,
		h.APITypeReportReason,
		total,
		limit,
		offset,
		pages,
		reqURL,
	)

	return reportReasonsArray, http.StatusOK, nil
}
