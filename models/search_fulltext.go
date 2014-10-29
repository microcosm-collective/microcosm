package models

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/lib/pq"

	merrors "github.com/microcosm-cc/microcosm/errors"
	h "github.com/microcosm-cc/microcosm/helpers"
)

// 2 queries are run, so a 5 second timeout would mean a worst case of 10
// seconds and so on
const searchTimeout = 15 * time.Second

func searchFullText(
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

	// Search options
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

	var filterFollowing string
	if m.Query.Following {
		filterFollowing = `
                  JOIN watchers w ON w.item_type_id = f.item_type_id
                                 AND w.item_id = f.item_id
                                 AND w.profile_id = $2`
	}

	fullTextScope := `document`
	var filterTitle string
	if m.Query.InTitle {
		fullTextScope = `title`
		filterTitle = `
              AND f.item_type_id <> 4`
	}

	var filterItemTypes string
	var filterItems string
	var includeComments bool
	if !m.Query.InTitle {
		includeComments = true
	}

	if len(m.Query.ItemTypeIds) > 0 {
		var itemTypeInList []string
		var itemTypeSansCommentsInList []string

		// Take care of the item types
		for _, v := range m.Query.ItemTypeIds {
			switch v {
			case h.ItemTypes[h.ItemTypeComment]:
				includeComments = true
				itemTypeInList = append(itemTypeInList, strconv.FormatInt(v, 10))
			default:
				itemTypeInList = append(itemTypeInList, strconv.FormatInt(v, 10))
				itemTypeSansCommentsInList = append(itemTypeSansCommentsInList, strconv.FormatInt(v, 10))
			}
		}

		if len(m.Query.ItemIds) == 0 {
			if len(m.Query.ItemTypeIds) == 1 {
				filterItemTypes = fmt.Sprintf(`
              AND f.item_type_id = %d`,
					m.Query.ItemTypeIds[0],
				)
			} else {
				if includeComments {
					filterItemTypes = `
              AND (   (f.item_type_id IN (` + strings.Join(itemTypeSansCommentsInList, `,`) + `))
                   OR (f.item_type_id = 4 AND f.parent_item_type_id IN (` + strings.Join(itemTypeSansCommentsInList, `,`) + `))
                 )`
				} else {
					filterItemTypes = `
              AND f.item_type_id IN (` + strings.Join(itemTypeInList, `,`) + `)`
				}
			}
		}

		// Take care of the item ids, which are only valid when we have item
		// types
		if len(m.Query.ItemIds) > 0 {
			var itemIdsInList []string
			for _, v := range m.Query.ItemIds {
				itemIdsInList = append(itemIdsInList, strconv.FormatInt(v, 10))
			}

			if len(m.Query.ItemIds) == 1 {
				if includeComments {
					filterItems = fmt.Sprintf(`
              AND (   (si.item_type_id IN (`+strings.Join(itemTypeSansCommentsInList, `,`)+`) AND si.item_id = %d)
                   OR (si.item_type_id = 4 AND si.parent_item_id = %d AND si.parent_item_type_id IN (`+strings.Join(itemTypeSansCommentsInList, `,`)+`))
                  )`,
						m.Query.ItemIds[0],
						m.Query.ItemIds[0],
					)
				} else {
					filterItems = fmt.Sprintf(`
              AND si.item_id = %d`,
						m.Query.ItemIds[0],
					)
				}
			} else {
				if includeComments {
					filterItems = `
              AND (   (si.item_type_id IN (` + strings.Join(itemTypeSansCommentsInList, `,`) + `) AND si.item_id IN (` + strings.Join(itemIdsInList, `,`) + `))
                   OR (si.item_type_id = 4 AND si.parent_item_type_id IN (` + strings.Join(itemTypeSansCommentsInList, `,`) + `) AND si.parent_item_id IN (` + strings.Join(itemIdsInList, `,`) + `))
                  )`
				} else {
					filterItems = `
              AND si.item_type_id IN (` + strings.Join(itemTypeInList, `,`) + `)
              AND si.item_id IN (` + strings.Join(itemIdsInList, `,`) + `)`
				}
			}
		}
	}

	// Note: hashtags being inserted into the query this way may appear
	// initially to be a vector for a SQL injection attack. However the
	// source of these hashtags is a regexp in hashtags.go which only
	// matches contiguous alphanum strings and does not permit spaces,
	// quotes, semicolons or any other escapable sequence that can be
	// utilised to create an attack.
	var filterHashTag string
	for _, hashtag := range m.Query.Hashtags {
		filterHashTag += `
              AND si.` + fullTextScope + `_text ~* '\W` + hashtag + `\W'`
	}

	var filterProfileId string
	if m.Query.ProfileId > 0 {
		filterProfileId = fmt.Sprintf(`
              AND si.profile_id = %d`, m.Query.ProfileId)
	}

	var filterMicrocosmIds string
	if len(m.Query.MicrocosmIds) > 0 {
		if len(m.Query.MicrocosmIds) == 1 {
			filterMicrocosmIds = fmt.Sprintf(`
   AND f.microcosm_id = %d`, m.Query.MicrocosmIds[0])
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

	sqlQuery := `
WITH m AS (
    SELECT m.microcosm_id
      FROM microcosms m
      LEFT JOIN ignores i ON i.profile_id = $2
                         AND i.item_type_id = 2
                         AND i.item_id = m.microcosm_id
     WHERE m.site_id = $1
       AND i.profile_id IS NULL
       AND (get_effective_permissions($1,m.microcosm_id,2,m.microcosm_id,$2)).can_read IS TRUE
)
SELECT total
      ,item_type_id
      ,item_id
      ,parent_item_type_id
      ,parent_item_id
      ,last_modified
      ,rank
      ,ts_headline(` + fullTextScope + `_text, query) AS highlight
      ,has_unread(item_type_id, item_id, $2)
  FROM (
           SELECT COUNT(*) OVER() AS total
                 ,f.item_type_id
                 ,f.item_id
                 ,f.parent_item_type_id
                 ,f.parent_item_id
                 ,f.last_modified
                 ,ts_rank_cd(si.` + fullTextScope + `_vector, query, 8) AS rank
                 ,si.` + fullTextScope + `_text
                 ,query.query
             FROM search_index si
                  JOIN flags f ON f.item_type_id = si.item_type_id
                              AND f.item_id = si.item_id
             LEFT JOIN ignores i ON i.profile_id = $2
                                AND i.item_type_id = f.item_type_id
                                AND i.item_id = f.item_id` +
		filterEventsJoin +
		filterFollowing + `
             LEFT JOIN huddle_profiles h ON (f.parent_item_type_id = 5 OR f.item_type_id = 5)
                                        AND h.huddle_id = COALESCE(f.parent_item_id, f.item_id)
                                        AND h.profile_id = $2
                 ,plainto_tsquery($3) AS query
            WHERE f.site_id = $1
              AND i.profile_id IS NULL` +
		filterModified +
		filterMicrocosmIds +
		filterTitle +
		filterItemTypes +
		filterItems +
		filterHashTag +
		filterEventsWhere +
		filterProfileId + `
              AND f.microcosm_is_deleted IS NOT TRUE
              AND f.microcosm_is_moderated IS NOT TRUE
              AND f.parent_is_deleted IS NOT TRUE
              AND f.parent_is_moderated IS NOT TRUE
              AND f.item_is_deleted IS NOT TRUE
              AND f.item_is_moderated IS NOT TRUE
              AND si.` + fullTextScope + `_vector @@ query` + `
              AND (
                      -- Things that are public by default
                      COALESCE(f.parent_item_type_id, f.item_type_id) = 3
                   OR -- Things in microcosms
                      COALESCE(f.microcosm_id, f.item_id) IN (SELECT microcosm_id FROM m)
                   OR -- Things in huddles
                      COALESCE(f.parent_item_id, f.item_id) = h.huddle_id
                  )
            ORDER BY ` + orderBy + `
            LIMIT $4
           OFFSET $5
       ) r
`

	db, err := h.GetConnection()
	if err != nil {
		glog.Errorf("h.GetConnection() %+v", err)
		return m, http.StatusInternalServerError, err
	}

	queryId := `Search` + randomString()
	queryTimer := time.NewTimer(searchTimeout)
	go func() {
		<-queryTimer.C
		db.Exec(`SELECT pg_cancel_backend(pid)
  FROM pg_stat_activity
 WHERE state = 'active'
   AND query LIKE '--` + queryId + `%'`)
	}()
	// This nested query is used to run the `has_unread` query on only the rows
	// that are returned, rather than on all rows in the underlying query before
	// limit has been applied.
	rows, err := db.Query(
		`--`+queryId+
			sqlQuery,
		siteId,
		profileId,
		m.Query.Query,
		limit,
		offset,
	)
	queryTimer.Stop()
	if err != nil {
		e, ok := err.(*pq.Error)

		if !ok {
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

		switch e.Code.Name() {
		case "query_canceled":
			glog.Errorf(
				"Query for '%s' took too long",
				m.Query.Query,
			)
			return m, http.StatusInternalServerError,
				merrors.MicrocosmError{
					ErrorCode:    24,
					ErrorMessage: "The search query took too long and has been cancelled",
				}
		default:
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
	}
	defer rows.Close()

	var total int64
	rs := []SearchResult{}
	for rows.Next() {
		var r SearchResult
		err = rows.Scan(
			&total,
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

// Copyright (c) 2011 Dmitry Chestnykh
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.
//
// https://github.com/dchest/uniuri

func randomString() string {

	const (
		// Standard length of uniuri string to achive ~95 bits of entropy.
		length = 16
		// Length of uniurl string to achive ~119 bits of entropy, closest
		// to what can be losslessly converted to UUIDv4 (122 bits).
		UUIDLen = 20
	)

	// Standard characters allowed in uniuri string.
	var chars = []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789")

	// NewLenChars returns a new random string of the provided length, consisting
	// of the provided byte slice of allowed characters (maximum 256).

	b := make([]byte, length)
	r := make([]byte, length+(length/4)) // storage for random bytes.
	clen := byte(len(chars))
	maxrb := byte(256 - (256 % len(chars)))
	i := 0
	for {
		if _, err := io.ReadFull(rand.Reader, r); err != nil {
			panic("error reading from random source: " + err.Error())
		}
		for _, c := range r {
			if c >= maxrb {
				// Skip this number to avoid modulo bias.
				continue
			}
			b[i] = chars[c%clen]
			i++
			if i == length {
				return string(b)
			}
		}
	}
}
