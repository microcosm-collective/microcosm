package models

import (
	"fmt"
	"net/http"
	"sort"
	"sync"

	"github.com/golang/glog"

	h "github.com/microcosm-collective/microcosm/helpers"
)

// IgnoredType is a collection of ignored items
type IgnoredType struct {
	Ignored h.ArrayType    `json:"ignored"`
	Meta    h.CoreMetaType `json:"meta"`
}

// IgnoreType is an ignored item
type IgnoreType struct {
	ProfileID  int64       `json:"-"`
	ItemTypeID int64       `json:"-"`
	ItemType   string      `json:"itemType,omitempty"`
	ItemID     int64       `json:"itemId,omitempty"`
	Item       interface{} `json:"item,omitempty"`
}

// Validate returns true if the item is valid
func (m *IgnoreType) Validate() (int, error) {

	if m.ProfileID <= 0 {
		return http.StatusBadRequest,
			fmt.Errorf(
				"profileID ('%d') must be a positive integer",
				m.ProfileID,
			)
	}

	if _, inMap := h.ItemTypes[m.ItemType]; !inMap {
		return http.StatusBadRequest,
			fmt.Errorf("you must specify a valid item type")
	}
	m.ItemTypeID = h.ItemTypes[m.ItemType]

	if m.ItemID <= 0 {
		return http.StatusBadRequest,
			fmt.Errorf("you must specify an Item ID this comment belongs to")
	}

	return http.StatusOK, nil
}

// Update saves the ignore to the database
func (m *IgnoreType) Update() (int, error) {
	status, err := m.Validate()
	if err != nil {
		return status, err
	}

	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	// Lack of error checking, it can only fail if it has been inserted already
	// and our answer "OK" remains the same if it exists after this action.
	_, err = tx.Exec(`--Create Ignore
INSERT INTO ignores (
    profile_id, item_type_id, item_id
) VALUES (
    $1, $2, $3
)`,
		m.ProfileID,
		m.ItemTypeID,
		m.ItemID,
	)
	if err == nil {
		tx.Commit()
	}

	return http.StatusOK, nil
}

// Delete removes an ignore
func (m *IgnoreType) Delete() (int, error) {
	status, err := m.Validate()
	if err != nil {
		return status, err
	}

	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	// Lack of error checking, it can only fail if it has been deleted already
	// and our answer "OK" remains the same if it exists after this action.
	_, err = tx.Exec(`--Delete Ignore
DELETE
  FROM ignores
 WHERE profile_id = $1
   AND item_type_id = $2
   AND item_id = $3`,
		m.ProfileID,
		m.ItemTypeID,
		m.ItemID,
	)
	if err == nil {
		tx.Commit()
	}

	return http.StatusOK, nil
}

// GetIgnored returns a collection of ignored items
func GetIgnored(
	siteID int64,
	profileID int64,
	limit int64,
	offset int64,
) (
	[]IgnoreType,
	int64,
	int64,
	int,
	error,
) {
	db, err := h.GetConnection()
	if err != nil {
		glog.Errorf("h.GetConnection() %+v", err)
		return []IgnoreType{}, 0, 0, http.StatusInternalServerError, err
	}

	// This query intentionally does not provide has_unread() status. This is
	// to pacify angry people ignoring things, then unignoring on updates and
	// subsequently getting in to fights.
	sqlQuery := `--Get Ignores
WITH m AS (
    SELECT m.microcosm_id
      FROM microcosms m
      LEFT JOIN permissions_cache p ON p.site_id = m.site_id
                                   AND p.item_type_id = 2
                                   AND p.item_id = m.microcosm_id
                                   AND p.profile_id = $2
     WHERE m.site_id = $1
       AND m.is_deleted IS NOT TRUE
       AND m.is_moderated IS NOT TRUE
       AND (
               (p.can_read IS NOT NULL AND p.can_read IS TRUE)
            OR (get_effective_permissions($1,m.microcosm_id,2,m.microcosm_id,$2)).can_read IS TRUE
           )
)
SELECT COUNT(*) OVER() AS total
      ,a.profile_id
      ,a.item_type_id
      ,a.item_id
  FROM (
           SELECT i.profile_id
                 ,i.item_type_id
                 ,i.item_id
             FROM ignores i
            INNER JOIN flags f ON f.item_type_id = i.item_type_id
                              AND f.item_id = i.item_id
            WHERE i.profile_id = $2
              AND f.site_id = $1
              AND (
                      f.microcosm_id IS NULL
                   OR f.microcosm_id IN (SELECT microcosm_id FROM m)
                  )
              AND f.microcosm_is_deleted IS NOT TRUE
              AND f.microcosm_is_moderated IS NOT TRUE
              AND f.parent_is_deleted IS NOT TRUE
              AND f.parent_is_moderated IS NOT TRUE
              AND f.item_is_deleted IS NOT TRUE
              AND f.item_is_moderated IS NOT TRUE
       ) a
 INNER JOIN search_index si ON si.item_type_id = a.item_type_id
                           AND si.item_id = a.item_id
 ORDER BY a.item_type_id ASC
         ,si.title_text ASC
 LIMIT $3
OFFSET $4`

	rows, err := db.Query(sqlQuery, siteID, profileID, limit, offset)
	if err != nil {
		glog.Errorf(
			"db.Query(%d, %d, %d, %d) %+v",
			siteID,
			profileID,
			limit,
			offset,
			err,
		)
		return []IgnoreType{}, 0, 0, http.StatusInternalServerError,
			fmt.Errorf("database query failed")
	}
	defer rows.Close()

	var total int64
	ems := []IgnoreType{}
	for rows.Next() {
		m := IgnoreType{}
		err = rows.Scan(
			&total,
			&m.ProfileID,
			&m.ItemTypeID,
			&m.ItemID,
		)
		if err != nil {
			glog.Errorf("rows.Scan() %+v", err)
			return []IgnoreType{}, 0, 0, http.StatusInternalServerError,
				fmt.Errorf("row parsing error")
		}

		itemType, err := h.GetItemTypeFromInt(m.ItemTypeID)
		if err != nil {
			glog.Errorf("h.GetItemTypeFromInt(%d) %+v", m.ItemTypeID, err)
			return []IgnoreType{}, 0, 0, http.StatusInternalServerError, err
		}
		m.ItemType = itemType

		ems = append(ems, m)
	}
	err = rows.Err()
	if err != nil {
		glog.Errorf("rows.Err() %+v", err)
		return []IgnoreType{}, 0, 0, http.StatusInternalServerError,
			fmt.Errorf("error fetching rows")
	}
	rows.Close()

	pages := h.GetPageCount(total, limit)
	maxOffset := h.GetMaxOffset(total, limit)

	if offset > maxOffset {
		glog.Infoln("offset > maxOffset")
		return []IgnoreType{}, 0, 0, http.StatusBadRequest,
			fmt.Errorf("not enough records, "+
				"offset (%d) would return an empty page", offset)
	}

	// Get the first round of summaries
	var wg1 sync.WaitGroup
	chan1 := make(chan SummaryContainerRequest)
	defer close(chan1)

	seq := 0
	for i := 0; i < len(ems); i++ {
		go HandleSummaryContainerRequest(
			siteID,
			ems[i].ItemTypeID,
			ems[i].ItemID,
			ems[i].ProfileID,
			seq,
			chan1,
		)
		wg1.Add(1)
		seq++
	}

	resps := []SummaryContainerRequest{}
	for i := 0; i < seq; i++ {
		resp := <-chan1
		wg1.Done()

		resps = append(resps, resp)
	}
	wg1.Wait()

	for _, resp := range resps {
		if resp.Err != nil {
			return []IgnoreType{}, 0, 0, resp.Status, resp.Err
		}
	}

	sort.Sort(SummaryContainerRequestsBySeq(resps))

	seq = 0
	for i := 0; i < len(ems); i++ {
		ems[i].Item = resps[seq].Item.Summary
		seq++
	}

	return ems, total, pages, http.StatusOK, nil
}
