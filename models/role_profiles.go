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

type RoleProfilesType struct {
	RoleProfiles h.ArrayType    `json:"profiles"`
	Meta         h.CoreMetaType `json:"meta"`
}

type RoleProfileType struct {
	Id      int64       `json:"id,omitempty"`
	Profile interface{} `json:"profile"`
}

func (m *RoleProfileType) GetLink(roleLink string) string {
	return fmt.Sprintf("%s/profiles/%d", roleLink, m.Id)
}

func (m *RoleProfileType) Validate(siteId int64) (int, error) {
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

func UpdateManyRoleProfiles(
	siteId int64,
	roleId int64,
	ems []RoleProfileType,
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
		status, err := m.insert(tx, siteId, roleId)
		if err != nil {
			return status, err
		}
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			errors.New(fmt.Sprintf("Transaction failed: %v", err.Error()))
	}

	go PurgeCache(h.ItemTypes[h.ItemTypeRole], roleId)

	return http.StatusOK, nil
}

func (m *RoleProfileType) Update(siteId int64, roleId int64) (int, error) {
	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	status, err := m.insert(tx, siteId, roleId)
	if err != nil {
		return status, err
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			errors.New(fmt.Sprintf("Transaction failed: %v", err.Error()))
	}

	go PurgeCache(h.ItemTypes[h.ItemTypeRole], roleId)

	return http.StatusOK, nil
}

func (m *RoleProfileType) insert(
	tx *sql.Tx,
	siteId int64,
	roleId int64,
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
INSERT INTO role_profiles
SELECT $1, $2
 WHERE NOT EXISTS (
       SELECT role_id
             ,profile_id
         FROM role_profiles
        WHERE role_id = $1
          AND profile_id = $2
       )`,
		roleId,
		m.Id,
	)
	if err != nil {
		return http.StatusInternalServerError,
			errors.New(fmt.Sprintf("Error executing upsert: %v", err.Error()))
	}

	return http.StatusOK, nil
}

func DeleteManyRoleProfiles(roleId int64, ems []RoleProfileType) (int, error) {
	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	if len(ems) == 0 {
		// Delete all named profiles
		_, err = tx.Exec(`
	DELETE FROM role_profiles
	 WHERE role_id = $1`,
			roleId,
		)
		if err != nil {
			return http.StatusInternalServerError,
				errors.New(fmt.Sprintf("Error executing delete: %+v", err))
		}
	} else {
		// Delete specific profiles
		for _, m := range ems {
			status, err := m.delete(tx, roleId)
			if err != nil {
				return status, err
			}
		}
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			errors.New(fmt.Sprintf("Transaction failed: %v", err.Error()))
	}

	go PurgeCache(h.ItemTypes[h.ItemTypeRole], roleId)

	return http.StatusOK, nil
}

func (m *RoleProfileType) Delete(roleId int64) (int, error) {
	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	status, err := m.delete(tx, roleId)
	if err != nil {
		return status, err
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			errors.New(fmt.Sprintf("Transaction failed: %v", err.Error()))
	}

	go PurgeCache(h.ItemTypes[h.ItemTypeRole], roleId)

	return http.StatusOK, nil
}

func (m *RoleProfileType) delete(tx *sql.Tx, roleId int64) (int, error) {

	// Only inserts once, cannot break the primary key
	_, err := tx.Exec(`
DELETE FROM role_profiles
 WHERE role_id = $1
   AND profile_id = $2`,
		roleId,
		m.Id,
	)
	if err != nil {
		return http.StatusInternalServerError,
			errors.New(fmt.Sprintf("Error executing delete: %v", err.Error()))
	}

	return http.StatusOK, nil
}

func GetRoleProfile(
	siteId int64,
	roleId int64,
	profileId int64,
) (
	ProfileSummaryType,
	int,
	error,
) {

	// Retrieve resources
	db, err := h.GetConnection()
	if err != nil {
		return ProfileSummaryType{}, http.StatusInternalServerError, err
	}

	rows, err := db.Query(`
SELECT profile_id
  FROM role_profiles
 WHERE role_id = $1
   AND profile_id = $2`,
		roleId,
		profileId,
	)
	if err != nil {
		return ProfileSummaryType{}, http.StatusInternalServerError,
			errors.New(
				fmt.Sprintf("Database query failed: %v", err.Error()),
			)
	}
	defer rows.Close()

	m := ProfileSummaryType{}
	for rows.Next() {
		err = rows.Scan(
			&m.Id,
		)
		if err != nil {
			return ProfileSummaryType{}, http.StatusInternalServerError,
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
			return ProfileSummaryType{}, resp.Status, resp.Err
		}

		m = resp.Item
	}
	err = rows.Err()
	if err != nil {
		return ProfileSummaryType{}, http.StatusInternalServerError,
			errors.New(
				fmt.Sprintf("Error fetching rows: %v", err.Error()),
			)
	}
	rows.Close()

	return m, http.StatusOK, nil
}

func GetRoleProfiles(
	siteId int64,
	roleId int64,
	limit int64,
	offset int64,
) (
	[]ProfileSummaryType,
	int64,
	int64,
	int,
	error,
) {

	db, err := h.GetConnection()
	if err != nil {
		return []ProfileSummaryType{}, 0, 0, http.StatusInternalServerError, err
	}

	rows, err := db.Query(`
SELECT COUNT(*) OVER() AS total
      ,rp.profile_id
  FROM role_profiles rp,
       roles r
 WHERE r.role_id = rp.role_id
   AND r.role_id = $1
 ORDER BY rp.profile_id ASC
 LIMIT $2
OFFSET $3`,
		roleId,
		limit,
		offset,
	)
	if err != nil {
		return []ProfileSummaryType{}, 0, 0, http.StatusInternalServerError,
			errors.New(
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
			return []ProfileSummaryType{}, 0, 0, http.StatusInternalServerError,
				errors.New(
					fmt.Sprintf("Row parsing error: %v", err.Error()),
				)
		}

		ids = append(ids, id)
	}
	err = rows.Err()
	if err != nil {
		return []ProfileSummaryType{}, 0, 0, http.StatusInternalServerError,
			errors.New(
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
		return []ProfileSummaryType{}, 0, 0, http.StatusBadRequest, errors.New(
			fmt.Sprintf(
				"not enough records, offset (%d) would return an empty page.",
				offset,
			),
		)
	}

	return ems, total, pages, http.StatusOK, nil

}
