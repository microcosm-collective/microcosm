package models

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang/glog"
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

// UserMembership is for managing user permissions
type UserMembership struct {
	Email     string `json:"email"`
	IsMember  bool   `json:"isMember"`
	userID    int64
	user      *UserType
	profileID int64
	profile   *ProfileType
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
		glog.Errorf("Cannot get connection: %s", err.Error())
		return http.StatusInternalServerError, err
	}

	// Create all users that we need to create first
	tx, err := db.Begin()
	if err != nil {
		glog.Errorf("Cannot start transaction: %s", err.Error())
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	var emails []string
	for _, m := range ems {
		emails = append(emails, fmt.Sprintf(`"%s"`, m.Email))
	}
	emailSQLArr := fmt.Sprintf(`{%s}`, strings.Join(emails, ","))

	// Return user ids for those users that exist, and return nulls for those
	// that don't.
	rows, err := tx.Query(`-- ManageUsers::GetUsers
WITH e AS (
    SELECT r.email
          ,canonical_email(r.email)
      FROM (
               SELECT UNNEST($1::text[]) AS email
           ) AS r
)
SELECT distinct ON (e.canonical_email)
       e.canonical_email
      ,e.email
      ,u.user_id
  FROM e
  LEFT JOIN users u ON u.canonical_email = e.canonical_email
 ORDER BY e.canonical_email, u.created ASC;`,
		emailSQLArr,
	)
	if err != nil {
		glog.Errorf("ManageUsers::GetUsers: %s", err.Error())
		return http.StatusInternalServerError, err
	}
	defer rows.Close()

	type User struct {
		Email  string
		UserID int64
	}
	var users []User

	for rows.Next() {
		var (
			canonicalEmail string
			email          string
			userID         sql.NullInt64
		)
		err = rows.Scan(
			&canonicalEmail,
			&email,
			&userID,
		)
		if err != nil {
			glog.Errorf("ManageUsers::GetUsers::Scan: %s", err.Error())
			return http.StatusInternalServerError, err
		}

		var u User
		u.Email = email
		if userID.Valid {
			u.UserID = userID.Int64
		}
		users = append(users, u)
	}
	if err = rows.Err(); err != nil {
		glog.Errorf("ManageUsers::GetUsers::Rows: %s", err.Error())
		return http.StatusInternalServerError, err
	}
	rows.Close()

	// Update the userIDs of those we have found, and for the rest mark them as
	// to be created
	var emailsToCreate []string
	for mi, m := range ems {
		for _, u := range users {
			if m.Email != u.Email {
				continue
			}

			if u.UserID > 0 {
				ems[mi].userID = u.UserID
			} else {
				emailsToCreate = append(emailsToCreate, fmt.Sprintf(`"%s"`, u.Email))
			}
		}
	}

	// Create the users that we need to create
	if len(emailsToCreate) > 0 {
		emailSQLArr = fmt.Sprintf(`{%s}`, strings.Join(emailsToCreate, ","))
		rows2, err := tx.Query(`--ManageUsers::CreateUsers
WITH e AS (
    SELECT r.email
          ,canonical_email(r.email)
      FROM (
               SELECT UNNEST($1::text[]) AS email
           ) AS r
)
INSERT INTO users (email, created, language, is_banned, password, password_date, canonical_email)
SELECT e.email, NOW(), '', false, '', NOW(), e.canonical_email
  FROM e
  RETURNING email, user_id;`,
			emailSQLArr,
		)
		if err != nil {
			glog.Errorf("ManageUsers::CreateUsers: %s", err.Error())
			return http.StatusInternalServerError, err
		}
		defer rows2.Close()

		users = []User{}
		for rows2.Next() {
			var u User
			err = rows2.Scan(
				&u.Email,
				&u.UserID,
			)
			if err != nil {
				glog.Errorf("ManageUsers::CreateUsers::Scan: %s", err.Error())
				return http.StatusInternalServerError, err
			}
			users = append(users, u)
		}
		if err = rows2.Err(); err != nil {
			glog.Errorf("ManageUsers::CreateUsers::Rows: %s", err.Error())
			return http.StatusInternalServerError, err
		}
		rows2.Close()

		// Update the remaining users so that after this every single user has a
		// valid user_id
		for mi, m := range ems {
			for _, u := range users {
				if m.Email != u.Email {
					continue
				}
				ems[mi].userID = u.UserID
			}
		}
	}

	// Assert that we are in a good place
	for _, m := range ems {
		if m.userID == 0 {
			glog.Errorf("%s does not have a user_id", m.Email)
			return http.StatusInternalServerError, fmt.Errorf("%s does not have a user_id", m.Email)
		}
	}

	// Find out which users do not yet have profiles, and get the profiles of
	// the users who do have profiles.
	var userIDStr []string
	for _, m := range ems {
		userIDStr = append(userIDStr, fmt.Sprintf(`%d`, m.userID))
	}

	rows3, err := tx.Query(`--ManageUsers::GetProfiles
with e AS (
    SELECT UNNEST($1::bigint[]) AS user_id
)
SELECT e.user_id
      ,p.profile_id
  FROM e
  LEFT outer JOIN profiles p ON p.user_id = e.user_id and site_id = $2;`,
		fmt.Sprintf(`{%s}`, strings.Join(userIDStr, ",")),
		site.ID,
	)
	if err != nil {
		glog.Errorf("ManageUsers::GetProfiles: %s", err.Error())
		return http.StatusInternalServerError, err
	}
	defer rows3.Close()

	type U2P struct {
		userID    int64
		profileID int64
	}
	var u2ps []U2P
	for rows3.Next() {
		var (
			u2p U2P
			pid sql.NullInt64
		)
		err = rows3.Scan(
			&u2p.userID,
			&pid,
		)
		if err != nil {
			glog.Errorf("ManageUsers::GetProfiles::Scan: %s", err.Error())
			return http.StatusInternalServerError, err
		}
		if pid.Valid {
			u2p.profileID = pid.Int64
		}
		u2ps = append(u2ps, u2p)
	}
	if err = rows3.Err(); err != nil {
		glog.Errorf("ManageUsers::GetProfiles::Rows: %s", err.Error())
		return http.StatusInternalServerError, err
	}
	rows3.Close()

	// Track the found profiles and
	type ProfileNeeded struct {
		UserID    int64
		Username  string
		Email     string
		ProfileID int64
	}
	var profilesToCreate []ProfileNeeded
	for mi, m := range ems {
		for _, u2p := range u2ps {
			if m.userID != u2p.userID {
				continue
			}
			if u2p.profileID > 0 {
				ems[mi].profileID = u2p.profileID
			} else {
				profilesToCreate = append(
					profilesToCreate,
					ProfileNeeded{
						UserID:   u2p.userID,
						Username: fmt.Sprintf("user%d", u2p.userID+UserIDOffset),
						Email:    m.Email,
					},
				)
			}
		}
	}
	if len(profilesToCreate) > 0 {
		profileOptions, status, err := GetProfileOptionsDefaults(site.ID)
		if err != nil {
			glog.Errorf("GetProfileOptionsDefaults: %s", err.Error())
			return status, err
		}

		for pi, p := range profilesToCreate {
			// Create profile
			var insertID int64
			err = tx.QueryRow(`--Create Profile
INSERT INTO profiles (
    site_id
   ,user_id
   ,profile_name
   ,gender
   ,is_visible

   ,style_id
   ,item_count
   ,comment_count
   ,avatar_url
   ,avatar_id

   ,created
   ,last_active
) VALUES (
    $1
   ,$2
   ,$3
   ,NULL
   ,TRUE

   ,0
   ,0
   ,0
   ,NULL
   ,NULL

   ,NOW()
   ,NOW()
) RETURNING profile_id`,
				site.ID,
				p.UserID,
				p.Username,
			).Scan(
				&insertID,
			)
			if err != nil {
				glog.Errorf("Create Profile: %s", err.Error())
				return http.StatusInternalServerError, err
			}

			profileOptions.ProfileID = insertID

			status, err = profileOptions.Insert(tx)
			if err != nil {
				glog.Errorf("profileOptions.Insert: %s", err.Error())
				return status,
					fmt.Errorf("Could not insert new profile options: %+v", err)
			}

			gravatarURL := MakeGravatarURL(p.Email)

			file, status, err := storeGravatar(tx, gravatarURL)
			if err != nil {
				glog.Errorf("storeGravatar: %s", err.Error())
				return status, err
			}

			// TODO: This needs to take the tx
			attachment := AttachmentType{}
			attachment.AttachmentMetaID = file.AttachmentMetaID
			attachment.FileHash = file.FileHash
			attachment.Created = time.Now()
			attachment.ItemTypeID = h.ItemTypes[h.ItemTypeProfile]
			attachment.ItemID = insertID
			attachment.ProfileID = insertID

			_, err = attachment.insert(tx)
			if err != nil {
				glog.Errorf("attachment.insert: %s", err.Error())
				return http.StatusInternalServerError,
					fmt.Errorf(
						"Could not create avatar attachment to profile: %+v",
						err,
					)
			}

			_, err = tx.Exec(
				`UPDATE profiles SET avatar_id = $2 WHERE profile_id = $1`,
				insertID,
				sql.NullInt64{
					Int64: attachment.AttachmentID,
					Valid: true,
				},
			)
			if err != nil {
				glog.Errorf("SET avatar_id: %s", err.Error())
				return http.StatusInternalServerError, err
			}

			// Update our profileID
			profilesToCreate[pi].ProfileID = insertID
		}

		// Update the profile IDs
		for mi, m := range ems {
			for _, p := range profilesToCreate {
				if m.userID != p.UserID {
					continue
				}
				ems[mi].profileID = p.ProfileID
			}
		}
	}

	// Assert that we have all of the profiles
	for _, m := range ems {
		if m.profileID == 0 {
			glog.Errorf("%s does not have a profile_id", m.Email)
			return http.StatusInternalServerError,
				fmt.Errorf("%s does not have a profile_id", m.Email)
		}
	}

	// Create a list of every profileID that we change permission on, as we
	// will need to flush the cached privileges later.
	var touchedProfiles []string

	// Handle removal of privileges
	var revocations []string
	for _, m := range ems {
		if !m.IsMember {
			revocations = append(revocations, fmt.Sprintf("%d", m.profileID))
			touchedProfiles = append(touchedProfiles, fmt.Sprintf("%d", m.profileID))
		}
	}
	if len(revocations) > 0 {
		_, err := tx.Exec(`--ManageUsers::Revoke
WITH a AS (
    SELECT attribute_id
      FROM attribute_keys ak
     WHERE item_type_id = 3
       AND item_id = ANY($1::bigint[])
       AND "key" = 'is_member'
), av AS (
    DELETE
      FROM attribute_values
     WHERE attribute_id IN (SELECT * FROM a)
)
DELETE
  FROM attribute_keys
 WHERE attribute_id IN (SELECT * FROM a)
`,
			fmt.Sprintf(`{%s}`, strings.Join(revocations, ",")),
		)
		if err != nil {
			glog.Errorf("ManageUsers::Revoke: %s", err.Error())
			return http.StatusInternalServerError,
				fmt.Errorf("Revoke failed: %s", err.Error())
		}
	}

	// Handle grant of privileges
	var grant []string
	for _, m := range ems {
		if m.IsMember {
			grant = append(grant, fmt.Sprintf("%d", m.profileID))
			touchedProfiles = append(touchedProfiles, fmt.Sprintf("%d", m.profileID))
		}
	}

	if len(grant) > 0 {
		// Check whether they need the permission granted (if they don't have it)
		rows4, err := tx.Query(`--ManageUsers::CheckGrant
WITH a AS (
    SELECT UNNEST($1::bigint[]) AS pid
)
SELECT a.pid
  FROM a
  LEFT JOIN attribute_keys ak ON ak.item_id = a.pid AND ak.item_type_id = 3 AND ak."key" = 'is_member'
 WHERE ak.item_id IS NULL;`,
			fmt.Sprintf(`{%s}`, strings.Join(grant, ",")),
		)
		if err != nil {
			glog.Errorf("ManageUsers::CheckGrant: %s", err.Error())
			return http.StatusInternalServerError,
				fmt.Errorf("Check grant failed: %s", err.Error())
		}
		defer rows4.Close()

		grant = []string{}
		for rows4.Next() {
			var pid int64
			if err := rows4.Scan(&pid); err != nil {
				glog.Errorf("ManageUsers::CheckGrant::Scan: %s", err.Error())
				return http.StatusInternalServerError,
					fmt.Errorf("ManageUsers::CheckGrant::Scan: %s", err.Error())
			}
			grant = append(grant, fmt.Sprintf("%d", pid))
		}
		if err := rows4.Err(); err != nil {
			glog.Errorf("ManageUsers::CheckGrant::Rows: %s", err.Error())
			return http.StatusInternalServerError,
				fmt.Errorf("ManageUsers::CheckGrant::Rows: %s", err.Error())
		}

		// Grant if we have any to grant
		if len(grant) > 0 {
			_, err := tx.Exec(`-- ManageUsers::Grant
WITH a AS (
    INSERT INTO attribute_keys (item_type_id, item_id, "key")
    SELECT 3, pid, 'is_member'
      FROM UNNEST($1::bigint[]) AS pid
 RETURNING attribute_id
)
INSERT INTO attribute_values (attribute_id, value_type_id, "boolean")
SELECT a.attribute_id, 4, TRUE
  FROM a;`,
				fmt.Sprintf(`{%s}`, strings.Join(grant, ",")),
			)
			if err != nil {
				glog.Errorf("ManageUsers::Grant: %s", err.Error())
				return http.StatusInternalServerError,
					fmt.Errorf("Grant failed: %s", err.Error())
			}
		}
	}

	// Now we flush the cache permissions
	if len(touchedProfiles) > 0 {
		_, err := tx.Exec(
			`DELETE FROM permissions_cache WHERE profile_id IN (SELECT UNNEST($1::bigint[]))`,
			fmt.Sprintf(`{%s}`, strings.Join(touchedProfiles, ",")),
		)
		if err != nil {
			glog.Errorf("ManageUsers::FlushPermissionsCache: %s", err.Error())
			return http.StatusInternalServerError,
				fmt.Errorf("FlushPermissionsCache failed: %s", err.Error())
		}
		_, err = tx.Exec(
			`DELETE FROM role_members_cache WHERE profile_id IN (SELECT UNNEST($1::bigint[]))`,
			fmt.Sprintf(`{%s}`, strings.Join(touchedProfiles, ",")),
		)
		if err != nil {
			glog.Errorf("ManageUsers::FlushRoleMembersCache: %s", err.Error())
			return http.StatusInternalServerError,
				fmt.Errorf("FlushRoleMembersCache failed: %s", err.Error())
		}
	}

	if err = tx.Commit(); err != nil {
		glog.Errorf("Commit: %s", err.Error())
		return http.StatusInternalServerError,
			fmt.Errorf("Commit failed: %s", err.Error())
	}

	return http.StatusOK, nil
}
