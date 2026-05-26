package models

import (
	"database/sql"
	"fmt"
	"time"

	c "github.com/microcosm-collective/microcosm/cache"
	h "github.com/microcosm-collective/microcosm/helpers"
)

const banCacheKey = `ban_s%d_u%d`
const banNotFoundCacheTTL int32 = 60

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

// TODO: Add a BanUser() func

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
