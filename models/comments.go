package models

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"

	c "github.com/microcosm-cc/microcosm/cache"
	h "github.com/microcosm-cc/microcosm/helpers"
)

const (
	MinimumPostLength int = 0
)

type CommentsType struct {
	Comments h.ArrayType    `json:"comments"`
	Meta     h.CoreMetaType `json:"meta"`
}

type CommentType struct {
	Id         int64  `json:"id"`
	ItemTypeId int64  `json:"-"`
	ItemType   string `json:"itemType"`
	ItemId     int64  `json:"itemId"`
	Revisions  int64  `json:"revisions"`

	InReplyToNullable sql.NullInt64  `json:"-"`
	InReplyTo         int64          `json:"inReplyTo"`
	Attachments       int64          `json:"attachments"`
	FirstLine         string         `json:"firstLine"`
	Markdown          string         `json:"markdown"`
	HTMLNullable      sql.NullString `json:"-"`
	HTML              string         `json:"html"`

	Files []h.AttachmentType   `json:"files,omitempty"`
	Meta  CommentFlagsMetaType `json:"meta"`
}

type CommentSummaryType struct {
	Id         int64  `json:"id"`
	ItemTypeId int64  `json:"-"`
	ItemType   string `json:"itemType"`
	ItemId     int64  `json:"itemId"`
	Revisions  int64  `json:"revisions"`

	InReplyToNullable sql.NullInt64  `json:"-"`
	InReplyTo         int64          `json:"inReplyTo,omitempty"`
	Attachments       int64          `json:"attachments,omitempty"`
	FirstLine         string         `json:"firstLine,omitempty"`
	Markdown          string         `json:"markdown"`
	HTMLNullable      sql.NullString `json:"-"`
	HTML              string         `json:"html"`

	Files []h.AttachmentType `json:"files,omitempty"`
	Meta  CommentMetaType    `json:"meta"`
}

type CommentMetaType struct {
	h.CreatedType
	h.EditedType
	Flags CommentFlagsType `json:"flags,omitempty"`
	h.CoreMetaType
}

type CommentFlagsType struct {
	Deleted   bool `json:"deleted"`
	Moderated bool `json:"moderated"`
	Visible   bool `json:"visible"`
	Unread    bool `json:"unread"`
}

type ThreadedMetaType struct {
	InReplyTo interface{}   `json:"inReplyTo,omitempty"`
	Replies   []interface{} `json:"replies,omitempty"`
	CommentMetaType
}

type CommentFlagsMetaType struct {
	h.CreatedType
	h.EditedType
	ThreadedMetaType
}

type CommentSummaryRequest struct {
	Item   CommentSummaryType
	Err    error
	Status int
	Seq    int
}

type CommentRequestBySeq []CommentSummaryRequest

func (v CommentRequestBySeq) Len() int           { return len(v) }
func (v CommentRequestBySeq) Swap(i, j int)      { v[i], v[j] = v[j], v[i] }
func (v CommentRequestBySeq) Less(i, j int) bool { return v[i].Seq < v[j].Seq }

func (m *CommentSummaryType) Validate(siteId int64, exists bool) (int, error) {
	if _, inMap := h.ItemTypesCommentable[m.ItemType]; !inMap {
		return http.StatusBadRequest,
			errors.New("You must specify a valid item type")
	} else {
		m.ItemTypeId = h.ItemTypesCommentable[m.ItemType]
	}
	if !exists && m.InReplyTo > 0 {
		parent, _, err := GetCommentSummary(siteId, m.InReplyTo)
		if err != nil {
			m.InReplyTo = 0
		}

		if m.ItemTypeId == parent.ItemTypeId && m.ItemId == parent.ItemId {
			m.InReplyToNullable = sql.NullInt64{Int64: m.InReplyTo, Valid: true}
		} else {
			m.InReplyTo = 0
		}
	}

	if m.ItemId <= 0 {
		return http.StatusBadRequest,
			errors.New("You must specify an Item ID this comment belongs to")
	}

	if strings.Trim(m.Markdown, " ") == "" ||
		len(m.Markdown) < MinimumPostLength {

		return http.StatusBadRequest, errors.New(
			fmt.Sprintf(
				"Markdown is a required field and "+
					"must be of decent length (more than %d chars)",
				MinimumPostLength,
			),
		)
	}

	// Prevent shouting on text fields
	m.Markdown = ShoutToWhisper(m.Markdown)

	return http.StatusOK, nil
}

func (m *CommentSummaryType) FetchProfileSummaries(siteId int64) (int, error) {

	profile, status, err := GetProfileSummary(siteId, m.Meta.CreatedById)
	if err != nil {
		return status, err
	}
	m.Meta.CreatedBy = profile

	if m.Meta.EditedByNullable.Valid {
		profile, status, err := GetProfileSummary(
			siteId,
			m.Meta.EditedByNullable.Int64,
		)
		if err != nil {
			return status, err
		}
		m.Meta.EditedBy = profile
	}

	if m.InReplyTo != 0 {

		// Replies are only valid when they come from the site item as that
		// implies the same site, same microcosm/huddle and the same permissions
		parent, status, _ := GetCommentSummary(siteId, m.InReplyTo)
		if status == http.StatusOK &&
			parent.ItemType == m.ItemType &&
			parent.ItemId == m.ItemId {

			inReplyToProfileTitle, _, _ := GetTitle(
				siteId,
				h.ItemTypes[h.ItemTypeProfile],
				parent.Meta.CreatedById,
				0,
			)

			m.Meta.Links = append(
				m.Meta.Links,
				h.GetLink("inReplyTo", "", h.ItemTypeComment, parent.Id),
			)
			m.Meta.Links = append(
				m.Meta.Links,
				h.GetLink(
					"inReplyToAuthor",
					inReplyToProfileTitle,
					h.ItemTypeProfile,
					parent.Meta.CreatedById,
				),
			)
		}
	}

	return http.StatusOK, nil

}

func (m *CommentSummaryType) Insert(siteId int64) (int, error) {

	status, err := m.Validate(siteId, false)
	if err != nil {
		return status, err
	}

	// Dupe checking
	dupeKey := "dupe_" + h.Md5sum(
		strconv.FormatInt(m.ItemTypeId, 10)+
			strconv.FormatInt(m.ItemId, 10)+
			m.Markdown+
			strconv.FormatInt(m.Meta.CreatedById, 10),
	)

	v, ok := c.CacheGetInt64(dupeKey)
	if ok {
		m.Id = v
		return http.StatusOK, nil
	}

	status, err = m.insert(siteId, false)
	// 5 minute dupe check
	c.CacheSetInt64(dupeKey, m.Id, 60*5)

	// If we're posting to a huddle, purge the counts for the users in the
	// huddle
	if m.ItemTypeId == h.ItemTypes[h.ItemTypeHuddle] {
		ems, _, _, _, err2 := GetHuddleParticipants(siteId, m.ItemId, 9999, 0)
		if err2 != nil {
			return status, err
		}

		for _, em := range ems {
			p, _, err := GetProfileSummary(siteId, em.ID)
			if err != nil {
				glog.Error(err)
				continue
			}
			p.UpdateUnreadHuddleCount()
		}
	}

	return status, err
}

func (m *CommentSummaryType) Import(siteId int64) (int, error) {
	status, err := m.Validate(siteId, true)
	if err != nil {
		return status, err
	}
	return m.insert(siteId, true)
}

func (m *CommentSummaryType) insert(siteId int64, isImport bool) (int, error) {

	tx, err := h.GetTransaction()
	if err != nil {
		glog.Error(err)
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	var insertId int64
	err = tx.QueryRow(`
INSERT INTO comments (
    item_type_id, item_id, profile_id, created, is_visible,
    is_moderated, is_deleted, in_reply_to, attachment_count, yay_count,
    meh_count, grr_count
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, 0, 0,
    0, 0
) RETURNING comment_id`,
		m.ItemTypeId,
		m.ItemId,
		m.Meta.CreatedById,
		m.Meta.Created,
		m.Meta.Flags.Visible,
		m.Meta.Flags.Moderated,
		m.Meta.Flags.Deleted,
		m.InReplyToNullable,
	).Scan(
		&insertId,
	)
	if err != nil {
		glog.Error(err)
		return http.StatusInternalServerError,
			errors.New(
				fmt.Sprintf("Error inserting data and returning ID: %+v", err),
			)
	}
	m.Id = insertId

	revisionId, status, err := m.CreateCommentRevision(
		tx,
		true,
		siteId,
		m.ItemTypeId,
		m.ItemId,
		isImport,
	)
	if err != nil {
		glog.Error(err)
		return status, err
	}

	err = tx.Commit()
	if err != nil {
		glog.Error(err)
		return http.StatusInternalServerError,
			errors.New(fmt.Sprintf("Transaction failed: %v", err.Error()))
	}

	EmbedAllMedia(revisionId)

	PurgeCache(h.ItemTypes[h.ItemTypeComment], m.Id)
	PurgeCache(m.ItemTypeId, m.ItemId)

	if !isImport {
		go IncrementProfileCommentCount(m.Meta.CreatedById)
		go IncrementItemCommentCount(m.ItemTypeId, m.ItemId)

		summary, status, err := GetSummary(
			siteId,
			m.ItemTypeId,
			m.ItemId,
			m.Meta.CreatedById,
		)
		if err != nil {
			glog.Error(err)
			return status, err
		}

		switch summary.(type) {
		case ConversationSummaryType:
			PurgeCache(
				h.ItemTypes[h.ItemTypeMicrocosm],
				summary.(ConversationSummaryType).MicrocosmID,
			)
		case EventSummaryType:
			PurgeCache(
				h.ItemTypes[h.ItemTypeMicrocosm],
				summary.(EventSummaryType).MicrocosmID,
			)
		case PollSummaryType:
			PurgeCache(
				h.ItemTypes[h.ItemTypeMicrocosm],
				summary.(PollSummaryType).MicrocosmID,
			)
		default:
		}
	}

	return http.StatusOK, nil
}

func (m *CommentSummaryType) CreateCommentRevision(
	tx *sql.Tx,
	isFirst bool,
	siteId int64,
	itemTypeId int64,
	itemId int64,
	isImport bool,
) (
	int64,
	int,
	error,
) {

	_, err := tx.Exec(`
UPDATE revisions
   SET is_current = false
 WHERE comment_id = $1
   AND is_current IS NOT FALSE`,
		m.Id,
	)
	if err != nil {
		return 0, http.StatusInternalServerError,
			errors.New(fmt.Sprintf("Update 'is_current = false' failed: %v", err.Error()))
	}

	sqlQuery := `
INSERT INTO revisions (
    comment_id, profile_id, raw, html, created,
    is_current
) VALUES (
    $1, $2, $3, NULL, $4,
    true
) RETURNING revision_id`

	var row *sql.Row
	if isFirst {
		row = tx.QueryRow(
			sqlQuery,
			m.Id,
			m.Meta.CreatedById,
			m.Markdown,
			m.Meta.Created,
		)
	} else {
		row = tx.QueryRow(
			sqlQuery,
			m.Id,
			m.Meta.EditedByNullable,
			m.Markdown,
			m.Meta.EditedNullable,
		)
	}

	var revisionId int64
	err = row.Scan(&revisionId)
	if err != nil {
		return 0, http.StatusInternalServerError,
			errors.New(fmt.Sprintf("Insert failed: %v", err.Error()))
	}

	html, err := ProcessCommentMarkdown(
		tx,
		revisionId,
		m.Markdown,
		siteId,
		itemTypeId,
		itemId,
		!isImport,
	)
	if err != nil {
		return revisionId, http.StatusInternalServerError, err
	}

	m.HTML = html

	_, err = tx.Exec(`
UPDATE revisions
   SET html = $2
 WHERE revision_id = $1`,
		revisionId,
		m.HTML,
	)
	if err != nil {
		return revisionId, http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Error updating HTML: %v", err.Error()),
		)
	}

	return revisionId, http.StatusOK, nil
}

func (m *CommentSummaryType) Update(siteId int64) (int, error) {

	status, err := m.Validate(siteId, true)
	if err != nil {
		return status, err
	}

	// Update resource
	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	revisionId, status, err := m.CreateCommentRevision(
		tx,
		false,
		siteId,
		m.ItemTypeId,
		m.ItemId,
		false,
	)
	if err != nil {
		return status, err
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			errors.New(fmt.Sprintf("Transaction failed: %+v", err))
	}

	EmbedAllMedia(revisionId)

	PurgeCache(h.ItemTypes[h.ItemTypeComment], m.Id)
	PurgeCache(h.ItemTypes[m.ItemType], m.ItemId)

	summary, status, err := GetSummary(
		siteId,
		m.ItemTypeId,
		m.ItemId,
		m.Meta.CreatedById,
	)
	if err != nil {
		return status, err
	}

	switch summary.(type) {
	case ConversationSummaryType:
		PurgeCache(
			h.ItemTypes[h.ItemTypeMicrocosm],
			summary.(ConversationSummaryType).MicrocosmID,
		)
	case EventSummaryType:
		PurgeCache(
			h.ItemTypes[h.ItemTypeMicrocosm],
			summary.(EventSummaryType).MicrocosmID,
		)
	case PollSummaryType:
		PurgeCache(
			h.ItemTypes[h.ItemTypeMicrocosm],
			summary.(PollSummaryType).MicrocosmID,
		)
	default:
	}

	return http.StatusOK, nil
}

func (ct *CommentSummaryType) Patch(
	siteId int64,
	ac AuthContext,
	patches []h.PatchType,
) (
	int,
	error,
) {

	// Update resource
	m, status, err := GetCommentSummary(siteId, ct.Id)
	if err != nil {
		return status, err
	}

	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	for _, patch := range patches {

		var column string
		patch.ScanRawValue()
		switch patch.Path {
		case "/meta/flags/deleted":
			column = "is_deleted"
			m.Meta.Flags.Deleted = patch.Bool.Bool
		case "/meta/flags/moderated":
			column = "is_moderated"
			m.Meta.Flags.Moderated = patch.Bool.Bool
		default:
			return http.StatusBadRequest,
				errors.New("Unsupported path in patch replace operation")
		}

		m.Meta.Flags.Visible = !(m.Meta.Flags.Moderated || m.Meta.Flags.Deleted)
		_, err = tx.Exec(`
UPDATE comments
   SET `+column+` = $2
      ,is_visible = $3
 WHERE comment_id = $1`,
			m.Id,
			patch.Bool.Bool,
			m.Meta.Flags.Visible,
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

	PurgeCache(h.ItemTypes[h.ItemTypeComment], m.Id)
	PurgeCache(m.ItemTypeId, m.ItemId)

	summary, status, err := GetSummary(
		siteId,
		m.ItemTypeId,
		m.ItemId,
		m.Meta.CreatedById,
	)
	if err != nil {
		return status, err
	}

	switch summary.(type) {
	case ConversationSummaryType:
		PurgeCache(
			h.ItemTypes[h.ItemTypeMicrocosm],
			summary.(ConversationSummaryType).MicrocosmID,
		)
	case EventSummaryType:
		PurgeCache(
			h.ItemTypes[h.ItemTypeMicrocosm],
			summary.(EventSummaryType).MicrocosmID,
		)
	case PollSummaryType:
		PurgeCache(
			h.ItemTypes[h.ItemTypeMicrocosm],
			summary.(PollSummaryType).MicrocosmID,
		)
	default:
	}

	return http.StatusOK, nil
}

func (ct *CommentSummaryType) Delete(siteId int64) (int, error) {

	m, status, err := GetCommentSummary(siteId, ct.Id)
	if err != nil {
		if status == http.StatusNotFound {
			return http.StatusOK, nil
		}

		glog.Error(err)
		return status, err
	}

	// We have something to delete
	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
UPDATE comments
   SET is_deleted = true
 WHERE comment_id = $1`,
		m.Id,
	)
	if err != nil {
		glog.Error(err)
		return http.StatusInternalServerError,
			errors.New(fmt.Sprintf("Delete failed: %+v", err))
	}

	_, err = tx.Exec(`
UPDATE flags f
   SET last_modified = item.last_modified
  FROM (
SELECT c1.item_type_id
      ,c1.item_id
      ,MAX(c.created) AS last_modified
  FROM comments c1
       JOIN comments c ON c.item_type_id = c1.item_type_id
                      AND c.item_id = c1.item_id
 WHERE c1.comment_id = $1
   AND c.is_deleted IS NOT TRUE
 GROUP BY c1.item_type_id, c1.item_id
       ) item
 WHERE f.item_type_id = item.item_type_id
   AND f.item_id = item.item_id`,
		m.Id,
	)
	if err != nil {
		glog.Error(err)
		return http.StatusInternalServerError,
			errors.New(fmt.Sprintf("Delete failed: %+v", err))
	}

	err = tx.Commit()
	if err != nil {
		glog.Error(err)
		return http.StatusInternalServerError,
			errors.New(fmt.Sprintf("Transaction failed: %+v", err))
	}

	go DecrementProfileCommentCount(m.Meta.CreatedById)
	go DecrementItemCommentCount(m.ItemTypeId, m.ItemId)
	PurgeCache(h.ItemTypes[h.ItemTypeComment], m.Id)

	tx2, err := h.GetTransaction()
	if err != nil {
		glog.Error(err)
		return http.StatusInternalServerError, err
	}
	defer tx2.Rollback()

	PurgeCache(m.ItemTypeId, m.ItemId)

	summary, status, err := GetSummary(
		siteId,
		m.ItemTypeId,
		m.ItemId,
		m.Meta.CreatedById,
	)
	if err != nil {
		glog.Error(err)
		return status, err
	}

	switch summary.(type) {
	case ConversationSummaryType:
		PurgeCache(
			h.ItemTypes[h.ItemTypeMicrocosm],
			summary.(ConversationSummaryType).MicrocosmID,
		)
	case EventSummaryType:
		PurgeCache(
			h.ItemTypes[h.ItemTypeMicrocosm],
			summary.(EventSummaryType).MicrocosmID,
		)
	case PollSummaryType:
		PurgeCache(
			h.ItemTypes[h.ItemTypeMicrocosm],
			summary.(PollSummaryType).MicrocosmID,
		)
	default:
	}

	return http.StatusOK, nil
}

func GetPageNumber(
	commentId int64,
	limit int64,
	profileId int64,
) (
	int64,
	int64,
	int64,
	int,
	error,
) {

	db, err := h.GetConnection()
	if err != nil {
		return 0, 0, 0, http.StatusInternalServerError, err
	}

	rows, err := db.Query(`--GetPageNumber
SELECT oc.item_type_id
      ,oc.item_id
      ,(CEIL(COUNT(*)::real / $2) - 1) * $2 AS offset
  FROM comments oc
  LEFT JOIN ignores i ON i.profile_id = $3
                     AND i.item_type_id = 3
                     AND i.item_id = oc.profile_id
      ,(
        SELECT item_type_id
              ,item_id
              ,created
              ,is_deleted
              ,is_moderated
          FROM comments
         WHERE comment_id = $1
           AND is_deleted = False
           AND is_moderated = False
       ) AS ic
 WHERE i.profile_id IS NULL
   AND oc.is_deleted = ic.is_deleted
   AND oc.is_moderated = ic.is_moderated
   AND oc.item_type_id = ic.item_type_id
   AND oc.item_id = ic.item_id
   AND oc.created <= ic.created
 GROUP BY oc.item_type_id
         ,oc.item_id`,
		commentId,
		limit,
		profileId,
	)
	if err != nil {
		return 0, 0, 0, http.StatusInternalServerError,
			errors.New(fmt.Sprintf("Get page link failed: %v", err.Error()))
	}
	defer rows.Close()

	var (
		itemTypeId int64
		itemId     int64
		offset     int64
	)
	for rows.Next() {
		err = rows.Scan(
			&itemTypeId,
			&itemId,
			&offset,
		)
		if err != nil {
			return 0, 0, 0, http.StatusInternalServerError, errors.New(
				fmt.Sprintf("Row parsing error: %v", err.Error()),
			)
		}
	}
	err = rows.Err()
	if err != nil {
		return 0, 0, 0, http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Error fetching rows: %v", err.Error()),
		)
	}
	rows.Close()

	if itemTypeId < 1 || itemId < 1 {
		return 0, 0, 0, http.StatusNotFound, errors.New("Comment not found")
	}

	return itemTypeId, itemId, offset, http.StatusOK, nil
}

func (m *CommentSummaryType) GetPageLink(
	limit int64,
	profileId int64,
) (
	h.LinkType,
	int,
	error,
) {

	itemTypeId, itemId, offset, status, err := GetPageNumber(
		m.Id,
		limit,
		profileId,
	)
	if err != nil {
		return h.LinkType{}, status, err
	}

	itemType, err := h.GetItemTypeFromInt(itemTypeId)
	if err != nil {
		return h.LinkType{}, http.StatusInternalServerError, err
	}

	link := h.GetLink("commentPage", "", itemType, itemId)

	href, err := url.Parse(link.Href)
	if err != nil {
		return h.LinkType{}, http.StatusInternalServerError, err
	}

	query := href.Query()

	if offset > 0 {
		query.Add("offset", strconv.FormatInt(offset, 10))

		if limit > 0 && limit != h.DefaultQueryLimit {
			query.Add("limit", strconv.FormatInt(limit, 10))
		}
	}

	href.RawQuery = query.Encode()

	link.Href = href.String()

	return link, http.StatusOK, nil
}

func HandleCommentRequest(
	siteId int64,
	commentId int64,
	seq int,
	out chan<- CommentSummaryRequest,
) {

	item, status, err := GetCommentSummary(siteId, commentId)

	response := CommentSummaryRequest{
		Item:   item,
		Status: status,
		Err:    err,
		Seq:    seq,
	}
	out <- response
}

func GetCommentSummary(
	siteId int64,
	commentId int64,
) (
	CommentSummaryType,
	int,
	error,
) {

	if commentId == 0 {
		return CommentSummaryType{}, http.StatusNotFound,
			errors.New("Comment not found")
	}

	// Get from cache if it's available
	mcKey := fmt.Sprintf(mcCommentKeys[c.CacheDetail], commentId)
	if val, ok := c.CacheGet(mcKey, CommentSummaryType{}); ok {
		m := val.(CommentSummaryType)

		m.FetchProfileSummaries(siteId)

		return m, http.StatusOK, nil
	}

	// It's not in cache, and so we're going to generate it. If we're unable to
	// parse the markdown we will need to re-try. But we will cache it to help
	// ensure we don't thrash the system resources. What this means is that
	// instead of a 1 week time-to-live we *may* need a much shorter TTL.
	//
	// This is what commentTtl stores... the default TTL to be over-ridden
	// with a shorter TTL is we cannot parse the Markdown.
	commentTtl := mcTTL

	db, err := h.GetConnection()
	if err != nil {
		glog.Errorf("h.GetConnection() %+v", err)
		return CommentSummaryType{}, http.StatusInternalServerError, err
	}

	// TODO(buro9): admins and mods could see this with isDeleted=true in the
	// querystring

	var revisionId int64
	m := CommentSummaryType{}
	err = db.QueryRow(`
SELECT c.comment_id
      ,c.item_type_id
      ,c.item_id
      ,c.created
      ,c.profile_id AS createdby
      ,(SELECT COUNT(*) FROM revisions WHERE comment_id = c.comment_id) AS revs
      ,r.revision_id
      ,r.created AS edited
      ,r.profile_id AS editedby
      ,c.in_reply_to
      ,(
           SELECT COUNT(a.*)
             FROM attachments a
            WHERE a.item_type_id = 4
              AND a.item_id = c.comment_id
       ) AS attachment_count
      ,c.is_deleted
      ,c.is_moderated
      ,(c.is_deleted OR c.is_moderated) IS NOT TRUE AS is_visible
      ,r.raw
      ,r.html
  FROM comments c
      ,revisions r
 WHERE c.comment_id = $1
   AND is_deleted(4, c.comment_id) IS FALSE
   AND c.comment_id = r.comment_id
   AND r.is_current IS NOT FALSE
 ORDER BY r.created DESC
 LIMIT 1
OFFSET 0`,
		commentId,
	).Scan(
		&m.Id,
		&m.ItemTypeId,
		&m.ItemId,
		&m.Meta.Created,
		&m.Meta.CreatedById,
		&m.Revisions,
		&revisionId,
		&m.Meta.EditedNullable,
		&m.Meta.EditedByNullable,
		&m.InReplyToNullable,
		&m.Attachments,
		&m.Meta.Flags.Deleted,
		&m.Meta.Flags.Moderated,
		&m.Meta.Flags.Visible,
		&m.Markdown,
		&m.HTMLNullable,
	)
	if err == sql.ErrNoRows {
		return CommentSummaryType{}, http.StatusNotFound, errors.New(
			fmt.Sprintf("Comment with ID %d not found", commentId),
		)

	} else if err != nil {
		glog.Errorf("db.QueryRow(%d) %+v", commentId, err)
		return CommentSummaryType{}, http.StatusInternalServerError,
			errors.New("Database query failed")
	}

	if m.Meta.EditedNullable.Valid &&
		m.Meta.EditedNullable.Time != m.Meta.Created {

		m.Meta.Edited = m.Meta.EditedNullable.Time.Format(time.RFC3339Nano)
		if m.Meta.EditReasonNullable.Valid {
			m.Meta.EditReason = m.Meta.EditReasonNullable.String
		}
	}

	m.ItemType, err = h.GetItemTypeFromInt(m.ItemTypeId)
	if err != nil {
		glog.Errorf("h.GetItemTypeFromInt(%d) %+v", m.ItemTypeId, err)
		return CommentSummaryType{}, http.StatusInternalServerError, err
	}

	if m.InReplyToNullable.Valid {
		m.InReplyTo = m.InReplyToNullable.Int64
	}

	// Edge case for reprocessing HTML if we change the processing mechanism
	if m.HTMLNullable.Valid && strings.Trim(m.HTMLNullable.String, " ") != "" {
		m.HTML = m.HTMLNullable.String

	} else {

		if strings.Trim(m.Markdown, " ") == "" {
			glog.Errorln(`strings.Trim(m.Markdown, " ") == ""`)
			return CommentSummaryType{}, http.StatusInternalServerError,
				errors.New("Markdown is empty")
		}

		tx, err := h.GetTransaction()
		if err != nil {
			glog.Errorf("h.GetTransaction() %+v", err)
			return CommentSummaryType{}, http.StatusInternalServerError, err
		}
		defer tx.Rollback()

		html, err := ProcessCommentMarkdown(
			tx,
			revisionId,
			m.Markdown,
			siteId,
			m.ItemTypeId,
			m.ItemId,
			false,
		)
		if err != nil {
			glog.Errorf(
				"ProcessCommentMarkdown(tx, %d, m.Markdown, siteId, "+
					"m.ItemTypeId, m.ItemId, false) %+v",
				revisionId,
				err,
			)
			return CommentSummaryType{}, http.StatusInternalServerError, err
		}

		m.HTML = html

		if strings.Trim(m.HTML, " ") != "" {
			_, err = tx.Exec(`
UPDATE revisions
   SET html = $2
 WHERE revision_id = $1`,
				revisionId,
				m.HTML,
			)
			if err != nil {
				tx.Rollback()

				glog.Errorf("tx.Exec(%d, m.HTML) %+v", revisionId, err)
				return CommentSummaryType{}, http.StatusInternalServerError,
					errors.New("Error updating HTML")
			}

			err = tx.Commit()
			if err != nil {
				glog.Errorf("tx.Commit() %+v", err)
				return CommentSummaryType{}, http.StatusInternalServerError,
					errors.New("Transaction failed")
			}

			EmbedAllMedia(revisionId)

		} else {

			glog.Errorf(`m.HTML == "" for commentId %d`, commentId)

			// A friendly error message
			m.HTML = "<em>Microcosm error: Comment not rendered, " +
				"please try again later</em>."

			commentTtl = 60 * 5 // 5 minutes
		}
	}

	if m.Id == 0 {
		glog.Warningf("m.Id == 0 (expected %d)", commentId)
		return CommentSummaryType{}, http.StatusNotFound, errors.New(
			fmt.Sprintf("Resource with ID %d not found", commentId),
		)
	}

	itemTitle, _, err := GetTitle(siteId, h.ItemTypes[m.ItemType], m.ItemId, 0)
	if err != nil {
		glog.Warningf(
			"GetTitle(%d, %d, %d, 0) %+v",
			siteId,
			h.ItemTypes[m.ItemType],
			m.ItemId,
			err,
		)
	}
	m.Meta.Links =
		[]h.LinkType{
			h.GetLink("self", "", h.ItemTypeComment, m.Id),
			h.GetLink(m.ItemType, itemTitle, m.ItemType, m.ItemId),
			h.GetLink("up", itemTitle, m.ItemType, m.ItemId),
		}

	// Update cache
	c.CacheSet(mcKey, m, commentTtl)

	// Profiles should be fetched after the item is cached.
	m.FetchProfileSummaries(siteId)

	return m, http.StatusOK, nil
}

func GetComments(
	siteId int64,
	itemType string,
	itemId int64,
	reqUrl *url.URL,
	profileId int64,
	itemCreated time.Time,
) (
	h.ArrayType,
	int,
	error,
) {

	query := reqUrl.Query()
	limit, offset, status, err := h.GetLimitAndOffset(query)
	if err != nil {
		return h.ArrayType{}, status, err
	}

	ems, total, pages, status, err := GetItemComments(
		siteId,
		itemType,
		itemId,
		limit,
		offset,
		profileId,
		itemCreated,
	)
	if err != nil {
		return h.ArrayType{}, status, err
	}

	commentArray := h.ConstructArray(
		ems,
		h.ApiTypeComment,
		total,
		limit,
		offset,
		pages,
		reqUrl,
	)

	return commentArray, http.StatusOK, nil
}

func GetLatestComments(
	siteId int64,
	itemType string,
	itemId int64,
	profileId int64,
	limit int64,
) (
	int64,
	int64,
	int,
	error,
) {

	lastRead, status, err :=
		GetLastReadTime(h.ItemTypes[itemType], itemId, profileId)
	if err != nil {
		return 0, 0, status, err
	}

	commentId, status, err :=
		GetNextOrLastCommentId(h.ItemTypes[itemType], itemId, lastRead, profileId)
	if err != nil {
		return 0, 0, status, err
	}

	_, _, offset, status, err :=
		GetPageNumber(commentId, limit, profileId)
	if err != nil {
		return 0, 0, status, err
	}

	return offset, commentId, http.StatusOK, nil
}

func GetItemComments(
	siteId int64,
	itemType string,
	itemId int64,
	limit int64,
	offset int64,
	profileId int64,
	itemCreated time.Time,
) (
	[]CommentSummaryType,
	int64,
	int64,
	int,
	error,
) {

	// Comments may be fetched for an individual item, or without that filter.
	var fetchForItem = false
	if itemType != "" {
		if _, exists := h.ItemTypesCommentable[itemType]; !exists {
			return []CommentSummaryType{}, 0, 0, http.StatusBadRequest,
				errors.New("You must specify a valid item type")
		}

		if itemId <= 0 {
			return []CommentSummaryType{}, 0, 0, http.StatusBadRequest,
				errors.New("If you provide an itemType, then you must " +
					"provide a non-zero and not negative itemId")
		}
		fetchForItem = true
	}

	db, err := h.GetConnection()
	if err != nil {
		return []CommentSummaryType{}, 0, 0, http.StatusInternalServerError, err
	}

	// Define WHERE/LIMIT clauses as they are used multiple times.
	var sqlWhere string
	var sqlLimit string

	if fetchForItem {
		sqlWhere = `
              AND f.parent_item_type_id = $1
              AND f.parent_item_id = $2`
		sqlLimit = `
            LIMIT $4
           OFFSET $5`
	} else {
		sqlLimit = `
            LIMIT $1
           OFFSET $2`
	}

	// Fetch comment IDs and read status.
	sqlQuery := `--GetItemComments
SELECT total
      ,item_id
      ,last_modified > last_read_time(item_type_id, item_id, $3) AS unread
  FROM (
           SELECT COUNT(*) OVER() AS total
                 ,f.item_type_id
                 ,f.item_id
                 ,f.last_modified
             FROM flags f
             LEFT JOIN ignores i ON i.profile_id = $3
                                AND i.item_type_id = 3
                                AND i.item_id = f.created_by
            WHERE f.item_type_id = 4
              AND i.profile_id IS NULL` + sqlWhere + `
              AND f.microcosm_is_deleted IS NOT TRUE
              AND f.microcosm_is_moderated IS NOT TRUE
              AND f.parent_is_deleted IS NOT TRUE
              AND f.parent_is_moderated IS NOT TRUE
              AND f.item_is_deleted IS NOT TRUE
              AND f.item_is_moderated IS NOT TRUE
            ORDER BY f.last_modified` + sqlLimit + `
       ) AS r`

	var rows *sql.Rows

	if fetchForItem {
		// Comment IDs.
		rows, err = db.Query(
			sqlQuery,
			h.ItemTypesCommentable[itemType],
			itemId,
			profileId,
			limit,
			offset,
		)
	} else {
		// Comment IDs.
		rows, err = db.Query(sqlQuery, limit, offset, profileId)
	}

	defer rows.Close()
	if err != nil {
		return []CommentSummaryType{}, 0, 0, http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Database query failed: %v", err.Error()),
		)
	}

	// Get a list of the identifiers of the items to return
	var total int64
	ids := []int64{}
	unread := map[int64]bool{}
	for rows.Next() {
		var (
			id       int64
			isUnread bool
		)
		err = rows.Scan(
			&total,
			&id,
			&isUnread,
		)
		if err != nil {
			return []CommentSummaryType{}, 0, 0, http.StatusInternalServerError,
				errors.New(
					fmt.Sprintf("Row parsing error: %v", err.Error()),
				)
		}

		unread[id] = isUnread
		ids = append(ids, id)
	}
	err = rows.Err()
	if err != nil {
		return []CommentSummaryType{}, 0, 0, http.StatusInternalServerError,
			errors.New(
				fmt.Sprintf("Error fetching rows: %v", err.Error()),
			)
	}
	rows.Close()

	// Make a request for each identifier
	var wg1 sync.WaitGroup
	req := make(chan CommentSummaryRequest)
	defer close(req)

	for seq, id := range ids {
		go HandleCommentRequest(siteId, id, seq, req)
		wg1.Add(1)
	}

	// Receive the responses and check for errors
	resps := []CommentSummaryRequest{}
	for i := 0; i < len(ids); i++ {
		resp := <-req
		wg1.Done()
		resps = append(resps, resp)
	}
	wg1.Wait()

	for _, resp := range resps {
		if resp.Err != nil {
			return []CommentSummaryType{}, 0, 0, resp.Status, resp.Err
		}
	}

	// Sort them
	sort.Sort(CommentRequestBySeq(resps))

	// Extract the values
	ems := []CommentSummaryType{}
	for _, resp := range resps {
		m := resp.Item
		m.Meta.Flags.Unread = unread[m.Id]
		ems = append(ems, m)
	}

	pages := h.GetPageCount(total, limit)
	maxOffset := h.GetMaxOffset(total, limit)

	if offset > maxOffset {
		return []CommentSummaryType{}, 0, 0, http.StatusBadRequest,
			errors.New(
				fmt.Sprintf("Not enough records, "+
					"offset (%d) would return an empty page.",
					offset,
				),
			)
	}

	return ems, total, pages, http.StatusOK, nil
}

func GetComment(
	siteId int64,
	commentId int64,
	profileId int64,
	limit int64,
) (
	CommentType,
	int,
	error,
) {

	if commentId == 0 {
		return CommentType{}, http.StatusNotFound,
			errors.New("Comment not found")
	}

	var m CommentType
	commentsummary, status, err := GetCommentSummary(siteId, commentId)
	if err != nil {
		return CommentType{}, status, err
	}

	// We are cheating by fetch stuff from an existing in-memory object and
	// mapping it now to the new data structure
	m.Id = commentsummary.Id
	m.ItemTypeId = commentsummary.ItemTypeId
	m.ItemType = commentsummary.ItemType
	m.ItemId = commentsummary.ItemId
	m.Revisions = commentsummary.Revisions
	m.InReplyToNullable = commentsummary.InReplyToNullable
	m.InReplyTo = commentsummary.InReplyTo
	m.Attachments = commentsummary.Attachments
	m.FirstLine = commentsummary.FirstLine
	m.Markdown = commentsummary.Markdown
	m.HTMLNullable = commentsummary.HTMLNullable
	m.HTML = commentsummary.HTML
	m.Files = commentsummary.Files
	m.Meta.Created = commentsummary.Meta.Created
	m.Meta.CreatedById = commentsummary.Meta.CreatedById
	m.Meta.CreatedBy = commentsummary.Meta.CreatedBy
	m.Meta.EditedNullable = commentsummary.Meta.EditedNullable
	m.Meta.Edited = commentsummary.Meta.Edited
	m.Meta.EditedByNullable = commentsummary.Meta.EditedByNullable
	m.Meta.EditedBy = commentsummary.Meta.EditedBy
	m.Meta.EditReasonNullable = commentsummary.Meta.EditReasonNullable
	m.Meta.EditReason = commentsummary.Meta.EditReason
	m.Meta.Flags = commentsummary.Meta.Flags
	m.Meta.Stats = commentsummary.Meta.Stats
	m.Meta.Links = commentsummary.Meta.Links
	m.Meta.Permissions = commentsummary.Meta.Permissions

	link, status, err := commentsummary.GetPageLink(limit, profileId)
	if err != nil {
		return CommentType{}, status, err
	}
	m.Meta.Links = append(m.Meta.Links, link)

	// We only fetch the immediate parent
	if m.InReplyTo != 0 {
		commentsummary, status, _ = GetCommentSummary(siteId, m.InReplyTo)
		if status == http.StatusOK {
			m.Meta.InReplyTo = commentsummary
		}
	}

	//GET Replies
	db, err := h.GetConnection()
	if err != nil {
		return CommentType{}, http.StatusInternalServerError, err
	}

	rows, err := db.Query(`
SELECT c.comment_id
  FROM comments c
  LEFT JOIN ignores i ON i.profile_id = $2
                     AND i.item_type_id = 3
                     AND i.item_id = c.profile_id
 WHERE c.in_reply_to = $1
   AND i.profile_id IS NULL
   AND c.is_moderated IS NOT TRUE
   AND c.is_deleted IS NOT TRUE
 ORDER BY c.created ASC`,
		commentId,
		profileId,
	)
	if err != nil {
		return CommentType{}, http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Database query failed: %v", err.Error()),
		)
	}
	defer rows.Close()

	ids := []int64{}

	for rows.Next() {
		var id int64
		err = rows.Scan(
			&id,
		)
		if err != nil {
			return CommentType{}, http.StatusInternalServerError, errors.New(
				fmt.Sprintf("Row parsing error: %v", err.Error()),
			)
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	if err != nil {
		return CommentType{}, http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Error fetching rows: %v", err.Error()),
		)
	}
	rows.Close()

	var wg1 sync.WaitGroup
	req := make(chan CommentSummaryRequest)
	defer close(req)

	for seq, id := range ids {
		go HandleCommentRequest(siteId, id, seq, req)
		wg1.Add(1)
	}

	// Receive the responses and check for errors
	resps := []CommentSummaryRequest{}
	for i := 0; i < len(ids); i++ {
		resp := <-req
		wg1.Done()
		resps = append(resps, resp)
	}
	wg1.Wait()

	for _, resp := range resps {
		if resp.Err != nil {
			return CommentType{}, resp.Status, resp.Err
		}
	}

	// Sort them
	sort.Sort(CommentRequestBySeq(resps))

	// Extract the values
	for _, resp := range resps {
		m.Meta.Replies = append(m.Meta.Replies, resp.Item)
	}

	return m, http.StatusOK, nil
}

func GetNextOrLastCommentId(
	itemTypeId int64,
	itemId int64,
	timestamp time.Time,
	profileId int64,
) (
	int64,
	int,
	error,
) {

	// Gets the id for the first comment after the given timestamp
	// If no such comment, give the id for the last comment
	db, err := h.GetConnection()
	if err != nil {
		return 0, http.StatusInternalServerError, err
	}

	var commentId int64

	err = db.QueryRow(`--GetNextOrLastCommentId
SELECT comment_id
  FROM (
           (
              -- Get next comment
              SELECT f.item_id AS comment_id
                    ,f.last_modified AS created
                FROM flags f
                LEFT JOIN ignores i ON i.profile_id = $4
                                   AND i.item_type_id = 3
                                   AND i.item_id = f.created_by
               WHERE i.profile_id IS NULL
                 AND f.parent_item_type_id = $1
                 AND f.parent_item_id = $2
                 AND f.item_type_id = 4
                 AND f.microcosm_is_deleted IS NOT TRUE
                 AND f.microcosm_is_moderated IS NOT TRUE
                 AND f.parent_is_deleted IS NOT TRUE
                 AND f.parent_is_moderated IS NOT TRUE
                 AND f.item_is_deleted IS NOT TRUE
                 AND f.item_is_moderated IS NOT TRUE
                 AND f.last_modified > $3
               ORDER BY f.last_modified ASC
               FETCH FIRST 1 ROWS ONLY
           )
            UNION
           (
              -- Get last comment
              SELECT f.item_id AS comment_id
                    ,f.last_modified AS created
                FROM flags f
                LEFT JOIN ignores i ON i.profile_id = $4
                                   AND i.item_type_id = 3
                                   AND i.item_id = f.created_by
               WHERE i.profile_id IS NULL
                 AND f.parent_item_type_id = $1
                 AND f.parent_item_id = $2
                 AND f.item_type_id = 4
                 AND f.microcosm_is_deleted IS NOT TRUE
                 AND f.microcosm_is_moderated IS NOT TRUE
                 AND f.parent_is_deleted IS NOT TRUE
                 AND f.parent_is_moderated IS NOT TRUE
                 AND f.item_is_deleted IS NOT TRUE
                 AND f.item_is_moderated IS NOT TRUE
               ORDER BY f.last_modified DESC
               FETCH FIRST 1 ROWS ONLY
           )
       ) AS nextandlast
 ORDER BY created ASC
 FETCH FIRST 1 ROWS ONLY`,
		itemTypeId,
		itemId,
		timestamp,
		profileId,
	).Scan(
		&commentId,
	)
	if err != nil {
		return 0, http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Error getting next commentid for item: %+v", err),
		)
	}

	return commentId, http.StatusOK, nil
}

func GetLastComment(itemTypeId int64, itemId int64) (LastComment, int, error) {

	db, err := h.GetConnection()
	if err != nil {
		glog.Errorf("h.GetConnection() %+v", err)
		return LastComment{}, http.StatusInternalServerError, err
	}

	rows, err := db.Query(`
SELECT c.comment_id
      ,c.profile_id
      ,c.created
  FROM flags f
       JOIN comments c ON c.comment_id = f.item_id
 WHERE f.item_type_id = 4
   AND f.parent_item_type_id = $1
   AND f.parent_item_id = $2
   AND f.microcosm_is_deleted IS NOT TRUE
   AND f.microcosm_is_moderated IS NOT TRUE
   AND f.parent_is_deleted IS NOT TRUE
   AND f.parent_is_moderated IS NOT TRUE
   AND f.item_is_deleted IS NOT TRUE
   AND f.item_is_moderated IS NOT TRUE
 ORDER BY f.last_modified DESC
 LIMIT 1`,
		itemTypeId,
		itemId,
	)
	if err != nil {
		glog.Errorf("db.Query(%d, %d) %+v", itemTypeId, itemId, err)
		return LastComment{}, http.StatusInternalServerError, err
	}
	defer rows.Close()

	lastComment := LastComment{}

	for rows.Next() {
		err = rows.Scan(
			&lastComment.ID,
			&lastComment.CreatedById,
			&lastComment.Created,
		)
		if err != nil {
			glog.Errorf("rows.Scan() %+v", err)
			return LastComment{}, http.StatusInternalServerError, err
		}

		lastComment.Valid = true
	}
	err = rows.Err()
	if err != nil {
		glog.Errorf("rows.Err() %+v", err)
		return LastComment{}, http.StatusInternalServerError, err
	}
	rows.Close()

	return lastComment, http.StatusOK, nil
}

// SetCommentInReplyTo updates the in_reply_to value of a comment. This is
// only for imports as it is never anticipated that this value will change once
// it has been set.
func SetCommentInReplyTo(siteId int64, commentId int64, inReplyTo int64) error {

	if siteId == 0 || commentId == 0 || inReplyTo == 0 {
		return fmt.Errorf("Cannot accept zero input value")
	}

	tx, err := h.GetTransaction()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`UPDATE comments SET in_reply_to = $2 WHERE comment_id = $1`,
		commentId,
		inReplyTo,
	)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	go PurgeCache(h.ItemTypes[h.ItemTypeComment], commentId)

	return nil
}
