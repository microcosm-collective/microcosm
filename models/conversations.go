package models

import (
	"database/sql"
	"errors"
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

type ConversationsType struct {
	Conversations h.ArrayType    `json:"conversations"`
	Meta          h.CoreMetaType `json:"meta"`
}

type ConversationSummaryType struct {
	ItemSummary
	ItemSummaryMeta
}

type ConversationType struct {
	ItemDetail
	ItemDetailCommentsAndMeta
}

func (m *ConversationType) Validate(
	siteId int64,
	profileId int64,
	exists bool,
	isImport bool,
) (
	int,
	error,
) {

	m.Title = SanitiseText(m.Title)

	if strings.Trim(m.Title, " ") == "" {
		return http.StatusBadRequest, errors.New("Title is a required field")
	}

	m.Title = ShoutToWhisper(m.Title)

	if !exists {
		// Does the Microcosm specified exist on this site?
		_, status, err := GetMicrocosmSummary(siteId, m.MicrocosmId, profileId)
		if err != nil {
			return status, err
		}
	}

	if exists && !isImport {
		if m.Id < 1 {
			return http.StatusBadRequest, errors.New(
				fmt.Sprintf(
					"The supplied ID ('%d') cannot be zero or negative.",
					m.Id,
				),
			)
		}

		if strings.Trim(m.Meta.EditReason, " ") == "" ||
			len(m.Meta.EditReason) == 0 {

			return http.StatusBadRequest,
				errors.New("You must provide a reason for the update")

		} else {
			m.Meta.EditReason = ShoutToWhisper(m.Meta.EditReason)
		}
	}

	if m.MicrocosmId <= 0 {
		return http.StatusBadRequest,
			errors.New("You must specify a Microcosm ID")
	}

	m.Meta.Flags.SetVisible()

	return http.StatusOK, nil
}

func (m *ConversationType) FetchSummaries(siteId int64) (int, error) {

	profile, status, err := GetProfileSummary(siteId, m.Meta.CreatedById)
	if err != nil {
		return status, err
	}
	m.Meta.CreatedBy = profile

	if m.Meta.EditedByNullable.Valid {
		profile, status, err :=
			GetProfileSummary(siteId, m.Meta.EditedByNullable.Int64)
		if err != nil {
			return status, err
		}
		m.Meta.EditedBy = profile
	}

	return http.StatusOK, nil
}

func (m *ConversationSummaryType) FetchProfileSummaries(
	siteId int64,
) (
	int,
	error,
) {

	profile, status, err := GetProfileSummary(siteId, m.Meta.CreatedById)
	if err != nil {
		return status, err
	}
	m.Meta.CreatedBy = profile

	switch m.LastComment.(type) {
	case LastComment:
		lastComment := m.LastComment.(LastComment)

		profile, status, err =
			GetProfileSummary(siteId, lastComment.CreatedById)
		if err != nil {
			return status, err
		}

		lastComment.CreatedBy = profile
		m.LastComment = lastComment
	}

	return http.StatusOK, nil
}

func (m *ConversationType) Insert(siteId int64, profileId int64) (int, error) {
	status, err := m.Validate(siteId, profileId, false, false)
	if err != nil {
		return status, err
	}

	dupeKey := "dupe_" + h.Md5sum(
		strconv.FormatInt(m.MicrocosmId, 10)+
			m.Title+
			strconv.FormatInt(m.Meta.CreatedById, 10),
	)
	v, ok := c.CacheGetInt64(dupeKey)
	if ok {
		m.Id = v
		return http.StatusOK, nil
	}

	status, err = m.insert(siteId, profileId)
	if status == http.StatusOK {
		// 5 minute dupe check
		c.CacheSetInt64(dupeKey, m.Id, 60*5)
	}

	return status, err
}

func (m *ConversationType) Import(siteId int64, profileId int64) (int, error) {
	status, err := m.Validate(siteId, profileId, true, true)
	if err != nil {
		return status, err
	}

	return m.insert(siteId, profileId)
}

func (m *ConversationType) insert(siteId int64, profileId int64) (int, error) {

	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	var insertId int64
	err = tx.QueryRow(`--Create Conversation
INSERT INTO conversations (
    microcosm_id, title, created, created_by, view_count,
    is_deleted, is_moderated, is_open, is_sticky
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9
) RETURNING conversation_id`,
		m.MicrocosmId,
		m.Title,
		m.Meta.Created,
		m.Meta.CreatedById,
		m.ViewCount,

		m.Meta.Flags.Deleted,
		m.Meta.Flags.Moderated,
		m.Meta.Flags.Open,
		m.Meta.Flags.Sticky,
	).Scan(
		&insertId,
	)
	if err != nil {
		return http.StatusInternalServerError, errors.New(
			fmt.Sprintf(
				"Error inserting data and returning ID: %v",
				err.Error(),
			),
		)
	}

	m.Id = insertId

	err = IncrementMicrocosmItemCount(tx, m.MicrocosmId)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Transaction failed: %v", err.Error()),
		)
	}

	PurgeCache(h.ItemTypes[h.ItemTypeConversation], m.Id)
	PurgeCache(h.ItemTypes[h.ItemTypeMicrocosm], m.MicrocosmId)

	return http.StatusOK, nil
}

func (m *ConversationType) Update(siteId int64, profileId int64) (int, error) {

	status, err := m.Validate(siteId, profileId, true, false)
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
		m.Id,
		m.MicrocosmId,
		m.Title,
		m.Meta.EditedNullable,
		m.Meta.EditedByNullable,
		m.Meta.EditReason,
	)
	if err != nil {
		return http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Update failed: %v", err.Error()),
		)
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Transaction failed: %v", err.Error()),
		)
	}

	PurgeCache(h.ItemTypes[h.ItemTypeConversation], m.Id)
	PurgeCache(h.ItemTypes[h.ItemTypeMicrocosm], m.MicrocosmId)

	return http.StatusOK, nil
}

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
		m.Meta.EditedByNullable = sql.NullInt64{Int64: ac.ProfileId, Valid: true}

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
				errors.New("Unsupported path in patch replace operation")
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
			m.Id,
			patch.Bool.Bool,
			m.Meta.Flags.Visible,
			m.Meta.EditedNullable,
			m.Meta.EditedByNullable,
			m.Meta.EditReason,
		)
		if err != nil {
			return http.StatusInternalServerError, errors.New(
				fmt.Sprintf("Update failed: %v", err.Error()),
			)
		}
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Transaction failed: %v", err.Error()),
		)
	}

	PurgeCache(h.ItemTypes[h.ItemTypeConversation], m.Id)
	PurgeCache(h.ItemTypes[h.ItemTypeMicrocosm], m.MicrocosmId)

	return http.StatusOK, nil
}

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
		m.Id,
	)
	if err != nil {
		return http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Delete failed: %v", err.Error()),
		)
	}

	err = DecrementMicrocosmItemCount(tx, m.MicrocosmId)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Transaction failed: %v", err.Error()),
		)
	}

	PurgeCache(h.ItemTypes[h.ItemTypeConversation], m.Id)
	PurgeCache(h.ItemTypes[h.ItemTypeMicrocosm], m.MicrocosmId)

	return http.StatusOK, nil
}

func GetConversation(
	siteId int64,
	id int64,
	profileId int64,
) (
	ConversationType,
	int,
	error,
) {

	if id == 0 {
		return ConversationType{}, http.StatusNotFound,
			errors.New("Conversation not found")
	}

	// Get from cache if it's available
	mcKey := fmt.Sprintf(mcConversationKeys[c.CacheDetail], id)
	if val, ok := c.CacheGet(mcKey, ConversationType{}); ok {

		m := val.(ConversationType)

		// TODO(buro9) 2014-05-05: We are not verifying that the cached
		// conversation belongs to this siteId

		m.FetchSummaries(siteId)

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
		siteId,
	).Scan(
		&m.Id,
		&m.MicrocosmId,
		&m.Title,
		&m.Meta.Created,
		&m.Meta.CreatedById,

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
			errors.New(fmt.Sprintf("Resource with ID %d not found", id))

	} else if err != nil {
		glog.Errorf("db.Query(%d) %+v", id, err)
		return ConversationType{}, http.StatusInternalServerError,
			errors.New("Database query failed")
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
			h.GetLink("self", "", h.ItemTypeConversation, m.Id),
			h.GetLink(
				"microcosm",
				GetMicrocosmTitle(m.MicrocosmId),
				h.ItemTypeMicrocosm,
				m.MicrocosmId,
			),
		}

	// Update cache
	c.CacheSet(mcKey, m, mcTtl)

	m.FetchSummaries(siteId)
	return m, http.StatusOK, nil
}

func GetConversationSummary(
	siteId int64,
	id int64,
	profileId int64,
) (
	ConversationSummaryType,
	int,
	error,
) {

	// Get from cache if it's available
	mcKey := fmt.Sprintf(mcConversationKeys[c.CacheSummary], id)
	if val, ok := c.CacheGet(mcKey, ConversationSummaryType{}); ok {
		m := val.(ConversationSummaryType)
		m.FetchProfileSummaries(siteId)
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
		&m.Id,
		&m.MicrocosmId,
		&m.Title,
		&m.Meta.Created,
		&m.Meta.CreatedById,
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
			errors.New(
				fmt.Sprintf("Resource with ID %d not found", id),
			)

	} else if err != nil {
		return ConversationSummaryType{}, http.StatusInternalServerError,
			errors.New(
				fmt.Sprintf("Database query failed: %v", err.Error()),
			)
	}

	lastComment, status, err :=
		GetLastComment(h.ItemTypes[h.ItemTypeConversation], m.Id)
	if err != nil {
		return ConversationSummaryType{}, status, errors.New(
			fmt.Sprintf("Error fetching last comment: %v", err.Error()),
		)
	}

	if lastComment.Valid {
		m.LastComment = lastComment
	}

	m.Meta.Links =
		[]h.LinkType{
			h.GetLink("self", "", h.ItemTypeConversation, m.Id),
			h.GetLink(
				"microcosm",
				GetMicrocosmTitle(m.MicrocosmId),
				h.ItemTypeMicrocosm,
				m.MicrocosmId,
			),
		}

	// Update cache
	c.CacheSet(mcKey, m, mcTtl)

	m.FetchProfileSummaries(siteId)
	return m, http.StatusOK, nil
}

func GetConversations(
	siteId int64,
	profileId int64,
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
      LEFT JOIN ignores i ON i.profile_id = $3
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
		siteId,
		h.ItemTypes[h.ItemTypeConversation],
		profileId,
		limit,
		offset,
	)
	if err != nil {
		return []ConversationSummaryType{}, 0, 0,
			http.StatusInternalServerError, errors.New(
				fmt.Sprintf("Database query failed: %v", err.Error()),
			)
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
				http.StatusInternalServerError, errors.New(
					fmt.Sprintf("Row parsing error: %v", err.Error()),
				)
		}

		m, status, err := GetConversationSummary(siteId, id, profileId)
		if err != nil {
			return []ConversationSummaryType{}, 0, 0, status, err
		}

		ems = append(ems, m)
	}
	err = rows.Err()
	if err != nil {
		return []ConversationSummaryType{}, 0, 0,
			http.StatusInternalServerError, errors.New(
				fmt.Sprintf("Error fetching rows: %v", err.Error()),
			)
	}
	rows.Close()

	pages := h.GetPageCount(total, limit)
	maxOffset := h.GetMaxOffset(total, limit)

	if offset > maxOffset {
		return []ConversationSummaryType{}, 0, 0,
			http.StatusBadRequest, errors.New(
				fmt.Sprintf(
					"not enough records, "+
						"offset (%d) would return an empty page.",
					offset,
				),
			)
	}

	return ems, total, pages, http.StatusOK, nil
}
