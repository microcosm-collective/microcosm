package models

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/lib/pq"

	c "github.com/microcosm-collective/microcosm/cache"
	h "github.com/microcosm-collective/microcosm/helpers"
)

const (
	ReportStateOpen      = "open"
	ReportStateReviewing = "reviewing"
	ReportStateClosed    = "closed"
	reportCountCacheTTL  = 3600
)

func reportCountCacheKey(siteID int64) string {
	return fmt.Sprintf("reports_count_%d", siteID)
}

type ReportType struct {
	ID    int64  `json:"id"`
	Key   string `json:"key"`
	Title string `json:"title"`
}

type ReporterType struct {
	ProfileID   int64      `json:"profileId"`
	ProfileName string     `json:"profileName"`
	Type        ReportType `json:"type"`
	Note        string     `json:"note,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
}

type ReportStateChange struct {
	ID        int64               `json:"id"`
	ChangedBy int64               `json:"changedByProfileId"`
	Profile   *ProfileSummaryType `json:"profile"`
	FromState *string             `json:"fromState,omitempty"`
	ToState   string              `json:"toState"`
	Note      string              `json:"note,omitempty"`
	CreatedAt time.Time           `json:"createdAt"`
}

type ReportCommentEdit struct {
	ProfileID int64               `json:"profileId"`
	Profile   *ProfileSummaryType `json:"profile"`
	CreatedAt time.Time           `json:"createdAt"`
}

type Report struct {
	ID           int64               `json:"id"`
	CommentID    int64               `json:"commentId"`
	CommentLink  string              `json:"commentLink"`
	Comment      *CommentSummaryType `json:"comment,omitempty"`
	CommentEdits []ReportCommentEdit `json:"commentEdits"`
	State        string              `json:"state"`
	CreatedAt    time.Time           `json:"createdAt"`
	UpdatedAt    time.Time           `json:"updatedAt"`
	ClosedAt     *time.Time          `json:"closedAt,omitempty"`
	Reporters    []ReporterType      `json:"reporters"`
	StateLog     []ReportStateChange `json:"stateChanges"`
}

type ReportsPage struct {
	Reports h.ArrayType    `json:"reports"`
	Meta    h.CoreMetaType `json:"meta"`
}

func (m Report) GetLink() string { return fmt.Sprintf("/api/v1/reports/%d", m.ID) }

func GetReportTypes() ([]ReportType, int, error) {
	db, err := h.GetConnection()
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	rows, err := db.Query(`SELECT report_type_id, key, title FROM report_types WHERE is_active IS TRUE ORDER BY sort_order, report_type_id`)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	defer rows.Close()
	result := []ReportType{}
	for rows.Next() {
		var t ReportType
		if err := rows.Scan(&t.ID, &t.Key, &t.Title); err != nil {
			return nil, http.StatusInternalServerError, err
		}
		result = append(result, t)
	}
	if err := rows.Err(); err != nil {
		return nil, http.StatusInternalServerError, err
	}
	return result, http.StatusOK, nil
}

// SubmitReport creates a report for a comment, or adds the reporter to the
// existing case. The conflict path leaves an existing report's state intact.
func SubmitReport(siteID, profileID, commentID int64, typeKey, note string) (Report, int, error) {
	if profileID < 1 || commentID < 1 || typeKey == "" {
		return Report{}, http.StatusBadRequest, fmt.Errorf("commentId and type are required")
	}
	tx, err := h.GetTransaction()
	if err != nil {
		return Report{}, http.StatusInternalServerError, err
	}
	defer tx.Rollback()
	var typeID int64
	err = tx.QueryRow(`SELECT report_type_id FROM report_types WHERE key = $1 AND is_active IS TRUE`, typeKey).Scan(&typeID)
	if err == sql.ErrNoRows {
		return Report{}, http.StatusBadRequest, fmt.Errorf("invalid report type")
	}
	if err != nil {
		return Report{}, http.StatusInternalServerError, err
	}
	var exists bool
	err = tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM flags WHERE site_id = $1 AND item_type_id = 4 AND item_id = $2)`, siteID, commentID).Scan(&exists)
	if err != nil {
		return Report{}, http.StatusInternalServerError, err
	}
	if !exists {
		return Report{}, http.StatusNotFound, fmt.Errorf("comment not found on this site")
	}
	var reportID int64
	var reportSiteID int64
	err = tx.QueryRow(`INSERT INTO reports (site_id, comment_id) VALUES ($1, $2)
ON CONFLICT (comment_id) DO NOTHING RETURNING report_id`, siteID, commentID).Scan(&reportID)
	if err == sql.ErrNoRows {
		err = tx.QueryRow(`SELECT report_id, site_id FROM reports WHERE comment_id=$1`, commentID).Scan(&reportID, &reportSiteID)
	} else if err == nil {
		reportSiteID = siteID
	}
	if err != nil {
		return Report{}, http.StatusInternalServerError, err
	}
	if reportSiteID != siteID {
		return Report{}, http.StatusNotFound, fmt.Errorf("comment not found on this site")
	}
	_, err = tx.Exec(`INSERT INTO reporters (report_id, profile_id, report_type_id, note) VALUES ($1, $2, $3, NULLIF($4,''))
ON CONFLICT (report_id, profile_id) DO NOTHING`, reportID, profileID, typeID, note)
	if err != nil {
		return Report{}, http.StatusInternalServerError, err
	}
	var state string
	var createdAt, updatedAt time.Time
	var closedAt sql.NullTime
	err = tx.QueryRow(`SELECT state, created_at, updated_at, closed_at FROM reports WHERE report_id = $1`, reportID).Scan(&state, &createdAt, &updatedAt, &closedAt)
	if err != nil {
		return Report{}, http.StatusInternalServerError, err
	}
	var savedReporter ReporterType
	err = tx.QueryRow(`SELECT rp.profile_id, p.profile_name, rt.report_type_id, rt.key, rt.title, COALESCE(rp.note,''), rp.created_at
FROM reporters rp JOIN profiles p ON p.profile_id=rp.profile_id JOIN report_types rt ON rt.report_type_id=rp.report_type_id
WHERE rp.report_id=$1 AND rp.profile_id=$2`, reportID, profileID).Scan(
		&savedReporter.ProfileID, &savedReporter.ProfileName, &savedReporter.Type.ID, &savedReporter.Type.Key,
		&savedReporter.Type.Title, &savedReporter.Note, &savedReporter.CreatedAt,
	)
	if err != nil {
		return Report{}, http.StatusInternalServerError, err
	}
	if err = tx.Commit(); err != nil {
		return Report{}, http.StatusInternalServerError, err
	}
	c.Delete(reportCountCacheKey(siteID))
	report := Report{ID: reportID, CommentID: commentID, CommentLink: fmt.Sprintf("%s/%d", h.APITypeComment, commentID), State: state, CreatedAt: createdAt, UpdatedAt: updatedAt, Reporters: []ReporterType{savedReporter}, StateLog: []ReportStateChange{}, CommentEdits: []ReportCommentEdit{}}
	if closedAt.Valid {
		report.ClosedAt = &closedAt.Time
	}
	return report, http.StatusOK, nil
}

// GetOpenReportCount returns the number of reports that are not closed for a
// site. The cached value is shared by all callers for that site.
func GetOpenReportCount(siteID int64) (int64, int, error) {
	key := reportCountCacheKey(siteID)
	if count, ok := c.GetInt64(key); ok {
		return count, http.StatusOK, nil
	}
	db, err := h.GetConnection()
	if err != nil {
		return 0, http.StatusInternalServerError, err
	}
	var count int64
	if err := db.QueryRow(`SELECT COUNT(*) FROM reports WHERE site_id=$1 AND state<>$2`, siteID, ReportStateClosed).Scan(&count); err != nil {
		return 0, http.StatusInternalServerError, err
	}
	c.SetInt64(key, count, reportCountCacheTTL)
	return count, http.StatusOK, nil
}

func GetReports(siteID int64, state string, limit, offset int64, singleID int64) ([]Report, int64, int64, int, error) {
	if state != "" && state != ReportStateOpen && state != ReportStateReviewing && state != ReportStateClosed {
		return nil, 0, 0, http.StatusBadRequest, fmt.Errorf("invalid report state")
	}
	db, err := h.GetConnection()
	if err != nil {
		return nil, 0, 0, http.StatusInternalServerError, err
	}
	var total int64
	err = db.QueryRow(`SELECT COUNT(*) FROM reports WHERE site_id=$1 AND ($2='' OR state=$2) AND ($3=0 OR report_id=$3)`, siteID, state, singleID).Scan(&total)
	if err != nil {
		return nil, 0, 0, http.StatusInternalServerError, err
	}
	rows, err := db.Query(`SELECT r.report_id, r.comment_id, r.state, r.created_at, r.updated_at, r.closed_at,
 COALESCE((SELECT jsonb_agg(jsonb_build_object('profileId', p.profile_id, 'profileName', p.profile_name, 'type', jsonb_build_object('id', rt.report_type_id, 'key', rt.key, 'title', rt.title), 'note', NULLIF(rp.note,''), 'createdAt', rp.created_at) ORDER BY rp.created_at)
 FROM reporters rp JOIN profiles p ON p.profile_id=rp.profile_id JOIN report_types rt ON rt.report_type_id=rp.report_type_id WHERE rp.report_id=r.report_id), '[]'::jsonb)
 FROM reports r WHERE r.site_id=$1 AND ($2='' OR r.state=$2) AND ($3=0 OR r.report_id=$3)
 ORDER BY r.created_at DESC, r.report_id DESC LIMIT $4 OFFSET $5`, siteID, state, singleID, limit, offset)
	if err != nil {
		return nil, 0, 0, http.StatusInternalServerError, err
	}
	defer rows.Close()
	items := []Report{}
	for rows.Next() {
		var item Report
		var closedAt sql.NullTime
		var reporterJSON []byte
		if err := rows.Scan(&item.ID, &item.CommentID, &item.State, &item.CreatedAt, &item.UpdatedAt, &closedAt, &reporterJSON); err != nil {
			return nil, 0, 0, http.StatusInternalServerError, err
		}
		item.CommentLink = fmt.Sprintf("%s/%d", h.APITypeComment, item.CommentID)
		if closedAt.Valid {
			item.ClosedAt = &closedAt.Time
		}
		if err := json.Unmarshal(reporterJSON, &item.Reporters); err != nil {
			return nil, 0, 0, http.StatusInternalServerError, err
		}
		item.Reporters = nonNilReporters(item.Reporters)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, 0, http.StatusInternalServerError, err
	}
	if len(items) > 0 {
		reportIDs := make([]int64, 0, len(items))
		commentIDs := make([]int64, 0, len(items))
		for _, item := range items {
			reportIDs = append(reportIDs, item.ID)
			commentIDs = append(commentIDs, item.CommentID)
		}
		changes, err := db.Query(`SELECT report_id, report_state_change_id, changed_by_profile_id, from_state, to_state, COALESCE(note,''), created_at FROM report_state_changes WHERE report_id = ANY($1) ORDER BY created_at, report_state_change_id`, pq.Array(reportIDs))
		if err != nil {
			return nil, 0, 0, http.StatusInternalServerError, err
		}
		profileIDs := map[int64]struct{}{}
		for changes.Next() {
			var change ReportStateChange
			var reportID int64
			var from sql.NullString
			if err := changes.Scan(&reportID, &change.ID, &change.ChangedBy, &from, &change.ToState, &change.Note, &change.CreatedAt); err != nil {
				changes.Close()
				return nil, 0, 0, http.StatusInternalServerError, err
			}
			if from.Valid {
				change.FromState = &from.String
			}
			for i := range items {
				if items[i].ID == reportID {
					items[i].StateLog = append(items[i].StateLog, change)
					break
				}
			}
			profileIDs[change.ChangedBy] = struct{}{}
		}
		if err := changes.Err(); err != nil {
			changes.Close()
			return nil, 0, 0, http.StatusInternalServerError, err
		}
		changes.Close()
		profiles := map[int64]ProfileSummaryType{}
		for profileID := range profileIDs {
			profile, status, err := GetProfileSummary(siteID, profileID)
			if err != nil {
				return nil, 0, 0, status, err
			}
			profiles[profileID] = profile
		}
		for i := range items {
			for j := range items[i].StateLog {
				profile := profiles[items[i].StateLog[j].ChangedBy]
				items[i].StateLog[j].Profile = &profile
			}
			if items[i].StateLog == nil {
				items[i].StateLog = []ReportStateChange{}
			}
		}
		comments := make(map[int64]CommentSummaryType, len(commentIDs))
		for _, commentID := range commentIDs {
			comment, status, err := GetCommentSummary(siteID, commentID)
			if err != nil {
				return nil, 0, 0, status, err
			}
			comments[commentID] = comment
		}
		for i := range items {
			comment := comments[items[i].CommentID]
			items[i].Comment = &comment
			items[i].CommentEdits = []ReportCommentEdit{}
		}
		edits, err := db.Query(`SELECT r.report_id, rev.profile_id, rev.created FROM reports r JOIN revisions rev ON rev.comment_id = r.comment_id WHERE r.report_id = ANY($1) AND rev.created > r.created_at ORDER BY rev.created, rev.revision_id`, pq.Array(reportIDs))
		if err != nil {
			return nil, 0, 0, http.StatusInternalServerError, err
		}
		editProfileIDs := map[int64]struct{}{}
		for edits.Next() {
			var reportID int64
			var edit ReportCommentEdit
			if err := edits.Scan(&reportID, &edit.ProfileID, &edit.CreatedAt); err != nil {
				edits.Close()
				return nil, 0, 0, http.StatusInternalServerError, err
			}
			for i := range items {
				if items[i].ID == reportID {
					items[i].CommentEdits = append(items[i].CommentEdits, edit)
					break
				}
			}
			editProfileIDs[edit.ProfileID] = struct{}{}
		}
		if err := edits.Err(); err != nil {
			edits.Close()
			return nil, 0, 0, http.StatusInternalServerError, err
		}
		edits.Close()
		editProfiles := map[int64]ProfileSummaryType{}
		for profileID := range editProfileIDs {
			profile, status, err := GetProfileSummary(siteID, profileID)
			if err != nil {
				return nil, 0, 0, status, err
			}
			editProfiles[profileID] = profile
		}
		for i := range items {
			for j := range items[i].CommentEdits {
				profile := editProfiles[items[i].CommentEdits[j].ProfileID]
				items[i].CommentEdits[j].Profile = &profile
			}
		}
	}
	pages := int64(0)
	if limit > 0 {
		pages = (total + limit - 1) / limit
	}
	return items, total, pages, http.StatusOK, nil
}

func nonNilReporters(reporters []ReporterType) []ReporterType {
	if reporters == nil {
		return []ReporterType{}
	}
	return reporters
}

func UpdateReportState(siteID, reportID, actorID int64, nextState, note string) (int, error) {
	if nextState != ReportStateOpen && nextState != ReportStateReviewing && nextState != ReportStateClosed {
		return http.StatusBadRequest, fmt.Errorf("invalid report state")
	}
	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()
	var previous string
	err = tx.QueryRow(`SELECT state FROM reports WHERE site_id=$1 AND report_id=$2 FOR UPDATE`, siteID, reportID).Scan(&previous)
	if err == sql.ErrNoRows {
		return http.StatusNotFound, fmt.Errorf("report not found")
	}
	if err != nil {
		return http.StatusInternalServerError, err
	}
	if previous == nextState {
		if err := tx.Commit(); err != nil {
			return http.StatusInternalServerError, err
		}
		return http.StatusOK, nil
	}
	var closedAt interface{}
	if nextState == ReportStateClosed {
		closedAt = time.Now()
	}
	_, err = tx.Exec(`UPDATE reports SET state=$3, updated_at=now(), closed_at=$4 WHERE site_id=$1 AND report_id=$2`, siteID, reportID, nextState, closedAt)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	_, err = tx.Exec(`INSERT INTO report_state_changes (report_id, changed_by_profile_id, from_state, to_state, note) VALUES ($1,$2,$3,$4,NULLIF($5,''))`, reportID, actorID, previous, nextState, note)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	if err = tx.Commit(); err != nil {
		return http.StatusInternalServerError, err
	}
	c.Delete(reportCountCacheKey(siteID))
	return http.StatusOK, nil
}
