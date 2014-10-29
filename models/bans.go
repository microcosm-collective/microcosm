package models

import (
	"database/sql"
	"fmt"

	c "github.com/microcosm-cc/microcosm/cache"
	h "github.com/microcosm-cc/microcosm/helpers"
)

const banCacheKey = `ban_s%d_u%d`

func IsBanned(siteId int64, userId int64) bool {

	if siteId == 0 || userId == 0 {
		return false
	}

	// Get from cache if it's available
	//
	// This map of siteId+userId = profileId is never expected to change, so
	// this cache key is unique and does not conform to the cache flushing
	// mechanism
	mcKey := fmt.Sprintf(banCacheKey, siteId, userId)
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
		siteId,
		userId,
	).Scan(
		&isBanned,
	)
	if err == sql.ErrNoRows {
		return false
	} else if err != nil {
		return false
	}

	c.CacheSetBool(mcKey, isBanned, mcTtl)

	return isBanned
}

// TODO: Add a BanUser() func
