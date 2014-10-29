package models

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	h "github.com/microcosm-cc/microcosm/helpers"
)

type ProfileOptionType struct {
	ProfileId     int64 `json:"profileId"`
	ShowDOB       bool  `json:"showDOB"`
	ShowDOBYear   bool  `json:"showDOBYear"`
	SendEMail     bool  `json:"sendEmail"`
	SendSMS       bool  `json:"sendSMS"`
	IsDiscouraged bool  `json:"isDiscouraged"`
}

func (m *ProfileOptionType) Insert(tx *sql.Tx) (int, error) {

	_, err := tx.Exec(`
INSERT INTO profile_options (
    profile_id
   ,show_dob_year
   ,show_dob_date
   ,send_email
   ,send_sms
   ,is_discouraged
) VALUES (
    $1
   ,$2
   ,$3
   ,$4
   ,$5
   ,$6
)`,
		m.ProfileId,
		m.ShowDOBYear,
		m.ShowDOB,
		m.SendEMail,
		m.SendSMS,
		m.IsDiscouraged,
	)
	if err != nil {
		tx.Rollback()
		return http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Error inserting data: %v", err.Error()),
		)
	}

	return http.StatusOK, nil
}

func (m *ProfileOptionType) Update() (int, error) {

	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Could not start transaction: %v", err.Error()),
		)
	}

	defer tx.Rollback()

	_, err = tx.Exec(`
UPDATE profile_options
    SET show_dob_year = $2
    ,show_dob_date = $3
    ,send_email = $4
    ,send_sms = $5
    ,is_discouraged = $6
WHERE profile_id = $1`,
		m.ProfileId,
		m.ShowDOBYear,
		m.ShowDOB,
		m.SendEMail,
		m.SendSMS,
		m.IsDiscouraged,
	)
	if err != nil {
		tx.Rollback()
		return http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Error inserting data: %v", err.Error()),
		)
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Transaction failed: %v", err.Error()),
		)
	}

	return http.StatusOK, nil
}

func GetProfileOptions(profileId int64) (ProfileOptionType, int, error) {

	db, err := h.GetConnection()
	if err != nil {
		return ProfileOptionType{}, http.StatusInternalServerError, err
	}

	var m ProfileOptionType
	err = db.QueryRow(`
SELECT profile_id
      ,show_dob_date
      ,show_dob_year
      ,send_email
      ,send_sms
      ,is_discouraged
  FROM profile_options
 WHERE profile_id = $1`,
		profileId,
	).Scan(
		&m.ProfileId,
		&m.ShowDOB,
		&m.ShowDOBYear,
		&m.SendEMail,
		&m.SendSMS,
		&m.IsDiscouraged,
	)
	if err == sql.ErrNoRows {
		return ProfileOptionType{}, http.StatusNotFound,
			errors.New(
				fmt.Sprintf("Resource with profile ID %d not found", profileId),
			)

	} else if err != nil {
		return ProfileOptionType{}, http.StatusInternalServerError,
			errors.New(
				fmt.Sprintf("Database query failed: %v", err.Error()),
			)
	}

	return m, http.StatusOK, nil
}

func GetProfileOptionsDefaults(siteId int64) (ProfileOptionType, int, error) {

	db, err := h.GetConnection()
	if err != nil {
		return ProfileOptionType{}, http.StatusInternalServerError, err
	}

	rows, err := db.Query(`
SELECT COALESCE(s.send_email, p.send_email) AS send_email
      ,COALESCE(s.send_sms, p.send_sms) AS send_sms
  FROM platform_options p
       LEFT JOIN (
           SELECT send_email
                 ,send_sms
             FROM site_options
            WHERE site_id = $1
       ) s ON 1=1`,
		siteId,
	)
	if err != nil {
		return ProfileOptionType{}, http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Database query failed: %v", err.Error()),
		)
	}
	defer rows.Close()

	var m ProfileOptionType

	for rows.Next() {
		m = ProfileOptionType{}
		err = rows.Scan(
			&m.SendEMail,
			&m.SendSMS,
		)
		if err != nil {
			return ProfileOptionType{}, http.StatusInternalServerError, errors.New(
				fmt.Sprintf("Row parsing error: %v", err.Error()),
			)
		}
	}
	err = rows.Err()
	if err != nil {
		return ProfileOptionType{}, http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Error fetching rows: %v", err.Error()),
		)
	}
	rows.Close()

	m.IsDiscouraged = false
	m.ShowDOB = false
	m.ShowDOBYear = false

	return m, http.StatusOK, nil
}
