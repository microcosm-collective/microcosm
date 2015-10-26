package models

import (
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"

	h "github.com/microcosm-cc/microcosm/helpers"
)

func searchEmail(
	siteID int64,
	searchURL url.URL,
	profileID int64,
	m SearchResults,
) (
	SearchResults,
	int,
	error,
) {

	limit, offset, status, err := h.GetLimitAndOffset(searchURL.Query())
	if err != nil {
		glog.Errorf("h.GetLimitAndOffset(searchURL.Query()) %+v", err)
		return m, status, err
	}

	start := time.Now()

	db, err := h.GetConnection()
	if err != nil {
		glog.Errorf("h.GetConnection() %+v", err)
		return m, http.StatusInternalServerError, err
	}

	rows, err := db.Query(`--SearchEmails
SELECT *
  FROM (
           SELECT COUNT(*) OVER() AS total
                 ,3 AS item_type
                 ,p.profile_id AS item_type_id
                 ,p.last_active
                 ,1
                 ,'' AS highlight
             FROM users u
             JOIN profiles p ON p.user_id = u.user_id
            WHERE u.email = ANY(string_to_array($2, '____'))
              AND p.site_id = $1
            ORDER BY p.profile_name ASC
       ) AS r
 LIMIT $3
OFFSET $4`,
		siteID,
		strings.Join(m.Query.Emails, `____`),
		limit,
		offset,
	)
	if err != nil {
		glog.Errorf(
			"stmt.Query(%d, %s, %d, %d) %+v",
			siteID,
			strings.Join(m.Query.Emails, `,`),
			limit,
			offset,
			err,
		)
		return m, http.StatusInternalServerError,
			fmt.Errorf("Database query failed")
	}
	defer rows.Close()

	var total int64
	rs := []SearchResult{}
	for rows.Next() {
		var r SearchResult
		err = rows.Scan(
			&total,
			&r.ItemTypeID,
			&r.ItemID,
			&r.LastModified,
			&r.Rank,
			&r.Highlight,
		)
		if err != nil {
			glog.Errorf("rows.Scan() %+v", err)
			return m, http.StatusInternalServerError,
				fmt.Errorf("Row parsing error")
		}

		r.ItemType = "profile"

		rs = append(rs, r)
	}
	err = rows.Err()
	if err != nil {
		glog.Errorf("rows.Err() %+v", err)
		return m, http.StatusInternalServerError,
			fmt.Errorf("Error fetching rows")
	}
	rows.Close()

	pages := h.GetPageCount(total, limit)
	maxOffset := h.GetMaxOffset(total, limit)

	if offset > maxOffset {
		glog.Infoln("offset > maxOffset")
		return m, http.StatusBadRequest,
			fmt.Errorf("not enough records, "+
				"offset (%d) would return an empty page.", offset)
	}

	// Extract the summaries
	var wg1 sync.WaitGroup
	req := make(chan SummaryContainerRequest)
	defer close(req)

	seq := 0
	for i := 0; i < len(rs); i++ {
		go HandleSummaryContainerRequest(
			siteID,
			rs[i].ItemTypeID,
			rs[i].ItemID,
			profileID,
			seq,
			req,
		)
		seq++
		wg1.Add(1)

		if rs[i].ParentItemID.Valid && rs[i].ParentItemID.Int64 > 0 {
			go HandleSummaryContainerRequest(
				siteID,
				rs[i].ParentItemTypeID.Int64,
				rs[i].ParentItemID.Int64,
				profileID,
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

		if rs[i].ParentItemID.Valid && rs[i].ParentItemID.Int64 > 0 {
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
		&searchURL,
	)

	// return milliseconds
	m.TimeTaken = time.Now().Sub(start).Nanoseconds() / 1000000

	return m, http.StatusOK, nil

}
