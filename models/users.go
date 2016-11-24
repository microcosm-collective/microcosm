package models

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	c "github.com/microcosm-cc/microcosm/cache"
	h "github.com/microcosm-cc/microcosm/helpers"
)

// UsersType offers an array of users
type UsersType struct {
	Users h.ArrayType    `json:"users"`
	Meta  h.CoreMetaType `json:"meta"`
}

// UserType encapsulates a user in the system
type UserType struct {
	ID             int64          `json:"userId"`
	Email          string         `json:"email"`
	Gender         sql.NullString `json:"gender,omitempty"`
	Language       string         `json:"language,omitempty"`
	Created        time.Time      `json:"created"`
	State          sql.NullString `json:"state"`
	Banned         bool           `json:"banned,omitempty"`
	Password       string         `json:"-"`
	PasswordDate   time.Time      `json:"-"`
	DobDay         sql.NullInt64  `json:"dobDay,omitempty"`
	DobMonth       sql.NullInt64  `json:"dobMonth,omitempty"`
	DobYear        sql.NullInt64  `json:"dobYear,omitempty"`
	CanonicalEmail string         `json:"canonicalEmail,omitempty"`

	Meta h.CoreMetaType `json:"meta"`
}

// UserMembership
type UserMembership struct {
	Email    string `json:"email"`
	IsMember bool   `json:"isMember"`
	user     *UserType
	profile  *ProfileType
}

// Validate checks that a given user has all the required information to be
// created or updated successfully
func (m *UserType) Validate(exists bool) (int, error) {

	if exists == false {
		if m.ID != 0 {
			return http.StatusBadRequest,
				fmt.Errorf("You cannot specify an ID")
		}

		if !m.Created.IsZero() {
			return http.StatusBadRequest,
				fmt.Errorf("You cannot specify creation time")
		}
	}

	if strings.Trim(m.Email, " ") == "" {
		return http.StatusBadRequest,
			fmt.Errorf("An email address must be provided")
	}

	return http.StatusOK, nil
}

// Insert creates a user
func (m *UserType) Insert() (int, error) {
	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("Could not start transaction: %v", err.Error())
	}
	defer tx.Rollback()

	status, err := m.insert(tx)
	if err != nil {
		return status, err
	}

	if err = tx.Commit(); err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("Transaction failed: %v", err.Error())
	}

	return http.StatusOK, nil
}

func (m *UserType) insert(q h.Er) (int, error) {

	status, err := m.Validate(false)
	if err != nil {
		return status, err
	}

	var insertID int64
	err = q.QueryRow(`
INSERT INTO users (
    email, created, language, is_banned, password,
    password_date, canonical_email
) VALUES (
    $1, NOW(), $2, false, $3,
    NOW(), canonical_email($1)
) RETURNING user_id`,
		m.Email,
		m.Language,
		m.Password,
	).Scan(
		&insertID,
	)
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("Error inserting data and returning ID: %+v", err)
	}
	m.ID = insertID

	return http.StatusOK, nil
}

// UserIsOnSite returns true if the given userId exists as a profile on the
// given site.
func UserIsOnSite(userID int64, siteID int64) bool {
	db, err := h.GetConnection()
	if err != nil {
		return false
	}

	var val bool
	err = db.QueryRow(`--UserIsOnSite
SELECT COUNT(*) > 0
  FROM profiles
 WHERE site_id = $1
   AND user_id = $2`,
		siteID,
		userID,
	).Scan(&val)
	if err != nil {
		return false
	}

	return val
}

// GetUser will fetch a user for a given ID
func GetUser(id int64) (UserType, int, error) {
	db, err := h.GetConnection()
	if err != nil {
		return UserType{}, http.StatusInternalServerError, err
	}

	return getUser(db, id)
}

func getUser(q h.Er, id int64) (UserType, int, error) {
	if id == 0 {
		return UserType{}, http.StatusNotFound, fmt.Errorf("User not found")
	}

	// Get from cache if it's available
	mcKey := fmt.Sprintf(mcProfileKeys[c.CacheUser], id)
	if val, ok := c.Get(mcKey, UserType{}); ok {
		return val.(UserType), http.StatusOK, nil
	}

	var m UserType
	err := q.QueryRow(`
SELECT user_id
      ,email
      ,gender
      ,language
      ,created
      ,state
      ,is_banned
      ,password
      ,password_date
      ,dob_day
      ,dob_month
      ,dob_year
      ,canonical_email
  FROM users
 WHERE user_id = $1`,
		id,
	).Scan(
		&m.ID,
		&m.Email,
		&m.Gender,
		&m.Language,
		&m.Created,
		&m.State,
		&m.Banned,
		&m.Password,
		&m.PasswordDate,
		&m.DobDay,
		&m.DobMonth,
		&m.DobYear,
		&m.CanonicalEmail,
	)
	if err == sql.ErrNoRows {
		return UserType{}, http.StatusNotFound,
			fmt.Errorf("Resource with id %v not found", id)
	} else if err != nil {
		return UserType{}, http.StatusInternalServerError,
			fmt.Errorf("Database query failed: %v", err.Error())
	}
	m.Meta.Links =
		[]h.LinkType{
			h.GetLink("self", "", h.ItemTypeUser, m.ID),
		}

	c.Set(mcKey, m, mcTTL)

	return m, http.StatusOK, nil
}

// Update is not yet implemented
func (m *UserType) Update() (int, error) {

	return http.StatusNotImplemented,
		fmt.Errorf("Update User is not yet implemented")
	/*
	   	if m.Id < 1 {
	   		return http.StatusBadRequest, fmt.Errorf(fmt.Sprintf(
	   			"Invalid ID supplied: %v", m.Id),
	   		)
	   	}

	   	tx, err := h.GetTransaction()
	   	if err != nil {
	   		return http.StatusInternalServerError, fmt.Errorf(
	   			fmt.Sprintf("Could not start transaction: %v", err.Error()),
	   		)
	   	}

	   	defer tx.Rollback()

	   	stmt, err := tx.Prepare(`
	   UPDATE users (
	   	email,
	   	gender,
	   	language,
	   	state,
	   	is_banned,
	   	password,
	   	password_date,
	   	dob_day,
	   	dob_month,
	   	dob_year
	   ) VALUES (
	   	$2,
	   	$3,
	   	$4,
	   	$5,
	   	$6,
	   	$7,
	   	NOW(),
	   	$8,
	   	$9,
	   	$10
	   ) WHERE user_id = $1;`,
	   	)

	   	if err != nil {
	   		return http.StatusInternalServerError, fmt.Errorf(
	   			fmt.Sprintf("Could not prepare insert statement: %v", err.Error()),
	   		)
	   	}
	   	defer stmt.Close()

	   	row := stmt.QueryRow(
	   		m.Email,
	   		m.Gender,
	   		m.Language,
	   		m.State,
	   		m.Banned,
	   		m.Password,
	   		m.DobDay,
	   		m.DobMonth,
	   		m.DobYear,
	   	)

	   	var insertId int64
	   	err = row.Scan(&insertId)
	   	if err != nil {
	   		return http.StatusInternalServerError, fmt.Errorf(
	   			fmt.Sprintf("Error updating : %v", err.Error()),
	   		)
	   	}

	   	err = tx.Commit()
	   	if err != nil {
	   		return http.StatusInternalServerError, fmt.Errorf(
	   			fmt.Sprintf("Transaction failed: %v", err.Error()),
	   		)
	   	}

	   	return http.StatusOK, nil
	*/
}

// Delete is not yet implemented
func (m *UserType) Delete() (int, error) {

	return http.StatusNotImplemented,
		fmt.Errorf("Delete User is not yet implemented")

	/*
	   	if m.Id < 1 {
	   		return http.StatusBadRequest, fmt.Errorf(
	   			fmt.Sprintf("The supplied ID ('%d') cannot be zero or negative.", m.Id),
	   		)
	   	}

	   	tx, err := h.GetTransaction()
	   	if err != nil {
	   		return http.StatusInternalServerError, err
	   	}
	   	defer tx.Rollback()

	   	stmt, err := tx.Prepare(`
	   DELETE FROM users
	    WHERE user_id = $1`,
	   	)
	   	if err != nil {
	   		return http.StatusInternalServerError, fmt.Errorf(
	   			fmt.Sprintf("Could not prepare statement: %v", err.Error()),
	   		)
	   	}

	   	_, err = stmt.Exec(m.Id)
	   	if err != nil {
	   		return http.StatusInternalServerError, fmt.Errorf(
	   			fmt.Sprintf("Delete failed: %v", err.Error()),
	   		)
	   	}

	   	err = tx.Commit()
	   	if err != nil {
	   		return http.StatusInternalServerError, fmt.Errorf(
	   			fmt.Sprintf("Transaction failed: %v", err.Error()),
	   		)
	   	}

	   	return http.StatusOK, nil
	*/
}

// CreateUserByEmailAddress creates a stub user from an email address
func CreateUserByEmailAddress(email string) (UserType, int, error) {

	if strings.Trim(email, " ") == "" {
		return UserType{}, http.StatusBadRequest,
			fmt.Errorf("You must specify an email address")
	}

	m := UserType{}
	m.Email = email

	status, err := m.Insert()
	if err != nil {
		return UserType{}, status, err
	}

	return m, http.StatusOK, nil

}

// GetUserByEmailAddress performs a case-insensitive search for any matching
// user and returns it.
func GetUserByEmailAddress(email string) (UserType, int, error) {
	if strings.Trim(email, " ") == "" {
		return UserType{}, http.StatusBadRequest,
			fmt.Errorf("You must specify an email address")
	}

	db, err := h.GetConnection()
	if err != nil {
		return UserType{}, http.StatusInternalServerError, err
	}

	return getUserByEmailAddress(db, email)
}

func getUserByEmailAddress(q h.Er, email string) (UserType, int, error) {

	// Note that if multiple accounts exist for a given canonical email address
	// then the oldest account wins.
	var m UserType
	err := q.QueryRow(`--get user by email
SELECT user_id
  FROM users
 WHERE canonical_email = canonical_email($1)
 ORDER BY created ASC
 LIMIT 1`,
		email,
	).Scan(
		&m.ID,
	)
	if err == sql.ErrNoRows {
		return UserType{}, http.StatusNotFound,
			fmt.Errorf("Resource with email %v not found", email)
	} else if err != nil {
		return UserType{}, http.StatusInternalServerError,
			fmt.Errorf("Database query failed: %+v", err)
	}

	return getUser(q, m.ID)
}

// ManageUsers provides a way to ensure that a batch of users exists as profiles
// on a site (given their email address), and to grant/deny membership using the
// attribute key/values.
func ManageUsers(site SiteType, ems []UserMembership) (int, error) {
	db, err := h.GetConnection()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	// Create all users that we need to create first
	tx, err := db.Begin()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	for ii, um := range ems {
		u, status, err := getUserByEmailAddress(tx, um.Email)
		if err == nil {
			ems[ii].user = &u
			continue
		}
		if status == http.StatusNotFound {
			u := UserType{
				Email: um.Email,
			}
			status, err := u.insert(tx)
			if err != nil {
				return status, err
			}

			u, status, err = getUserByEmailAddress(tx, um.Email)
			if err != nil {
				return status, err
			}
			ems[ii].user = &u
			continue
		}
		return status, err
	}

	if err = tx.Commit(); err != nil {
		return http.StatusInternalServerError, err
	}

	// Create all profiles and grant/revoke permissions
	for _, um := range ems {
		if um.user == nil {
			continue
		}

		p, status, err := GetOrCreateProfile(site, *um.user)
		if err != nil {
			return status, err
		}

		key, status, err := GetAttributeID(h.ItemTypes[h.ItemTypeProfile], p.ID, "is_member")
		if err != nil && status != http.StatusNotFound {
			return status, err
		}

		if um.IsMember {
			// Grant access
			if key > 0 {
				// already granted
				continue
			}

			var at AttributeType
			at.Key = "is_member"
			at.Value = true
			status, err := at.Update(h.ItemTypes[h.ItemTypeProfile], p.ID)
			if err != nil {
				return status, err
			}
		} else {
			// Deny access
			if key == 0 {
				// already denied
				continue
			}

			var at AttributeType
			at.ID = key
			status, err := at.Delete()
			if err != nil {
				return status, err
			}
		}
	}

	return http.StatusOK, nil
}
