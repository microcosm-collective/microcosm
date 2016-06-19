package models

import (
	"fmt"
	"net/http"
	"time"

	"github.com/lib/pq"

	h "github.com/microcosm-cc/microcosm/helpers"
)

// UpdateRecipient distills watchers and communications options into a single
// actionable and switchable thing
type UpdateRecipient struct {
	Watcher              WatcherType
	ForProfile           ProfileSummaryType
	SendEmail            bool
	SendSMS              bool
	LastNotifiedNullable pq.NullTime
	LastNotified         time.Time
}

// GetUpdateRecipients returns the update recipients for a given item that has
// been updated
func GetUpdateRecipients(
	siteID int64,
	itemTypeID int64,
	itemID int64,
	updateTypeID int64,
	createdByID int64,
) (
	[]UpdateRecipient,
	int,
	error,
) {

	var (
		includeMicrocosmWatchers bool
		includeSiteWatchers      bool
		includeHuddleWatchers    bool
	)

	switch updateTypeID {
	case h.UpdateTypes[h.UpdateTypeNewComment]:
		includeMicrocosmWatchers = true
		includeSiteWatchers = true
		includeHuddleWatchers = false

	case h.UpdateTypes[h.UpdateTypeNewItem]:
		includeMicrocosmWatchers = true
		includeSiteWatchers = true
		includeHuddleWatchers = false

	case h.UpdateTypes[h.UpdateTypeNewCommentInHuddle]:
		includeMicrocosmWatchers = false
		includeSiteWatchers = false
		includeHuddleWatchers = true

	default:
	}

	db, err := h.GetConnection()
	if err != nil {
		return []UpdateRecipient{}, http.StatusInternalServerError, err
	}

	var sql string
	sql = `--GetUpdateRecipients
SELECT w.watcher_id
      ,w.profile_id
      ,w.last_notified
      ,(w.send_email AND i.profile_id IS NULL) AS send_email
      ,w.send_sms
  FROM (
SELECT CASE WHEN BIT_AND(a.item_watcher) > 0 THEN
           BIT_AND(a.item_watcher)
       WHEN BIT_AND(a.microcosm_watcher) > 0 THEN
           BIT_AND(a.microcosm_watcher)
       ELSE
           BIT_AND(a.site_watcher)
       END AS watcher_id
      ,a.profile_id
      ,MAX(a.last_notified) AS last_notified
      ,BOOL_AND(a.send_email AND po.send_email) AS send_email
      ,BOOL_AND(a.send_sms AND po.send_sms) AS send_sms
  FROM (
           -- Explicitly watching the item in question
           SELECT watcher_id AS item_watcher
                 ,CAST(NULL AS BIGINT) AS microcosm_watcher
                 ,CAST(NULL AS BIGINT) AS site_watcher
                 ,profile_id
                 ,last_notified
                 ,send_email
                 ,send_sms
             FROM watchers
            WHERE item_type_id = $2
              AND item_id = $3`

	if includeMicrocosmWatchers {
		sql = sql + `
           -- Watching the microcosm the item belongs to
            UNION
           SELECT CAST(NULL AS BIGINT) AS item_watcher
                 ,w.watcher_id AS microcosm_watcher
                 ,CAST(NULL AS BIGINT) AS site_watcher
                 ,w.profile_id
                 ,w.last_notified
                 ,(w.send_email AND COUNT(i.profile_id) = 0) AS send_email
                 ,w.send_sms
             FROM watchers w
             LEFT JOIN ignores i ON i.profile_id = w.profile_id
                                AND (
                                        (i.item_type_id = w.item_type_id AND i.item_id = w.item_id)
                                     OR (i.item_type_id = $2 AND i.item_id = $3)
                                    )
            WHERE w.item_type_id = 2 -- Microcosm
              AND w.item_id IN (
                      SELECT microcosm_id
                        FROM flags
                       WHERE item_type_id = $2
                         AND item_id = $3
                  )
            GROUP BY w.watcher_id
                    ,w.profile_id
                    ,w.last_notified
                    ,w.send_email
                    ,w.send_sms`
	}

	if includeSiteWatchers {
		sql = sql + `
           -- Watching the site
            UNION
           SELECT CAST(NULL AS BIGINT) AS item_watcher
                 ,CAST(NULL AS BIGINT) AS microcosm_watcher
                 ,watcher_id AS site_watcher
                 ,profile_id
                 ,last_notified
                 ,send_email
                 ,send_sms
             FROM watchers
            WHERE item_type_id = 1
              AND item_id = $1`
	}

	if includeHuddleWatchers {
		sql = sql + `
           -- Watching the huddle
            UNION
           SELECT CAST(NULL AS BIGINT) AS item_watcher
                 ,CAST(NULL AS BIGINT) AS microcosm_watcher
                 ,w.watcher_id AS site_watcher
                 ,w.profile_id
                 ,w.last_notified
                 ,w.send_email
                 ,w.send_sms
             FROM watchers w
                 ,flags f
            WHERE f.item_type_id = $2
              AND f.item_id = $3
              AND f.site_id = $1
              AND f.microcosm_is_deleted IS NOT TRUE
              AND f.microcosm_is_moderated IS NOT TRUE
              AND f.item_is_deleted IS NOT TRUE
              AND f.item_is_moderated IS NOT TRUE
              AND f.parent_is_deleted IS NOT TRUE
              AND f.parent_is_moderated IS NOT TRUE
              AND w.item_type_id = f.parent_item_type_id
              AND w.item_id = f.parent_item_id`
	}

	sql = sql + `
       ) AS a
       JOIN profile_options po ON po.profile_id = a.profile_id
 GROUP BY a.profile_id
 ORDER BY a.profile_id
       ) AS w
  LEFT JOIN ignores i ON i.profile_id = w.profile_id
                     AND i.item_type_id = 3 -- profile
                     AND i.item_id = $4 -- created by
      ,flags f
 WHERE f.site_id = $1
   AND f.item_type_id = $2
   AND f.item_id = $3
   AND (get_effective_permissions($1, COALESCE(f.microcosm_id, 0), $2, $3, w.profile_id)).can_read IS TRUE`

	if updateTypeID == h.UpdateTypes[h.UpdateTypeNewUser] {
		sql = `--NewUserEmails
SELECT w.watcher_id
      ,w.profile_id
      ,w.last_notified
      ,w.send_email
      ,w.send_sms
  FROM watchers w
  JOIN profiles p ON p.profile_id = w.profile_id AND p.site_id = $1
 WHERE w.item_type_id = $2
   AND w.item_id = 0
   AND w.send_email IS TRUE
   AND $3::bigint = $4::bigint`
	}

	rows, err := db.Query(sql, siteID, itemTypeID, itemID, createdByID)
	if err != nil {
		return []UpdateRecipient{}, http.StatusInternalServerError,
			fmt.Errorf("Database query failed: %v", err.Error())
	}
	defer rows.Close()

	ems := []UpdateRecipient{}
	for rows.Next() {

		m := UpdateRecipient{}

		var (
			profileID int64
			watcherID int64
		)
		err = rows.Scan(
			&watcherID,
			&profileID,
			&m.LastNotifiedNullable,
			&m.SendEmail,
			&m.SendSMS,
		)
		if err != nil {
			return []UpdateRecipient{}, http.StatusInternalServerError,
				fmt.Errorf("Row parsing error: %v", err.Error())
		}

		watcher, status, err := GetWatcher(watcherID, siteID)
		if err != nil {
			return []UpdateRecipient{}, status, err
		}
		m.Watcher = watcher

		if m.LastNotifiedNullable.Valid {
			m.LastNotified = m.LastNotifiedNullable.Time
		}

		profile, status, err := GetProfileSummary(siteID, profileID)
		if err != nil {
			return []UpdateRecipient{}, status, err
		}
		m.ForProfile = profile

		ems = append(ems, m)
	}
	err = rows.Err()
	if err != nil {
		return []UpdateRecipient{}, http.StatusInternalServerError,
			fmt.Errorf("Error fetching rows: %v", err.Error())
	}
	rows.Close()

	return ems, http.StatusOK, nil
}
