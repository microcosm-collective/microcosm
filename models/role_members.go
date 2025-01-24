package models

import (
	"database/sql"
	"fmt"
	"net/http"
	"sort"
	"sync"

	h "github.com/microcosm-collective/microcosm/helpers"
)

// FlushRoleMembersCacheByProfileID clears the database caches
func FlushRoleMembersCacheByProfileID(
	tx *sql.Tx,
	profileID int64,
) (
	int,
	error,
) {
	_, err := tx.Exec(
		`DELETE FROM permissions_cache WHERE profile_id = $1`,
		profileID,
	)
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("error executing statement: %v", err.Error())
	}

	_, err = tx.Exec(
		`DELETE FROM role_members_cache WHERE profile_id = $1`,
		profileID,
	)
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("error executing statement: %v", err.Error())
	}

	return http.StatusOK, nil
}

// FlushRoleMembersCacheByRoleID clears the database caches
func FlushRoleMembersCacheByRoleID(tx *sql.Tx, roleID int64) (int, error) {
	_, err := tx.Exec(`TRUNCATE permissions_cache`)
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("error executing statement: %v", err.Error())
	}

	_, err = tx.Exec(
		`DELETE FROM role_members_cache WHERE role_id = $1`,
		roleID,
	)
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("error executing statement: %v", err.Error())
	}

	return http.StatusOK, nil
}

// GetRoleMembers fetches all profiles who are part of this role, either
// explicitly so or implicitly so.
func GetRoleMembers(
	siteID int64,
	roleID int64,
	limit int64,
	offset int64,
) (
	[]ProfileSummaryType,
	int64,
	int64,
	int,
	error,
) {
	// Retrieve resources
	db, err := h.GetConnection()
	if err != nil {
		return []ProfileSummaryType{}, 0, 0, http.StatusInternalServerError, err
	}

	rows, err := db.Query(`
SELECT COUNT(*) OVER() AS total
      ,profile_id
  FROM get_role_profiles($1, $2) AS profile_id
 WHERE profile_id > 0
 LIMIT $3
OFFSET $4`,
		siteID,
		roleID,
		limit,
		offset,
	)
	if err != nil {
		return []ProfileSummaryType{}, 0, 0, http.StatusInternalServerError,
			fmt.Errorf("database query failed: %v", err.Error())
	}
	defer rows.Close()

	// Get a list of the identifiers of the items to return
	var total int64
	ids := []int64{}
	for rows.Next() {
		var id int64
		err = rows.Scan(
			&total,
			&id,
		)
		if err != nil {
			return []ProfileSummaryType{}, 0, 0, http.StatusInternalServerError,
				fmt.Errorf("row parsing error: %v", err.Error())
		}

		ids = append(ids, id)
	}
	err = rows.Err()
	if err != nil {
		return []ProfileSummaryType{}, 0, 0, http.StatusInternalServerError,
			fmt.Errorf("error fetching rows: %v", err.Error())
	}
	rows.Close()

	// Make a request for each identifier
	var wg1 sync.WaitGroup
	req := make(chan ProfileSummaryRequest)
	defer close(req)

	for seq, id := range ids {
		go HandleProfileSummaryRequest(siteID, id, seq, req)
		wg1.Add(1)
	}

	// Receive the responses and check for errors
	resps := []ProfileSummaryRequest{}
	for i := 0; i < len(ids); i++ {
		resp := <-req
		wg1.Done()
		resps = append(resps, resp)
	}
	wg1.Wait()

	for _, resp := range resps {
		if resp.Err != nil {
			return []ProfileSummaryType{}, 0, 0, resp.Status, resp.Err
		}
	}

	// Sort them
	sort.Sort(ProfileSummaryRequestBySeq(resps))

	// Extract the values
	ems := []ProfileSummaryType{}
	for _, resp := range resps {
		ems = append(ems, resp.Item)
	}

	pages := h.GetPageCount(total, limit)
	maxOffset := h.GetMaxOffset(total, limit)

	if offset > maxOffset {
		return []ProfileSummaryType{}, 0, 0, http.StatusBadRequest,
			fmt.Errorf(
				"not enough records, offset (%d) would return an empty page",
				offset,
			)
	}

	return ems, total, pages, http.StatusOK, nil
}
