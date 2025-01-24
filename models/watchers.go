package models

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/golang/glog"
	"github.com/lib/pq"

	c "github.com/microcosm-collective/microcosm/cache"
	h "github.com/microcosm-collective/microcosm/helpers"
)

// WatchersType offers an array of watchers
type WatchersType struct {
	Watchers h.ArrayType    `json:"watchers"`
	Meta     h.CoreMetaType `json:"meta"`
}

// WatcherType encapsulates a single instance of an item being watched by a
// profile and the communication preferences
type WatcherType struct {
	ID                   int64       `json:"id"`
	ProfileID            int64       `json:"-"`
	ItemTypeID           int64       `json:"itemTypeId"`
	ItemID               int64       `json:"itemId"`
	LastNotifiedNullable pq.NullTime `json:"-"`
	LastNotified         time.Time   `json:"lastNotified,omitempty"`
	SendEmail            bool        `json:"sendEmail"`
	SendSMS              bool        `json:"sendSMS"`
	Item                 interface{} `json:"item"`
	ItemType             string      `json:"itemType"`
}

func (m *WatcherType) validate(exists bool) (int, error) {

	if exists {
		if m.ID < 1 {
			return http.StatusBadRequest,
				fmt.Errorf(
					"the supplied ID ('%d') cannot be zero or negative",
					m.ID,
				)
		}
	} else {
		if m.ID < 0 || m.ID > 0 {
			return http.StatusBadRequest,
				fmt.Errorf("you cannot specify an ID when creating a resource")
		}
	}

	return http.StatusOK, nil
}

// Insert stores the WatcherType in the database
func (m *WatcherType) Insert() (int, error) {
	return m.insert(false)
}

// Import stores the watcher type without performing validation
func (m *WatcherType) Import() (int, error) {
	return m.insert(true)
}

// Insert creates the watcher
func (m *WatcherType) insert(imported bool) (int, error) {

	status, err := m.validate(imported)
	if err != nil {
		glog.Error(err)
		return status, err
	}

	tx, err := h.GetTransaction()
	if err != nil {
		glog.Error(err)
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	var insertID int64
	err = tx.QueryRow(`
INSERT INTO watchers (
    profile_id
   ,item_type_id
   ,item_id
   ,send_email
   ,send_sms
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5
) RETURNING watcher_id`,
		m.ProfileID,
		m.ItemTypeID,
		m.ItemID,
		m.SendEmail,
		m.SendSMS,
	).Scan(
		&insertID,
	)
	if err != nil {
		glog.Error(err)
		return http.StatusInternalServerError,
			fmt.Errorf("error inserting data and returning ID: %+v", err)
	}
	m.ID = insertID

	err = tx.Commit()
	if err != nil {
		glog.Error(err)
		return http.StatusInternalServerError,
			fmt.Errorf("transaction failed: %v", err.Error())
	}

	PurgeCache(h.ItemTypes[h.ItemTypeWatcher], m.ID)

	return http.StatusOK, nil
}

// Update updates the watcher in the database
func (m *WatcherType) Update() (int, error) {

	status, err := m.validate(true)
	if err != nil {
		glog.Info(err)
		return status, err
	}

	tx, err := h.GetTransaction()
	if err != nil {
		glog.Error(err)
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
UPDATE watchers
   SET send_email = $2,
       send_sms = $3
 WHERE watcher_id = $1`,
		m.ID,
		m.SendEmail,
		m.SendSMS,
	)
	if err != nil {
		glog.Error(err)
		return http.StatusInternalServerError,
			fmt.Errorf("update failed: %v", err.Error())
	}

	err = tx.Commit()
	if err != nil {
		glog.Error(err)
		return http.StatusInternalServerError,
			fmt.Errorf("transaction failed: %v", err.Error())
	}

	PurgeCache(h.ItemTypes[h.ItemTypeWatcher], m.ID)

	return http.StatusOK, nil
}

// Delete removes a watcher from the database and cache
func (m *WatcherType) Delete() (int, error) {

	tx, err := h.GetTransaction()
	if err != nil {
		glog.Error(err)
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	if m.ID > 0 {
		_, err = tx.Exec(`
DELETE
  FROM watchers
 WHERE watcher_id = $1`,
			m.ID,
		)
		if err != nil {
			glog.Error(err)
			return http.StatusInternalServerError,
				fmt.Errorf("delete failed: %v", err.Error())
		}
	} else {
		err = tx.QueryRow(`
WITH t1 AS (
         SELECT watcher_id
           FROM watchers
          WHERE profile_id = $1
            AND item_type_id = $2
            AND item_id = $3
     )
    ,t2 AS (
         DELETE
           FROM watchers
          WHERE profile_id = $1
            AND item_type_id = $2
            AND item_id = $3
     )
SELECT watcher_id
  FROM t1`,
			m.ProfileID,
			m.ItemTypeID,
			m.ItemID,
		).Scan(&m.ID)
		if err != nil {
			if err == sql.ErrNoRows {
				return http.StatusOK, nil
			}

			glog.Error(err)
			return http.StatusInternalServerError,
				fmt.Errorf("delete failed: %v", err.Error())
		}
	}

	err = tx.Commit()
	if err != nil {
		glog.Error(err)
		return http.StatusInternalServerError,
			fmt.Errorf("transaction failed: %v", err.Error())
	}

	PurgeCache(h.ItemTypes[h.ItemTypeWatcher], m.ID)

	return http.StatusOK, nil
}

// GetWatcherAndIgnoreStatus will return the communication preferences
// for a given item and profile
func GetWatcherAndIgnoreStatus(
	itemTypeID int64,
	itemID int64,
	profileID int64,
) (
	int64,
	bool,
	bool,
	bool,
	int,
	error,
) {

	db, err := h.GetConnection()
	if err != nil {
		glog.Error(err)
		return 0, false, false, false, http.StatusInternalServerError, err
	}

	// Returns a watched id if a watcher exists, or zero
	rows, err := db.Query(`--GetWatcherAndIgnoreStatus
SELECT COALESCE(w.watcher_id, 0) AS watcher_id
      ,COALESCE(w.send_email, FALSE) AS send_email
      ,COALESCE(w.send_sms, FALSE) AS send_sms
      ,CASE WHEN i.profile_id IS NULL THEN FALSE
              ELSE TRUE
       END AS ignored
  FROM flags f
  LEFT JOIN watchers w ON w.item_type_id = f.item_type_id
                      AND w.item_id = f.item_id
                      AND w.profile_id = $3
  LEFT JOIN ignores_expanded i ON i.item_type_id = f.item_type_id
                              AND i.item_id = f.item_id
                              AND i.profile_id = $3
 WHERE f.item_type_id = $1
   AND f.item_id = $2
 UNION
-- Deals with the item_id = 0 case as a flags row won't exist
SELECT watcher_id
      ,send_email
      ,send_sms
      ,FALSE
  FROM watchers
 WHERE item_type_id = $1
   AND item_id = $2
   AND item_id = 0
   AND profile_id = $3
 ORDER BY 1 DESC
 LIMIT 1`,
		itemTypeID,
		itemID,
		profileID,
	)
	if err != nil {
		glog.Error(err)
		return 0, false, false, false, http.StatusInternalServerError,
			fmt.Errorf("database query failed: %v", err.Error())
	}
	defer rows.Close()

	var watcherID int64
	var sendEmail bool
	var sendSMS bool
	var ignored bool

	for rows.Next() {
		err = rows.Scan(
			&watcherID,
			&sendEmail,
			&sendSMS,
			&ignored,
		)
		if err != nil {
			glog.Error(err)
			return 0, false, false, false, http.StatusInternalServerError,
				fmt.Errorf("row parsing error: %v", err.Error())
		}
	}
	err = rows.Err()
	if err != nil {
		glog.Error(err)
		return 0, false, false, false, http.StatusInternalServerError,
			fmt.Errorf("error fetching rows: %v", err.Error())
	}
	rows.Close()

	return watcherID, sendEmail, sendSMS, ignored, http.StatusOK, nil
}

// UpdateLastNotified updates a watcher according to the last time that watcher
// triggered a notification. The purpose is to record any contact so that we
// can avoid multiple notifications for a given item
func (m *WatcherType) UpdateLastNotified() (int, error) {

	tx, err := h.GetTransaction()
	if err != nil {
		glog.Error(err)
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
UPDATE watchers
   SET last_notified = NOW()
 WHERE watcher_id = $1`,
		m.ID,
	)
	if err != nil {
		glog.Error(err)
		tx.Rollback()
		return http.StatusInternalServerError, err
	}

	err = tx.Commit()
	if err != nil {
		glog.Error(err)
		return http.StatusInternalServerError, err
	}

	PurgeCache(h.ItemTypes[h.ItemTypeWatcher], m.ID)

	return http.StatusOK, nil
}

// GetWatcher returns a given watcher
func GetWatcher(watcherID int64, siteID int64) (WatcherType, int, error) {

	// Get from cache if it's available
	mcKey := fmt.Sprintf(mcWatcherKeys[c.CacheDetail], watcherID)
	if val, ok := c.Get(mcKey, WatcherType{}); ok {
		return val.(WatcherType), http.StatusOK, nil
	}

	db, err := h.GetConnection()
	if err != nil {
		glog.Error(err)
		return WatcherType{}, http.StatusInternalServerError, err
	}

	var m WatcherType
	err = db.QueryRow(`
SELECT watcher_id,
       profile_id,
       item_type_id,
       item_id,
       last_notified,
       send_email,
       send_sms
  FROM watchers
 WHERE watcher_id = $1`,
		watcherID,
	).Scan(
		&m.ID,
		&m.ProfileID,
		&m.ItemTypeID,
		&m.ItemID,
		&m.LastNotifiedNullable,
		&m.SendEmail,
		&m.SendSMS,
	)
	if err == sql.ErrNoRows {
		return WatcherType{}, http.StatusNotFound,
			fmt.Errorf("resource with ID %d not found", watcherID)
	} else if err != nil {
		glog.Error(err)
		return WatcherType{}, http.StatusInternalServerError,
			fmt.Errorf("database query failed: %v", err.Error())
	}

	if m.LastNotifiedNullable.Valid {
		m.LastNotified = m.LastNotifiedNullable.Time
	}

	// Fetch item data
	itemType, err := h.GetItemTypeFromInt(m.ItemTypeID)
	if err != nil {
		glog.Error(err)
		return WatcherType{}, http.StatusInternalServerError, err
	}
	m.ItemType = itemType

	// Only fetch the item itself if valid siteId is given
	if siteID > 0 {
		if m.ItemTypeID != 2 {
			if m.ItemID != 0 {
				item, _, err := GetSummary(
					siteID,
					m.ItemTypeID,
					m.ItemID,
					m.ProfileID,
				)
				if err != nil {
					glog.Error(err)
					return WatcherType{}, http.StatusInternalServerError, err
				}
				m.Item = item
			}
		} else {
			microcosm, _, err := GetMicrocosmSummary(
				siteID,
				m.ItemID,
				m.ProfileID,
			)
			if err != nil {
				glog.Error(err)
				return WatcherType{}, http.StatusInternalServerError, err
			}
			m.Item = microcosm
		}
	}

	// Update cache
	c.Set(mcKey, m, mcTTL)

	return m, http.StatusOK, nil
}

// GetProfileWatchers fetches all watchers registered to a particular profile.
// This is mainly used for showing a list of watchers to the user.
func GetProfileWatchers(
	profileID int64,
	siteID int64,
	limit int64,
	offset int64,
) (
	[]WatcherType,
	int64,
	int64,
	int,
	error,
) {

	db, err := h.GetConnection()
	if err != nil {
		glog.Error(err)
		return []WatcherType{}, 0, 0, http.StatusInternalServerError, err
	}

	rows, err := db.Query(`
SELECT COUNT(*) OVER() AS total
      ,watcher_id
  FROM watchers
 WHERE profile_id = $1
 ORDER BY last_notified DESC
         ,item_type_id ASC
         ,item_id DESC
 LIMIT $2
OFFSET $3`,
		profileID,
		limit,
		offset,
	)
	if err != nil {
		glog.Error(err)
		return []WatcherType{}, 0, 0, http.StatusInternalServerError,
			fmt.Errorf("database query failed: %v", err.Error())
	}
	defer rows.Close()

	var ems []WatcherType

	var total int64
	for rows.Next() {
		var id int64
		err = rows.Scan(
			&total,
			&id,
		)
		if err != nil {
			glog.Error(err)
			return []WatcherType{}, 0, 0, http.StatusInternalServerError,
				fmt.Errorf("row parsing error: %v", err.Error())
		}

		m, status, err := GetWatcher(id, siteID)
		if err != nil {
			glog.Error(err)
			return []WatcherType{}, 0, 0, status, err
		}

		ems = append(ems, m)
	}
	err = rows.Err()
	if err != nil {
		glog.Error(err)
		return []WatcherType{}, 0, 0, http.StatusInternalServerError,
			fmt.Errorf("error fetching rows: %v", err.Error())
	}
	rows.Close()

	pages := h.GetPageCount(total, limit)
	maxOffset := h.GetMaxOffset(total, limit)

	if offset > maxOffset {
		return []WatcherType{}, 0, 0, http.StatusBadRequest,
			fmt.Errorf("not enough records, "+
				"offset (%d) would return an empty page", offset)
	}

	return ems, total, pages, http.StatusOK, nil
}

// RegisterWatcher is offers an idempotent operation for creating a watcher on
// a specific item
func RegisterWatcher(
	profileID int64,
	updateTypeID int64,
	itemID int64,
	itemTypeID int64,
	siteID int64,
) (
	bool,
	int,
	error,
) {

	// Don't do it if it exists.
	watcherID, sendEmail, _, _, status, err := GetWatcherAndIgnoreStatus(
		itemTypeID,
		itemID,
		profileID,
	)
	if err != nil {
		glog.Error(err)
		return false, status,
			fmt.Errorf("failed to get watcher: %s", err.Error())
	}
	if watcherID > 0 {
		return sendEmail, http.StatusOK, nil
	}

	// Does not exist, get default prefs, build watcher and insert
	updateOptions, status, err := GetCommunicationOptions(
		siteID,
		profileID,
		updateTypeID,
		itemTypeID,
		itemID,
	)
	if err != nil {
		glog.Error(err)
		return false, status,
			fmt.Errorf("failed to get update options for profile %d: %s", profileID, err.Error())
	}

	m := WatcherType{}
	m.ProfileID = profileID
	m.ItemTypeID = itemTypeID
	m.ItemID = itemID
	m.SendEmail = updateOptions.SendEmail
	m.SendSMS = updateOptions.SendSMS

	status, err = m.Insert()
	if err != nil {
		glog.Error(err)
		return false, status,
			fmt.Errorf("failed to set watcher: %s", err.Error())
	}

	return updateOptions.SendEmail, http.StatusOK, nil
}
