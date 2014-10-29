package models

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"sync"

	h "github.com/microcosm-cc/microcosm/helpers"
)

type HuddleParticipantsType struct {
	HuddleParticipants h.ArrayType    `json:"participants"`
	Meta               h.CoreMetaType `json:"meta"`
}

type HuddleParticipantType struct {
	Id      int64       `json:"id,omitempty"`
	Profile interface{} `json:"participant"`
}

func (m *HuddleParticipantType) GetLink(link string) string {
	return fmt.Sprintf("%s/participants/%d", link, m.Id)
}

func (m *HuddleParticipantType) Validate(siteId int64) (int, error) {
	if m.Id < 1 {
		return http.StatusBadRequest,
			errors.New("profile id needs to be a positive integer")
	}

	profileSummary, status, err := GetProfileSummary(siteId, m.Id)
	if err != nil {
		return status, err
	}
	m.Profile = profileSummary

	return http.StatusOK, nil
}

func UpdateManyHuddleParticipants(
	siteId int64,
	huddleId int64,
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
		status, err := m.insert(tx, siteId, huddleId)
		if err != nil {
			return status, err
		}
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			errors.New(fmt.Sprintf("Transaction failed: %+v", err))
	}

	go PurgeCache(h.ItemTypes[h.ItemTypeHuddle], huddleId)

	return http.StatusOK, nil
}

func (m *HuddleParticipantType) Update(
	siteId int64,
	huddleId int64,
) (
	int,
	error,
) {

	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	status, err := m.insert(tx, siteId, huddleId)
	if err != nil {
		return status, err
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			errors.New(fmt.Sprintf("Transaction failed: %v", err.Error()))
	}

	return http.StatusOK, nil
}

func (m *HuddleParticipantType) insert(
	tx *sql.Tx,
	siteId int64,
	huddleId int64,
) (
	int,
	error,
) {

	status, err := m.Validate(siteId)
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
		huddleId,
		m.Id,
	)
	if err != nil {
		return http.StatusInternalServerError,
			errors.New(fmt.Sprintf("Error executing upsert: %v", err.Error()))
	}

	go RegisterWatcher(m.Id, 4, huddleId, h.ItemTypes[h.ItemTypeHuddle], siteId)
	go UpdateUnreadHuddleCount(m.Id)

	return http.StatusOK, nil
}

func (m *HuddleParticipantType) Delete(huddleId int64) (int, error) {

	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	status, err := m.delete(tx, huddleId)
	if err != nil {
		return status, err
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			errors.New(fmt.Sprintf("Transaction failed: %v", err.Error()))
	}

	return http.StatusOK, nil
}

func (m *HuddleParticipantType) delete(
	tx *sql.Tx,
	huddleId int64,
) (
	int,
	error,
) {

	_, err := tx.Exec(`
DELETE FROM huddle_profiles
 WHERE huddle_id = $1
   AND profile_id = $2`,
		huddleId,
		m.Id,
	)
	if err != nil {
		return http.StatusInternalServerError,
			errors.New(fmt.Sprintf("Error executing delete: %+v", err))
	}

	go UpdateUnreadHuddleCount(m.Id)

	return http.StatusOK, nil
}

func GetHuddleParticipant(
	siteId int64,
	huddleId int64,
	profileId int64,
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
		huddleId,
		profileId,
	)
	if err != nil {
		return HuddleParticipantType{}, http.StatusInternalServerError,
			errors.New(
				fmt.Sprintf("Database query failed: %v", err.Error()),
			)
	}
	defer rows.Close()

	m := HuddleParticipantType{}
	for rows.Next() {
		err = rows.Scan(
			&m.Id,
		)
		if err != nil {
			return HuddleParticipantType{}, http.StatusInternalServerError,
				errors.New(
					fmt.Sprintf("Error fetching row: %v", err.Error()),
				)
		}

		// Make a request the profile summary
		req := make(chan ProfileSummaryRequest)
		defer close(req)

		go HandleProfileSummaryRequest(siteId, m.Id, 0, req)

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
			errors.New(
				fmt.Sprintf("Error fetching rows: %v", err.Error()),
			)
	}
	rows.Close()

	return m, http.StatusOK, nil
}

func GetHuddleParticipants(
	siteId int64,
	huddleId int64,
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
		huddleId,
		limit,
		offset,
	)
	if err != nil {
		return []HuddleParticipantType{}, 0, 0,
			http.StatusInternalServerError, errors.New(
				fmt.Sprintf("Database query failed: %v", err.Error()),
			)
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
				http.StatusInternalServerError, errors.New(
					fmt.Sprintf("Row parsing error: %v", err.Error()),
				)
		}

		ids = append(ids, id)
	}
	err = rows.Err()
	if err != nil {
		return []HuddleParticipantType{}, 0, 0,
			http.StatusInternalServerError, errors.New(
				fmt.Sprintf("Error fetching rows: %v", err.Error()),
			)
	}
	rows.Close()

	// Make a request for each identifier
	var wg1 sync.WaitGroup
	req := make(chan ProfileSummaryRequest)
	defer close(req)

	for seq, id := range ids {
		go HandleProfileSummaryRequest(siteId, id, seq, req)
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
		rp.Id = resp.Item.Id
		rp.Profile = resp.Item

		ems = append(ems, rp)
	}

	pages := h.GetPageCount(total, limit)
	maxOffset := h.GetMaxOffset(total, limit)

	if offset > maxOffset {
		return []HuddleParticipantType{}, 0, 0,
			http.StatusBadRequest,
			errors.New(
				fmt.Sprintf(
					"not enough records, offset (%d) would return an empty page.",
					offset,
				),
			)
	}

	return ems, total, pages, http.StatusOK, nil
}
