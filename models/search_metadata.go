package models

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/golang/glog"

	h "github.com/microcosm-cc/microcosm/helpers"
)

func searchMetaData(
	siteId int64,
	searchUrl url.URL,
	profileId int64,
	m SearchResults,
) (
	SearchResults,
	int,
	error,
) {

	limit, offset, status, err := h.GetLimitAndOffset(searchUrl.Query())
	if err != nil {
		glog.Errorf("h.GetLimitAndOffset(searchUrl.Query()) %+v", err)
		return m, status, err
	}

	start := time.Now()

	// The goal is to produce a piece of SQL that looks at just the flags table
	// and fetches a list of the items that we care about.
	//
	// Our target SQL should look roughly like this (fetches all viewable comments):
	//
	// WITH m AS (
	//     SELECT microcosm_id
	//       FROM microcosms
	//      WHERE site_id = 2
	//        AND (get_effective_permissions(2,microcosm_id,2,microcosm_id,7)).can_read IS TRUE
	// ), h AS (
	//     SELECT huddle_id
	//       FROM huddle_profiles
	//      WHERE profile_id = 7
	// )
	// SELECT item_type_id
	//       ,item_id
	//   FROM flags
	//  WHERE item_type_id = 4
	//    AND (
	//            site_is_deleted
	//        AND microcosm_is_deleted
	//        AND parent_is_deleted
	//        AND item_is_deleted
	//        ) IS NOT TRUE
	//    AND (
	//        (-- Things that are public by default and low in quantity
	//            item_type_id IN (1,3)
	//         OR parent_item_type_id IN (3)
	//        )
	//     OR (-- Things directly in microcosms
	//            item_type_id IN (2,6,7,9)
	//        AND COALESCE(microcosm_id, item_id) IN (SELECT microcosm_id FROM m)
	//        )
	//     OR (-- Comments on things in microcosms
	//            item_type_id = 4
	//        AND parent_item_type_id IN (6,7,9)
	//        AND microcosm_id IN (SELECT microcosm_id FROM m)
	//        )
	//     OR (-- Huddles
	//            item_type_id = 5
	//        AND item_id IN (SELECT huddle_id FROM h)
	//        )
	//     OR (-- Comments on things in huddles
	//            item_type_id = 4
	//        AND parent_item_type_id = 5
	//        AND parent_item_id IN (SELECT huddle_id FROM h)
	//        )
	//        )
	//  ORDER BY last_modified DESC
	//  LIMIT 25;
	//
	// The key is to only put into the query the bits that will definitely be
	// used.

	// Process search options

	var filterFollowing string
	var filterItemTypes string
	var filterItems string
	var includeHuddles bool
	var includeComments bool
	var joinEvents bool

	orderBy := `rank DESC
         ,f.last_modified DESC`

	switch m.Query.Sort {
	case "date":
		orderBy = `f.last_modified DESC`
	case "oldest":
		joinEvents = true
		orderBy = `e."when" ASC`
	case "newest":
		joinEvents = true
		orderBy = `e."when" DESC`
	}

	if m.Query.Following {
		filterFollowing = `
       JOIN watchers w ON w.item_type_id = f.item_type_id
                      AND w.item_id = f.item_id
                      AND w.profile_id = $2`
	}

	if len(m.Query.ItemTypeIds) > 0 {
		var inList string

		// Take care of the item types
		for i, v := range m.Query.ItemTypeIds {

			switch v {
			case h.ItemTypes[h.ItemTypeComment]:
				includeComments = true
			case h.ItemTypes[h.ItemTypeHuddle]:
				includeHuddles = true
			}

			inList += strconv.FormatInt(v, 10)
			if i < len(m.Query.ItemTypeIds)-1 {
				inList += `,`
			}
		}

		if len(m.Query.ItemTypeIds) == 1 {
			filterItemTypes = fmt.Sprintf(`
   AND f.item_type_id = %d`,
				m.Query.ItemTypeIds[0],
			)
		} else {
			if includeComments {
				filterItemTypes = `
   AND (   (f.item_type_id <> 4 AND f.item_type_id IN (` + inList + `))
        OR (f.item_type_id = 4 AND f.parent_item_type_id IN (` + inList + `))
       )`
			} else {
				filterItemTypes = `
   AND f.item_type_id IN (` + inList + `)`
			}
		}

		// Take care of the item ids, which are only valid when we have item
		// types
		if len(m.Query.ItemIds) > 0 {

			if len(m.Query.ItemIds) == 1 {
				if includeComments {
					filterItems = fmt.Sprintf(`
   AND (   (f.item_type_id <> 4 AND f.item_id = %d)
        OR (f.item_type_id = 4 AND f.parent_item_id = %d)
       )`,
						m.Query.ItemIds[0],
						m.Query.ItemIds[0],
					)
				} else {
					filterItems = fmt.Sprintf(`
   AND f.item_id = %d`,
						m.Query.ItemIds[0],
					)
				}
			} else {
				var inList = ``
				for i, v := range m.Query.ItemIds {
					inList += strconv.FormatInt(v, 10)
					if i < len(m.Query.ItemIds)-1 {
						inList += `,`
					}
				}

				if includeComments {
					filterItems = `
   AND (   (f.item_type_id <> 4 AND f.item_id IN (` + inList + `))
        OR (f.item_type_id = 4 AND f.parent_item_id IN (` + inList + `))
       )`
				} else {
					filterItems = `
   AND f.item_id IN (` + inList + `)`
				}
			}
		}
	}

	var filterProfileId string
	if m.Query.ProfileId > 0 {
		filterProfileId = fmt.Sprintf(`
   AND f.created_by = %d`, m.Query.ProfileId)
	}

	var filterMicrocosmIds string
	if len(m.Query.MicrocosmIds) > 0 {
		if len(m.Query.MicrocosmIds) == 1 {
			filterMicrocosmIds = fmt.Sprintf(`
   AND f.microcosm_id = %d`, m.Query.MicrocosmIds[0])
			includeHuddles = false
		} else {
			var inList = ``

			for i, v := range m.Query.MicrocosmIds {
				inList += strconv.FormatInt(v, 10)
				if i < len(m.Query.MicrocosmIds)-1 {
					inList += `,`
				}
			}
			filterMicrocosmIds = `
   AND f.microcosm_id IN (` + inList + `)`
		}
	}

	var filterModified string
	if !m.Query.SinceTime.IsZero() || !m.Query.UntilTime.IsZero() {
		if m.Query.UntilTime.IsZero() {
			filterModified = fmt.Sprintf(`
   AND f.last_modified > to_timestamp(%d)`,
				m.Query.SinceTime.Unix(),
			)

		} else if m.Query.SinceTime.IsZero() {
			filterModified = fmt.Sprintf(`
   AND f.last_modified < to_timestamp(%d)`,
				m.Query.UntilTime.Unix(),
			)
		} else {
			filterModified = fmt.Sprintf(`
   AND f.last_modified BETWEEN to_timestamp(%d) AND to_timestamp(%d)`,
				m.Query.SinceTime.Unix(),
				m.Query.UntilTime.Unix(),
			)
		}
	}

	var (
		filterEventsJoin  string
		filterEventsWhere string
	)
	if !m.Query.EventAfterTime.IsZero() || !m.Query.EventBeforeTime.IsZero() {
		joinEvents = true

		if m.Query.EventBeforeTime.IsZero() {
			filterModified = fmt.Sprintf(`
   AND e."when" > to_timestamp(%d)`,
				m.Query.EventAfterTime.Unix(),
			)

		} else if m.Query.EventAfterTime.IsZero() {
			filterModified = fmt.Sprintf(`
   AND e."when" < to_timestamp(%d)`,
				m.Query.EventBeforeTime.Unix(),
			)
		} else {
			filterModified = fmt.Sprintf(`
   AND e."when" BETWEEN to_timestamp(%d) AND to_timestamp(%d)`,
				m.Query.EventAfterTime.Unix(),
				m.Query.EventBeforeTime.Unix(),
			)
		}
	}

	if joinEvents || m.Query.Attendee {
		filterEventsJoin = `
       JOIN events e ON e.event_id = f.item_id`

		if m.Query.Attendee {
			filterEventsJoin += `
       JOIN attendees a ON a.event_id = e.event_id
                       AND a.profile_id = ` + strconv.FormatInt(profileId, 10) + `
                       AND a.state_id = 1`
		}
	}

	// These make up our SQL query
	sqlSelect := `
SELECT 0,0,0,NULL,NULL,NOW(),0,''`
	sqlFromWhere := `
  FROM flags WHERE 1=2`

	// Query with only meta data
	sqlWith := `
WITH m AS (
    SELECT m.microcosm_id
      FROM microcosms m
      LEFT JOIN permissions_cache p ON p.site_id = m.site_id
                                   AND p.item_type_id = 2
                                   AND p.item_id = m.microcosm_id
                                   AND p.profile_id = $2
           LEFT JOIN ignores i ON i.profile_id = $2
                              AND i.item_type_id = 2
                              AND i.item_id = m.microcosm_id
     WHERE m.site_id = $1
       AND m.is_deleted IS NOT TRUE
       AND m.is_moderated IS NOT TRUE
       AND i.profile_id IS NULL
       AND (
               (p.can_read IS NOT NULL AND p.can_read IS TRUE)
            OR (get_effective_permissions($1,m.microcosm_id,2,m.microcosm_id,$2)).can_read IS TRUE
           )
)`
	if includeHuddles || includeComments {
		if filterModified != "" {
			sqlWith += `, h AS (
    SELECT hp.huddle_id
      FROM huddle_profiles hp
      JOIN flags f ON f.item_type_id = 5
                  AND f.item_id = hp.huddle_id
     WHERE hp.profile_id = $2` + filterModified + `
)`
		} else {
			sqlWith += `, h AS (
    SELECT huddle_id
      FROM huddle_profiles
     WHERE profile_id = $2
)`
		}

	}

	sqlSelect = `
SELECT f.item_type_id
      ,f.item_id
      ,f.parent_item_type_id
      ,f.parent_item_id
      ,f.last_modified
      ,0.5 AS rank
      ,'' AS highlight`

	sqlFromWhere = `
  FROM flags f
  LEFT JOIN ignores i ON i.profile_id = $2
                     AND i.item_type_id = f.item_type_id
                     AND i.item_id = f.item_id` +
		filterFollowing +
		filterEventsJoin + `
 WHERE f.site_id = $1
   AND i.profile_id IS NULL` +
		filterModified +
		filterMicrocosmIds +
		filterItemTypes +
		filterItems +
		filterProfileId +
		filterEventsWhere + `
   AND f.microcosm_is_deleted IS NOT TRUE
   AND f.microcosm_is_moderated IS NOT TRUE
   AND f.parent_is_deleted IS NOT TRUE
   AND f.parent_is_moderated IS NOT TRUE
   AND f.item_is_deleted IS NOT TRUE
   AND f.item_is_moderated IS NOT TRUE
   AND (
       (-- Things that are public by default and low in quantity
           f.item_type_id IN (1,3)
        OR f.parent_item_type_id IN (3)
       )
    OR (-- Things directly in microcosms
           f.item_type_id IN (2,6,7,9)
       AND COALESCE(f.microcosm_id, f.item_id) IN (SELECT microcosm_id FROM m)
       )`

	if includeComments {
		sqlFromWhere += `
    OR (-- Comments on things in microcosms
           f.item_type_id = 4
       AND f.parent_item_type_id IN (6,7,9)
       AND f.microcosm_id IN (SELECT microcosm_id FROM m)
       )
    OR (-- Comments on things in huddles
           f.item_type_id = 4
       AND f.parent_item_type_id = 5
       AND f.parent_item_id IN (SELECT huddle_id FROM h)
       )`
	}

	if includeHuddles {
		sqlFromWhere += `
    OR (-- Huddles
           f.item_type_id = 5
       AND f.item_id IN (SELECT huddle_id FROM h)
       )`
	}

	sqlFromWhere += `
       )`

	sqlOrderLimit := `
 ORDER BY ` + orderBy + `
 LIMIT $3
OFFSET $4`

	db, err := h.GetConnection()
	if err != nil {
		glog.Errorf("h.GetConnection() %+v", err)
		return m, http.StatusInternalServerError, err
	}

	var total int64
	err = db.QueryRow(
		sqlWith+`SELECT COUNT(*)`+sqlFromWhere,
		siteId,
		profileId,
	).Scan(&total)
	if err != nil {
		glog.Error(err)
		return m, http.StatusInternalServerError, err
	}

	// This nested query is used to run the `has_unread` query on only the rows
	// that are returned, rather than on all rows in the underlying query before
	// limit has been applied.
	rows, err := db.Query(`
SELECT item_type_id
      ,item_id
      ,parent_item_type_id
      ,parent_item_id
      ,last_modified
      ,rank
      ,highlight
      ,has_unread(item_type_id, item_id, $2)
  FROM (`+
		sqlWith+
		sqlSelect+
		sqlFromWhere+
		sqlOrderLimit+
		`) r`,
		siteId,
		profileId,
		limit,
		offset,
	)
	if err != nil {
		glog.Errorf(
			"stmt.Query(%d, %s, %d, %d, %d) %+v",
			siteId,
			m.Query.Query,
			profileId,
			limit,
			offset,
			err,
		)
		return m, http.StatusInternalServerError,
			errors.New("Database query failed")
	}
	defer rows.Close()

	rs := []SearchResult{}
	for rows.Next() {
		var r SearchResult
		err = rows.Scan(
			&r.ItemTypeId,
			&r.ItemId,
			&r.ParentItemTypeId,
			&r.ParentItemId,
			&r.LastModified,
			&r.Rank,
			&r.Highlight,
			&r.Unread,
		)
		if err != nil {
			glog.Errorf("rows.Scan() %+v", err)
			return m, http.StatusInternalServerError,
				errors.New("Row parsing error")
		}

		itemType, err := h.GetMapStringFromInt(h.ItemTypes, r.ItemTypeId)
		if err != nil {
			glog.Errorf(
				"h.GetMapStringFromInt(h.ItemTypes, %d) %+v",
				r.ItemTypeId,
				err,
			)
			return m, http.StatusInternalServerError, err
		}
		r.ItemType = itemType

		if r.ParentItemTypeId.Valid {
			parentItemType, err :=
				h.GetMapStringFromInt(h.ItemTypes, r.ParentItemTypeId.Int64)
			if err != nil {
				glog.Errorf(
					"h.GetMapStringFromInt(h.ItemTypes, %d) %+v",
					r.ParentItemTypeId.Int64,
					err,
				)
				return m, http.StatusInternalServerError, err
			}
			r.ParentItemType = parentItemType
		}

		rs = append(rs, r)
	}
	err = rows.Err()
	if err != nil {
		glog.Errorf("rows.Err() %+v", err)
		return m, http.StatusInternalServerError,
			errors.New("Error fetching rows")
	}
	rows.Close()

	pages := h.GetPageCount(total, limit)
	maxOffset := h.GetMaxOffset(total, limit)

	if offset > maxOffset {
		glog.Infoln("offset > maxOffset")
		return m, http.StatusBadRequest, errors.New(
			fmt.Sprintf("not enough records, "+
				"offset (%d) would return an empty page.", offset),
		)
	}

	// Extract the summaries
	var wg1 sync.WaitGroup
	req := make(chan SummaryContainerRequest)
	defer close(req)

	seq := 0
	for i := 0; i < len(rs); i++ {
		go HandleSummaryContainerRequest(
			siteId,
			rs[i].ItemTypeId,
			rs[i].ItemId,
			profileId,
			seq,
			req,
		)
		seq++
		wg1.Add(1)

		if rs[i].ParentItemId.Valid && rs[i].ParentItemId.Int64 > 0 {
			go HandleSummaryContainerRequest(
				siteId,
				rs[i].ParentItemTypeId.Int64,
				rs[i].ParentItemId.Int64,
				profileId,
				seq,
				req,
			)
			seq++
			wg1.Add(1)
		}
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
			return m, resp.Status, resp.Err
		}
	}

	sort.Sort(SummaryContainerRequestsBySeq(resps))

	seq = 0
	for i := 0; i < len(rs); i++ {

		rs[i].Item = resps[seq].Item.Summary
		seq++

		if rs[i].ParentItemId.Valid && rs[i].ParentItemId.Int64 > 0 {
			rs[i].ParentItem = resps[seq].Item.Summary
			seq++
		}
	}

	m.Results = h.ConstructArray(
		rs,
		"result",
		total,
		limit,
		offset,
		pages,
		&searchUrl,
	)

	// return milliseconds
	m.TimeTaken = time.Now().Sub(start).Nanoseconds() / 1000000

	return m, http.StatusOK, nil

}
