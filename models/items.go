package models

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"sync"

	"github.com/golang/glog"
	"github.com/lib/pq"

	h "github.com/microcosm-cc/microcosm/helpers"
)

// Abstract row from the database ITEMS table
type Item struct {
	MicrocosmId int64  `json:"microcosmId"`
	ItemTypeId  int64  `json:"-"`
	ItemType    string `json:"itemType"`

	Id    int64  `json:"id"`
	Title string `json:"title"`

	CommentCount int64 `json:"totalComments"`
	ViewCount    int64 `json:"totalViews"`

	LastCommentIdNullable        sql.NullInt64 `json:"-"`
	LastCommentId                int64         `json:"lastCommentId,omitempty"`
	LastCommentCreatedByNullable sql.NullInt64 `json:"-"`
	LastCommentCreatedBy         interface{}   `json:"lastCommentCreatedBy,omitempty"`
	LastCommentCreatedNullable   pq.NullTime   `json:"-"`
	LastCommentCreated           string        `json:"lastCommentCreated,omitempty"`

	Meta h.DefaultMetaType `json:"meta"`
}

type ItemRequest struct {
	Item   Item
	Err    error
	Status int
	Seq    int
}

type ItemRequestBySeq []ItemRequest

func (v ItemRequestBySeq) Len() int           { return len(v) }
func (v ItemRequestBySeq) Swap(i, j int)      { v[i], v[j] = v[j], v[i] }
func (v ItemRequestBySeq) Less(i, j int) bool { return v[i].Seq < v[j].Seq }

// Core item types used by all things that can be a child of a Microcosm
type ItemSummary struct {
	// Common Fields
	Id          int64  `json:"id"`
	MicrocosmId int64  `json:"microcosmId,omitempty"`
	Title       string `json:"title"`
}

type ItemSummaryMeta struct {
	CommentCount int64             `json:"totalComments"`
	ViewCount    int64             `json:"totalViews"`
	LastComment  interface{}       `json:"lastComment,omitempty"`
	Meta         h.SummaryMetaType `json:"meta"`
}

type LastComment struct {
	Id int64 `json:"id"`
	h.CreatedType
	Valid bool `json:"-"`
}

type ItemDetail struct {
	// Common Fields
	Id          int64  `json:"id"`
	MicrocosmId int64  `json:"microcosmId,omitempty"`
	Title       string `json:"title"`

	// Used during import to set the view count
	ViewCount int64 `json:"-"`
}

type ItemDetailCommentsAndMeta struct {
	// Comments
	Comments h.ArrayType `json:"comments"`

	// Meta
	Meta h.DefaultMetaType `json:"meta"`
}

func (m *Item) FetchProfileSummaries(siteId int64) (int, error) {

	profile, status, err := GetProfileSummary(siteId, m.Meta.CreatedById)
	if err != nil {
		return status, err
	}
	m.Meta.CreatedBy = profile

	if m.LastCommentCreatedByNullable.Valid {
		profile, status, err :=
			GetProfileSummary(siteId, m.LastCommentCreatedByNullable.Int64)
		if err != nil {
			return status, err
		}
		m.LastCommentCreatedBy = profile
	}

	return http.StatusOK, nil
}

func IncrementViewCount(itemTypeId int64, itemId int64) {

	// No transaction as we don't care for accuracy on these updates
	// Note: This function doesn't even return errors, we don't even care
	// if the occasional INSERT fails.
	db, err := h.GetConnection()
	if err != nil {
		glog.Error(err)
		return
	}

	// Integrity insert, gets rolled up and updated on cron
	_, err = db.Exec(
		`INSERT INTO views(item_type_id, item_id) VALUES ($1, $2)`,
		itemTypeId,
		itemId,
	)
	if err != nil {
		glog.Error(err)
		return
	}
}

func IncrementItemCommentCount(itemTypeId int64, itemId int64) {

	if itemTypeId != h.ItemTypes[h.ItemTypeConversation] &&
		itemTypeId != h.ItemTypes[h.ItemTypeEvent] &&
		itemTypeId != h.ItemTypes[h.ItemTypePoll] {
		return
	}

	db, err := h.GetConnection()
	if err != nil {
		glog.Error(err)
		return
	}

	switch itemTypeId {
	case h.ItemTypes[h.ItemTypeConversation]:
		_, err = db.Exec(`--Update Conversation Comment Count
UPDATE conversations
   SET comment_count = comment_count + 1
 WHERE conversation_id = $1`,
			itemId,
		)
		if err != nil {
			glog.Error(err)
			return
		}
	case h.ItemTypes[h.ItemTypeEvent]:
		_, err = db.Exec(`--Update Event Comment Count
UPDATE events
   SET comment_count = comment_count + 1
 WHERE event_id = $1`,
			itemId,
		)
		if err != nil {
			glog.Error(err)
			return
		}
	case h.ItemTypes[h.ItemTypePoll]:
		_, err = db.Exec(`--Update Poll Comment Count
UPDATE polls
   SET comment_count = comment_count + 1
 WHERE poll_id = $1`,
			itemId,
		)
		if err != nil {
			glog.Error(err)
			return
		}
	default:
		return
	}

	PurgeCache(itemTypeId, itemId)

	microcosmId := GetMicrocosmIdForItem(itemTypeId, itemId)
	if microcosmId > 0 {
		_, err = db.Exec(`--Update Microcosm Comment Count
UPDATE microcosms
   SET comment_count = comment_count + 1
 WHERE microcosm_id = $1`,
			microcosmId,
		)
		if err != nil {
			glog.Error(err)
			return
		}

		PurgeCache(h.ItemTypes[h.ItemTypeMicrocosm], microcosmId)
	}
}

func DecrementItemCommentCount(itemTypeId int64, itemId int64) {

	db, err := h.GetConnection()
	if err != nil {
		glog.Error(err)
		return
	}

	switch itemTypeId {
	case h.ItemTypes[h.ItemTypeConversation]:
		_, err = db.Exec(`--Update Conversation Comment Count
UPDATE conversations
   SET comment_count = comment_count - 1
 WHERE conversation_id = $1`,
			itemId,
		)
		if err != nil {
			glog.Error(err)
			return
		}
	case h.ItemTypes[h.ItemTypeEvent]:
		_, err = db.Exec(`--Update Event Comment Count
UPDATE events
   SET comment_count = comment_count - 1
 WHERE event_id = $1`,
			itemId,
		)
		if err != nil {
			glog.Error(err)
			return
		}
	case h.ItemTypes[h.ItemTypePoll]:
		_, err = db.Exec(`--Update Poll Comment Count
UPDATE polls
   SET comment_count = comment_count - 1
 WHERE poll_id = $1`,
			itemId,
		)
		if err != nil {
			glog.Error(err)
			return
		}
	default:
		glog.Error("Not yet implemented")
		return
	}

	PurgeCache(itemTypeId, itemId)

	microcosmId := GetMicrocosmIdForItem(itemTypeId, itemId)
	if microcosmId > 0 {
		_, err = db.Exec(`--Update Microcosm Comment Count
UPDATE microcosms
   SET comment_count = comment_count - 1
 WHERE microcosm_id = $1`,
			microcosmId,
		)
		if err != nil {
			glog.Error(err)
			return
		}

		PurgeCache(h.ItemTypes[h.ItemTypeMicrocosm], microcosmId)
	}
}

func GetAllItems(
	siteId int64,
	microcosmId int64,
	profileId int64,
	limit int64,
	offset int64,
) (
	[]SummaryContainer,
	int64,
	int64,
	int,
	error,
) {

	// Retrieve resources
	db, err := h.GetConnection()
	if err != nil {
		return []SummaryContainer{}, 0, 0, http.StatusInternalServerError, err
	}

	sqlFromWhere := `
          FROM flags f
          LEFT JOIN ignores i ON i.profile_id = $3
                             AND i.item_type_id = f.item_type_id
                             AND i.item_id = f.item_id
         WHERE f.microcosm_id = (
                   SELECT $2::bigint AS microcosm_id
                    WHERE (get_effective_permissions($1, $2, 2, $2, $3)).can_read IS TRUE
               )
           AND (f.item_type_id = 6 OR f.item_type_id = 9)
           AND f.site_id = $1
           AND i.profile_id IS NULL
           AND f.microcosm_is_deleted IS NOT TRUE
           AND f.microcosm_is_moderated IS NOT TRUE
           AND f.parent_is_deleted IS NOT TRUE
           AND f.parent_is_moderated IS NOT TRUE
           AND f.item_is_deleted IS NOT TRUE
           AND f.item_is_moderated IS NOT TRUE`

	var total int64
	err = db.QueryRow(`
SELECT COUNT(*) AS total`+sqlFromWhere,
		siteId,
		microcosmId,
		profileId,
	).Scan(
		&total,
	)
	if err != nil {
		glog.Error(err)
		return []SummaryContainer{}, 0, 0, http.StatusInternalServerError,
			errors.New(
				fmt.Sprintf("Database query failed: %v", err.Error()),
			)
	}

	rows, err := db.Query(`
SELECT item_type_id
      ,item_id
      ,has_unread(item_type_id, item_id, $3)
      ,CASE WHEN item_type_id = 9
           THEN is_attending(item_id, $3)
           ELSE FALSE
       END AS is_attending
  FROM (
        SELECT f.item_type_id
              ,f.item_id`+sqlFromWhere+`
         ORDER BY f.item_is_sticky DESC
                 ,f.last_modified DESC
         LIMIT $4
        OFFSET $5
       ) r`,
		siteId,
		microcosmId,
		profileId,
		limit,
		offset,
	)
	if err != nil {
		glog.Error(err)
		return []SummaryContainer{}, 0, 0, http.StatusInternalServerError,
			errors.New(
				fmt.Sprintf("Database query failed: %v", err.Error()),
			)
	}
	defer rows.Close()

	var wg1 sync.WaitGroup
	req := make(chan SummaryContainerRequest)
	defer close(req)

	// [itemTypeId_itemId] = hasUnread
	unread := map[string]bool{}
	attending := map[int64]bool{}

	seq := 0
	for rows.Next() {
		var (
			itemTypeId  int64
			itemId      int64
			hasUnread   bool
			isAttending bool
		)

		err = rows.Scan(
			&itemTypeId,
			&itemId,
			&hasUnread,
			&isAttending,
		)
		if err != nil {
			return []SummaryContainer{}, 0, 0, http.StatusInternalServerError,
				errors.New(
					fmt.Sprintf("Row parsing error: %v", err.Error()),
				)
		}

		unread[strconv.FormatInt(itemTypeId, 10)+`_`+
			strconv.FormatInt(itemId, 10)] = hasUnread

		if itemTypeId == h.ItemTypes[h.ItemTypeEvent] {
			attending[itemId] = isAttending
		}

		go HandleSummaryContainerRequest(
			siteId,
			itemTypeId,
			itemId,
			profileId,
			seq,
			req,
		)
		seq++
		wg1.Add(1)
	}
	err = rows.Err()
	if err != nil {
		return []SummaryContainer{}, 0, 0, http.StatusInternalServerError,
			errors.New(
				fmt.Sprintf("Error fetching rows: %v", err.Error()),
			)
	}
	rows.Close()

	resps := []SummaryContainerRequest{}
	for i := 0; i < seq; i++ {
		resp := <-req
		wg1.Done()
		resps = append(resps, resp)
	}
	wg1.Wait()

	for _, resp := range resps {
		if resp.Err != nil {
			return []SummaryContainer{}, 0, 0, resp.Status, resp.Err
		}
	}

	sort.Sort(SummaryContainerRequestsBySeq(resps))

	ems := []SummaryContainer{}
	for _, resp := range resps {
		m := resp.Item

		switch m.Summary.(type) {
		case ConversationSummaryType:
			summary := m.Summary.(ConversationSummaryType)
			summary.Meta.Flags.Unread =
				unread[strconv.FormatInt(m.ItemTypeID, 10)+`_`+
					strconv.FormatInt(m.ItemID, 10)]

			m.Summary = summary

		case EventSummaryType:
			summary := m.Summary.(EventSummaryType)
			summary.Meta.Flags.Attending = attending[m.ItemID]
			summary.Meta.Flags.Unread =
				unread[strconv.FormatInt(m.ItemTypeID, 10)+`_`+
					strconv.FormatInt(m.ItemID, 10)]

			m.Summary = summary

		case PollSummaryType:
			summary := m.Summary.(PollSummaryType)
			summary.Meta.Flags.Unread =
				unread[strconv.FormatInt(m.ItemTypeID, 10)+`_`+
					strconv.FormatInt(m.ItemID, 10)]

			m.Summary = summary

		default:
		}

		ems = append(ems, m)
	}

	pages := h.GetPageCount(total, limit)
	maxOffset := h.GetMaxOffset(total, limit)

	if offset > maxOffset {
		return []SummaryContainer{}, 0, 0, http.StatusBadRequest,
			errors.New(
				fmt.Sprintf("Not enough records, "+
					"offset (%d) would return an empty page.",
					offset,
				),
			)
	}

	return ems, total, pages, http.StatusOK, nil
}

func GetMostRecentItem(
	siteId int64,
	microcosmId int64,
	profileId int64,
) (
	SummaryContainer,
	int,
	error,
) {

	// Retrieve resources
	db, err := h.GetConnection()
	if err != nil {
		glog.Errorf("h.GetConnection() %+v", err)
		return SummaryContainer{}, http.StatusInternalServerError, err
	}

	rows, err := db.Query(`
SELECT item_type_id
      ,item_id
  FROM flags
 WHERE microcosm_id = $1
   AND microcosm_is_deleted IS NOT TRUE
   AND microcosm_is_moderated IS NOT TRUE
   AND parent_is_deleted IS NOT TRUE
   AND parent_is_moderated IS NOT TRUE
   AND item_is_deleted IS NOT TRUE
   AND item_is_moderated IS NOT TRUE
   AND (item_type_id = 6
    OR item_type_id = 9)
 ORDER BY last_modified DESC
 LIMIT 1`,
		microcosmId,
	)
	if err != nil {
		glog.Errorf("db.Query(%d) %+v", microcosmId, err)
		return SummaryContainer{},
			http.StatusInternalServerError,
			errors.New("Database query failed")
	}
	defer rows.Close()

	var (
		m          SummaryContainer
		status     int
		itemTypeId int64
		itemId     int64
	)
	for rows.Next() {
		err = rows.Scan(
			&itemTypeId,
			&itemId,
		)
		if err != nil {
			glog.Errorf("rows.Scan() %+v", err)
			return SummaryContainer{},
				http.StatusInternalServerError,
				errors.New("Row parsing error")
		}

		m, status, err =
			GetSummaryContainer(siteId, itemTypeId, itemId, profileId)
		if err != nil {
			glog.Errorf(
				"GetSummaryContainer(%d, %d, %d, %d) %+v",
				siteId,
				itemTypeId,
				itemId,
				profileId,
				err,
			)
			return SummaryContainer{}, status, err
		}
	}
	err = rows.Err()
	if err != nil {
		glog.Errorf("rows.Err() %+v", err)
		return SummaryContainer{},
			http.StatusInternalServerError,
			errors.New("Error fetching rows")
	}
	rows.Close()

	if itemId == 0 {
		glog.Info("itemId == 0")
		return SummaryContainer{}, http.StatusNotFound, nil
	}

	return m, http.StatusOK, nil
}

func GetItems(
	siteId int64,
	microcosmId int64,
	profileId int64,
	reqUrl *url.URL,
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

	ems, total, pages, status, err :=
		GetAllItems(siteId, microcosmId, profileId, limit, offset)
	if err != nil {
		return h.ArrayType{}, status, err
	}

	m := h.ConstructArray(
		ems,
		h.ApiTypeComment,
		total,
		limit,
		offset,
		pages,
		reqUrl,
	)

	return m, http.StatusOK, nil
}
