package models

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	c "github.com/microcosm-collective/microcosm/cache"
	h "github.com/microcosm-collective/microcosm/helpers"
)

const banCacheKey = `ban_s%d_u%d`
const banNotFoundCacheTTL int32 = 60

// BansType is a collection of bans
type BansType struct {
	Bans h.ArrayType    `json:"bans"`
	Meta h.CoreMetaType `json:"meta"`
}

// BanType describes a site ban for a user
type BanType struct {
	ID            int64      `json:"id"`
	SiteID        int64      `json:"siteId,omitempty"`
	UserID        int64      `json:"userId"`
	Created       time.Time  `json:"created"`
	Expires       *time.Time `json:"expires,omitempty"`
	DisplayReason string     `json:"displayReason,omitempty"`
	AdminReason   string     `json:"adminReason,omitempty"`

	Meta h.CoreMetaType `json:"meta"`
}

// GetLink returns the link to this ban
func (m *BanType) GetLink() string {
	return fmt.Sprintf("%s/%d", h.APITypeBan, m.ID)
}

// Validate returns true if the ban is valid
func (m *BanType) Validate(siteID int64, exists bool) (int, error) {
	if exists && m.ID < 1 {
		return http.StatusBadRequest, fmt.Errorf("invalid ban ID")
	}

	if m.UserID < 1 {
		return http.StatusBadRequest, fmt.Errorf("userId must be supplied")
	}

	if !UserIsOnSite(m.UserID, siteID) {
		return http.StatusBadRequest, fmt.Errorf("user is not a member of this site")
	}

	if UserOwnsSite(m.UserID, siteID) {
		return http.StatusForbidden, fmt.Errorf("the site owner cannot be banned")
	}

	if !exists && m.Expires != nil && !m.Expires.After(time.Now()) {
		return http.StatusBadRequest, fmt.Errorf("expires must be in the future")
	}

	m.DisplayReason = CleanBlockText(strings.TrimSpace(m.DisplayReason))
	m.AdminReason = CleanBlockText(strings.TrimSpace(m.AdminReason))

	return http.StatusOK, nil
}

// Insert saves a ban to the database
func (m *BanType) Insert(siteID int64) (int, error) {
	status, err := m.Validate(siteID, false)
	if err != nil {
		return status, err
	}

	exists, status, err := ActiveBanExists(siteID, m.UserID)
	if err != nil {
		return status, err
	}
	if exists {
		return http.StatusConflict, fmt.Errorf("user already has an active ban on this site")
	}

	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("could not start transaction: %v", err.Error())
	}
	defer tx.Rollback()

	var insertID int64
	err = tx.QueryRow(`--InsertBan
INSERT INTO bans (
    site_id, user_id, created, expires, display_reason, admin_reason
) VALUES (
    $1, $2, NOW(), $3, $4, $5
) RETURNING ban_id, created`,
		siteID,
		m.UserID,
		m.Expires,
		m.DisplayReason,
		m.AdminReason,
	).Scan(
		&insertID,
		&m.Created,
	)
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("error inserting data and returning ID: %+v", err)
	}
	m.ID = insertID
	m.SiteID = siteID

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("transaction failed: %v", err.Error())
	}

	c.Delete(fmt.Sprintf(banCacheKey, siteID, m.UserID))

	return http.StatusOK, nil
}

// Update saves a ban to the database
func (m *BanType) Update(siteID int64) (int, error) {
	status, err := m.Validate(siteID, true)
	if err != nil {
		return status, err
	}

	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("could not start transaction: %v", err.Error())
	}
	defer tx.Rollback()

	_, err = tx.Exec(`--UpdateBan
UPDATE bans
   SET expires = $3
      ,display_reason = $4
      ,admin_reason = $5
 WHERE ban_id = $1
   AND site_id = $2`,
		m.ID,
		siteID,
		m.Expires,
		m.DisplayReason,
		m.AdminReason,
	)
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("update failed: %v", err.Error())
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("transaction failed: %v", err.Error())
	}

	c.Delete(fmt.Sprintf(banCacheKey, siteID, m.UserID))

	return http.StatusOK, nil
}

// Delete removes a ban from the database
func (m *BanType) Delete(siteID int64) (int, error) {
	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("could not start transaction: %v", err.Error())
	}
	defer tx.Rollback()

	_, err = tx.Exec(`--DeleteBan
DELETE FROM bans
 WHERE ban_id = $1
   AND site_id = $2`,
		m.ID,
		siteID,
	)
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("delete failed: %v", err.Error())
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("transaction failed: %v", err.Error())
	}

	c.Delete(fmt.Sprintf(banCacheKey, siteID, m.UserID))

	return http.StatusOK, nil
}

// GetBan fetches a ban by ID
func GetBan(siteID int64, id int64) (BanType, int, error) {
	if id < 1 {
		return BanType{}, http.StatusBadRequest, fmt.Errorf("invalid ban ID")
	}

	db, err := h.GetConnection()
	if err != nil {
		return BanType{}, http.StatusInternalServerError,
			fmt.Errorf("connection failed: %v", err.Error())
	}

	var (
		m             BanType
		expires       sql.NullTime
		displayReason sql.NullString
		adminReason   sql.NullString
	)

	err = db.QueryRow(`--GetBan
SELECT ban_id
      ,site_id
      ,user_id
      ,created
      ,expires
      ,display_reason
      ,admin_reason
  FROM bans
 WHERE site_id = $1
   AND ban_id = $2`,
		siteID,
		id,
	).Scan(
		&m.ID,
		&m.SiteID,
		&m.UserID,
		&m.Created,
		&expires,
		&displayReason,
		&adminReason,
	)
	if err == sql.ErrNoRows {
		return BanType{}, http.StatusNotFound, fmt.Errorf("ban not found")
	} else if err != nil {
		return BanType{}, http.StatusInternalServerError,
			fmt.Errorf("database query failed: %v", err.Error())
	}

	hydrateBan(&m, expires, displayReason, adminReason)

	return m, http.StatusOK, nil
}

// GetBans fetches bans for a site
func GetBans(siteID int64, limit int64, offset int64) ([]BanType, int64, int64, int, error) {
	db, err := h.GetConnection()
	if err != nil {
		return []BanType{}, 0, 0, http.StatusInternalServerError,
			fmt.Errorf("connection failed: %v", err.Error())
	}

	rows, err := db.Query(`--GetBans
SELECT COUNT(*) OVER() AS total
      ,ban_id
      ,site_id
      ,user_id
      ,created
      ,expires
      ,display_reason
      ,admin_reason
  FROM bans
 WHERE site_id = $1
 ORDER BY created DESC, ban_id DESC
 LIMIT $2
OFFSET $3`,
		siteID,
		limit,
		offset,
	)
	if err != nil {
		return []BanType{}, 0, 0, http.StatusInternalServerError,
			fmt.Errorf("database query failed: %v", err.Error())
	}
	defer rows.Close()

	var (
		ems   []BanType
		total int64
	)
	for rows.Next() {
		var (
			m             BanType
			expires       sql.NullTime
			displayReason sql.NullString
			adminReason   sql.NullString
		)

		err = rows.Scan(
			&total,
			&m.ID,
			&m.SiteID,
			&m.UserID,
			&m.Created,
			&expires,
			&displayReason,
			&adminReason,
		)
		if err != nil {
			return []BanType{}, 0, 0, http.StatusInternalServerError,
				fmt.Errorf("row parsing error: %v", err.Error())
		}

		hydrateBan(&m, expires, displayReason, adminReason)
		ems = append(ems, m)
	}
	err = rows.Err()
	if err != nil {
		return []BanType{}, 0, 0, http.StatusInternalServerError,
			fmt.Errorf("error fetching rows: %v", err.Error())
	}
	rows.Close()

	pages := h.GetPageCount(total, limit)
	maxOffset := h.GetMaxOffset(total, limit)

	if offset > maxOffset {
		return []BanType{}, 0, 0, http.StatusBadRequest,
			fmt.Errorf("not enough records, "+
				"offset (%d) would return an empty page", offset)
	}

	return ems, total, pages, http.StatusOK, nil
}

// ActiveBanExists returns true if the user has an active site ban
func ActiveBanExists(siteID int64, userID int64) (bool, int, error) {
	db, err := h.GetConnection()
	if err != nil {
		return false, http.StatusInternalServerError,
			fmt.Errorf("connection failed: %v", err.Error())
	}

	var exists bool
	err = db.QueryRow(`--ActiveBanExists
SELECT EXISTS(
SELECT 1
  FROM bans
 WHERE site_id = $1
   AND user_id = $2
   AND (expires IS NULL OR expires > NOW())
)`,
		siteID,
		userID,
	).Scan(
		&exists,
	)
	if err != nil {
		return false, http.StatusInternalServerError,
			fmt.Errorf("database query failed: %v", err.Error())
	}

	return exists, http.StatusOK, nil
}

// UserOwnsSite returns true if the user owns the given site
func UserOwnsSite(userID int64, siteID int64) bool {
	db, err := h.GetConnection()
	if err != nil {
		return false
	}

	var val bool
	err = db.QueryRow(`--UserOwnsSite
SELECT COUNT(*) > 0
  FROM sites s
  JOIN profiles p ON p.profile_id = s.owned_by
 WHERE s.site_id = $1
   AND p.user_id = $2`,
		siteID,
		userID,
	).Scan(&val)
	if err != nil {
		return false
	}

	return val
}

// IsBanned returns true if the user is banned for the given site
func IsBanned(siteID int64, userID int64) bool {

	if siteID == 0 || userID == 0 {
		return false
	}

	// Get from cache if it's available
	//
	// Active temporary bans are cached only until their expiry time.
	mcKey := fmt.Sprintf(banCacheKey, siteID, userID)
	if val, ok := c.GetBool(mcKey); ok {
		return val
	}

	var expires sql.NullTime
	db, err := h.GetConnection()
	if err != nil {
		return false
	}

	err = db.QueryRow(`--IsBanned
SELECT expires
  FROM bans
 WHERE site_id = $1
   AND user_id = $2
   AND (expires IS NULL OR expires > NOW())
 ORDER BY expires NULLS LAST
 LIMIT 1`,
		siteID,
		userID,
	).Scan(
		&expires,
	)
	if err == sql.ErrNoRows {
		c.SetBool(mcKey, false, banNotFoundCacheTTL)
		return false
	} else if err != nil {
		return false
	}

	c.SetBool(mcKey, true, activeBanCacheTTL(expires))

	return true
}

func activeBanCacheTTL(expires sql.NullTime) int32 {
	if !expires.Valid {
		return mcTTL
	}

	until := time.Until(expires.Time)
	if until < time.Second {
		return 1
	}

	maxTTL := time.Duration(mcTTL) * time.Second
	if until > maxTTL {
		return mcTTL
	}

	return int32(until / time.Second)
}

func hydrateBan(
	m *BanType,
	expires sql.NullTime,
	displayReason sql.NullString,
	adminReason sql.NullString,
) {
	if expires.Valid {
		t := expires.Time
		m.Expires = &t
	}

	if displayReason.Valid {
		m.DisplayReason = displayReason.String
	}

	if adminReason.Valid {
		m.AdminReason = adminReason.String
	}

	m.Meta.Links = []h.LinkType{
		{Rel: "self", Href: m.GetLink()},
	}
}
