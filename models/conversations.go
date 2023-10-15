package models

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/lib/pq"

	c "github.com/microcosm-cc/microcosm/cache"
	h "github.com/microcosm-cc/microcosm/helpers"
)

// ConversationsType is a collection of conversations
type ConversationsType struct {
	Conversations h.ArrayType    `json:"conversations"`
	Meta          h.CoreMetaType `json:"meta"`
}

// ConversationSummaryType is a summary of a conversation
type ConversationSummaryType struct {
	ItemSummary
	ItemSummaryMeta
}

// ConversationType is a conversation
type ConversationType struct {
	ItemDetail
	ItemDetailCommentsAndMeta
}

// Validate returns true if a conversation is valid
func (m *ConversationType) Validate(
	siteID int64,
	profileID int64,
	exists bool,
	isImport bool,
) (
	int,
	error,
) {
	preventShouting := true
	m.Title = CleanSentence(m.Title, preventShouting)

	if strings.Trim(m.Title, " ") == "" {
		return http.StatusBadRequest, fmt.Errorf("title is a required field")
	}

	if !exists {
		// Does the Microcosm specified exist on this site?
		_, status, err := GetMicrocosmSummary(
			siteID,
			m.MicrocosmID,
			profileID,
		)
		if err != nil {
			return status, err
		}
	}

	if exists && !isImport {
		if m.ID < 1 {
			return http.StatusBadRequest, fmt.Errorf(
				"the supplied ID ('%d') cannot be zero or negative",
				m.ID,
			)
		}

		if strings.Trim(m.Meta.EditReason, " ") == "" ||
			len(m.Meta.EditReason) == 0 {

			return http.StatusBadRequest,
				fmt.Errorf("you must provide a reason for the update")

		}

		m.Meta.EditReason = CleanSentence(m.Meta.EditReason, preventShouting)
	}

	if m.MicrocosmID <= 0 {
		return http.StatusBadRequest,
			fmt.Errorf("you must specify a Microcosm ID")
	}

	m.Meta.Flags.SetVisible()

	return http.StatusOK, nil
}

// Hydrate populates a partially populated struct
func (m *ConversationType) Hydrate(siteID int64) (int, error) {

	profile, status, err := GetProfileSummary(siteID, m.Meta.CreatedByID)
	if err != nil {
		return status, err
	}
	m.Meta.CreatedBy = profile

	if m.Meta.EditedByNullable.Valid {
		profile, status, err :=
			GetProfileSummary(siteID, m.Meta.EditedByNullable.Int64)
		if err != nil {
			return status, err
		}
		m.Meta.EditedBy = profile
	}

	if status, err := m.FetchBreadcrumb(); err != nil {
		return status, err
	}

	return http.StatusOK, nil
}

// Hydrate populates a partially populated struct
func (m *ConversationSummaryType) Hydrate(
	siteID int64,
) (
	int,
	error,
) {
	profile, status, err := GetProfileSummary(siteID, m.Meta.CreatedByID)
	if err != nil {
		return status, err
	}
	m.Meta.CreatedBy = profile

	switch m.LastComment.(type) {
	case LastComment:
		lastComment := m.LastComment.(LastComment)

		profile, status, err =
			GetProfileSummary(siteID, lastComment.CreatedByID)
		if err != nil {
			return status, err
		}

		lastComment.CreatedBy = profile
		m.LastComment = lastComment
	}

	if status, err := m.FetchBreadcrumb(); err != nil {
		return status, err
	}

	return http.StatusOK, nil
}

// Insert saves a conversation
func (m *ConversationType) Insert(siteID int64, profileID int64) (int, error) {
	status, err := m.Validate(siteID, profileID, false, false)
	if err != nil {
		return status, err
	}

	dupeKey := "dupe_" + h.MD5Sum(
		strconv.FormatInt(m.MicrocosmID, 10)+
			m.Title+
			strconv.FormatInt(m.Meta.CreatedByID, 10),
	)
	v, ok := c.GetInt64(dupeKey)
	if ok {
		m.ID = v
		return http.StatusOK, nil
	}

	status, err = m.insert(siteID, profileID)
	if status == http.StatusOK {
		// 5 minute dupe check
		c.SetInt64(dupeKey, m.ID, 60*5)
	}

	return status, err
}

// Import saves a conversation with duplicate checking
func (m *ConversationType) Import(siteID int64, profileID int64) (int, error) {
	status, err := m.Validate(siteID, profileID, true, true)
	if err != nil {
		return status, err
	}

	return m.insert(siteID, profileID)
}

func (m *ConversationType) insert(siteID int64, profileID int64) (int, error) {
	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	var insertID int64
	err = tx.QueryRow(`--Create Conversation
INSERT INTO conversations (
    microcosm_id, title, created, created_by, view_count,
    is_deleted, is_moderated, is_open, is_sticky
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9
) RETURNING conversation_id`,
		m.MicrocosmID,
		m.Title,
		m.Meta.Created,
		m.Meta.CreatedByID,
		m.ViewCount,

		m.Meta.Flags.Deleted,
		m.Meta.Flags.Moderated,
		m.Meta.Flags.Open,
		m.Meta.Flags.Sticky,
	).Scan(
		&insertID,
	)
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf(
				"error inserting data and returning ID: %v",
				err.Error(),
			)
	}

	m.ID = insertID

	err = IncrementMicrocosmItemCount(tx, m.MicrocosmID)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("transaction failed: %v", err.Error())
	}

	PurgeCache(h.ItemTypes[h.ItemTypeConversation], m.ID)
	PurgeCache(h.ItemTypes[h.ItemTypeMicrocosm], m.MicrocosmID)

	return http.StatusOK, nil
}

// Update updates a conversation
func (m *ConversationType) Update(siteID int64, profileID int64) (int, error) {

	status, err := m.Validate(siteID, profileID, true, false)
	if err != nil {
		return status, err
	}

	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`--Update Conversation
UPDATE conversations
   SET microcosm_id = $2,
       title = $3,
       edited = $4,
       edited_by = $5,
       edit_reason = $6
 WHERE conversation_id = $1`,
		m.ID,
		m.MicrocosmID,
		m.Title,
		m.Meta.EditedNullable,
		m.Meta.EditedByNullable,
		m.Meta.EditReason,
	)
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("update failed: %v", err.Error())
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("transaction failed: %v", err.Error())
	}

	PurgeCache(h.ItemTypes[h.ItemTypeConversation], m.ID)
	PurgeCache(h.ItemTypes[h.ItemTypeMicrocosm], m.MicrocosmID)

	return http.StatusOK, nil
}

// Patch partially updates a saved conversation
func (m *ConversationType) Patch(
	ac AuthContext,
	patches []h.PatchType,
) (
	int,
	error,
) {
	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	for _, patch := range patches {

		m.Meta.EditedNullable = pq.NullTime{Time: time.Now(), Valid: true}
		m.Meta.EditedByNullable = sql.NullInt64{Int64: ac.ProfileID, Valid: true}

		var column string
		patch.ScanRawValue()
		switch patch.Path {
		case "/meta/flags/sticky":
			column = "is_sticky"
			m.Meta.Flags.Sticky = patch.Bool.Bool
			m.Meta.EditReason =
				fmt.Sprintf("Set sticky to %t", m.Meta.Flags.Sticky)
		case "/meta/flags/open":
			column = "is_open"
			m.Meta.Flags.Open = patch.Bool.Bool
			m.Meta.EditReason =
				fmt.Sprintf("Set open to %t", m.Meta.Flags.Open)
		case "/meta/flags/deleted":
			column = "is_deleted"
			m.Meta.Flags.Deleted = patch.Bool.Bool
			m.Meta.EditReason =
				fmt.Sprintf("Set delete to %t", m.Meta.Flags.Deleted)
		case "/meta/flags/moderated":
			column = "is_moderated"
			m.Meta.Flags.Moderated = patch.Bool.Bool
			m.Meta.EditReason =
				fmt.Sprintf("Set moderated to %t", m.Meta.Flags.Moderated)
		default:
			return http.StatusBadRequest,
				fmt.Errorf("unsupported path in patch replace operation")
		}

		m.Meta.Flags.SetVisible()

		_, err = tx.Exec(`--Update Conversation Flags
UPDATE conversations
   SET `+column+` = $2
      ,is_visible = $3
      ,edited = $4
      ,edited_by = $5
      ,edit_reason = $6
 WHERE conversation_id = $1`,
			m.ID,
			patch.Bool.Bool,
			m.Meta.Flags.Visible,
			m.Meta.EditedNullable,
			m.Meta.EditedByNullable,
			m.Meta.EditReason,
		)
		if err != nil {
			return http.StatusInternalServerError,
				fmt.Errorf("update failed: %v", err.Error())
		}
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("transaction failed: %v", err.Error())
	}

	PurgeCache(h.ItemTypes[h.ItemTypeConversation], m.ID)
	PurgeCache(h.ItemTypes[h.ItemTypeMicrocosm], m.MicrocosmID)

	return http.StatusOK, nil
}

// Delete deletes a conversation
func (m *ConversationType) Delete() (int, error) {
	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`--Delete Conversation
UPDATE conversations
   SET is_deleted = true
      ,is_visible = false
 WHERE conversation_id = $1`,
		m.ID,
	)
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("delete failed: %v", err.Error())
	}

	err = DecrementMicrocosmItemCount(tx, m.MicrocosmID)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("transaction failed: %v", err.Error())
	}

	PurgeCache(h.ItemTypes[h.ItemTypeConversation], m.ID)
	PurgeCache(h.ItemTypes[h.ItemTypeMicrocosm], m.MicrocosmID)

	return http.StatusOK, nil
}

// GetConversation fetches a conversation
func GetConversation(
	siteID int64,
	id int64,
	profileID int64,
) (
	ConversationType,
	int,
	error,
) {
	if id == 0 {
		return ConversationType{}, http.StatusNotFound,
			fmt.Errorf("conversation not found")
	}

	// Get from cache if it's available
	mcKey := fmt.Sprintf(mcConversationKeys[c.CacheDetail], id)
	if val, ok := c.Get(mcKey, ConversationType{}); ok {
		m := val.(ConversationType)

		m.Hydrate(siteID)

		return m, http.StatusOK, nil
	}

	// Retrieve resource
	db, err := h.GetConnection()
	if err != nil {
		glog.Errorf("h.GetConnection() %+v", err)
		return ConversationType{}, http.StatusInternalServerError, err
	}

	// TODO(buro9): admins and mods could see this with isDeleted=true in the
	// querystring
	var m ConversationType

	err = db.QueryRow(`--GetConversation
SELECT c.conversation_id
      ,c.microcosm_id
      ,c.title
      ,c.created
      ,c.created_by

      ,c.edited
      ,c.edited_by
      ,c.edit_reason
      ,c.is_sticky
      ,c.is_open
      
      ,c.is_deleted
      ,c.is_moderated
      ,c.is_visible
  FROM conversations c
       JOIN flags f ON f.site_id = $2
                   AND f.item_type_id = 6
                   AND f.item_id = c.conversation_id
 WHERE c.conversation_id = $1
   AND is_deleted(6, c.conversation_id) IS FALSE`,
		id,
		siteID,
	).Scan(
		&m.ID,
		&m.MicrocosmID,
		&m.Title,
		&m.Meta.Created,
		&m.Meta.CreatedByID,

		&m.Meta.EditedNullable,
		&m.Meta.EditedByNullable,
		&m.Meta.EditReasonNullable,
		&m.Meta.Flags.Sticky,
		&m.Meta.Flags.Open,

		&m.Meta.Flags.Deleted,
		&m.Meta.Flags.Moderated,
		&m.Meta.Flags.Visible,
	)
	if err == sql.ErrNoRows {
		glog.Warningf("Conversation not found for id %d", id)
		return ConversationType{}, http.StatusNotFound,
			fmt.Errorf("resource with ID %d not found", id)

	} else if err != nil {
		glog.Errorf("db.Query(%d) %+v", id, err)
		return ConversationType{}, http.StatusInternalServerError,
			fmt.Errorf("database query failed")
	}

	if m.Meta.EditReasonNullable.Valid {
		m.Meta.EditReason = m.Meta.EditReasonNullable.String
	}

	if m.Meta.EditedNullable.Valid {
		m.Meta.Edited =
			m.Meta.EditedNullable.Time.Format(time.RFC3339Nano)
	}

	m.Meta.Links =
		[]h.LinkType{
			h.GetLink("self", "", h.ItemTypeConversation, m.ID),
			h.GetLink(
				"microcosm",
				GetMicrocosmTitle(m.MicrocosmID),
				h.ItemTypeMicrocosm,
				m.MicrocosmID,
			),
		}

	// Update cache
	c.Set(mcKey, m, mcTTL)

	m.Hydrate(siteID)
	return m, http.StatusOK, nil
}

// GetConversationSummary fetches a summary of a conversation
func GetConversationSummary(
	siteID int64,
	id int64,
	profileID int64,
) (
	ConversationSummaryType,
	int,
	error,
) {
	// Get from cache if it's available
	mcKey := fmt.Sprintf(mcConversationKeys[c.CacheSummary], id)
	if val, ok := c.Get(mcKey, ConversationSummaryType{}); ok {
		m := val.(ConversationSummaryType)
		m.Hydrate(siteID)
		return m, http.StatusOK, nil
	}

	// Retrieve resource
	db, err := h.GetConnection()
	if err != nil {
		return ConversationSummaryType{}, http.StatusInternalServerError, err
	}

	// TODO(buro9): admins and mods could see this with isDeleted=true in the
	// querystring
	var m ConversationSummaryType
	err = db.QueryRow(`--GetConversationSummary
SELECT conversation_id
      ,microcosm_id
      ,title
      ,created
      ,created_by
      ,is_sticky
      ,is_open
      ,is_deleted
      ,is_moderated
      ,is_visible
      ,(SELECT COUNT(*) AS total_comments
          FROM flags
         WHERE parent_item_type_id = 6
           AND parent_item_id = $1
           AND microcosm_is_deleted IS NOT TRUE
           AND microcosm_is_moderated IS NOT TRUE
           AND parent_is_deleted IS NOT TRUE
           AND parent_is_moderated IS NOT TRUE
           AND item_is_deleted IS NOT TRUE
           AND item_is_moderated IS NOT TRUE) AS comment_count
      ,view_count
  FROM conversations
 WHERE conversation_id = $1
   AND is_deleted(6, $1) IS FALSE`,
		id,
	).Scan(
		&m.ID,
		&m.MicrocosmID,
		&m.Title,
		&m.Meta.Created,
		&m.Meta.CreatedByID,
		&m.Meta.Flags.Sticky,
		&m.Meta.Flags.Open,
		&m.Meta.Flags.Deleted,
		&m.Meta.Flags.Moderated,
		&m.Meta.Flags.Visible,
		&m.CommentCount,
		&m.ViewCount,
	)
	if err == sql.ErrNoRows {
		return ConversationSummaryType{}, http.StatusNotFound,
			fmt.Errorf("resource with ID %d not found", id)

	} else if err != nil {
		return ConversationSummaryType{}, http.StatusInternalServerError,
			fmt.Errorf("database query failed: %v", err.Error())
	}

	lastComment, status, err :=
		GetLastComment(h.ItemTypes[h.ItemTypeConversation], m.ID)
	if err != nil {
		return ConversationSummaryType{}, status,
			fmt.Errorf("error fetching last comment: %v", err.Error())
	}

	if lastComment.Valid {
		m.LastComment = lastComment
	}

	m.Meta.Links =
		[]h.LinkType{
			h.GetLink("self", "", h.ItemTypeConversation, m.ID),
			h.GetLink(
				"microcosm",
				GetMicrocosmTitle(m.MicrocosmID),
				h.ItemTypeMicrocosm,
				m.MicrocosmID,
			),
		}

	// Update cache
	c.Set(mcKey, m, mcTTL)

	m.Hydrate(siteID)
	return m, http.StatusOK, nil
}

// GetConversations returns a collection of conversations
func GetConversations(
	siteID int64,
	profileID int64,
	limit int64,
	offset int64,
) (
	[]ConversationSummaryType,
	int64,
	int64,
	int,
	error,
) {
	// Retrieve resources
	db, err := h.GetConnection()
	if err != nil {
		return []ConversationSummaryType{}, 0, 0,
			http.StatusInternalServerError, err
	}

	rows, err := db.Query(`--GetConversations
WITH m AS (
    SELECT m.microcosm_id
      FROM microcosms m
      LEFT JOIN ignores_expanded i ON i.profile_id = $3
                                  AND i.item_type_id = 2
                                  AND i.item_id = m.microcosm_id
     WHERE i.profile_id IS NULL
       AND (get_effective_permissions(m.site_id, m.microcosm_id, 2, m.microcosm_id, $3)).can_read IS TRUE
)
SELECT COUNT(*) OVER() AS total
      ,f.item_id
  FROM flags f
  LEFT JOIN ignores i ON i.profile_id = $3
                     AND i.item_type_id = f.item_type_id
                     AND i.item_id = f.item_id
 WHERE f.site_id = $1
   AND i.profile_id IS NULL
   AND f.item_type_id = $2
   AND f.microcosm_is_deleted IS NOT TRUE
   AND f.microcosm_is_moderated IS NOT TRUE
   AND f.parent_is_deleted IS NOT TRUE
   AND f.parent_is_moderated IS NOT TRUE
   AND f.item_is_deleted IS NOT TRUE
   AND f.item_is_moderated IS NOT TRUE
   AND f.microcosm_id IN (SELECT * FROM m)
 ORDER BY f.item_is_sticky DESC
         ,f.last_modified DESC
 LIMIT $4
OFFSET $5`,
		siteID,
		h.ItemTypes[h.ItemTypeConversation],
		profileID,
		limit,
		offset,
	)
	if err != nil {
		return []ConversationSummaryType{}, 0, 0,
			http.StatusInternalServerError,
			fmt.Errorf("database query failed: %v", err.Error())
	}
	defer rows.Close()

	var ems []ConversationSummaryType

	var total int64
	for rows.Next() {
		var id int64
		err = rows.Scan(
			&total,
			&id,
		)
		if err != nil {
			return []ConversationSummaryType{}, 0, 0,
				http.StatusInternalServerError,
				fmt.Errorf("row parsing error: %v", err.Error())
		}

		m, status, err := GetConversationSummary(siteID, id, profileID)
		if err != nil {
			return []ConversationSummaryType{}, 0, 0, status, err
		}

		ems = append(ems, m)
	}
	err = rows.Err()
	if err != nil {
		return []ConversationSummaryType{}, 0, 0,
			http.StatusInternalServerError,
			fmt.Errorf("error fetching rows: %v", err.Error())
	}
	rows.Close()

	pages := h.GetPageCount(total, limit)
	maxOffset := h.GetMaxOffset(total, limit)

	if offset > maxOffset {
		return []ConversationSummaryType{}, 0, 0,
			http.StatusBadRequest, fmt.Errorf(
				"not enough records, "+
					"offset (%d) would return an empty page",
				offset,
			)
	}

	return ems, total, pages, http.StatusOK, nil
}
