package models

import (
	"database/sql"
	"fmt"
	"net/http"
	"sort"
	"sync"

	h "github.com/microcosm-cc/microcosm/helpers"
)

// RoleProfilesType is an array of RoleProfileType
type RoleProfilesType struct {
	RoleProfiles h.ArrayType    `json:"profiles"`
	Meta         h.CoreMetaType `json:"meta"`
}

// RoleProfileType describes a profile that belongs to a role
type RoleProfileType struct {
	ID      int64       `json:"id,omitempty"`
	Profile interface{} `json:"profile"`
}

// GetLink returns an API link to this role profile
func (m *RoleProfileType) GetLink(roleLink string) string {
	return fmt.Sprintf("%s/profiles/%d", roleLink, m.ID)
}

// Validate returns true if the role profile is valid
func (m *RoleProfileType) Validate(siteID int64) (int, error) {
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

// UpdateManyRoleProfiles allows many role profiles to be added at the
// same time
func UpdateManyRoleProfiles(
	siteID int64,
	roleID int64,
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
		status, err := m.insert(tx, siteID, roleID)
		if err != nil {
			return status, err
		}
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("transaction failed: %v", err.Error())
	}

	go PurgeCache(h.ItemTypes[h.ItemTypeRole], roleID)

	return http.StatusOK, nil
}

// Update adds a single profile to the role
func (m *RoleProfileType) Update(siteID int64, roleID int64) (int, error) {
	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	status, err := m.insert(tx, siteID, roleID)
	if err != nil {
		return status, err
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("transaction failed: %v", err.Error())
	}

	go PurgeCache(h.ItemTypes[h.ItemTypeRole], roleID)

	return http.StatusOK, nil
}

// insert saves a profile to a role
func (m *RoleProfileType) insert(
	tx *sql.Tx,
	siteID int64,
	roleID int64,
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
INSERT INTO role_profiles
SELECT $1, $2
 WHERE NOT EXISTS (
       SELECT role_id
             ,profile_id
         FROM role_profiles
        WHERE role_id = $1
          AND profile_id = $2
       )`,
		roleID,
		m.ID,
	)
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("error executing upsert: %v", err.Error())
	}

	return http.StatusOK, nil
}

// DeleteManyRoleProfiles removes many profiles from a role at once
func DeleteManyRoleProfiles(roleID int64, ems []RoleProfileType) (int, error) {
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
			roleID,
		)
		if err != nil {
			return http.StatusInternalServerError,
				fmt.Errorf("error executing delete: %+v", err)
		}
	} else {
		// Delete specific profiles
		for _, m := range ems {
			status, err := m.delete(tx, roleID)
			if err != nil {
				return status, err
			}
		}
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("transaction failed: %v", err.Error())
	}

	go PurgeCache(h.ItemTypes[h.ItemTypeRole], roleID)

	return http.StatusOK, nil
}

// Delete removes a profile from a role
func (m *RoleProfileType) Delete(roleID int64) (int, error) {
	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	status, err := m.delete(tx, roleID)
	if err != nil {
		return status, err
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("transaction failed: %v", err.Error())
	}

	go PurgeCache(h.ItemTypes[h.ItemTypeRole], roleID)

	return http.StatusOK, nil
}

// delete deletes a profile from a role
func (m *RoleProfileType) delete(tx *sql.Tx, roleID int64) (int, error) {
	_, err := tx.Exec(`
DELETE FROM role_profiles
 WHERE role_id = $1
   AND profile_id = $2`,
		roleID,
		m.ID,
	)
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("error executing delete: %v", err.Error())
	}

	return http.StatusOK, nil
}

// GetRoleProfile returns a single profile for a role
func GetRoleProfile(
	siteID int64,
	roleID int64,
	profileID int64,
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
		roleID,
		profileID,
	)
	if err != nil {
		return ProfileSummaryType{}, http.StatusInternalServerError,
			fmt.Errorf("database query failed: %v", err.Error())
	}
	defer rows.Close()

	m := ProfileSummaryType{}
	for rows.Next() {
		err = rows.Scan(
			&m.ID,
		)
		if err != nil {
			return ProfileSummaryType{}, http.StatusInternalServerError,
				fmt.Errorf("error fetching row: %v", err.Error())
		}

		// Make a request the profile summary
		req := make(chan ProfileSummaryRequest)
		defer close(req)

		go HandleProfileSummaryRequest(siteID, m.ID, 0, req)

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
			fmt.Errorf("error fetching rows: %v", err.Error())
	}
	rows.Close()

	return m, http.StatusOK, nil
}

// GetRoleProfiles fetches multiple profiles belonging to a role
func GetRoleProfiles(
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
