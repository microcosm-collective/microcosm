package models

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/golang/glog"
	"github.com/lib/pq"

	c "github.com/microcosm-collective/microcosm/cache"
	h "github.com/microcosm-collective/microcosm/helpers"
)

// ModeratorActionsType is a collection of moderator actions
type ModeratorActionsType struct {
	ModeratorActions h.ArrayType    `json:"moderatorActions"`
	Meta             h.CoreMetaType `json:"meta"`
}

// ModeratorActionType encapsulates a moderator action
type ModeratorActionType struct {
	ID                   int64        `json:"id"`
	ModeratorActionTypeID int64       `json:"moderatorActionTypeId"`
	ModeratorProfileID   int64        `json:"moderatorProfileId"`
	CommentID            int64        `json:"commentId"`
	Created              time.Time    `json:"-"`
	ValidFrom            time.Time    `json:"validFrom"`
	ExpiresNullable      pq.NullTime  `json:"-"`
	Expires              string       `json:"expires,omitempty"`

	// Populated fields
	ModeratorActionType ModeratorActionTypeType `json:"moderatorActionType,omitempty"`
	ModeratorProfile    ProfileSummaryType      `json:"moderatorProfile,omitempty"`
	Comment             CommentSummaryType      `json:"comment,omitempty"`

	Meta                ModeratorActionMetaType `json:"meta"`
}

// ModeratorActionMetaType is the meta struct of a moderator action
type ModeratorActionMetaType struct {
	h.CreatedType
	h.CoreMetaType
}

// Validate returns true if a moderator action is valid
func (m *ModeratorActionType) Validate(siteID int64) (int, error) {
	if m.ModeratorActionTypeID <= 0 {
		return http.StatusBadRequest, fmt.Errorf("moderatorActionTypeId is required")
	}

	if m.ModeratorProfileID <= 0 {
		return http.StatusBadRequest, fmt.Errorf("moderatorProfileId is required")
	}

	if m.CommentID <= 0 {
		return http.StatusBadRequest, fmt.Errorf("commentId is required")
	}

	// Verify the comment exists
	_, status, err := GetCommentSummary(siteID, m.CommentID)
	if err != nil {
		return status, fmt.Errorf("invalid commentId: %v", err)
	}

	// Verify the moderator action type exists
	_, status, err = GetModeratorActionType(m.ModeratorActionTypeID)
	if err != nil {
		return status, fmt.Errorf("invalid moderatorActionTypeId: %v", err)
	}

	// Verify the moderator profile exists
	_, status, err = GetProfileSummary(siteID, m.ModeratorProfileID)
	if err != nil {
		return status, fmt.Errorf("invalid moderatorProfileId: %v", err)
	}

	return http.StatusOK, nil
}

// Insert saves a moderator action
func (m *ModeratorActionType) Insert(siteID int64) (int, error) {
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

	if m.ValidFrom.IsZero() {
		m.ValidFrom = now
	}

	var insertID int64
	err = tx.QueryRow(`
INSERT INTO moderator_actions (
    moderator_action_type_id, moderator_profile_id, comment_id, created, valid_from, expires
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING moderator_action_id`,
		m.ModeratorActionTypeID,
		m.ModeratorProfileID,
		m.CommentID,
		now,
		m.ValidFrom,
		m.ExpiresNullable,
	).Scan(&insertID)

	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error inserting data: %v", err)
	}

	m.ID = insertID

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("transaction failed: %v", err)
	}

	PurgeCache(h.ItemTypes[h.ItemTypeModeratorAction], m.ID)
	PurgeCache(h.ItemTypes[h.ItemTypeComment], m.CommentID)

	return http.StatusOK, nil
}

// Update saves changes to a moderator action
func (m *ModeratorActionType) Update(siteID int64) (int, error) {
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
UPDATE moderator_actions
   SET moderator_action_type_id = $2,
       moderator_profile_id = $3,
       comment_id = $4,
       valid_from = $5,
       expires = $6
 WHERE moderator_action_id = $1`,
		m.ID,
		m.ModeratorActionTypeID,
		m.ModeratorProfileID,
		m.CommentID,
		m.ValidFrom,
		m.ExpiresNullable,
	)

	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error updating data: %v", err)
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("transaction failed: %v", err)
	}

	PurgeCache(h.ItemTypes[h.ItemTypeModeratorAction], m.ID)
	PurgeCache(h.ItemTypes[h.ItemTypeComment], m.CommentID)

	return http.StatusOK, nil
}

// Delete removes a moderator action
func (m *ModeratorActionType) Delete() (int, error) {
	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
DELETE FROM moderator_actions
 WHERE moderator_action_id = $1`,
		m.ID,
	)

	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error deleting data: %v", err)
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("transaction failed: %v", err)
	}

	PurgeCache(h.ItemTypes[h.ItemTypeModeratorAction], m.ID)
	PurgeCache(h.ItemTypes[h.ItemTypeComment], m.CommentID)

	return http.StatusOK, nil
}

// Hydrate populates a partially populated struct
func (m *ModeratorActionType) Hydrate(siteID int64) (int, error) {
	// Get moderator action type
	actionType, status, err := GetModeratorActionType(m.ModeratorActionTypeID)
	if err != nil {
		return status, err
	}
	m.ModeratorActionType = actionType

	// Get moderator profile
	profile, status, err := GetProfileSummary(siteID, m.ModeratorProfileID)
	if err != nil {
		return status, err
	}
	m.ModeratorProfile = profile
	m.Meta.CreatedBy = profile

	// Get comment
	comment, status, err := GetCommentSummary(siteID, m.CommentID)
	if err != nil {
		return status, err
	}
	m.Comment = comment

	return http.StatusOK, nil
}

// GetModeratorAction returns a moderator action
func GetModeratorAction(siteID int64, actionID int64) (ModeratorActionType, int, error) {
	if actionID == 0 {
		return ModeratorActionType{}, http.StatusNotFound, fmt.Errorf("moderator action not found")
	}

	// Try to get from cache
	mcKey := fmt.Sprintf("moderatoraction_%d", actionID)
	if val, ok := c.Get(mcKey, ModeratorActionType{}); ok {
		m := val.(ModeratorActionType)
		status, err := m.Hydrate(siteID)
		if err != nil {
			return m, status, err
		}
		return m, http.StatusOK, nil
	}

	db, err := h.GetConnection()
	if err != nil {
		return ModeratorActionType{}, http.StatusInternalServerError, err
	}

	m := ModeratorActionType{}
	err = db.QueryRow(`
SELECT ma.moderator_action_id, ma.moderator_action_type_id, ma.moderator_profile_id, 
       ma.comment_id, ma.created, ma.valid_from, ma.expires
  FROM moderator_actions ma
 WHERE ma.moderator_action_id = $1`,
		actionID,
	).Scan(
		&m.ID,
		&m.ModeratorActionTypeID,
		&m.ModeratorProfileID,
		&m.CommentID,
		&m.Created,
		&m.ValidFrom,
		&m.ExpiresNullable,
	)

	if err == sql.ErrNoRows {
		return ModeratorActionType{}, http.StatusNotFound, fmt.Errorf("moderator action with ID %d not found", actionID)
	} else if err != nil {
		return ModeratorActionType{}, http.StatusInternalServerError, fmt.Errorf("database query failed: %v", err)
	}

	m.Meta.Created = m.Created
	m.Meta.CreatedByID = m.ModeratorProfileID

	if m.ExpiresNullable.Valid {
		m.Expires = m.ExpiresNullable.Time.Format(time.RFC3339)
	}

	m.Meta.Links = []h.LinkType{
		h.GetLink("self", "", h.ItemTypeModeratorAction, m.ID),
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

// GetModeratorActions returns a collection of moderator actions
func GetModeratorActions(siteID int64, reqURL *url.URL) (h.ArrayType, int, error) {
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
SELECT ma.moderator_action_id, ma.moderator_action_type_id, ma.moderator_profile_id, 
       ma.comment_id, ma.created, ma.valid_from, ma.expires
  FROM moderator_actions ma
 ORDER BY ma.created DESC
 LIMIT $1 OFFSET $2`,
		limit,
		offset,
	)
	if err != nil {
		return h.ArrayType{}, http.StatusInternalServerError, fmt.Errorf("database query failed: %v", err)
	}
	defer rows.Close()

	var actions []interface{}
	for rows.Next() {
		m := ModeratorActionType{}
		err = rows.Scan(
			&m.ID,
			&m.ModeratorActionTypeID,
			&m.ModeratorProfileID,
			&m.CommentID,
			&m.Created,
			&m.ValidFrom,
			&m.ExpiresNullable,
		)
		if err != nil {
			return h.ArrayType{}, http.StatusInternalServerError, fmt.Errorf("row parsing error: %v", err)
		}

		m.Meta.Created = m.Created
		m.Meta.CreatedByID = m.ModeratorProfileID

		if m.ExpiresNullable.Valid {
			m.Expires = m.ExpiresNullable.Time.Format(time.RFC3339)
		}

		m.Meta.Links = []h.LinkType{
			h.GetLink("self", "", h.ItemTypeModeratorAction, m.ID),
			h.GetLink("comment", "", h.ItemTypeComment, m.CommentID),
		}

		// Hydrate with related data
		_, err := m.Hydrate(siteID)
		if err != nil {
			glog.Warningf("Error hydrating moderator action %d: %v", m.ID, err)
		}

		actions = append(actions, m)
	}
	err = rows.Err()
	if err != nil {
		return h.ArrayType{}, http.StatusInternalServerError, fmt.Errorf("error fetching rows: %v", err)
	}

	// Get total count
	var total int64
	err = db.QueryRow(`SELECT COUNT(*) FROM moderator_actions`).Scan(&total)
	if err != nil {
		return h.ArrayType{}, http.StatusInternalServerError, fmt.Errorf("count query failed: %v", err)
	}

	pages := h.GetPageCount(total, limit)

	actionsArray := h.ConstructArray(
		actions,
		h.APITypeModeratorAction,
		total,
		limit,
		offset,
		pages,
		reqURL,
	)

	return actionsArray, http.StatusOK, nil
}

// GetModeratorActionsByComment returns moderator actions for a specific comment
func GetModeratorActionsByComment(siteID int64, commentID int64, reqURL *url.URL) (h.ArrayType, int, error) {
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
SELECT ma.moderator_action_id, ma.moderator_action_type_id, ma.moderator_profile_id, 
       ma.created, ma.valid_from, ma.expires
  FROM moderator_actions ma
 WHERE ma.comment_id = $1
 ORDER BY ma.created DESC
 LIMIT $2 OFFSET $3`,
		commentID,
		limit,
		offset,
	)
	if err != nil {
		return h.ArrayType{}, http.StatusInternalServerError, fmt.Errorf("database query failed: %v", err)
	}
	defer rows.Close()

	var actions []interface{}
	for rows.Next() {
		m := ModeratorActionType{}
		m.CommentID = commentID
		err = rows.Scan(
			&m.ID,
			&m.ModeratorActionTypeID,
			&m.ModeratorProfileID,
			&m.Created,
			&m.ValidFrom,
			&m.ExpiresNullable,
		)
		if err != nil {
			return h.ArrayType{}, http.StatusInternalServerError, fmt.Errorf("row parsing error: %v", err)
		}

		m.Meta.Created = m.Created
		m.Meta.CreatedByID = m.ModeratorProfileID

		if m.ExpiresNullable.Valid {
			m.Expires = m.ExpiresNullable.Time.Format(time.RFC3339)
		}

		m.Meta.Links = []h.LinkType{
			h.GetLink("self", "", h.ItemTypeModeratorAction, m.ID),
			h.GetLink("comment", "", h.ItemTypeComment, m.CommentID),
		}

		// Hydrate with related data
		_, err := m.Hydrate(siteID)
		if err != nil {
			glog.Warningf("Error hydrating moderator action %d: %v", m.ID, err)
		}

		actions = append(actions, m)
	}
	err = rows.Err()
	if err != nil {
		return h.ArrayType{}, http.StatusInternalServerError, fmt.Errorf("error fetching rows: %v", err)
	}

	// Get total count
	var total int64
	err = db.QueryRow(`
SELECT COUNT(*) 
  FROM moderator_actions 
 WHERE comment_id = $1`,
		commentID,
	).Scan(&total)
	if err != nil {
		return h.ArrayType{}, http.StatusInternalServerError, fmt.Errorf("count query failed: %v", err)
	}

	pages := h.GetPageCount(total, limit)

	actionsArray := h.ConstructArray(
		actions,
		h.APITypeModeratorAction,
		total,
		limit,
		offset,
		pages,
		reqURL,
	)

	return actionsArray, http.StatusOK, nil
}

// GetActiveModeratorActionsByComment returns active moderator actions for a specific comment
func GetActiveModeratorActionsByComment(siteID int64, commentID int64) ([]ModeratorActionType, int, error) {
	db, err := h.GetConnection()
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	now := time.Now()

	rows, err := db.Query(`
SELECT ma.moderator_action_id, ma.moderator_action_type_id, ma.moderator_profile_id, 
       ma.created, ma.valid_from, ma.expires
  FROM moderator_actions ma
 WHERE ma.comment_id = $1
   AND ma.valid_from <= $2
   AND (ma.expires IS NULL OR ma.expires > $2)
 ORDER BY ma.created DESC`,
		commentID,
		now,
	)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("database query failed: %v", err)
	}
	defer rows.Close()

	var actions []ModeratorActionType
	for rows.Next() {
		m := ModeratorActionType{}
		m.CommentID = commentID
		err = rows.Scan(
			&m.ID,
			&m.ModeratorActionTypeID,
			&m.ModeratorProfileID,
			&m.Created,
			&m.ValidFrom,
			&m.ExpiresNullable,
		)
		if err != nil {
			return nil, http.StatusInternalServerError, fmt.Errorf("row parsing error: %v", err)
		}

		m.Meta.Created = m.Created
		m.Meta.CreatedByID = m.ModeratorProfileID

		if m.ExpiresNullable.Valid {
			m.Expires = m.ExpiresNullable.Time.Format(time.RFC3339)
		}

		m.Meta.Links = []h.LinkType{
			h.GetLink("self", "", h.ItemTypeModeratorAction, m.ID),
			h.GetLink("comment", "", h.ItemTypeComment, m.CommentID),
		}

		// Hydrate with related data
		_, err := m.Hydrate(siteID)
		if err != nil {
			glog.Warningf("Error hydrating moderator action %d: %v", m.ID, err)
		}

		actions = append(actions, m)
	}
	err = rows.Err()
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("error fetching rows: %v", err)
	}

	return actions, http.StatusOK, nil
}
