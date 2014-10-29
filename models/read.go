package models

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/golang/glog"
	"github.com/lib/pq"

	h "github.com/microcosm-cc/microcosm/helpers"
)

type ReadType struct {
	Id         int64
	ProfileId  int64
	ItemTypeId int64
	ItemId     int64
	Read       time.Time
}

type ReadScopeType struct {
	ItemId     int64  `json:"itemId"`
	ItemType   string `json:"itemType"`
	ItemTypeId int64
}

func GetLastReadTime(
	itemTypeId int64,
	itemId int64,
	profileId int64,
) (
	time.Time,
	int,
	error,
) {

	var lastRead time.Time

	if profileId == 0 {
		return lastRead, http.StatusOK, nil
	}

	db, err := h.GetConnection()
	if err != nil {
		glog.Errorf("h.GetConnection() %+v", err)
		return lastRead, http.StatusInternalServerError, err
	}

	rows, err := db.Query(`
SELECT MAX(read)
  FROM read
 WHERE (
           (item_type_id = $1 AND item_id = $2) 
        OR (item_type_id = 2 AND item_id = (
             SELECT microcosm_id 
               FROM flags
              WHERE item_type_id = $1
                AND item_id = $2
               )
           )
       )
   AND profile_id = $3`,
		itemTypeId,
		itemId,
		profileId,
	)
	if err != nil {
		glog.Errorf(
			"db.Query(%d, %d, %d) %+v",
			itemTypeId,
			itemId,
			profileId,
			err,
		)
		return lastRead, http.StatusInternalServerError,
			errors.New("Database query failed")
	}
	defer rows.Close()

	var read pq.NullTime
	for rows.Next() {
		err = rows.Scan(&read)
		if err != nil {
			glog.Errorf("rows.Scan() %+v", err)
			return lastRead, http.StatusInternalServerError,
				errors.New("Row parsing error")
		}
	}
	err = rows.Err()
	if err != nil {
		glog.Errorf("rows.Err() %+v", err)
		return lastRead, http.StatusInternalServerError,
			errors.New("Error fetching rows")
	}
	rows.Close()

	if read.Valid {
		lastRead = read.Time
	}

	return lastRead, http.StatusOK, nil
}

// Used by the importer
func MarkAllHuddlesForAllProfilesAsReadOnSite(siteId int64) error {
	tx, err := h.GetTransaction()
	if err != nil {
		glog.Error(err)
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
DELETE FROM read
 WHERE read_id IN (
SELECT r.read_id
  FROM read r
       JOIN profiles p ON p.profile_id = r.profile_id
 WHERE p.site_id = $1
   AND r.item_type_id = 5
)`,
		siteId,
	)
	if err != nil {
		glog.Error(err)
		return err
	}

	_, err = tx.Exec(`
INSERT INTO read (
       item_type_id
      ,item_id
      ,profile_id
      ,read
)
SELECT 5 AS item_type_id
      ,0 AS item_id
      ,p.profile_id
      ,NOW() AS read
  FROM profiles p
 WHERE site_id = $1`,
		siteId,
	)
	if err != nil {
		glog.Error(err)
		return err
	}

	_, err = tx.Exec(`
UPDATE profiles
   SET unread_huddles = 0
 WHERE site_id = $1`,
		siteId,
	)
	if err != nil {
		glog.Error(err)
		return err
	}

	err = tx.Commit()
	if err != nil {
		glog.Error(err)
		return err
	}

	return nil
}

// Used by the importer
func MarkAllMicrocosmsForAllProfilesAsReadOnSite(siteId int64) error {
	tx, err := h.GetTransaction()
	if err != nil {
		glog.Error(err)
		return err
	}
	defer tx.Rollback()

	// Delete all conversation, poll, event markers
	_, err = tx.Exec(`
DELETE
  FROM read r
 WHERE read_id IN (
           SELECT r.read_id
             FROM read r
             JOIN profiles p ON p.site_id = $1
                            AND p.profile_id = r.profile_id
            WHERE item_type_id IN (2,6,7,9)
       )`,
		siteId,
	)
	if err != nil {
		glog.Error(err)
		return err
	}

	_, err = tx.Exec(`
INSERT INTO read (
       item_type_id
      ,item_id
      ,profile_id
      ,read
)
SELECT 2 AS item_type_id
      ,m.microcosm_id AS item_id
      ,p.profile_id
      ,NOW() AS read
  FROM profiles p
  JOIN microcosms m ON m.site_id = p.site_id
 WHERE p.site_id = $1`,
		siteId,
	)
	if err != nil {
		glog.Error(err)
		return err
	}

	err = tx.Commit()
	if err != nil {
		glog.Error(err)
		return err
	}

	return nil
}

func MarkAsRead(
	itemTypeId int64,
	itemId int64,
	profileId int64,
	updateTime time.Time,
) (
	int,
	error,
) {

	// Some validation
	if profileId == 0 {
		glog.Infof("profileId == 0")
		return http.StatusOK, nil
	}

	if (itemTypeId != h.ItemTypes[h.ItemTypeUpdate] && itemId == 0) ||
		itemTypeId == 0 {

		glog.Errorln(
			"(itemTypeId != h.ItemTypes[h.ItemTypeUpdate] && itemId == 0) || " +
				"itemTypeId == 0",
		)
		return http.StatusExpectationFailed,
			errors.New("itemTypeId and/or itemId was null when MarkAsRead " +
				"was called. This is illogical, you need to tell us what " +
				"is being marked as read.")
	}

	if updateTime.IsZero() {
		glog.Errorln("updateTime.IsZero()")
		return http.StatusExpectationFailed,
			errors.New("MarkAsRead has been called but the time supplied " +
				"is null. You need to tell us when the item was read.")
	}

	// Do the deed
	tx, err := h.GetTransaction()
	if err != nil {
		glog.Errorf("h.GetTransaction() %+v", err)
		return http.StatusInternalServerError,
			errors.New("Could not start transaction")
	}
	defer tx.Rollback()

	m := ReadType{
		ItemTypeId: itemTypeId,
		ItemId:     itemId,
		ProfileId:  profileId,
		Read:       updateTime,
	}
	status, err := m.upsert(tx)
	if err != nil {
		glog.Errorf("m.upsert(tx) %+v", err)
		return status, err
	}

	switch itemTypeId {
	case h.ItemTypes[h.ItemTypeSite]:
		// Site has been marked read, which means all Microcosms and items are
		// implicitly read. So we should delete those records, but not the rows
		// referring to huddles or the site itself.

		_, err = tx.Exec(`
DELETE FROM read
 WHERE item_type_id NOT IN (1, 5) -- 1 = site, 5 = huddle
   AND profile_id = $1`,
			m.ProfileId,
		)
		if err != nil {
			glog.Errorf("tx.Exec(%d) %+v", m.ProfileId, err)
			return http.StatusInternalServerError,
				errors.New("Deletion of read items failed")
		}
	case h.ItemTypes[h.ItemTypeMicrocosm]:
		// Microcosm has been marked read, so delete all item level rows that
		// belong to this Microcosm as those are implicitly read as of now
		_, err = tx.Exec(`
DELETE FROM read
 WHERE read_id IN (
           SELECT read_id 
            FROM (
                     SELECT item_id
                           ,item_type_id 
                       FROM flags
                      WHERE microcosm_id = $1
                        AND parent_item_id IS NULL
                 ) i
            LEFT JOIN read
              ON read.item_id = i.item_id
             AND read.item_type_id = i.item_type_id
       )
   AND profile_id = $2`,
			m.ItemId,
			m.ProfileId,
		)
		if err != nil {
			glog.Errorf("tx.Exec(%d, %d) %+v", m.ItemId, m.ProfileId, err)
			return http.StatusInternalServerError,
				errors.New("Deletion of read items failed")
		}
	case h.ItemTypes[h.ItemTypeHuddle]:
		if itemId == 0 {
			// All huddles have been marked read, so we should delete the
			// individual records for older microcosms *and* set the unread
			// huddle count as 0 on the profile.
			_, err = tx.Exec(`
DELETE FROM read
 WHERE item_type_id = 5 -- 5 = huddle
   AND item_id <> 0
   AND profile_id = $1`,
				m.ProfileId,
			)
			if err != nil {
				glog.Errorf("tx.Exec(%d) %+v", m.ProfileId, err)
				return http.StatusInternalServerError,
					errors.New("Deletion of read items failed")
			}

			updateUnreadHuddleCount(tx, profileId)
		}
	}

	err = tx.Commit()
	if err != nil {
		glog.Errorf("tx.Commit() %+v", err)
		return http.StatusInternalServerError,
			errors.New("Transaction failed")
	}

	return http.StatusOK, nil
}

func (m *ReadType) upsert(tx *sql.Tx) (int, error) {

	res, err := tx.Exec(`
UPDATE read
   SET read = GREATEST(read, $4)
 WHERE item_type_id = $1
   AND item_id = $2
   AND profile_id = $3`,
		m.ItemTypeId,
		m.ItemId,
		m.ProfileId,
		m.Read,
	)
	if err != nil {
		glog.Errorf(
			"tx.Exec(%d, %d, %d, %v) %+v",
			m.ItemTypeId,
			m.ItemId,
			m.ProfileId,
			m.Read,
			err,
		)
		return http.StatusInternalServerError,
			errors.New("Error executing update")
	}

	// If update did not create any rows then we need to insert
	if rowsAffected, _ := res.RowsAffected(); rowsAffected == 0 {

		// Note that even though we just proved that it didn't exist, we could
		// have multiple transactions doing this and we'are playing safe and
		// defensively by doing a NOT EXISTS check, this shouldn't fail at all.
		_, err = tx.Exec(`
INSERT INTO read
    (item_type_id, item_id, profile_id, read)
SELECT $1, $2, $3, $4
 WHERE NOT EXISTS (
           SELECT read_id
             FROM read
            WHERE item_type_id = $1
              AND item_id = $2
              AND profile_id = $3
       )`,
			m.ItemTypeId,
			m.ItemId,
			m.ProfileId,
			m.Read,
		)
		if err != nil {
			glog.Errorf(
				"tx.Exec(%d, %d, %d, %v) %+v",
				m.ItemTypeId,
				m.ItemId,
				m.ProfileId,
				m.Read,
				err,
			)
			tx.Rollback()
			return http.StatusInternalServerError,
				errors.New("Error inserting data")
		}
	}

	return http.StatusOK, nil
}

func MarkScopeAsRead(profileId int64, rs ReadScopeType) (int, error) {
	if rs.ItemTypeId == h.ItemTypes[h.ItemTypeSite] {
		return MarkAllAsRead(profileId)

	} else if rs.ItemTypeId == h.ItemTypes[h.ItemTypeHuddle] && rs.ItemId >= 0 {
		// It's fine to mark huddleId = 0 as read, this is equivalent to marking
		// all huddles as read
		return MarkAsRead(rs.ItemTypeId, rs.ItemId, profileId, time.Now())

	} else if rs.ItemTypeId > 0 && rs.ItemId > 0 {
		return MarkAsRead(rs.ItemTypeId, rs.ItemId, profileId, time.Now())

	} else {
		return http.StatusBadRequest,
			errors.New(
				"ItemTypeId and ItemId must be specified to mark an item read",
			)
	}
}

func MarkAllAsRead(profileId int64) (int, error) {
	// This method lies... we mark everything except huddles as read
	tx, err := h.GetTransaction()
	if err != nil {
		glog.Errorf("h.GetTransaction() %+v", err)
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
DELETE FROM read
 WHERE profile_id = $1
   AND item_type_id IN (2, 6, 9, 16)`,
		profileId,
	)
	if err != nil {
		glog.Errorf("tx.Exec(%d) %+v", profileId, err)
		tx.Rollback()
		return http.StatusInternalServerError, err
	}

	_, err = tx.Exec(`
INSERT INTO READ
       (profile_id, item_type_id, item_id, "read")
SELECT p.profile_id
      ,2 AS item_type_id
      ,m.microcosm_id AS item_id
      ,NOW() AS "read"
  FROM profiles p
      ,microcosms m
 WHERE p.profile_id = $1
   AND m.site_id = p.site_id`,
		profileId,
	)
	if err != nil {
		glog.Errorf("stmt2.Exec(%d) %+v", profileId, err)
		tx.Rollback()
		return http.StatusInternalServerError, err
	}

	err = tx.Commit()
	if err != nil {
		glog.Errorf("tx.Commit() %+v", err)
		tx.Rollback()
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}
