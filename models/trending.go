package models

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"sync"

	"github.com/golang/glog"

	h "github.com/microcosm-collective/microcosm/helpers"
)

// TrendingItems is an array of TrendingItem
type TrendingItems struct {
	Items h.ArrayType    `json:"items"`
	Meta  h.CoreMetaType `json:"meta"`
}

// TrendingItem encapsulates a list of items currently trending
type TrendingItem struct {
	ItemType   string      `json:"itemType"`
	ItemTypeID int64       `json:"-"`
	ItemID     int64       `json:"-"`
	Item       interface{} `json:"item"`
	Score      int64       `json:"-"`
}

// GetTrending returns a paginated list of items on a site ordered by their
// activity score. Profile ID is used to check read permission on each item.
func GetTrending(
	siteID int64,
	profileID int64,
	limit int64,
	offset int64,
) (
	[]TrendingItem,
	int64,
	int64,
	int,
	error,
) {

	db, err := h.GetConnection()
	if err != nil {
		glog.Errorf("h.GetConnection() %+v", err)
		return []TrendingItem{}, 0, 0, http.StatusInternalServerError, err
	}

	// Fetch items with an activity score, check permissions and whether
	// it has been deleted.
	//
	// This implements Newton's Law of Cooling, basically we measure a
	// temperature (comment count) and then decay the temperature along
	// an exponential curve according to the age (in half-days) since the
	// last comment was made.
	//
	// We penalise older items by dividing the score by the age of the item
	rows, err := db.Query(`
WITH m AS (
   SELECT microcosm_id
     FROM microcosms
    WHERE site_id = $1
      AND (get_effective_permissions($1,microcosm_id,2,microcosm_id,$2)).can_read IS TRUE
)
SELECT item_id
      ,item_type_id
      ,has_unread(item_type_id, item_id, $2) AS unread
  FROM (
           SELECT f1.item_id
                 ,f1.item_type_id
                 ,COUNT(f2.item_id) -- temperature
                     *
                     (
                      EXP(
                          -0.25 -- cooling rate
                          *
                          EXTRACT(EPOCH FROM (NOW()-f1.last_modified)::interval)/60/60/12  -- half days since last update
                      ) / EXTRACT(EPOCH FROM (NOW()-COALESCE(c.created, e.created, p.created))::interval)/60/60/24/12 -- months since creation
                     )
                  AS score
             FROM flags f1
                  JOIN flags f2 ON f2.site_id = f1.site_id
                                    AND f2.parent_item_type_id = f1.item_type_id
                                    AND f2.parent_item_id = f1.item_id
                                    AND f2.last_modified > NOW()-'2 weeks'::interval -- comments counting towards temperature
             LEFT JOIN conversations c ON f1.item_type_id = 6
                                      AND c.conversation_id = f1.item_id
             LEFT JOIN events e ON f1.item_type_id = 9
                               AND e.event_id = f1.item_id
             LEFT JOIN polls p ON f1.item_type_id = 6
                              AND p.poll_id = f1.item_id
            WHERE f1.site_id = $1
              AND f1.microcosm_is_deleted IS NOT TRUE
              AND f1.microcosm_is_moderated IS NOT TRUE
              AND f1.item_is_deleted IS NOT TRUE
              AND f1.item_is_moderated IS NOT TRUE
              AND f1.parent_is_deleted IS NOT TRUE
              AND f1.parent_is_moderated IS NOT TRUE
              AND f1.item_type_id IN (6, 7, 9)
              AND f1.last_modified > NOW()-'1 months'::interval -- items to be included in results
              AND f1.microcosm_id IN (SELECT microcosm_id FROM m)
            GROUP BY f1.item_type_id
                    ,f1.item_id
                    ,f1.last_modified
                    ,COALESCE(c.created, e.created, p.created)
            ORDER BY score DESC
            FETCH FIRST 25 ROWS ONLY
       ) AS trending`,
		siteID,
		profileID,
	)
	if err != nil {
		return []TrendingItem{}, 0, 0, http.StatusInternalServerError,
			fmt.Errorf("database query failed: %v", err.Error())
	}
	defer rows.Close()

	var rowCount int64
	trendingItems := []TrendingItem{}
	// [itemTypeID_itemID] = hasUnread
	unread := map[string]bool{}
	for rows.Next() {
		var t TrendingItem
		var hasUnread bool
		err = rows.Scan(
			&t.ItemID,
			&t.ItemTypeID,
			&hasUnread,
		)
		if err != nil {
			glog.Errorf("Trending rows.Scan(): %+v", err)
			return []TrendingItem{}, 0, 0, http.StatusInternalServerError,
				fmt.Errorf("trending: row parsing error")
		}

		unread[strconv.FormatInt(t.ItemTypeID, 10)+`_`+
			strconv.FormatInt(t.ItemID, 10)] = hasUnread

		itemType, err := h.GetMapStringFromInt(h.ItemTypes, t.ItemTypeID)
		if err != nil {
			glog.Errorf(
				"h.GetMapStringFromInt(h.ItemTypes, %d) %+v",
				t.ItemTypeID,
				err,
			)
			return []TrendingItem{}, 0, 0, http.StatusInternalServerError, err
		}
		t.ItemType = itemType
		trendingItems = append(trendingItems, t)

		rowCount++
	}
	err = rows.Err()
	if err != nil {
		glog.Errorf("Trending rows.Err(): %+v", err)
		return []TrendingItem{}, 0, 0, http.StatusInternalServerError,
			fmt.Errorf("error fetching rows")
	}
	rows.Close()

	maxOffset := h.GetMaxOffset(rowCount, 10)
	if offset > maxOffset {
		glog.Infoln("offset > maxOffset")
		return []TrendingItem{}, 0, 0, http.StatusBadRequest,
			fmt.Errorf("not enough records, "+
				"offset (%d) would return an empty page", offset)
	}

	// Fetch summary for each item.
	var wg1 sync.WaitGroup
	req := make(chan SummaryContainerRequest)
	defer close(req)

	seq := 0
	for i := 0; i < len(trendingItems); i++ {
		go HandleSummaryContainerRequest(
			siteID,
			trendingItems[i].ItemTypeID,
			trendingItems[i].ItemID,
			profileID,
			seq,
			req,
		)
		seq++
		wg1.Add(1)
	}

	resps := []SummaryContainerRequest{}
	for i := 0; i < seq; i++ {
		resp := <-req
		wg1.Done()
		resps = append(resps, resp)
	}
	wg1.Wait()

	for _, resp := range resps {
		if resp.Err != nil {
			return []TrendingItem{}, 0, 0, resp.Status, resp.Err
		}
	}

	sort.Sort(SummaryContainerRequestsBySeq(resps))

	seq = 0
	for i := 0; i < len(trendingItems); i++ {
		m := resps[seq].Item

		switch m.Summary.(type) {
		case ConversationSummaryType:
			summary := m.Summary.(ConversationSummaryType)
			summary.Meta.Flags.Unread =
				unread[strconv.FormatInt(m.ItemTypeID, 10)+`_`+
					strconv.FormatInt(m.ItemID, 10)]

			m.Summary = summary

		case EventSummaryType:
			summary := m.Summary.(EventSummaryType)
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

		trendingItems[i].Item = m.Summary

		seq++
	}

	pages := h.GetPageCount(rowCount, rowCount)
	return trendingItems, rowCount, pages, http.StatusOK, nil
}
