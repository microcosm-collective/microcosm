package models

import (
	"database/sql"
	"errors"
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

type HuddlesType struct {
	Huddles h.ArrayType    `json:"huddles"`
	Meta    h.CoreMetaType `json:"meta"`
}

type HuddleSummaryType struct {
	Id     int64  `json:"id"`
	SiteId int64  `json:"siteId,omitempty"`
	Title  string `json:"title"`

	CommentCount int64 `json:"totalComments"`

	Participants []ProfileSummaryType `json:"participants"`

	LastCommentIdNullable        sql.NullInt64 `json:"-"`
	LastCommentId                int64         `json:"lastCommentId,omitempty"`
	LastCommentCreatedByNullable sql.NullInt64 `json:"-"`
	LastCommentCreatedBy         interface{}   `json:"lastCommentCreatedBy,omitempty"`
	LastCommentCreatedNullable   pq.NullTime   `json:"-"`
	LastCommentCreated           string        `json:"lastCommentCreated,omitempty"`

	Meta h.SummaryMetaType `json:"meta"`
}

type HuddleType struct {
	Id             int64  `json:"id"`
	SiteId         int64  `json:"siteId,omitempty"`
	Title          string `json:"title"`
	IsConfidential bool   `json:"isConfidential"`

	Participants []ProfileSummaryType `json:"participants"`

	Comments h.ArrayType `json:"comments"`

	Meta h.SummaryMetaType `json:"meta"`
}

func (m *HuddleType) GetLink() string {
	return fmt.Sprintf("%s/%d", h.ApiTypeHuddle, m.Id)
}

func (m *HuddleType) Validate(siteId int64, exists bool) (int, error) {

	m.Title = SanitiseText(m.Title)

	if exists {
		if m.Id < 1 {
			return http.StatusBadRequest, errors.New(
				fmt.Sprintf(
					"The supplied ID ('%d') cannot be zero or negative.",
					m.Id,
				),
			)
		}
	}

	for _, p := range m.Participants {
		_, status, err := GetProfileSummary(siteId, p.ID)
		if err != nil {
			return status, err
		}
	}

	if strings.Trim(m.Title, " ") == "" {
		return http.StatusBadRequest, errors.New("Title is a required field")
	}

	m.Title = ShoutToWhisper(m.Title)

	return http.StatusOK, nil
}

func (m *HuddleType) FetchProfileSummaries(siteId int64) (int, error) {

	profile, status, err := GetProfileSummary(siteId, m.Meta.CreatedById)
	if err != nil {
		return status, err
	}
	m.Meta.CreatedBy = profile

	for i, v := range m.Participants {
		profile, status, err = GetProfileSummary(siteId, v.ID)
		if err != nil {
			return status, err
		}
		m.Participants[i] = profile
	}

	return http.StatusOK, nil
}

func (m *HuddleSummaryType) FetchProfileSummaries(siteId int64) (int, error) {

	profile, status, err := GetProfileSummary(siteId, m.Meta.CreatedById)
	if err != nil {
		return status, err
	}
	m.Meta.CreatedBy = profile

	for i, v := range m.Participants {
		profile, status, err = GetProfileSummary(siteId, v.ID)
		if err != nil {
			return status, err
		}
		m.Participants[i] = profile
	}

	if m.LastCommentCreatedByNullable.Valid {
		profile, status, err :=
			GetProfileSummary(siteId, m.LastCommentCreatedByNullable.Int64)

		if err != nil {
			return status, err
		}
		m.LastCommentCreatedBy = profile
	}

	return http.StatusOK, nil
}

func (m *HuddleType) Import(siteId int64) (int, error) {
	return m.insert(siteId)
}

func (m *HuddleType) Insert(siteId int64) (int, error) {

	dupeKey := "dupe_" + h.Md5sum(
		m.Title+
			strconv.FormatInt(m.Meta.CreatedById, 10),
	)

	v, ok := c.CacheGetInt64(dupeKey)
	if ok {
		m.Id = v
		return http.StatusOK, nil
	}

	status, err := m.insert(siteId)

	// 5 second dupe check just to catch people hitting submit multiple times
	c.CacheSetInt64(dupeKey, m.Id, 5)

	return status, err
}

func (m *HuddleType) insert(siteId int64) (int, error) {

	status, err := m.Validate(siteId, false)
	if err != nil {
		return status, err
	}
	tx, err := h.GetTransaction()
	if err != nil {
		glog.Error(err)
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	var insertId int64
	err = tx.QueryRow(`
INSERT INTO huddles (
    site_id, title, created, created_by, is_confidential
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING huddle_id`,
		siteId,
		m.Title,
		m.Meta.Created,
		m.Meta.CreatedById,
		m.IsConfidential,
	).Scan(
		&insertId,
	)
	if err != nil {
		glog.Error(err)
		return http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Error inserting data and returning ID: %+v", err),
		)
	}
	m.Id = insertId

	// This is a candidate for a prepared statement, though we currently lack
	// faith that lib/pq is managing this right.

	// Author is a participant too
	_, err = tx.Exec(`
INSERT INTO huddle_profiles (
    huddle_id, profile_id
) VALUES (
    $1, $2
)`,
		m.Id,
		m.Meta.CreatedById,
	)
	if err != nil {
		glog.Error(err)
		return http.StatusInternalServerError,
			fmt.Errorf("Error creating huddle author")
	}

	// As are all of the explicitly added participants
	for _, p := range m.Participants {
		// Cannot add the author twice
		if p.ID != m.Meta.CreatedById {
			_, err := tx.Exec(`
INSERT INTO huddle_profiles (
    huddle_id, profile_id
) VALUES (
    $1, $2
)`,
				m.Id,
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
		return http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Transaction failed: %v", err.Error()),
		)
	}

	PurgeCache(h.ItemTypes[h.ItemTypeHuddle], m.Id)

	return http.StatusOK, nil
}

func (m *HuddleType) Delete(siteId int64, profileId int64) (int, error) {

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
		m.Id,
		profileId,
	)
	if err != nil {
		return http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Delete failed: %v", err.Error()),
		)
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Transaction failed: %v", err.Error()),
		)
	}

	PurgeCache(h.ItemTypes[h.ItemTypeHuddle], m.Id)

	return http.StatusOK, nil
}

func GetHuddle(
	siteId int64,
	profileId int64,
	id int64,
) (
	HuddleType,
	int,
	error,
) {

	// Get from cache if it's available
	mcKey := fmt.Sprintf(mcHuddleKeys[c.CacheDetail], id)
	if val, ok := c.CacheGet(mcKey, HuddleType{}); ok {

		m := val.(HuddleType)

		m.FetchProfileSummaries(siteId)

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
		siteId,
		id,
	)
	if err != nil {
		return HuddleType{}, http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Database query failed: %v", err.Error()),
		)
	}
	defer rows.Close()

	var m HuddleType

	m = HuddleType{}
	for rows.Next() {
		var participantId int64
		err = rows.Scan(
			&m.Id,
			&m.Title,
			&m.Meta.Created,
			&m.Meta.CreatedById,
			&m.IsConfidential,
			&participantId,
		)
		if err != nil {
			return HuddleType{}, http.StatusInternalServerError, errors.New(
				fmt.Sprintf("Row parsing error: %v", err.Error()),
			)
		}

		m.Participants = append(
			m.Participants,
			ProfileSummaryType{ID: participantId},
		)
	}
	err = rows.Err()
	if err != nil {
		return HuddleType{}, http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Error fetching rows: %v", err.Error()),
		)
	}
	rows.Close()

	if m.Id == 0 {
		return HuddleType{}, http.StatusNotFound, errors.New(
			fmt.Sprintf("Resource with ID %d not found", id),
		)
	}

	m.Meta.Links =
		[]h.LinkType{
			h.GetLink("self", "", h.ItemTypeHuddle, m.Id),
		}

	// Update cache
	c.CacheSet(mcKey, m, mcTtl)

	m.FetchProfileSummaries(siteId)

	return m, http.StatusOK, nil
}

func GetHuddleTitle(id int64) string {

	// Get from cache if it's available
	mcKey := fmt.Sprintf(mcHuddleKeys[c.CacheTitle], id)
	if val, ok := c.CacheGetString(mcKey); ok {
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
	c.CacheSetString(mcKey, title, mcTtl)

	return title
}

func GetHuddleSummary(
	siteId int64,
	profileId int64,
	id int64,
) (
	HuddleSummaryType,
	int,
	error,
) {

	// Get from cache if it's available
	mcKey := fmt.Sprintf(mcHuddleKeys[c.CacheSummary], id)
	if val, ok := c.CacheGet(mcKey, HuddleSummaryType{}); ok {

		m := val.(HuddleSummaryType)

		m.FetchProfileSummaries(siteId)

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
		siteId,
		id,
		profileId,
		h.ItemTypes[h.ItemTypeHuddle],
	)
	if err != nil {
		glog.Errorf(
			"db.Query(%d, %d, %d, h.ItemTypes[h.ItemTypeHuddle]) %+v",
			siteId, id, profileId, err,
		)
		return HuddleSummaryType{}, http.StatusInternalServerError,
			errors.New("Database query failed")
	}
	defer rows.Close()

	var m HuddleSummaryType

	m = HuddleSummaryType{}
	for rows.Next() {
		var participantId int64
		err = rows.Scan(
			&m.Id,
			&m.Title,
			&m.Meta.Created,
			&m.Meta.CreatedById,
			&participantId,
			&m.LastCommentIdNullable,
			&m.LastCommentCreatedNullable,
			&m.LastCommentCreatedByNullable,
			&m.CommentCount,
		)
		if err != nil {
			glog.Errorf("rows.Scan() %+v", err)
			return HuddleSummaryType{}, http.StatusInternalServerError,
				errors.New("Row parsing error")
		}

		m.Participants = append(
			m.Participants,
			ProfileSummaryType{ID: participantId},
		)

		if m.LastCommentIdNullable.Valid {
			m.LastCommentId = m.LastCommentIdNullable.Int64
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
			errors.New("Error fetching rows")
	}
	rows.Close()

	if m.Id == 0 {
		glog.Warningf("m.Id == 0")
		return HuddleSummaryType{}, http.StatusNotFound,
			errors.New("Resource not found")
	}

	m.Meta.Links =
		[]h.LinkType{
			h.GetLink("self", "", h.ItemTypeHuddle, m.Id),
		}

	// Update cache
	c.CacheSet(mcKey, m, mcTtl)

	m.FetchProfileSummaries(siteId)

	return m, http.StatusOK, nil
}

func GetHuddles(
	siteId int64,
	profileId int64,
	limit int64,
	offset int64,
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

	var total int64
	err = db.QueryRow(`
SELECT COUNT(*) AS total
  FROM huddles h
  JOIN huddle_profiles hp ON hp.huddle_id = h.huddle_id
  LEFT JOIN ignores i ON i.profile_id = $1
                     AND i.item_type_id = 3
                     AND i.item_id = h.created_by
 WHERE hp.profile_id = $1
   AND i.profile_id IS NULL`,
		profileId,
	).Scan(&total)
	if err != nil {
		glog.Error(err)
		return []HuddleSummaryType{}, 0, 0, http.StatusInternalServerError,
			errors.New("Database query failed")
	}

	rows, err := db.Query(`
SELECT huddle_id
      ,has_unread(5, huddle_id, $1)
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
       ) r`,
		profileId,
		limit,
		offset,
	)
	if err != nil {
		glog.Errorf(
			"db.Query(%d, %d, %d, %d) %+v",
			siteId,
			profileId,
			limit,
			offset,
			err,
		)
		return []HuddleSummaryType{}, 0, 0, http.StatusInternalServerError,
			errors.New("Database query failed")
	}
	defer rows.Close()

	var ems []HuddleSummaryType

	for rows.Next() {
		var (
			id        int64
			hasUnread bool
		)
		err = rows.Scan(
			&id,
			&hasUnread,
		)
		if err != nil {
			glog.Errorf("rows.Scan() %+v", err)
			return []HuddleSummaryType{}, 0, 0, http.StatusInternalServerError,
				errors.New("Row parsing error")
		}

		m, status, err := GetHuddleSummary(siteId, profileId, id)
		if err != nil {
			glog.Errorf(
				"GetHuddleSummary(%d, %d, %d) %+v",
				siteId,
				profileId,
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
			errors.New("Error fetching rows")
	}
	rows.Close()

	pages := h.GetPageCount(total, limit)
	maxOffset := h.GetMaxOffset(total, limit)

	if offset > maxOffset {
		glog.Warningf("offset > maxOffset")
		return []HuddleSummaryType{}, 0, 0, http.StatusBadRequest,
			errors.New(
				fmt.Sprintf("not enough records, "+
					"offset (%d) would return an empty page.",
					offset,
				),
			)
	}

	return ems, total, pages, http.StatusOK, nil
}
