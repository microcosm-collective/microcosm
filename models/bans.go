package models

import (
	"database/sql"
	"fmt"

	c "github.com/microcosm-cc/microcosm/cache"
	h "github.com/microcosm-cc/microcosm/helpers"
)

const banCacheKey = `ban_s%d_u%d`

// IsBanned returns true if the user is banned for the given site
func IsBanned(siteID int64, userID int64) bool {

	if siteID == 0 || userID == 0 {
		return false
	}

	// Get from cache if it's available
	//
	// This map of siteID+userID = profileId is never expected to change, so
	// this cache key is unique and does not conform to the cache flushing
	// mechanism
	mcKey := fmt.Sprintf(banCacheKey, siteID, userID)
	if val, ok := c.CacheGetBool(mcKey); ok {
		return val
	}

	var isBanned bool
	db, err := h.GetConnection()
	if err != nil {
		return false
	}

	err = db.QueryRow(`--IsBanned
SELECT EXISTS(
SELECT 1
  FROM bans
 WHERE site_id = $1
   AND user_id = $2
)`,
		siteID,
		userID,
	).Scan(
		&isBanned,
	)
	if err == sql.ErrNoRows {
		return false
	} else if err != nil {
		return false
	}

	c.CacheSetBool(mcKey, isBanned, mcTTL)

	return isBanned
}

// TODO: Add a BanUser() func
