package models

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/lib/pq"

	c "github.com/microcosm-cc/microcosm/cache"
	h "github.com/microcosm-cc/microcosm/helpers"
)

// HuddlesType is a collection of huddles
type HuddlesType struct {
	Huddles h.ArrayType    `json:"huddles"`
	Meta    h.CoreMetaType `json:"meta"`
}

// HuddleSummaryType is a summary of a huddle
type HuddleSummaryType struct {
	ID     int64  `json:"id"`
	SiteID int64  `json:"siteId,omitempty"`
	Title  string `json:"title"`

	CommentCount int64 `json:"totalComments"`

	Participants []ProfileSummaryType `json:"participants"`

	LastCommentIDNullable        sql.NullInt64 `json:"-"`
	LastCommentID                int64         `json:"lastCommentId,omitempty"`
	LastCommentCreatedByNullable sql.NullInt64 `json:"-"`
	LastCommentCreatedBy         interface{}   `json:"lastCommentCreatedBy,omitempty"`
	LastCommentCreatedNullable   pq.NullTime   `json:"-"`
	LastCommentCreated           string        `json:"lastCommentCreated,omitempty"`

	Meta h.SummaryMetaType `json:"meta"`
}

// HuddleType is a huddle
type HuddleType struct {
	ID             int64  `json:"id"`
	SiteID         int64  `json:"siteId,omitempty"`
	Title          string `json:"title"`
	IsConfidential bool   `json:"isConfidential"`

	Participants []ProfileSummaryType `json:"participants"`

	Comments h.ArrayType `json:"comments"`

	Meta h.SummaryMetaType `json:"meta"`
}

// GetLink returns the link to this huddle
func (m *HuddleType) GetLink() string {
	return fmt.Sprintf("%s/%d", h.APITypeHuddle, m.ID)
}

// Validate returns true if the huddle if valid
func (m *HuddleType) Validate(siteID int64, exists bool) (int, error) {

	preventShouting := true
	m.Title = CleanSentence(m.Title, preventShouting)

	if exists {
		if m.ID < 1 {
			return http.StatusBadRequest, fmt.Errorf(
				"The supplied ID ('%d') cannot be zero or negative",
				m.ID,
			)
		}
	}

	for _, p := range m.Participants {
		_, status, err := GetProfileSummary(siteID, p.ID)
		if err != nil {
			return status, err
		}
	}

	if strings.Trim(m.Title, " ") == "" {
		return http.StatusBadRequest, fmt.Errorf("Title is a required field")
	}

	return http.StatusOK, nil
}

// Hydrate populates the partially populated huddle
func (m *HuddleType) Hydrate(siteID int64) (int, error) {
	profile, status, err := GetProfileSummary(siteID, m.Meta.CreatedByID)
	if err != nil {
		return status, err
	}
	m.Meta.CreatedBy = profile

	for i, v := range m.Participants {
		profile, status, err = GetProfileSummary(siteID, v.ID)
		if err != nil {
			return status, err
		}
		m.Participants[i] = profile
	}

	return http.StatusOK, nil
}

// Hydrate populates the partially populated huddle summary
func (m *HuddleSummaryType) Hydrate(siteID int64) (int, error) {

	profile, status, err := GetProfileSummary(siteID, m.Meta.CreatedByID)
	if err != nil {
		return status, err
	}
	m.Meta.CreatedBy = profile

	for i, v := range m.Participants {
		profile, status, err = GetProfileSummary(siteID, v.ID)
		if err != nil {
			return status, err
		}
		m.Participants[i] = profile
	}

	if m.LastCommentCreatedByNullable.Valid {
		profile, status, err :=
			GetProfileSummary(siteID, m.LastCommentCreatedByNullable.Int64)

		if err != nil {
			return status, err
		}
		m.LastCommentCreatedBy = profile
	}

	return http.StatusOK, nil
}

// Import allows a huddle to be added without performing dupe checks
func (m *HuddleType) Import(siteID int64) (int, error) {
	return m.insert(siteID)
}

// Insert saves a huddle to the database
func (m *HuddleType) Insert(siteID int64) (int, error) {

	dupeKey := "dupe_" + h.MD5Sum(
		m.Title+
			strconv.FormatInt(m.Meta.CreatedByID, 10),
	)

	v, ok := c.GetInt64(dupeKey)
	if ok {
		m.ID = v
		return http.StatusOK, nil
	}

	status, err := m.insert(siteID)

	// 5 second dupe check just to catch people hitting submit multiple times
	c.SetInt64(dupeKey, m.ID, 5)

	return status, err
}

// insert saves the huddle to the database
func (m *HuddleType) insert(siteID int64) (int, error) {
	status, err := m.Validate(siteID, false)
	if err != nil {
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
INSERT INTO huddles (
    site_id, title, created, created_by, is_confidential
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING huddle_id`,
		siteID,
		m.Title,
		m.Meta.Created,
		m.Meta.CreatedByID,
		m.IsConfidential,
	).Scan(
		&insertID,
	)
	if err != nil {
		glog.Error(err)
		return http.StatusInternalServerError,
			fmt.Errorf("Error inserting data and returning ID: %+v", err)
	}
	m.ID = insertID

	// Author is a participant too
	_, err = tx.Exec(`
INSERT INTO huddle_profiles (
    huddle_id, profile_id
) VALUES (
    $1, $2
)`,
		m.ID,
		m.Meta.CreatedByID,
	)
	if err != nil {
		glog.Error(err)
		return http.StatusInternalServerError,
			fmt.Errorf("Error creating huddle author")
	}

	// As are all of the explicitly added participants
	for _, p := range m.Participants {
		// Cannot add the author twice
		if p.ID != m.Meta.CreatedByID {
			_, err := tx.Exec(`
INSERT INTO huddle_profiles (
    huddle_id, profile_id
) VALUES (
    $1, $2
)`,
				m.ID,
				p.ID,
			)
			if err != nil {
				glog.Error(err)
				return http.StatusInternalServerError,
					fmt.Errorf("Error creating huddle participant")
			}
		}
	}

	err = tx.Commit()
	if err != nil {
		glog.Error(err)
		return http.StatusInternalServerError,
			fmt.Errorf("Transaction failed: %v", err.Error())
	}

	PurgeCache(h.ItemTypes[h.ItemTypeHuddle], m.ID)

	return http.StatusOK, nil
}

// Delete removes a participant from a huddle
func (m *HuddleType) Delete(siteID int64, profileID int64) (int, error) {
	// Synonym for removing participant. If all participants are removed then
	// this will physically delete the huddle
	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
DELETE FROM huddle_profiles
 WHERE huddle_id = $1
   AND profile_id = $2`,
		m.ID,
		profileID,
	)
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("Delete failed: %v", err.Error())
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("Transaction failed: %v", err.Error())
	}

	PurgeCache(h.ItemTypes[h.ItemTypeHuddle], m.ID)

	return http.StatusOK, nil
}

// GetHuddle returns a huddle
func GetHuddle(
	siteID int64,
	profileID int64,
	id int64,
) (
	HuddleType,
	int,
	error,
) {

	// Get from cache if it's available
	mcKey := fmt.Sprintf(mcHuddleKeys[c.CacheDetail], id)
	if val, ok := c.Get(mcKey, HuddleType{}); ok {
		m := val.(HuddleType)
		m.Hydrate(siteID)
		return m, http.StatusOK, nil
	}

	// Retrieve resource
	db, err := h.GetConnection()
	if err != nil {
		return HuddleType{}, http.StatusInternalServerError, err
	}

	rows, err := db.Query(`
SELECT h.huddle_id
      ,h.title
      ,h.created
      ,h.created_by
      ,is_confidential
      ,hp.profile_id
  FROM huddles h
       JOIN huddle_profiles hp ON h.huddle_id = hp.huddle_id
 WHERE h.site_id = $1
   AND h.huddle_id = $2`,
		siteID,
		id,
	)
	if err != nil {
		return HuddleType{}, http.StatusInternalServerError,
			fmt.Errorf("Database query failed: %v", err.Error())
	}
	defer rows.Close()

	var m HuddleType

	m = HuddleType{}
	for rows.Next() {
		var participantID int64
		err = rows.Scan(
			&m.ID,
			&m.Title,
			&m.Meta.Created,
			&m.Meta.CreatedByID,
			&m.IsConfidential,
			&participantID,
		)
		if err != nil {
			return HuddleType{}, http.StatusInternalServerError,
				fmt.Errorf("Row parsing error: %v", err.Error())
		}

		m.Participants = append(
			m.Participants,
			ProfileSummaryType{ID: participantID},
		)
	}
	err = rows.Err()
	if err != nil {
		return HuddleType{}, http.StatusInternalServerError,
			fmt.Errorf("Error fetching rows: %v", err.Error())
	}
	rows.Close()

	if m.ID == 0 {
		return HuddleType{}, http.StatusNotFound,
			fmt.Errorf("Resource with ID %d not found", id)
	}

	m.Meta.Links =
		[]h.LinkType{
			h.GetLink("self", "", h.ItemTypeHuddle, m.ID),
		}

	// Update cache
	c.Set(mcKey, m, mcTTL)

	m.Hydrate(siteID)

	return m, http.StatusOK, nil
}

// GetHuddleTitle returns the title of the huddle
func GetHuddleTitle(id int64) string {
	// Get from cache if it's available
	mcKey := fmt.Sprintf(mcHuddleKeys[c.CacheTitle], id)
	if val, ok := c.GetString(mcKey); ok {
		return val
	}

	// Retrieve resource
	db, err := h.GetConnection()
	if err != nil {
		glog.Errorf("h.GetConmection() %+v", err)
		return ""
	}

	var title string
	err = db.QueryRow(`
SELECT title
  FROM huddles
 WHERE huddle_id = $1`,
		id,
	).Scan(
		&title,
	)
	if err != nil {
		glog.Errorf("row.Scan() %+v", err)
		return ""
	}

	// Update cache
	c.SetString(mcKey, title, mcTTL)

	return title
}

// GetHuddleSummary returns a summary of a huddle
func GetHuddleSummary(
	siteID int64,
	profileID int64,
	id int64,
) (
	HuddleSummaryType,
	int,
	error,
) {
	// Get from cache if it's available
	mcKey := fmt.Sprintf(mcHuddleKeys[c.CacheSummary], id)
	if val, ok := c.Get(mcKey, HuddleSummaryType{}); ok {
		m := val.(HuddleSummaryType)
		m.Hydrate(siteID)
		return m, http.StatusOK, nil
	}

	// Retrieve resource
	db, err := h.GetConnection()
	if err != nil {
		glog.Errorf("h.GetConnection() %+v", err)
		return HuddleSummaryType{}, http.StatusInternalServerError, err
	}

	rows, err := db.Query(`
SELECT h.huddle_id
      ,h.title
      ,h.created
      ,h.created_by
      ,hp.profile_id
      ,lc.comment_id
      ,lc.created
      ,lc.profile_id
      ,(SELECT COUNT(*) AS total_comments
          FROM flags
         WHERE parent_item_type_id = 5
           AND parent_item_id = $2
           AND site_id = $1
           AND microcosm_is_deleted IS NOT TRUE
           AND microcosm_is_moderated IS NOT TRUE
           AND parent_is_deleted IS NOT TRUE
           AND parent_is_moderated IS NOT TRUE
           AND item_is_deleted IS NOT TRUE
           AND item_is_moderated IS NOT TRUE) AS comment_count
  FROM huddles h
       LEFT OUTER JOIN (
           SELECT c.created
                 ,c.comment_id
                 ,c.profile_id
                 ,c.item_id
             FROM comments c
                  JOIN revisions r ON c.comment_id = r.comment_id
            WHERE r.is_current
              AND c.is_deleted IS NOT TRUE
              AND c.item_type_id = $4
              AND c.item_id = $2
            ORDER BY 1 DESC, 2 DESC
            LIMIT 1
       ) lc ON lc.item_id = h.huddle_id
      ,huddle_profiles hp
      ,(
           SELECT *
             FROM huddle_profiles 
            WHERE huddle_id = $2
              AND profile_id = $3
       ) AS hp2
 WHERE h.site_id = $1
   AND h.huddle_id = $2
   AND h.huddle_id = hp.huddle_id`,
		siteID,
		id,
		profileID,
		h.ItemTypes[h.ItemTypeHuddle],
	)
	if err != nil {
		glog.Errorf(
			"db.Query(%d, %d, %d, h.ItemTypes[h.ItemTypeHuddle]) %+v",
			siteID, id, profileID, err,
		)
		return HuddleSummaryType{}, http.StatusInternalServerError,
			fmt.Errorf("Database query failed")
	}
	defer rows.Close()

	var m HuddleSummaryType

	m = HuddleSummaryType{}
	for rows.Next() {
		var participantID int64
		err = rows.Scan(
			&m.ID,
			&m.Title,
			&m.Meta.Created,
			&m.Meta.CreatedByID,
			&participantID,
			&m.LastCommentIDNullable,
			&m.LastCommentCreatedNullable,
			&m.LastCommentCreatedByNullable,
			&m.CommentCount,
		)
		if err != nil {
			glog.Errorf("rows.Scan() %+v", err)
			return HuddleSummaryType{}, http.StatusInternalServerError,
				fmt.Errorf("Row parsing error")
		}

		m.Participants = append(
			m.Participants,
			ProfileSummaryType{ID: participantID},
		)

		if m.LastCommentIDNullable.Valid {
			m.LastCommentID = m.LastCommentIDNullable.Int64
		}

		if m.LastCommentCreatedNullable.Valid {
			m.LastCommentCreated =
				m.LastCommentCreatedNullable.Time.Format(time.RFC3339Nano)
		}
	}
	err = rows.Err()
	if err != nil {
		glog.Errorf("rows.Err() %+v", err)
		return HuddleSummaryType{}, http.StatusInternalServerError,
			fmt.Errorf("Error fetching rows")
	}
	rows.Close()

	if m.ID == 0 {
		glog.Warningf("m.Id == 0")
		return HuddleSummaryType{}, http.StatusNotFound,
			fmt.Errorf("Resource not found")
	}

	m.Meta.Links =
		[]h.LinkType{
			h.GetLink("self", "", h.ItemTypeHuddle, m.ID),
		}

	// Update cache
	c.Set(mcKey, m, mcTTL)

	m.Hydrate(siteID)

	return m, http.StatusOK, nil
}

// GetHuddles returns a collection of huddles
func GetHuddles(
	siteID int64,
	profileID int64,
	limit int64,
	offset int64,
	filterUnread bool,
) (
	[]HuddleSummaryType,
	int64,
	int64,
	int,
	error,
) {
	// Retrieve resources
	db, err := h.GetConnection()
	if err != nil {
		glog.Errorf("h.GetConnection() %+v", err)
		return []HuddleSummaryType{}, 0, 0, http.StatusInternalServerError, err
	}

	var (
		total int64
		query string
	)
	if filterUnread {
		query = `--unreadHuddles
SELECT COUNT(*) OVER() AS total
      ,h.huddle_id
      ,TRUE AS unread
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
 GROUP BY h.huddle_id, ff.last_modified
 ORDER BY ff.last_modified DESC
 LIMIT $2
OFFSET $3`
	} else {
		query = `--huddles
SELECT 0 AS total
      ,huddle_id
      ,has_unread(5, huddle_id, $1) AS unread
  FROM (
            SELECT h.huddle_id
              FROM huddles h
              JOIN huddle_profiles hp ON hp.huddle_id = h.huddle_id
              JOIN flags f ON f.item_type_id = 5
                          AND f.item_id = h.huddle_id
              LEFT JOIN ignores i ON i.profile_id = $1
                                 AND i.item_type_id = 3
                                 AND i.item_id = h.created_by
             WHERE hp.profile_id = $1
               AND i.profile_id IS NULL
             ORDER BY f.last_modified DESC
             LIMIT $2
            OFFSET $3
       ) r`

		// We also need to get the total as the above query doesn't do it for
		// performance reasons (potentially a huge number of huddles)
		err = db.QueryRow(`
SELECT COUNT(*) AS total
  FROM huddles h
  JOIN huddle_profiles hp ON hp.huddle_id = h.huddle_id
  LEFT JOIN ignores i ON i.profile_id = $1
                     AND i.item_type_id = 3
                     AND i.item_id = h.created_by
 WHERE hp.profile_id = $1
   AND i.profile_id IS NULL`,
			profileID,
		).Scan(&total)
		if err != nil {
			glog.Error(err)
			return []HuddleSummaryType{}, 0, 0, http.StatusInternalServerError,
				fmt.Errorf("Database query failed")
		}
	}

	rows, err := db.Query(query, profileID, limit, offset)
	if err != nil {
		glog.Errorf(
			"db.Query(%d, %d, %d, %d) %+v",
			siteID,
			profileID,
			limit,
			offset,
			err,
		)
		return []HuddleSummaryType{}, 0, 0, http.StatusInternalServerError,
			fmt.Errorf("Database query failed")
	}
	defer rows.Close()

	var ems []HuddleSummaryType

	for rows.Next() {
		var (
			tmpTotal  int64
			id        int64
			hasUnread bool
		)
		err = rows.Scan(
			&tmpTotal,
			&id,
			&hasUnread,
		)
		if err != nil {
			glog.Errorf("rows.Scan() %+v", err)
			return []HuddleSummaryType{}, 0, 0, http.StatusInternalServerError,
				fmt.Errorf("Row parsing error")
		}

		if filterUnread {
			total = tmpTotal
		}

		m, status, err := GetHuddleSummary(siteID, profileID, id)
		if err != nil {
			glog.Errorf(
				"GetHuddleSummary(%d, %d, %d) %+v",
				siteID,
				profileID,
				id,
				err,
			)
			return []HuddleSummaryType{}, 0, 0, status, err
		}

		m.Meta.Flags.Unread = hasUnread

		ems = append(ems, m)
	}
	err = rows.Err()
	if err != nil {
		glog.Errorf("rows.Err() %+v", err)
		return []HuddleSummaryType{}, 0, 0, http.StatusInternalServerError,
			fmt.Errorf("Error fetching rows")
	}
	rows.Close()

	pages := h.GetPageCount(total, limit)
	maxOffset := h.GetMaxOffset(total, limit)

	if offset > maxOffset {
		glog.Warningf("offset > maxOffset")
		return []HuddleSummaryType{}, 0, 0, http.StatusBadRequest,
			fmt.Errorf(
				fmt.Sprintf("not enough records, "+
					"offset (%d) would return an empty page.",
					offset,
				),
			)
	}

	return ems, total, pages, http.StatusOK, nil
}
