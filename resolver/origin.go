package resolver

import (
	"database/sql"
	"fmt"

	"github.com/golang/glog"

	"github.com/microcosm-cc/microcosm/cache"
	h "github.com/microcosm-cc/microcosm/helpers"
)

const ttl int32 = (60 * 60 * 24 * 365) / 12 // 1 Month

type Origin struct {
	OriginID int64
	SiteID   int64
	Product  string
}

func getOrigin(siteID int64) *Origin {

	key := fmt.Sprintf("site_origin_%d", siteID)

	if val, ok := cache.CacheGet(key, Origin{}); ok {
		origin := val.(Origin)
		return &origin
	}

	origin := Origin{SiteID: siteID}

	db, err := h.GetConnection()
	if err != nil {
		glog.Error(err)
		return nil
	}

	err = db.QueryRow(`
SELECT origin_id
      ,product
  FROM import_origins
 WHERE site_id = $1`,
		siteID,
	).Scan(
		&origin.OriginID,
		&origin.Product,
	)
	if err != nil {
		if err != sql.ErrNoRows {
			glog.Error(err)
		}
		return nil
	}

	cache.CacheSet(key, origin, ttl)

	return &origin
}

func getNewID(originID int64, itemTypeID int64, oldID int64) int64 {

	db, err := h.GetConnection()
	if err != nil {
		glog.Error(err)
		return 0
	}

	var newID int64
	err = db.QueryRow(`
SELECT item_id
 FROM imported_items
WHERE origin_id = $1
  AND item_type_id = $2
  AND old_id::bigint = $3`,
		originID,
		itemTypeID,
		oldID,
	).Scan(
		&newID,
	)
	if err != nil {
		if err != sql.ErrNoRows {
			glog.Error(err)
		}
		return 0
	}

	return newID
}
