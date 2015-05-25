package models

import (
	"database/sql"
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

// Item is a set of minimal and common fields used by items that exist on the
// site, usually things that exist within a microcosm
type Item struct {
	MicrocosmID int64  `json:"microcosmId"`
	ItemTypeID  int64  `json:"-"`
	ItemType    string `json:"itemType"`

	ID    int64  `json:"id"`
	Title string `json:"title"`

	CommentCount int64 `json:"totalComments"`
	ViewCount    int64 `json:"totalViews"`

	LastCommentIDNullable        sql.NullInt64 `json:"-"`
	LastCommentID                int64         `json:"lastCommentId,omitempty"`
	LastCommentCreatedByNullable sql.NullInt64 `json:"-"`
	LastCommentCreatedBy         interface{}   `json:"lastCommentCreatedBy,omitempty"`
	LastCommentCreatedNullable   pq.NullTime   `json:"-"`
	LastCommentCreated           string        `json:"lastCommentCreated,omitempty"`

	Meta h.DefaultMetaType `json:"meta"`
}

// ItemRequest is an envelope to request an item via a channel
type ItemRequest struct {
	Item   Item
	Err    error
	Status int
	Seq    int
}

// ItemRequestBySeq it a sorted collection of ItemRequests
type ItemRequestBySeq []ItemRequest

// Len returns the length of the collection
func (v ItemRequestBySeq) Len() int { return len(v) }

// Swap exchanges two adjacent items in the collection
func (v ItemRequestBySeq) Swap(i, j int) { v[i], v[j] = v[j], v[i] }

// Less determines whether an item is greater in sequence than another
func (v ItemRequestBySeq) Less(i, j int) bool { return v[i].Seq < v[j].Seq }

// ItemSummary is used by all things that can be a child of a microcosm
type ItemSummary struct {
	// Common Fields
	ID          int64  `json:"id"`
	MicrocosmID int64  `json:"microcosmId,omitempty"`
	Title       string `json:"title"`
}

// ItemSummaryMeta is the meta object for an ItemSummary
type ItemSummaryMeta struct {
	CommentCount int64             `json:"totalComments"`
	ViewCount    int64             `json:"totalViews"`
	LastComment  interface{}       `json:"lastComment,omitempty"`
	Meta         h.SummaryMetaType `json:"meta"`
}

// LastComment encapsulates the last common on an item within a microcosm
type LastComment struct {
	ID int64 `json:"id"`
	h.CreatedType
	Valid bool `json:"-"`
}

// ItemDetail describes an item and the microcosm it belongs to
type ItemDetail struct {
	// Common Fields
	ID          int64  `json:"id"`
	MicrocosmID int64  `json:"microcosmId,omitempty"`
	Title       string `json:"title"`

	// Used during import to set the view count
	ViewCount int64 `json:"-"`
}

// ItemDetailCommentsAndMeta provides the comments for an item
type ItemDetailCommentsAndMeta struct {
	// Comments
	Comments h.ArrayType `json:"comments"`

	// Meta
	Meta h.DefaultMetaType `json:"meta"`
}

// FetchProfileSummaries populates a partially populated Item
func (m *Item) FetchProfileSummaries(siteID int64) (int, error) {

	profile, status, err := GetProfileSummary(siteID, m.Meta.CreatedByID)
	if err != nil {
		return status, err
	}
	m.Meta.CreatedBy = profile

	if m.LastCommentCreatedByNullable.Valid {
		profile, status, err :=
			GetProfileSummary(siteID, m.LastCommentCreatedByNullable.Int64)
		if err != nil {
			return status, err
		}
		m.LastCommentCreatedBy = profile
	}

	return http.StatusOK, nil
}

// IncrementViewCount increments the views of an item
func IncrementViewCount(itemTypeID int64, itemID int64) {

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
		itemTypeID,
		itemID,
	)
	if err != nil {
		glog.Error(err)
		return
	}
}

// IncrementItemCommentCount increments the comments of an item
func IncrementItemCommentCount(itemTypeID int64, itemID int64) {

	if itemTypeID != h.ItemTypes[h.ItemTypeConversation] &&
		itemTypeID != h.ItemTypes[h.ItemTypeEvent] &&
		itemTypeID != h.ItemTypes[h.ItemTypePoll] {
		return
	}

	db, err := h.GetConnection()
	if err != nil {
		glog.Error(err)
		return
	}

	switch itemTypeID {
	case h.ItemTypes[h.ItemTypeConversation]:
		_, err = db.Exec(`--Update Conversation Comment Count
UPDATE conversations
   SET comment_count = comment_count + 1
 WHERE conversation_id = $1`,
			itemID,
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
			itemID,
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
			itemID,
		)
		if err != nil {
			glog.Error(err)
			return
		}
	default:
		return
	}

	PurgeCache(itemTypeID, itemID)

	microcosmID := GetMicrocosmIDForItem(itemTypeID, itemID)
	if microcosmID > 0 {
		_, err = db.Exec(`--Update Microcosm Comment Count
UPDATE microcosms
   SET comment_count = comment_count + 1
 WHERE microcosm_id = $1`,
			microcosmID,
		)
		if err != nil {
			glog.Error(err)
			return
		}

		PurgeCache(h.ItemTypes[h.ItemTypeMicrocosm], microcosmID)
	}
}

// DecrementItemCommentCount is called when a comment is deleted from an item
func DecrementItemCommentCount(itemTypeID int64, itemID int64) {

	db, err := h.GetConnection()
	if err != nil {
		glog.Error(err)
		return
	}

	switch itemTypeID {
	case h.ItemTypes[h.ItemTypeConversation]:
		_, err = db.Exec(`--Update Conversation Comment Count
UPDATE conversations
   SET comment_count = comment_count - 1
 WHERE conversation_id = $1`,
			itemID,
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
			itemID,
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
			itemID,
		)
		if err != nil {
			glog.Error(err)
			return
		}
	default:
		glog.Error("Not yet implemented")
		return
	}

	PurgeCache(itemTypeID, itemID)

	microcosmID := GetMicrocosmIDForItem(itemTypeID, itemID)
	if microcosmID > 0 {
		_, err = db.Exec(`--Update Microcosm Comment Count
UPDATE microcosms
   SET comment_count = comment_count - 1
 WHERE microcosm_id = $1`,
			microcosmID,
		)
		if err != nil {
			glog.Error(err)
			return
		}

		PurgeCache(h.ItemTypes[h.ItemTypeMicrocosm], microcosmID)
	}
}

// GetAllItems fetches items within a microcosm
func GetAllItems(
	siteID int64,
	microcosmID int64,
	profileID int64,
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
		siteID,
		microcosmID,
		profileID,
	).Scan(
		&total,
	)
	if err != nil {
		glog.Error(err)
		return []SummaryContainer{}, 0, 0, http.StatusInternalServerError,
			fmt.Errorf("Database query failed: %v", err.Error())
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
		siteID,
		microcosmID,
		profileID,
		limit,
		offset,
	)
	if err != nil {
		glog.Error(err)
		return []SummaryContainer{}, 0, 0, http.StatusInternalServerError,
			fmt.Errorf("Database query failed: %v", err.Error())
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
			itemTypeID  int64
			itemID      int64
			hasUnread   bool
			isAttending bool
		)

		err = rows.Scan(
			&itemTypeID,
			&itemID,
			&hasUnread,
			&isAttending,
		)
		if err != nil {
			return []SummaryContainer{}, 0, 0, http.StatusInternalServerError,
				fmt.Errorf("Row parsing error: %v", err.Error())
		}

		unread[strconv.FormatInt(itemTypeID, 10)+`_`+
			strconv.FormatInt(itemID, 10)] = hasUnread

		if itemTypeID == h.ItemTypes[h.ItemTypeEvent] {
			attending[itemID] = isAttending
		}

		go HandleSummaryContainerRequest(
			siteID,
			itemTypeID,
			itemID,
			profileID,
			seq,
			req,
		)
		seq++
		wg1.Add(1)
	}
	err = rows.Err()
	if err != nil {
		return []SummaryContainer{}, 0, 0, http.StatusInternalServerError,
			fmt.Errorf("Error fetching rows: %v", err.Error())
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
			fmt.Errorf("Not enough records, "+
				"offset (%d) would return an empty page",
				offset,
			)
	}

	return ems, total, pages, http.StatusOK, nil
}

// GetMostRecentItem fetches the most recently updated item within a microcosm
func GetMostRecentItem(
	siteID int64,
	microcosmID int64,
	profileID int64,
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
		microcosmID,
	)
	if err != nil {
		glog.Errorf("db.Query(%d) %+v", microcosmID, err)
		return SummaryContainer{},
			http.StatusInternalServerError,
			fmt.Errorf("Database query failed")
	}
	defer rows.Close()

	var (
		m          SummaryContainer
		status     int
		itemTypeID int64
		itemID     int64
	)
	for rows.Next() {
		err = rows.Scan(
			&itemTypeID,
			&itemID,
		)
		if err != nil {
			glog.Errorf("rows.Scan() %+v", err)
			return SummaryContainer{},
				http.StatusInternalServerError,
				fmt.Errorf("Row parsing error")
		}

		m, status, err =
			GetSummaryContainer(siteID, itemTypeID, itemID, profileID)
		if err != nil {
			glog.Errorf(
				"GetSummaryContainer(%d, %d, %d, %d) %+v",
				siteID,
				itemTypeID,
				itemID,
				profileID,
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
			fmt.Errorf("Error fetching rows")
	}
	rows.Close()

	if itemID == 0 {
		glog.Info("itemId == 0")
		return SummaryContainer{}, http.StatusNotFound, nil
	}

	return m, http.StatusOK, nil
}

// GetItems fetches items for a microcosm
func GetItems(
	siteID int64,
	microcosmID int64,
	profileID int64,
	reqURL *url.URL,
) (
	h.ArrayType,
	int,
	error,
) {
	query := reqURL.Query()
	limit, offset, status, err := h.GetLimitAndOffset(query)
	if err != nil {
		return h.ArrayType{}, status, err
	}

	ems, total, pages, status, err :=
		GetAllItems(siteID, microcosmID, profileID, limit, offset)
	if err != nil {
		return h.ArrayType{}, status, err
	}

	m := h.ConstructArray(
		ems,
		h.APITypeComment,
		total,
		limit,
		offset,
		pages,
		reqURL,
	)

	return m, http.StatusOK, nil
}
