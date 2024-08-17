package models

import (
	"database/sql"
	"fmt"
	"net/http"
	"sort"
	"sync"

	h "github.com/microcosm-cc/microcosm/helpers"
)

// HuddleParticipantsType is a collection of huddle participants
type HuddleParticipantsType struct {
	HuddleParticipants h.ArrayType    `json:"participants"`
	Meta               h.CoreMetaType `json:"meta"`
}

// HuddleParticipantType describes a person who is able to view and interact
// with a huddle
type HuddleParticipantType struct {
	ID      int64       `json:"id,omitempty"`
	Profile interface{} `json:"participant"`
}

// GetLink returns the API link to a participant
func (m *HuddleParticipantType) GetLink(link string) string {
	return fmt.Sprintf("%s/participants/%d", link, m.ID)
}

// Validate returns true if the huddle participant is valid
func (m *HuddleParticipantType) Validate(siteID int64) (int, error) {
	if m.ID < 1 {
		return http.StatusBadRequest,
			fmt.Errorf("profile id needs to be a positive integer")
	}

	profileSummary, status, err := GetProfileSummary(siteID, m.ID)
	if err != nil {
		return status, err
	}
	m.Profile = profileSummary

	return http.StatusOK, nil
}

// UpdateManyHuddleParticipants saves many participants
func UpdateManyHuddleParticipants(
	siteID int64,
	huddleID int64,
	ems []HuddleParticipantType,
) (
	int,
	error,
) {
	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	for _, m := range ems {
		status, err := m.insert(tx, siteID, huddleID)
		if err != nil {
			return status, err
		}
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("transaction failed: %+v", err)
	}

	go PurgeCache(h.ItemTypes[h.ItemTypeHuddle], huddleID)

	return http.StatusOK, nil
}

// Update saves a single participant
func (m *HuddleParticipantType) Update(
	siteID int64,
	huddleID int64,
) (
	int,
	error,
) {
	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	status, err := m.insert(tx, siteID, huddleID)
	if err != nil {
		return status, err
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("transaction failed: %v", err.Error())
	}

	return http.StatusOK, nil
}

// insert saves a participant
func (m *HuddleParticipantType) insert(
	tx *sql.Tx,
	siteID int64,
	huddleID int64,
) (
	int,
	error,
) {
	status, err := m.Validate(siteID)
	if err != nil {
		return status, err
	}

	// upsert
	_, err = tx.Exec(`
INSERT INTO huddle_profiles
SELECT $1, $2
 WHERE NOT EXISTS (
       SELECT huddle_id
             ,profile_id
         FROM huddle_profiles
        WHERE huddle_id = $1
          AND profile_id = $2
       )`,
		huddleID,
		m.ID,
	)
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("error executing upsert: %v", err.Error())
	}

	go RegisterWatcher(m.ID, 4, huddleID, h.ItemTypes[h.ItemTypeHuddle], siteID)
	go UpdateUnreadHuddleCount(m.ID)

	return http.StatusOK, nil
}

// Delete removes a participant from a huddle
func (m *HuddleParticipantType) Delete(huddleID int64) (int, error) {
	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	status, err := m.delete(tx, huddleID)
	if err != nil {
		return status, err
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("transaction failed: %v", err.Error())
	}

	return http.StatusOK, nil
}

func (m *HuddleParticipantType) delete(
	tx *sql.Tx,
	huddleID int64,
) (
	int,
	error,
) {

	_, err := tx.Exec(`
DELETE FROM huddle_profiles
 WHERE huddle_id = $1
   AND profile_id = $2`,
		huddleID,
		m.ID,
	)
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("error executing delete: %+v", err)
	}

	go UpdateUnreadHuddleCount(m.ID)

	return http.StatusOK, nil
}

// GetHuddleParticipant fetches a huddle participant
func GetHuddleParticipant(
	siteID int64,
	huddleID int64,
	profileID int64,
) (
	HuddleParticipantType,
	int,
	error,
) {
	// Retrieve resources
	db, err := h.GetConnection()
	if err != nil {
		return HuddleParticipantType{}, http.StatusInternalServerError, err
	}

	rows, err := db.Query(`
SELECT profile_id
  FROM huddle_profiles
 WHERE huddle_id = $1
   AND profile_id = $2`,
		huddleID,
		profileID,
	)
	if err != nil {
		return HuddleParticipantType{}, http.StatusInternalServerError,
			fmt.Errorf("database query failed: %v", err.Error())
	}
	defer rows.Close()

	m := HuddleParticipantType{}
	for rows.Next() {
		err = rows.Scan(
			&m.ID,
		)
		if err != nil {
			return HuddleParticipantType{}, http.StatusInternalServerError,
				fmt.Errorf("error fetching row: %v", err.Error())
		}

		// Make a request the profile summary
		req := make(chan ProfileSummaryRequest)
		defer close(req)

		go HandleProfileSummaryRequest(siteID, m.ID, 0, req)

		// Receive the response
		resp := <-req
		if resp.Err != nil {
			return HuddleParticipantType{}, resp.Status, resp.Err
		}

		m.Profile = resp.Item
	}
	err = rows.Err()
	if err != nil {
		return HuddleParticipantType{}, http.StatusInternalServerError,
			fmt.Errorf("error fetching rows: %v", err.Error())
	}
	rows.Close()

	return m, http.StatusOK, nil
}

// GetHuddleParticipants returns a collection of huddle participants
func GetHuddleParticipants(
	siteID int64,
	huddleID int64,
	limit int64,
	offset int64,
) (
	[]HuddleParticipantType,
	int64,
	int64,
	int,
	error,
) {
	// Retrieve resources
	db, err := h.GetConnection()
	if err != nil {
		return []HuddleParticipantType{}, 0, 0,
			http.StatusInternalServerError, err
	}

	rows, err := db.Query(`
SELECT COUNT(*) OVER() AS total
      ,profile_id
  FROM huddle_profiles
 WHERE huddle_id = $1
 ORDER BY profile_id ASC
 LIMIT $2
OFFSET $3`,
		huddleID,
		limit,
		offset,
	)
	if err != nil {
		return []HuddleParticipantType{}, 0, 0,
			http.StatusInternalServerError,
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
			return []HuddleParticipantType{}, 0, 0,
				http.StatusInternalServerError,
				fmt.Errorf("row parsing error: %v", err.Error())
		}

		ids = append(ids, id)
	}
	err = rows.Err()
	if err != nil {
		return []HuddleParticipantType{}, 0, 0,
			http.StatusInternalServerError,
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
			return []HuddleParticipantType{}, 0, 0, resp.Status, resp.Err
		}
	}

	// Sort them
	sort.Sort(ProfileSummaryRequestBySeq(resps))

	// Extract the values
	ems := []HuddleParticipantType{}
	for _, resp := range resps {
		rp := HuddleParticipantType{}
		rp.ID = resp.Item.ID
		rp.Profile = resp.Item

		ems = append(ems, rp)
	}

	pages := h.GetPageCount(total, limit)
	maxOffset := h.GetMaxOffset(total, limit)

	if offset > maxOffset {
		return []HuddleParticipantType{}, 0, 0,
			http.StatusBadRequest,
			fmt.Errorf(
				"not enough records, offset (%d) would return an empty page",
				offset,
			)
	}

	return ems, total, pages, http.StatusOK, nil
}
