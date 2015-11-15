package models

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/golang/glog"

	c "github.com/microcosm-cc/microcosm/cache"
	h "github.com/microcosm-cc/microcosm/helpers"
)

// URLGravatar is the first part of the URL which Gravatar uses to serve
// avatars
const URLGravatar string = "https://secure.gravatar.com/avatar/"

// ProfilesType encapsulates a collection of profiles
type ProfilesType struct {
	Profiles h.ArrayType    `json:"profiles"`
	Meta     h.CoreMetaType `json:"meta"`
}

// ProfileSummaryType describes a profile, in summary
type ProfileSummaryType struct {
	ID                int64              `json:"id"`
	SiteID            int64              `json:"siteId,omitempty"`
	UserID            int64              `json:"userId"`
	ProfileName       string             `json:"profileName"`
	Visible           bool               `json:"visible"`
	AvatarURLNullable sql.NullString     `json:"-"`
	AvatarURL         string             `json:"avatar"`
	AvatarIDNullable  sql.NullInt64      `json:"-"`
	AvatarID          int64              `json:"-"`
	Meta              h.ExtendedMetaType `json:"meta"`
}

// ProfileType describes a profile
type ProfileType struct {
	ID                int64              `json:"id"`
	SiteID            int64              `json:"siteId,omitempty"`
	UserID            int64              `json:"userId"`
	Email             string             `json:"email,omitempty"`
	ProfileName       string             `json:"profileName"`
	Member            bool               `json:"member,omitempty"`
	GenderNullable    sql.NullString     `json:"-"`
	Gender            string             `json:"gender,omitempty"`
	Visible           bool               `json:"visible"`
	StyleID           int64              `json:"styleId"`
	ItemCount         int32              `json:"itemCount"`
	CommentCount      int32              `json:"commentCount"`
	ProfileComment    interface{}        `json:"profileComment"`
	Created           time.Time          `json:"created"`
	LastActive        time.Time          `json:"lastActive"`
	AvatarURLNullable sql.NullString     `json:"-"`
	AvatarURL         string             `json:"avatar"`
	AvatarIDNullable  sql.NullInt64      `json:"-"`
	AvatarID          int64              `json:"-"`
	Meta              h.ExtendedMetaType `json:"meta"`
}

// ProfileSearchOptions describes the available ways to search and filter
// profiles
type ProfileSearchOptions struct {
	OrderByCommentCount bool
	IsFollowing         bool
	IsOnline            bool
	StartsWith          string
	ProfileID           int64
}

// ProfileSummaryRequest is an envelope to request a profile
type ProfileSummaryRequest struct {
	Item   ProfileSummaryType
	Err    error
	Status int
	Seq    int
}

// ProfileSummaryRequestBySeq is a collection of profile requests
type ProfileSummaryRequestBySeq []ProfileSummaryRequest

// Len returns the length of the array
func (v ProfileSummaryRequestBySeq) Len() int {
	return len(v)
}

// Swap exchanges two items in the array
func (v ProfileSummaryRequestBySeq) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

// Less determines whether one profile is lower in sequence than another
func (v ProfileSummaryRequestBySeq) Less(i, j int) bool {
	return v[i].Seq < v[j].Seq
}

// ValidateProfileName returns true if the profile name is valid and not taken
// by another user on this site already
func ValidateProfileName(name string) (string, int, error) {
	// Note: We are not preventing shouting in usernames as some people will
	// use their initials for their username
	name = SanitiseText(strings.Trim(name, " "))

	if name == "" {
		return name, http.StatusBadRequest,
			fmt.Errorf("You must supply a profile name")
	}

	nameLen := utf8.RuneCountInString(name)
	if nameLen < 2 {
		return name, http.StatusBadRequest,
			fmt.Errorf("Profile name is too short, " +
				"it must be 2 characters or more")
	}

	if nameLen > 25 {
		return name, http.StatusBadRequest,
			fmt.Errorf("Profile name is too long, " +
				"it must be 25 or fewer characters in length")
	}

	if strings.Contains(name, " ") {
		return name, http.StatusBadRequest,
			fmt.Errorf("Profile name cannot contain a space, " +
				"have you considered using an underscore instead?")
	}

	if strings.Contains(name, "@") {
		return name, http.StatusBadRequest,
			fmt.Errorf("Profile name cannot contain an @, " +
				"have you considered using an underscore instead?")
	}

	if strings.Contains(name, "+") {
		return name, http.StatusBadRequest,
			fmt.Errorf("Profile name cannot contain a +, " +
				"have you considered using an underscore instead?")
	}

	return name, http.StatusOK, nil
}

// Validate returns true if the profile is valid
func (m *ProfileType) Validate(exists bool) (int, error) {

	m.Gender = SanitiseText(m.Gender)

	if m.SiteID < 1 {
		return http.StatusBadRequest, fmt.Errorf("Invalid site ID supplied")
	}

	if m.UserID < 1 {
		return http.StatusBadRequest, fmt.Errorf("Invalid user ID supplied")
	}

	if m.StyleID < 0 {
		return http.StatusBadRequest, fmt.Errorf("Invalid style ID supplied")
	}

	name, status, err := ValidateProfileName(m.ProfileName)
	if err != nil {
		return status, err
	}
	m.ProfileName = name

	profileNameTaken, status, err :=
		IsProfileNameTaken(m.SiteID, m.UserID, m.ProfileName)
	if err != nil {
		return status, err
	}

	if profileNameTaken {
		// Suggest an alternative
		user, status, err := GetUser(m.UserID)
		if err != nil {
			return status, err
		}

		m.ProfileName = SuggestProfileName(user)
	}

	if !exists {
		if m.ID != 0 {
			return http.StatusBadRequest,
				fmt.Errorf("You cannot specify an ID")
		}

		if m.ItemCount != 0 {
			return http.StatusBadRequest,
				fmt.Errorf("You cannot specify item count")
		}

		if m.CommentCount != 0 {
			return http.StatusBadRequest,
				fmt.Errorf("You cannot specify comment count")
		}

		if !m.Created.IsZero() {
			return http.StatusBadRequest,
				fmt.Errorf("You cannot specify creation time")
		}

		if !m.LastActive.IsZero() {
			return http.StatusBadRequest,
				fmt.Errorf("You cannot specify last active time")
		}
	}

	return http.StatusOK, nil
}

// Insert provides a public interface for creating a profile.
//
// Insert performs strict validation and will return an error if the data is
// not very good (i.e. username contains a space and created date was supplied)
func (m *ProfileType) Insert() (int, error) {
	status, err := m.Validate(false)
	if err != nil {
		return status, err
	}

	m.Created = time.Now()
	m.LastActive = time.Now()

	return m.insert(false)
}

// Import provides a public interface for creating a profile by importing an
// existing profile.
//
// Import performs permissive validation and will return an error only if the
// data is fundamentally crap. It will repair and fix any data it can, i.e.
// by replacing spaces in usernames
func (m *ProfileType) Import() (int, error) {

	// Microcosm usernames cannot contain spaces
	m.ProfileName = strings.Replace(m.ProfileName, " ", "_", -1)

	// Validates as if it already exists to avoid any of that messy "you can't
	// set the created data" rubbish
	status, err := m.Validate(true)
	if err != nil {
		return status, err
	}

	// If the user has never been active, use the date they were created
	if m.LastActive.Unix() < m.Created.Unix() {
		m.LastActive = m.Created
	}

	return m.insert(true)
}

// insert actually inserts the profile into the database
func (m *ProfileType) insert(isImport bool) (int, error) {

	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf(
			fmt.Sprintf("Could not start transaction: %v", err.Error()),
		)
	}

	defer tx.Rollback()

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
   ,$4
   ,$5

   ,$6
   ,$7
   ,$8
   ,$9
   ,$10

   ,$11
   ,$12
) RETURNING profile_id`,
		m.SiteID,
		m.UserID,
		m.ProfileName,
		m.GenderNullable,
		m.Visible,

		m.StyleID,
		m.ItemCount,
		m.CommentCount,
		m.AvatarURLNullable,
		m.AvatarIDNullable,

		m.Created,
		m.LastActive,
	).Scan(
		&insertID,
	)

	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("Error inserting data and returning ID: %+v", err)
	}
	m.ID = insertID

	// Set options for profile
	profileOptions, status, err := GetProfileOptionsDefaults(m.SiteID)
	if err != nil {
		return status,
			fmt.Errorf("Could not load default profile options: %+v", err)
	}
	profileOptions.ProfileID = insertID

	status, err = profileOptions.Insert(tx)
	if err != nil {
		return status,
			fmt.Errorf("Could not insert new profile options: %+v", err)
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("Transaction failed: %+v", err)
	}

	// Fetch gravatar (or default to pattern based on email address)
	user, _, err := GetUser(m.UserID)
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("No user found for profile: %+v", err)
	}

	// Create attachment for avatar and attach it to profile
	avatarURL := MakeGravatarURL(user.Email)
	if !isImport {
		fm, _, err := StoreGravatar(avatarURL)
		if err != nil {
			return http.StatusInternalServerError,
				fmt.Errorf("Could not store gravatar for profile: %+v", err)
		}

		// Attach avatar to profile
		attachment, status, err := AttachAvatar(m.ID, fm)
		if err != nil {
			return status,
				fmt.Errorf("Could not attach avatar to profile: %+v", err)
		}
		m.AvatarIDNullable = sql.NullInt64{
			Int64: attachment.AttachmentID,
			Valid: true,
		}
		filePath := fm.FileHash
		if fm.FileExt != "" {
			filePath += `.` + fm.FileExt
		}
		avatarURL = fmt.Sprintf("%s/%s", h.APITypeFile, filePath)
	}

	// Construct URL to avatar, update profile with Avatar ID and URL
	m.AvatarURLNullable = sql.NullString{
		String: avatarURL,
		Valid:  true,
	}
	status, err = m.Update()
	if err != nil {
		return status,
			fmt.Errorf("Could not update profile with avatar: %+v", err)
	}

	go PurgeCache(h.ItemTypes[h.ItemTypeProfile], m.ID)
	go MarkAllAsRead(m.ID)

	return http.StatusOK, nil
}

// Delete removes a profile from the database
func (m *ProfileType) Delete() (int, error) {
	return http.StatusNotImplemented,
		fmt.Errorf("Delete Profile is not yet implemented")
}

// Update saves the current version of the profile to the database
func (m *ProfileType) Update() (int, error) {

	status, err := m.Validate(true)
	if err != nil {
		return status, err
	}

	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("Could not start transaction: %v", err.Error())
	}
	defer tx.Rollback()

	_, err = tx.Exec(`--Update Profile
UPDATE profiles
   SET profile_name = $2
      ,gender = $3
      ,is_visible = $4
      ,style_id = $5
      ,item_count = $6
      ,comment_count = $7
      ,last_active = $8
      ,avatar_url = $9
      ,avatar_id = $10
 WHERE profile_id = $1`,
		m.ID,
		m.ProfileName,
		m.GenderNullable,
		m.Visible,
		m.StyleID,
		m.ItemCount,
		m.CommentCount,
		m.LastActive,
		m.AvatarURLNullable,
		m.AvatarIDNullable,
	)
	if err != nil {
		tx.Rollback()
		return http.StatusInternalServerError,
			fmt.Errorf("Update of profile failed: %v", err.Error())
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("Transaction failed: %v", err.Error())
	}

	PurgeCache(h.ItemTypes[h.ItemTypeProfile], m.ID)

	return http.StatusOK, nil
}

// UpdateLastActive marks a profile as being active
func UpdateLastActive(profileID int64, lastActive time.Time) (int, error) {

	db, err := h.GetConnection()
	if err != nil {
		glog.Errorf("h.GetConnection() %+v", err)
		return http.StatusInternalServerError,
			fmt.Errorf("Could not get a database connection: %v", err.Error())
	}

	tx, err := db.Begin()
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("Could not start transaction: %v", err.Error())
	}
	defer tx.Rollback()

	_, err = tx.Exec(`--UpdateLastActive
UPDATE profiles
   SET last_active = $2
 WHERE profile_id = $1;`,
		profileID,
		lastActive,
	)
	if err != nil {
		nerr := tx.Rollback()
		if nerr != nil {
			glog.Errorf("Cannot rollback: %+v", nerr)
		}

		return http.StatusInternalServerError,
			fmt.Errorf("Update of last active failed: %v", err.Error())
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("Transaction failed: %v", err.Error())
	}

	PurgeCacheByScope(c.CacheDetail, h.ItemTypes[h.ItemTypeProfile], profileID)

	return http.StatusOK, nil
}

// Patch partially updates a profile
func (m *ProfileType) Patch(
	ac AuthContext,
	patches []h.PatchType,
) (
	int,
	error,
) {
	// Update resource
	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	for _, patch := range patches {
		patch.ScanRawValue()
		switch patch.Path {
		case "/member":
			if !ac.IsSiteOwner {
				return http.StatusUnauthorized, nil
			}

			// Revoke any existing status
			var attributeID int64
			err = tx.QueryRow(`--GetMemberStatus
SELECT k.attribute_id
  FROM attribute_keys k
  JOIN attribute_values v ON v.attribute_id = k.attribute_id
 WHERE k.item_type_id = 3
   AND k.item_id = $1
   AND k.key = 'is_member'`,
				m.ID,
			).Scan(&attributeID)
			if err != nil && err != sql.ErrNoRows {
				return http.StatusInternalServerError, err
			}

			if attributeID != 0 {
				attr, status, err := GetAttribute(attributeID)
				if err != nil {
					return status, err
				}

				status, err = attr.delete(tx)
				if err != nil {
					return status, err
				}
			}

			if patch.Bool.Bool {
				// Grant membership status if necessary
				attr := AttributeType{}
				attr.Key = "is_member"
				attr.Type = tBoolean
				attr.Value = patch.Bool.Bool
				// upsert manages the roles flush
				status, err := attr.upsert(tx, h.ItemTypes[h.ItemTypeProfile], m.ID)
				if err != nil {
					return status, err
				}
			} else {
				status, err := FlushRoleMembersCacheByProfileID(tx, m.ID)
				if err != nil {
					return status,
						fmt.Errorf("Error flushing role members cache: %+v", err)
				}
			}
		default:
			return http.StatusBadRequest,
				fmt.Errorf("Unsupported path in patch replace operation")
		}
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("Transaction failed: %v", err.Error())
	}

	PurgeCache(h.ItemTypes[h.ItemTypeProfile], m.ID)

	return http.StatusOK, nil
}

// IncrementProfileCommentCount increments the comment count
func IncrementProfileCommentCount(profileID int64) {

	db, err := h.GetConnection()
	if err != nil {
		glog.Error(err)
		return
	}

	_, err = db.Exec(`--Update Profile Comment Count
UPDATE profiles
   SET comment_count = comment_count + 1
 WHERE profile_id = $1`,
		profileID,
	)
	if err != nil {
		glog.Error(err)
		return
	}

	PurgeCacheByScope(c.CacheDetail, h.ItemTypes[h.ItemTypeProfile], profileID)
}

// DecrementProfileCommentCount decrements the comment count
func DecrementProfileCommentCount(profileID int64) {

	db, err := h.GetConnection()
	if err != nil {
		glog.Error(err)
		return
	}

	_, err = db.Exec(`--Update Profile Comment Count
UPDATE profiles
   SET comment_count = comment_count - 1
 WHERE profile_id = $1`,
		profileID,
	)
	if err != nil {
		glog.Error(err)
		return
	}

	PurgeCacheByScope(c.CacheDetail, h.ItemTypes[h.ItemTypeProfile], profileID)
}

// UpdateCommentCountForAllProfiles is intended as an import/admin task only.
// It is relatively expensive due to calling is_deleted() for every comment on
// a site.
func UpdateCommentCountForAllProfiles(siteID int64) (int, error) {

	db, err := h.GetConnection()
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("Could not get db connection: %v", err.Error())
	}

	_, err = db.Exec(`-- Reset Profile Counts for All Profiles on Site
UPDATE profiles
   SET comment_count = 0
      ,item_count = 0
 WHERE site_id = $1`,
		siteID,
	)
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("Update of comment count failed: %v", err.Error())
	}

	_, err = db.Exec(`-- Update Comment Counts for All Profiles on Site
UPDATE profiles AS p
   SET comment_count = c.comment_count
  FROM (
 SELECT created_by AS profile_id
       ,COUNT(*) AS comment_count
   FROM flags
  WHERE site_id = $1
    AND item_type_id = 4
    AND microcosm_is_deleted IS NOT TRUE
    AND microcosm_is_moderated IS NOT TRUE
    AND parent_is_deleted IS NOT TRUE
    AND parent_is_moderated IS NOT TRUE
    AND item_is_deleted IS NOT TRUE
    AND item_is_moderated IS NOT TRUE
  GROUP BY created_by
       ) AS c
 WHERE p.profile_id = c.profile_id`,
		siteID,
	)
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("Update of comment count failed: %v", err.Error())
	}

	_, err = db.Exec(`-- Update Item Counts for All Profiles on Site
UPDATE profiles AS p
   SET item_count = c.item_count
  FROM (
 SELECT created_by AS profile_id
       ,COUNT(*) AS item_count
   FROM flags
  WHERE site_id = $1
    AND item_type_id IN (6,9)
    AND microcosm_is_deleted IS NOT TRUE
    AND microcosm_is_moderated IS NOT TRUE
    AND parent_is_deleted IS NOT TRUE
    AND parent_is_moderated IS NOT TRUE
    AND item_is_deleted IS NOT TRUE
    AND item_is_moderated IS NOT TRUE
  GROUP BY created_by
       ) AS c
 WHERE p.profile_id = c.profile_id`,
		siteID,
	)
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("Update of comment count failed: %v", err.Error())
	}

	return http.StatusOK, nil
}

// GetProfileEmail fetches the email address for a profile
func GetProfileEmail(siteID int64, profileID int64) (email string) {
	db, err := h.GetConnection()
	if err != nil {
		glog.Error(err)
		return
	}

	err = db.QueryRow(`--GetProfileEmail
SELECT u.email
  FROM users u
  JOIN profiles p ON p.user_id = u.user_id
 WHERE p.profile_id = $2
   AND p.site_id = $1`,
		siteID,
		profileID,
	).Scan(&email)
	if err != nil {
		glog.Error(err)
		return
	}

	return
}

// GetProfile fetches a single profile
func GetProfile(siteID int64, id int64) (ProfileType, int, error) {

	if id == 0 {
		return ProfileType{}, http.StatusNotFound,
			fmt.Errorf("Profile not found")
	}

	// Get from cache if it's available
	mcKey := fmt.Sprintf(mcProfileKeys[c.CacheDetail], id)
	if val, ok := c.Get(mcKey, ProfileType{}); ok {
		m := val.(ProfileType)

		if m.SiteID != siteID {
			return ProfileType{}, http.StatusNotFound, fmt.Errorf("Not found")
		}
		return m, http.StatusOK, nil
	}

	db, err := h.GetConnection()
	if err != nil {
		glog.Error(err)
		return ProfileType{}, http.StatusInternalServerError, err
	}

	var m ProfileType
	var profileCommentID int64

	err = db.QueryRow(`--GetProfile
SELECT p.profile_id
      ,p.site_id
      ,p.user_id
      ,p.profile_name
      ,p.gender
      ,p.is_visible
      ,si.item_count
      ,p.comment_count
      ,COALESCE(
           (SELECT item_id
              FROM flags
             WHERE item_type_id = 4
               AND parent_item_type_id = 3
               AND parent_item_id = p.profile_id
               AND microcosm_is_deleted IS NOT TRUE
               AND microcosm_is_moderated IS NOT TRUE
               AND parent_is_deleted IS NOT TRUE
               AND parent_is_moderated IS NOT TRUE
               AND item_is_deleted IS NOT TRUE
               AND item_is_moderated IS NOT TRUE
             ORDER BY last_modified DESC
             LIMIT 1),
           0
       ) as profile_comment_id
      ,p.created
      ,p.last_active
      ,p.avatar_url
      ,p.avatar_id
  FROM profiles p,
       (
           SELECT COUNT(*) as item_count
             FROM flags
            WHERE site_id = $1
              AND created_by = $2
              AND item_type_id = 6
              AND item_is_deleted IS NOT TRUE
              AND item_is_moderated IS NOT TRUE
              AND parent_is_deleted IS NOT TRUE
              AND parent_is_moderated IS NOT TRUE
              AND microcosm_is_deleted IS NOT TRUE
              AND microcosm_is_moderated IS NOT TRUE
       ) AS si
 WHERE p.site_id = $1
   AND p.profile_id = $2`,
		siteID,
		id,
	).Scan(
		&m.ID,
		&m.SiteID,
		&m.UserID,
		&m.ProfileName,
		&m.GenderNullable,
		&m.Visible,
		&m.ItemCount,
		&m.CommentCount,
		&profileCommentID,
		&m.Created,
		&m.LastActive,
		&m.AvatarURLNullable,
		&m.AvatarIDNullable,
	)

	if err == sql.ErrNoRows {
		return ProfileType{}, http.StatusNotFound,
			fmt.Errorf("Resource with profile ID %d not found", id)

	} else if err != nil {
		glog.Error(err)
		return ProfileType{}, http.StatusInternalServerError,
			fmt.Errorf("Database query failed: %v", err.Error())
	}

	if m.GenderNullable.Valid {
		m.Gender = m.GenderNullable.String
	}
	if m.AvatarIDNullable.Valid {
		m.AvatarID = m.AvatarIDNullable.Int64
	}
	if m.AvatarURLNullable.Valid {
		m.AvatarURL = m.AvatarURLNullable.String
	}

	if profileCommentID > 0 {
		comment, status, err := GetCommentSummary(siteID, profileCommentID)
		if err != nil {
			glog.Error(err)
			return ProfileType{}, status, err
		}
		m.ProfileComment = comment
	}

	m.Meta.Links =
		[]h.LinkType{
			h.GetLink("self", "", h.ItemTypeProfile, m.ID),
			h.GetLink("site", "", h.ItemTypeSite, m.SiteID),
		}

	// Update cache
	c.Set(mcKey, m, mcTTL)

	return m, http.StatusOK, nil
}

// UpdateUnreadHuddleCount updates the unread huddle count
func (m *ProfileType) UpdateUnreadHuddleCount() {
	UpdateUnreadHuddleCount(m.ID)
}

// UpdateUnreadHuddleCount updates the unread huddle count
func (m *ProfileSummaryType) UpdateUnreadHuddleCount() {
	UpdateUnreadHuddleCount(m.ID)
}

// UpdateUnreadHuddleCount updates the unread huddle count
func UpdateUnreadHuddleCount(profileID int64) {
	tx, err := h.GetTransaction()
	if err != nil {
		glog.Error(err)
		return
	}
	defer tx.Rollback()

	updateUnreadHuddleCount(tx, profileID)

	err = tx.Commit()
	if err != nil {
		glog.Error(err)
		return
	}
}

// updateUnreadHuddleCount updates the unread huddle count
func updateUnreadHuddleCount(tx *sql.Tx, profileID int64) {
	_, err := tx.Exec(`--updateUnreadHuddleCount
UPDATE profiles
   SET unread_huddles = (
           SELECT COUNT(*) OVER() AS total
             FROM flags ff
             JOIN (
                      SELECT hp.huddle_id
                            ,f.last_modified
                        FROM huddle_profiles hp
                             JOIN flags f ON f.item_type_id = 5
                                         AND f.item_id = hp.huddle_id
                        LEFT JOIN read r ON r.profile_id = $1
                                        AND r.item_type_id = 5
                                        AND r.item_id = f.item_id
                        LEFT JOIN read r2 ON r2.profile_id = $1
                                         AND r2.item_type_id = 5
                                         AND r2.item_id = 0
                       WHERE hp.profile_id = $1
                         AND f.last_modified > COALESCE(
                                                   COALESCE(
                                                       r.read,
                                                       r2.read
                                                   ),
                                                   TIMESTAMP WITH TIME ZONE '1970-01-01 12:00:00'
                                               )
                  ) AS h ON ff.parent_item_id = h.huddle_id
                        AND ff.parent_item_type_id = 5
                        AND ff.last_modified >= h.last_modified
             LEFT JOIN ignores i ON i.profile_id = $1
                                AND i.item_type_id = 3
                                AND i.item_id = ff.created_by
            WHERE i.profile_id IS NULL
            GROUP BY h.huddle_id
            LIMIT 1
       )
 WHERE profile_id = $1`,
		profileID,
	)
	if err != nil {
		glog.Error(err)
		return
	}

	PurgeCacheByScope(c.CacheCounts, h.ItemTypes[h.ItemTypeProfile], profileID)
}

// GetUnreadHuddleCount fetches the current unread huddle count
func (m *ProfileType) GetUnreadHuddleCount() (int, error) {

	// Get from cache if it's available
	mcKey := fmt.Sprintf(mcProfileKeys[c.CacheCounts], m.ID)
	if i, ok := c.GetInt64(mcKey); ok {

		m.Meta.Stats = append(
			m.Meta.Stats,
			h.StatType{Metric: "unreadHuddles", Value: i},
		)

		return http.StatusOK, nil
	}

	db, err := h.GetConnection()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	var unreadHuddles int64
	err = db.QueryRow(`--GetUnreadHuddleCount
SELECT unread_huddles
  FROM profiles
 WHERE profile_id = $1`,
		m.ID,
	).Scan(
		&unreadHuddles,
	)
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("Error fetching row: %v", err.Error())
	}

	m.Meta.Stats = append(
		m.Meta.Stats,
		h.StatType{Metric: "unreadHuddles", Value: unreadHuddles},
	)

	c.SetInt64(mcKey, unreadHuddles, mcTTL)

	return http.StatusOK, nil
}

// HandleProfileSummaryRequest is a wrapper to fetch a profile summary
func HandleProfileSummaryRequest(
	siteID int64,
	id int64,
	seq int,
	out chan<- ProfileSummaryRequest,
) {

	item, status, err := GetProfileSummary(siteID, id)

	response := ProfileSummaryRequest{
		Item:   item,
		Status: status,
		Err:    err,
		Seq:    seq,
	}
	out <- response
}

// GetProfileSummary fetches a summary of a profile
func GetProfileSummary(
	siteID int64,
	id int64,
) (
	ProfileSummaryType,
	int,
	error,
) {

	if id == 0 {
		return ProfileSummaryType{}, http.StatusNotFound,
			fmt.Errorf("Profile not found")
	}

	// Get from cache if it's available
	mcKey := fmt.Sprintf(mcProfileKeys[c.CacheSummary], id)
	if val, ok := c.Get(mcKey, ProfileSummaryType{}); ok {
		m := val.(ProfileSummaryType)
		if m.SiteID != siteID {
			return ProfileSummaryType{}, http.StatusNotFound,
				fmt.Errorf("Not found")
		}
		return m, http.StatusOK, nil
	}

	db, err := h.GetConnection()
	if err != nil {
		glog.Error(err)
		return ProfileSummaryType{}, http.StatusInternalServerError, err
	}

	var m ProfileSummaryType
	err = db.QueryRow(`--GetProfileSummary
SELECT profile_id
      ,site_id
      ,user_id
      ,profile_name
      ,is_visible
      ,avatar_url
      ,avatar_id
  FROM profiles
 WHERE site_id = $1
   AND profile_id = $2`,
		siteID,
		id,
	).Scan(
		&m.ID,
		&m.SiteID,
		&m.UserID,
		&m.ProfileName,
		&m.Visible,
		&m.AvatarURLNullable,
		&m.AvatarIDNullable,
	)
	if err == sql.ErrNoRows {
		glog.Warning(err)
		return ProfileSummaryType{}, http.StatusNotFound,
			fmt.Errorf("Resource with profile ID %d not found", id)

	} else if err != nil {
		glog.Error(err)
		return ProfileSummaryType{}, http.StatusInternalServerError,
			fmt.Errorf("Database query failed: %v", err.Error())
	}

	if m.AvatarIDNullable.Valid {
		m.AvatarID = m.AvatarIDNullable.Int64
	}
	if m.AvatarURLNullable.Valid {
		m.AvatarURL = m.AvatarURLNullable.String
	}
	m.Meta.Links =
		[]h.LinkType{
			h.GetLink("self", "", h.ItemTypeProfile, m.ID),
			h.GetLink("site", "", h.ItemTypeSite, m.SiteID),
		}

	// Update cache
	c.Set(mcKey, m, mcTTL)

	return m, http.StatusOK, nil
}

// GetProfileID fetches a profileID given a userID
func GetProfileID(siteID int64, userID int64) (int64, int, error) {

	if siteID == 0 || userID == 0 {
		return 0, http.StatusOK, nil
	}

	// Get from cache if it's available
	//
	// This map of siteId+userId = profileId is never expected to change, so
	// this cache key is unique and does not conform to the cache flushing
	// mechanism
	mcKey := fmt.Sprintf("s%d_u%d", siteID, userID)
	if val, ok := c.GetInt64(mcKey); ok {
		return val, http.StatusOK, nil
	}

	var profileID int64
	db, err := h.GetConnection()
	if err != nil {
		glog.Error(err)
		return profileID, http.StatusInternalServerError, err
	}

	err = db.QueryRow(`--GetProfileId
SELECT profile_id
  FROM profiles
 WHERE site_id = $1
   AND user_id = $2`,
		siteID,
		userID,
	).Scan(
		&profileID,
	)
	if err == sql.ErrNoRows {
		glog.Warning(err)
		return profileID, http.StatusNotFound,
			fmt.Errorf(
				"Profile for site (%d) and user (%d) not found.",
				siteID,
				userID,
			)

	} else if err != nil {
		glog.Error(err)
		return profileID, http.StatusInternalServerError,
			fmt.Errorf("Database query failed: %v", err.Error())
	}

	c.SetInt64(mcKey, profileID, mcTTL)

	return profileID, http.StatusOK, nil
}

// GetOrCreateProfile is called for new logins, to either fetch a profile or
// create a new one
func GetOrCreateProfile(
	site SiteType,
	user UserType,
) (
	ProfileType,
	int,
	error,
) {

	profileID, status, err := GetProfileID(site.ID, user.ID)
	if status == http.StatusOK {
		return GetProfile(site.ID, profileID)
	}
	if err != nil && status != http.StatusNotFound {
		return ProfileType{}, http.StatusInternalServerError,
			fmt.Errorf("Error fetching profile: %v", err.Error())
	}

	// Profile not found, so create one
	p := ProfileType{}
	p.SiteID = site.ID
	p.UserID = user.ID
	// Create randomised username unless the profile is for Site ID 1 (root site)
	if p.SiteID == 1 {
		p.ProfileName = strings.Split(user.Email, "@")[0]
	} else {
		p.ProfileName = SuggestProfileName(user)
	}
	p.Visible = true

	status, err = p.Insert()
	if err != nil {
		glog.Errorf("Creation of profile failed: %+v\n", err)
		return ProfileType{}, status, err
	}
	return p, http.StatusOK, nil
}

// GetProfiles returns a collection of profiles
func GetProfiles(
	siteID int64,
	so ProfileSearchOptions,
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
		glog.Errorf("h.GetConnection() %+v", err)
		return []ProfileSummaryType{}, 0, 0, http.StatusInternalServerError, err
	}

	var following string
	if so.IsFollowing {
		following = `
       JOIN watchers w ON w.profile_id = ` + strconv.FormatInt(so.ProfileID, 10) + `
                      AND w.item_type_id = 3
                      AND p.profile_id = w.item_id`
	}

	var online string
	if so.IsOnline {
		online = `
   AND p.last_active > NOW() - interval '90 minute'`
	}

	var selectCountArgs []interface{}
	var selectArgs []interface{}
	//                                        $1      $2            $3     $4
	selectCountArgs = append(selectCountArgs, siteID, so.ProfileID, limit, offset)
	//                              $1      $2            $3     $4
	selectArgs = append(selectArgs, siteID, so.ProfileID, limit, offset)

	var startsWith string
	var startsWithOrderBy string
	if so.StartsWith != "" {
		//                                        $5
		selectCountArgs = append(selectCountArgs, so.StartsWith+`%`)
		//                              $5                 $6
		selectArgs = append(selectArgs, so.StartsWith+`%`, so.StartsWith)
		startsWith = `
   AND p.profile_name ILIKE $5`
		startsWithOrderBy = `p.profile_name ILIKE $6 DESC
         ,`
	}

	// Construct the query
	sqlSelect := `--GetProfiles
SELECT p.profile_id`

	sqlFromWhere := `
  FROM profiles p
  LEFT JOIN ignores i ON i.profile_id = $2
                     AND i.item_type_id = 3
                     AND i.item_id = p.profile_id` + following + `
 WHERE p.site_id = $1
   AND i.profile_id IS NULL
   AND p.profile_name <> 'deleted'` + online + startsWith

	var sqlOrderLimit string
	if so.OrderByCommentCount {
		sqlOrderLimit = `
 ORDER BY ` + startsWithOrderBy + `p.comment_count DESC
         ,p.profile_name ASC
 LIMIT $3
OFFSET $4`
	} else {
		sqlOrderLimit = `
 ORDER BY ` + startsWithOrderBy + `p.profile_name ASC
 LIMIT $3
OFFSET $4`
	}

	var total int64
	err = db.QueryRow(
		`SELECT COUNT(*)`+sqlFromWhere+`
   AND $3 > 0
   AND $4 >= 0`,
		selectCountArgs...,
	).Scan(
		&total,
	)
	if err != nil {
		glog.Error(err)
		return []ProfileSummaryType{}, 0, 0, http.StatusInternalServerError,
			fmt.Errorf("Database query failed")
	}

	rows, err := db.Query(
		sqlSelect+sqlFromWhere+sqlOrderLimit,
		selectArgs...,
	)
	if err != nil {
		glog.Errorf(
			"stmt.Query(%d, `%s`, %d, %d) %+v",
			siteID,
			so.StartsWith+`%`,
			limit,
			offset,
			err,
		)
		return []ProfileSummaryType{}, 0, 0, http.StatusInternalServerError,
			fmt.Errorf("Database query failed")
	}
	defer rows.Close()

	ids := []int64{}

	for rows.Next() {
		var id int64
		err = rows.Scan(&id)
		if err != nil {
			glog.Errorf("rows.Scan() %+v", err)
			return []ProfileSummaryType{}, 0, 0, http.StatusInternalServerError,
				fmt.Errorf("Row parsing error")
		}

		ids = append(ids, id)
	}
	err = rows.Err()
	if err != nil {
		glog.Errorf("rows.Err() %+v", err)
		return []ProfileSummaryType{}, 0, 0, http.StatusInternalServerError,
			fmt.Errorf("Error fetching rows")
	}
	rows.Close()

	var wg1 sync.WaitGroup
	req := make(chan ProfileSummaryRequest)
	defer close(req)

	for seq, id := range ids {
		go HandleProfileSummaryRequest(siteID, id, seq, req)
		wg1.Add(1)
	}

	resps := []ProfileSummaryRequest{}
	for i := 0; i < len(ids); i++ {
		resp := <-req
		wg1.Done()
		resps = append(resps, resp)
	}
	wg1.Wait()

	for _, resp := range resps {
		if resp.Err != nil {
			glog.Errorf("resp.Err != nil %+v", resp.Err)
			return []ProfileSummaryType{}, 0, 0, resp.Status, resp.Err
		}
	}

	sort.Sort(ProfileSummaryRequestBySeq(resps))

	ems := []ProfileSummaryType{}
	for _, resp := range resps {
		ems = append(ems, resp.Item)
	}

	pages := h.GetPageCount(total, limit)
	maxOffset := h.GetMaxOffset(total, limit)

	if offset > maxOffset {
		glog.Infoln("offset > maxOffset")
		return []ProfileSummaryType{}, 0, 0, http.StatusBadRequest,
			fmt.Errorf("not enough records, "+
				"offset (%d) would return an empty page.", offset)
	}

	return ems, total, pages, http.StatusOK, nil
}

// MakeGravatarURL hashes the email and creates the Gravatar URL
func MakeGravatarURL(email string) string {
	return fmt.Sprintf(
		"%s%s?d=identicon",
		URLGravatar,
		h.MD5Sum(strings.ToLower(strings.Trim(email, " "))),
	)
}

// StoreGravatar stores the gravatar file in the database
func StoreGravatar(gravatarURL string) (FileMetadataType, int, error) {

	// TODO(matt): reduce duplication with models.FileController
	resp, err := http.Get(gravatarURL)

	if err != nil {
		glog.Errorf("http.Get(`%s`) %+v", gravatarURL, err)
		return FileMetadataType{}, http.StatusInternalServerError,
			fmt.Errorf("Could not retrieve gravatar")
	}
	defer resp.Body.Close()

	fileContent, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		glog.Errorf("ioutil.ReadAll(resp.Body) %+v", err)
		return FileMetadataType{}, http.StatusInternalServerError,
			fmt.Errorf("Could not read gravatar response")
	}

	metadata := FileMetadataType{}
	metadata.Content = fileContent
	metadata.FileSize = int32(len(metadata.Content))
	metadata.FileHash, err = h.SHA1(metadata.Content)
	if err != nil {
		glog.Errorf("h.Sha1(metadata.Content) %+v", err)
		return FileMetadataType{}, http.StatusInternalServerError,
			fmt.Errorf("Could not generate file SHA-1")
	}
	metadata.MimeType = resp.Header.Get("Content-Type")
	metadata.Created = time.Now()
	metadata.AttachCount++

	status, err := metadata.Insert(AvatarMaxWidth, AvatarMaxHeight)
	if err != nil {
		glog.Errorf("metadata.Insert(%d, %d) %+v", AvatarMaxWidth, AvatarMaxHeight, err)
		return FileMetadataType{}, status,
			fmt.Errorf("Could not insert gravatar file metadata")
	}

	return metadata, http.StatusOK, nil
}

// AttachAvatar allows file uploading of a custom avatar
func AttachAvatar(
	profileID int64,
	fileMetadata FileMetadataType,
) (
	AttachmentType,
	int,
	error,
) {

	attachment := AttachmentType{}
	attachment.AttachmentMetaID = fileMetadata.AttachmentMetaID
	attachment.FileHash = fileMetadata.FileHash
	attachment.Created = time.Now()
	attachment.ItemTypeID = h.ItemTypes[h.ItemTypeProfile]
	attachment.ItemID = profileID
	attachment.ProfileID = profileID

	_, err := attachment.Insert()
	if err != nil {
		return AttachmentType{}, http.StatusInternalServerError,
			fmt.Errorf(
				"Could not create avatar attachment to profile: %+v",
				err,
			)
	}

	return attachment, http.StatusOK, nil
}

// SuggestProfileName will create a semi-random username for a new user
func SuggestProfileName(user UserType) string {
	// TODO(buro9): Change this to just have a global blacklist of user names
	//   i.e. root, admin, god, moderator
	// The old method reserved usernames to email addresses and is no longer
	// needed
	if _, inMap := reservedProfileNames[user.Email]; inMap {
		return reservedProfileNames[user.Email]
	}

	// TODO(buro9): This is not duplication safe, and we will need to do a
	// multiple pass generation thing eventually.
	return "user" + strconv.FormatInt(user.ID+5830, 10)
}

// IsProfileNameTaken checks whether a profile name is taken for a given site,
// If the profile name is taken by the user specified then it's considered
// to be available (as in... updating your own profile won't fail this check)
// Errors in this method will return "true" for the check as data integrity
// is everything
func IsProfileNameTaken(
	siteID int64,
	userID int64,
	profileName string,
) (
	bool,
	int,
	error,
) {

	profileName = strings.ToLower(profileName)

	db, err := h.GetConnection()
	if err != nil {
		return true, http.StatusInternalServerError, err
	}

	rows, err := db.Query(`
SELECT u.email
      ,p.exists
  FROM users u
      ,(
        SELECT NOT COUNT(*) = 0 AS exists
          FROM profiles
         WHERE site_id = $1
           AND LOWER(profile_name) = $3
           AND user_id != $2
       ) AS p
 WHERE u.user_id = $2`,
		siteID,
		userID,
		profileName,
	)
	if err != nil {
		return true, http.StatusInternalServerError,
			fmt.Errorf("Database query failed: %v", err.Error())
	}
	defer rows.Close()

	var (
		email  string
		exists bool
	)
	for rows.Next() {
		err = rows.Scan(
			&email,
			&exists,
		)
		if err != nil {
			return true, http.StatusInternalServerError,
				fmt.Errorf("Row parsing error: %v", err.Error())
		}
	}
	err = rows.Err()
	if err != nil {
		return true, http.StatusInternalServerError,
			fmt.Errorf("Error fetching rows: %v", err.Error())
	}
	rows.Close()

	// Is it already in the database?
	if exists {
		return true, http.StatusOK, nil
	}

	// Is it in the reserved list, but not for the given email?
	for e, n := range reservedProfileNames {
		if strings.ToLower(n) == profileName && email != e {
			return true, http.StatusOK, nil
		}
	}

	return false, http.StatusOK, nil
}

// GetProfileSearchOptions fetches the options in the querystring that are being
// used to filter and search for profiles
func GetProfileSearchOptions(query url.Values) ProfileSearchOptions {

	so := ProfileSearchOptions{}

	if query.Get("top") != "" {
		inTop, err := strconv.ParseBool(query.Get("top"))
		if err == nil {
			so.OrderByCommentCount = inTop
		}
	}

	if query.Get("q") != "" {
		startsWith := strings.TrimLeft(query.Get("q"), "+@")
		if startsWith != "" {
			so.StartsWith = startsWith
		}
	}

	if query.Get("following") != "" {
		inFollowing, err := strconv.ParseBool(query.Get("following"))
		if err == nil {
			so.IsFollowing = inFollowing
		}
	}

	if query.Get("online") != "" {
		inFollowing, err := strconv.ParseBool(query.Get("online"))
		if err == nil {
			so.IsOnline = inFollowing
		}
	}

	return so
}

// Allows you to define a list of profile names that are reserved.
// i.e. var reservedProfileNames = map[string]string{
//    "someone@example.com": "someone",
// }
// That would result in the username 'someone' only being available to the
// person whose email address is 'someone@example.com'. This applies across
// all sites, and can be used to prohibit certain profile names from being
// used at all, i.e. misleading names like God, Admin, or root, or names that
// are profane and would harm the community standards.
var reservedProfileNames = map[string]string{}
