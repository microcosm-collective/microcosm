package models

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/url"

	c "github.com/microcosm-collective/microcosm/cache"
	h "github.com/microcosm-collective/microcosm/helpers"
)

// ModeratorActionTypesType is a collection of moderator action types
type ModeratorActionTypesType struct {
	ModeratorActionTypes h.ArrayType    `json:"moderatorActionTypes"`
	Meta                 h.CoreMetaType `json:"meta"`
}

// ModeratorActionTypeType encapsulates a moderator action type
type ModeratorActionTypeType struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`

	Meta h.CoreMetaType `json:"meta"`
}

// Validate returns true if a moderator action type is valid
func (m *ModeratorActionTypeType) Validate() (int, error) {
	if m.Title == "" {
		return http.StatusBadRequest, fmt.Errorf("title is required")
	}

	return http.StatusOK, nil
}

// Insert saves a moderator action type
func (m *ModeratorActionTypeType) Insert() (int, error) {
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
INSERT INTO moderator_action_types (
    title, description
) VALUES (
    $1, $2
) RETURNING moderator_action_type_id`,
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

	PurgeCache(h.ItemTypes[h.ItemTypeModeratorActionType], m.ID)

	return http.StatusOK, nil
}

// Update saves changes to a moderator action type
func (m *ModeratorActionTypeType) Update() (int, error) {
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
UPDATE moderator_action_types
   SET title = $2,
       description = $3
 WHERE moderator_action_type_id = $1`,
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

	PurgeCache(h.ItemTypes[h.ItemTypeModeratorActionType], m.ID)

	return http.StatusOK, nil
}

// Delete removes a moderator action type
func (m *ModeratorActionTypeType) Delete() (int, error) {
	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
DELETE FROM moderator_action_types
 WHERE moderator_action_type_id = $1`,
		m.ID,
	)

	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error deleting data: %v", err)
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("transaction failed: %v", err)
	}

	PurgeCache(h.ItemTypes[h.ItemTypeModeratorActionType], m.ID)

	return http.StatusOK, nil
}

// GetModeratorActionType returns a moderator action type
func GetModeratorActionType(actionTypeID int64) (ModeratorActionTypeType, int, error) {
	if actionTypeID == 0 {
		return ModeratorActionTypeType{}, http.StatusNotFound, fmt.Errorf("moderator action type not found")
	}

	// Try to get from cache
	mcKey := fmt.Sprintf("moderatoractiontype_%d", actionTypeID)
	if val, ok := c.Get(mcKey, ModeratorActionTypeType{}); ok {
		m := val.(ModeratorActionTypeType)
		return m, http.StatusOK, nil
	}

	db, err := h.GetConnection()
	if err != nil {
		return ModeratorActionTypeType{}, http.StatusInternalServerError, err
	}

	m := ModeratorActionTypeType{}
	err = db.QueryRow(`
SELECT moderator_action_type_id, title, description
  FROM moderator_action_types
 WHERE moderator_action_type_id = $1`,
		actionTypeID,
	).Scan(
		&m.ID,
		&m.Title,
		&m.Description,
	)

	if err == sql.ErrNoRows {
		return ModeratorActionTypeType{}, http.StatusNotFound, fmt.Errorf("moderator action type with ID %d not found", actionTypeID)
	} else if err != nil {
		return ModeratorActionTypeType{}, http.StatusInternalServerError, fmt.Errorf("database query failed: %v", err)
	}

	m.Meta.Links = []h.LinkType{
		h.GetLink("self", "", h.ItemTypeModeratorActionType, m.ID),
	}

	// Update cache
	c.Set(mcKey, m, 60*60*24) // Cache for 24 hours

	return m, http.StatusOK, nil
}

// GetModeratorActionTypes returns a collection of moderator action types
func GetModeratorActionTypes(reqURL *url.URL) (h.ArrayType, int, error) {
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
SELECT moderator_action_type_id, title, description
  FROM moderator_action_types
 ORDER BY title
 LIMIT $1 OFFSET $2`,
		limit,
		offset,
	)
	if err != nil {
		return h.ArrayType{}, http.StatusInternalServerError, fmt.Errorf("database query failed: %v", err)
	}
	defer rows.Close()

	var actionTypes []interface{}
	for rows.Next() {
		m := ModeratorActionTypeType{}
		err = rows.Scan(
			&m.ID,
			&m.Title,
			&m.Description,
		)
		if err != nil {
			return h.ArrayType{}, http.StatusInternalServerError, fmt.Errorf("row parsing error: %v", err)
		}

		m.Meta.Links = []h.LinkType{
			h.GetLink("self", "", h.ItemTypeModeratorActionType, m.ID),
		}

		actionTypes = append(actionTypes, m)
	}
	err = rows.Err()
	if err != nil {
		return h.ArrayType{}, http.StatusInternalServerError, fmt.Errorf("error fetching rows: %v", err)
	}

	// Get total count
	var total int64
	err = db.QueryRow(`SELECT COUNT(*) FROM moderator_action_types`).Scan(&total)
	if err != nil {
		return h.ArrayType{}, http.StatusInternalServerError, fmt.Errorf("count query failed: %v", err)
	}

	pages := h.GetPageCount(total, limit)

	actionTypesArray := h.ConstructArray(
		actionTypes,
		h.APITypeModeratorActionType,
		total,
		limit,
		offset,
		pages,
		reqURL,
	)

	return actionTypesArray, http.StatusOK, nil
}
