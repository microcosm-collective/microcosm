package models

import (
	"database/sql"
	"fmt"
	"net/http"

	h "github.com/microcosm-cc/microcosm/helpers"
)

// UpdateDefaultOptionType contains the default communication options
type UpdateDefaultOptionType struct {
	UpdateTypeID int64 `json:"id"`
	SendEmail    bool  `json:"sendEmail"`
	SendSMS      bool  `json:"sendSMS"`
}

// UpdateOptionType contains a user preference for a communication option
type UpdateOptionType struct {
	ProfileID    int64          `json:"profileId"`
	UpdateTypeID int64          `json:"id"`
	Description  string         `json:"description"`
	SendEmail    bool           `json:"sendEmail"`
	SendSMS      bool           `json:"sendSMS"`
	Meta         h.CoreMetaType `json:"meta"`
}

// Validate returns true if the integrity of the communications option is valid
func (m *UpdateOptionType) Validate() (int, error) {

	if m.ProfileID < 1 {
		return http.StatusBadRequest,
			fmt.Errorf(
				"ID ('%d') cannot be zero or negative",
				m.ProfileID,
			)
	}

	if m.UpdateTypeID < 1 {
		return http.StatusBadRequest,
			fmt.Errorf(
				"Update type ID ('%d') cannot be zero or negative",
				m.UpdateTypeID,
			)
	}

	return http.StatusOK, nil
}

// Insert creates a communication option to the database
func (m *UpdateOptionType) Insert() (int, error) {

	status, err := m.Validate()
	if err != nil {
		return status, err
	}

	// Check that a corresponding profile_options record exists,
	// otherwise profile_id constraint will fail.
	// TODO(lewi): check receive_email doesn't contradict
	// profile_options.email_updates
	_, status, err = GetProfileOptions(m.ProfileID)
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("Insert of update preference failed: %v", err.Error())
	}

	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
INSERT INTO update_options (
    profile_id
   ,update_type_id
   ,send_email
   ,send_sms
) VALUES (
    $1,
    $2,
    $3,
    $4
)`,
		m.ProfileID,
		m.UpdateTypeID,
		m.SendEmail,
		m.SendSMS,
	)
	if err != nil {
		tx.Rollback()
		return http.StatusInternalServerError,
			fmt.Errorf("Insert of update option failed: %v", err.Error())
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("Transaction failed: %v", err.Error())
	}

	return http.StatusOK, nil

}

// Update saves the communication preferences for an UpdateOptionType to the
// database
func (m *UpdateOptionType) Update() (int, error) {

	status, err := m.Validate()
	if err != nil {
		return status, err
	}

	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("Could not start transaction: %v", err.Error())
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
UPDATE update_options
   SET send_email = $3
      ,send_sms = $4
 WHERE profile_id = $1
   AND update_type_id = $2`,
		m.ProfileID,
		m.UpdateTypeID,
		m.SendEmail,
		m.SendSMS,
	)
	if err != nil {
		tx.Rollback()
		return http.StatusInternalServerError,
			fmt.Errorf("Update of update option failed: %v", err.Error())
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("Transaction failed: %v", err.Error())
	}

	return http.StatusOK, nil

}

// Delete removes an update option record for a specific user. Generally this
// will be unnecessary unless a user needs to clear their preferences completely
// and return to the defaults.
func (m *UpdateOptionType) Delete() (int, error) {

	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
DELETE FROM update_options
 WHERE profile_id = $1
   AND update_type_id = $2`,
		m.ProfileID,
		m.UpdateTypeID,
	)
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("Delete failed: %v", err.Error())
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("Transaction failed: %v", err.Error())
	}

	return http.StatusOK, nil

}

// GetUpdateOptionByUpdateType returns the notification settings (email, sms)
// for a given user and update type.
func GetUpdateOptionByUpdateType(
	profileID int64,
	updateTypeID int64,
) (
	UpdateOptionType,
	int,
	error,
) {

	db, err := h.GetConnection()
	if err != nil {
		return UpdateOptionType{}, http.StatusInternalServerError, err
	}

	var m UpdateOptionType
	err = db.QueryRow(`
SELECT uo.profile_id
      ,uo.update_type_id
      ,ut.description
      ,uo.send_email
      ,uo.send_sms
  FROM update_options uo
       LEFT JOIN update_types ut ON uo.update_type_id = ut.update_type_id
 WHERE uo.profile_id = $1
   AND uo.update_type_id = $2`,
		profileID,
		updateTypeID,
	).Scan(
		&m.ProfileID,
		&m.UpdateTypeID,
		&m.Description,
		&m.SendEmail,
		&m.SendSMS,
	)
	if err == sql.ErrNoRows {
		return UpdateOptionType{}, http.StatusNotFound,
			fmt.Errorf("Update options for profile ID %d not found", profileID)
	} else if err != nil {
		return UpdateOptionType{}, http.StatusInternalServerError,
			fmt.Errorf("Database query failed: %v", err.Error())
	}

	return m, http.StatusOK, nil
}

// GetUpdateOptions retrieves a user's alert preferences for all available
// alert types. This is not paginated since the collection should always fit on
// a single page.
func GetUpdateOptions(
	siteID int64,
	profileID int64,
) (
	[]UpdateOptionType,
	int,
	error,
) {

	// We can't guarantee that the profile_options exists for this
	_, status, err := GetProfileOptions(profileID)
	if err != nil {
		return []UpdateOptionType{}, status, err
	}

	var ems []UpdateOptionType

	for _, updateTypeID := range h.UpdateTypes {
		//-1, -1 will always return the default settings as there will be no
		// item-specific options found
		m, status, err := GetCommunicationOptions(
			siteID,
			profileID,
			updateTypeID,
			-1,
			-1,
		)
		if err != nil {
			return []UpdateOptionType{}, status, err
		}
		ems = append(ems, m)
	}

	return ems, http.StatusOK, nil
}
