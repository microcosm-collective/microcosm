package models

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"sync"

	"github.com/golang/glog"

	c "github.com/microcosm-cc/microcosm/cache"
	h "github.com/microcosm-cc/microcosm/helpers"
)

type UpdatesType struct {
	Updates h.ArrayType    `json:"updates"`
	Meta    h.CoreMetaType `json:"meta"`
}

type UpdateType struct {
	Id               int64             `json:"id"`
	SiteId           int64             `json:"-"`
	ForProfileId     int64             `json:"-"`
	UpdateTypeId     int64             `json:"-"`
	UpdateType       string            `json:"updateType"`
	ItemTypeId       int64             `json:"-"`
	ItemType         string            `json:"itemType"`
	ItemId           int64             `json:"-"`
	Item             interface{}       `json:"item,omitempty"`
	ParentItemTypeId int64             `json:"-"`
	ParentItemType   string            `json:"parentItemType,omitempty"`
	ParentItemId     int64             `json:"-"`
	ParentItem       interface{}       `json:"parentItem,omitempty"`
	Meta             h.CreatedMetaType `json:"meta"`
}

func (m *UpdateType) Validate(exists bool) (int, error) {

	if m.ForProfileId < 0 {
		return http.StatusBadRequest,
			errors.New(
				fmt.Sprintf(
					"forProfileId ('%d') cannot be negative.",
					m.ForProfileId,
				),
			)
	}

	if m.UpdateTypeId < 0 {
		return http.StatusBadRequest,
			errors.New(
				fmt.Sprintf(
					"updateTypeId ('%d') cannot be negative.",
					m.UpdateTypeId,
				),
			)
	}

	return http.StatusOK, nil
}

// FetchSummaries fetches profile/item summary for a update entry.
// Called post SELECT or post-GetFromCache
func (m *UpdateType) FetchSummaries(siteId int64) (int, error) {

	profile, status, err := GetSummary(
		siteId,
		h.ItemTypes[h.ItemTypeProfile],
		m.Meta.CreatedById,
		m.ForProfileId,
	)
	if err != nil {
		return status, err
	}
	m.Meta.CreatedBy = profile

	itemSummary, status, err := GetSummary(
		siteId,
		m.ItemTypeId,
		m.ItemId,
		m.ForProfileId,
	)
	if err != nil {
		return status, err
	}
	m.Item = itemSummary

	if m.ItemTypeId == h.ItemTypes[h.ItemTypeComment] {
		comment := itemSummary.(CommentSummaryType)
		parent, status, err := GetSummary(
			siteId,
			comment.ItemTypeId,
			comment.ItemId,
			m.ForProfileId,
		)
		if err != nil {
			return status, err
		}
		m.ParentItem = parent
		m.ParentItemTypeId = comment.ItemTypeId
		parentItemType, err := h.GetMapStringFromInt(
			h.ItemTypes,
			comment.ItemTypeId,
		)
		if err != nil {
			return http.StatusInternalServerError, err
		}
		m.ParentItemType = parentItemType
		m.ParentItemId = comment.ItemId
	}

	updateType, status, err := GetUpdateType(m.UpdateTypeId)
	if err != nil {
		return status, err
	}
	m.UpdateType = updateType.Title

	return http.StatusOK, nil
}

func (m *UpdateType) Insert() (int, error) {

	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Could not start transaction: %v", err.Error()),
		)
	}
	defer tx.Rollback()

	status, err := m.insert(tx)
	if err != nil {
		return status, err
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Transaction failed: %v", err.Error()),
		)
	}

	return http.StatusOK, nil
}

// Exists to allow multiple inserts from update_dispatcher.go to be made within
// a single transaction
func (m *UpdateType) insert(tx *sql.Tx) (int, error) {

	status, err := m.Validate(false)
	if err != nil {
		return status, errors.New(
			fmt.Sprintf("Insert did not validate: %v", err.Error()),
		)
	}

	var insertId int64
	err = tx.QueryRow(`
INSERT INTO updates (
    site_id
   ,for_profile_id
   ,update_type_id
   ,item_type_id
   ,item_id

   ,created_by
   ,created
) VALUES (
   $1, $2, $3, $4, $5,
   $6, NOW()
) RETURNING update_id`,
		m.SiteId,
		m.ForProfileId,
		m.UpdateTypeId,
		m.ItemTypeId,
		m.ItemId,

		m.Meta.CreatedById,
	).Scan(
		&insertId,
	)
	if err != nil {
		return http.StatusInternalServerError,
			errors.New(
				fmt.Sprintf(
					"Error inserting data and returning ID: %v",
					err.Error(),
				),
			)
	}
	m.Id = insertId

	return http.StatusOK, nil
}

// Exists to allow multiple inserts from update_dispatcher.go to be made within
// a single transaction.
//
// This upsert method is used when an existing update *may* already exist...
// such as when a comment revision includes a mention.
//
// No update_id is returned
func (m *UpdateType) upsert(tx *sql.Tx) (int, error) {

	status, err := m.Validate(false)
	if err != nil {
		return status, errors.New(
			fmt.Sprintf("Update did not validate: %v", err.Error()),
		)
	}

	_, err = tx.Exec(`
INSERT INTO updates (
    site_id
   ,for_profile_id
   ,update_type_id
   ,item_type_id
   ,item_id

   ,created_by
   ,created
) VALUES (
    $1, $2, $3, $4, $5,
    $6, NOW()
) WHERE NOT EXISTS (
    SELECT *
      FROM updates
     WHERE site_id = $1
       AND for_profile_id = $2
       AND update_type_id = $3
       AND item_type_id = $4
       AND item_id = $5
       AND created_by = $6
)`,
		m.SiteId,
		m.ForProfileId,
		m.UpdateTypeId,
		m.ItemTypeId,
		m.ItemId,

		m.Meta.CreatedById,
	)
	if err != nil {
		return http.StatusInternalServerError,
			errors.New(
				fmt.Sprintf("Error inserting data and returning ID: %+v", err),
			)
	}

	return http.StatusOK, nil
}

func GetUpdate(
	siteId int64,
	updateId int64,
	profileId int64,
) (
	UpdateType,
	int,
	error,
) {

	// Try fetching from cache
	mcKey := fmt.Sprintf(mcUpdateKeys[c.CacheDetail], updateId)
	if val, ok := c.CacheGet(mcKey, UpdateType{}); ok {
		m := val.(UpdateType)
		m.FetchSummaries(siteId)
		return m, http.StatusOK, nil
	}

	db, err := h.GetConnection()
	if err != nil {
		return UpdateType{}, http.StatusInternalServerError, err
	}

	var m UpdateType
	err = db.QueryRow(`
SELECT update_id
      ,for_profile_id
      ,update_type_id
      ,item_type_id
      ,item_id
      ,created_by
      ,created
      ,site_id
  FROM updates
 WHERE site_id = $1
   AND update_id = $2
   AND for_profile_id = $3`,
		siteId,
		updateId,
		profileId,
	).Scan(
		&m.Id,
		&m.ForProfileId,
		&m.UpdateTypeId,
		&m.ItemTypeId,
		&m.ItemId,
		&m.Meta.CreatedById,
		&m.Meta.Created,
		&m.SiteId,
	)
	if err == sql.ErrNoRows {
		return UpdateType{}, http.StatusNotFound,
			errors.New(
				fmt.Sprintf("Update not found: %v", err.Error()),
			)
	} else if err != nil {
		return UpdateType{}, http.StatusInternalServerError,
			errors.New(
				fmt.Sprintf("Error fetching update: %v", err.Error()),
			)
	}

	itemType, err := h.GetItemTypeFromInt(m.ItemTypeId)
	if err != nil {
		return UpdateType{}, http.StatusInternalServerError, err
	}
	m.ItemType = itemType
	m.FetchSummaries(siteId)

	c.CacheSet(mcKey, m, mcTtl)
	return m, http.StatusOK, nil
}

func GetUpdates(
	siteId int64,
	profileId int64,
	limit int64,
	offset int64,
) (
	[]UpdateType,
	int64,
	int64,
	int,
	error,
) {

	db, err := h.GetConnection()
	if err != nil {
		glog.Errorf("h.GetConnection() %+v", err)
		return []UpdateType{}, 0, 0, http.StatusInternalServerError, err
	}

	sqlQuery := `--GetUpdates
WITH m AS (
    SELECT m.microcosm_id
      FROM microcosms m
      LEFT JOIN permissions_cache p ON p.site_id = m.site_id
                                   AND p.item_type_id = 2
                                   AND p.item_id = m.microcosm_id
                                   AND p.profile_id = $2
           LEFT JOIN ignores i ON i.profile_id = $2
                              AND i.item_type_id = 2
                              AND i.item_id = m.microcosm_id
     WHERE m.site_id = $1
       AND m.is_deleted IS NOT TRUE
       AND m.is_moderated IS NOT TRUE
       AND i.profile_id IS NULL
       AND (
               (p.can_read IS NOT NULL AND p.can_read IS TRUE)
            OR (get_effective_permissions($1,m.microcosm_id,2,m.microcosm_id,$2)).can_read IS TRUE
           )
)
SELECT total
      ,update_id
      ,for_profile_id
      ,update_type_id
      ,item_type_id
      ,item_id
      ,created_by
      ,created
      ,site_id
      ,has_unread(COALESCE(parent_item_type_id, item_type_id), COALESCE(parent_item_id, item_id), $2)
  FROM (
          SELECT COUNT(*) OVER() AS total
                ,rollup.update_id
                ,rollup.for_profile_id
                ,rollup.update_type_id
                ,rollup.item_type_id
                ,rollup.item_id
                ,rollup.created_by
                ,rollup.created
                ,rollup.site_id
                ,f.parent_item_type_id
                ,f.parent_item_id
            FROM flags f
                 JOIN (
                          -- 1;'new_comment';'When a comment has been posted in an item you are watching'
                          -- 4;'new_comment_in_huddle';'When you receive a new comment in a private message'
                          SELECT u.update_id
                                ,u.for_profile_id
                                ,u.update_type_id
                                ,u.item_type_id
                                ,u.item_id
                                ,u.created_by
                                ,u.created
                                ,$1 AS site_id
                            FROM updates u
                                 JOIN (
                                          SELECT MAX(u.update_id) AS update_id
                                                ,f.parent_item_type_id AS item_type_id
                                                ,f.parent_item_id AS item_id
                                            FROM updates u
                                                 JOIN flags f ON f.item_type_id = u.item_type_id
                                                             AND f.item_id = u.item_id
                                            LEFT JOIN ignores i ON i.profile_id = $2
                                                               AND (
                                                                       (i.item_type_id = 3 AND i.item_id = u.created_by)
                                                                    OR (i.item_type_id = f.parent_item_type_id AND i.item_id = f.parent_item_id)
                                                                   )
                                            LEFT JOIN huddle_profiles hp ON hp.huddle_id = f.parent_item_id
                                                                        AND hp.profile_id = u.for_profile_id
                                                                        AND f.parent_item_type_id = 5
                                           WHERE u.for_profile_id = $2
                                             AND i.profile_id IS NULL
                                             AND u.update_type_id IN (1, 4)
                                             AND f.microcosm_is_deleted IS NOT TRUE
                                             AND f.microcosm_is_moderated IS NOT TRUE
                                             AND f.item_is_deleted IS NOT TRUE
                                             AND f.item_is_moderated IS NOT TRUE
                                             AND f.parent_is_deleted IS NOT TRUE
                                             AND f.parent_is_moderated IS NOT TRUE
                                             AND (
                                                     f.microcosm_id IN (SELECT microcosm_id FROM m)
                                                  OR hp.profile_id = u.for_profile_id
                                                 )
                                           GROUP BY f.parent_item_type_id
                                                   ,f.parent_item_id
                                                   ,f.site_id
                                      ) r ON r.update_id = u.update_id
                                 JOIN watchers w ON w.profile_id = $2
                                                AND w.item_type_id = r.item_type_id
                                                AND w.item_id = r.item_id
                           UNION
                          -- 2;'reply_to_comment';'When a comment of yours is replied to'
                          -- 3;'mentioned';'When you are @mentioned in a comment'
                          SELECT u.update_id
                                ,u.for_profile_id
                                ,u.update_type_id
                                ,u.item_type_id
                                ,u.item_id
                                ,u.created_by
                                ,u.created
                                ,$1 AS site_id
                            FROM updates u
                           WHERE update_id IN (
                                     SELECT MAX(u.update_id)
                                       FROM updates u
                                            JOIN flags f ON f.item_type_id = u.item_type_id
                                                        AND f.item_id = u.item_id
                                            LEFT JOIN huddle_profiles hp ON hp.huddle_id = f.parent_item_id
                                                                        AND hp.profile_id = u.for_profile_id
                                                                        AND f.parent_item_type_id = 5
                                            LEFT JOIN ignores i ON i.profile_id = $2
                                                               AND (
                                                                       (i.item_type_id = 3 AND i.item_id = u.created_by)
                                                                    OR (i.item_type_id = f.parent_item_type_id AND i.item_id = f.parent_item_id)
                                                                   )
                                      WHERE u.for_profile_id = $2
                                        AND i.profile_id IS NULL
                                        AND (u.update_type_id = 2 OR u.update_type_id = 3) -- replies (2) & mentions (3)
                                        AND f.site_id = $1
                                        AND f.microcosm_is_deleted IS NOT TRUE
                                        AND f.microcosm_is_moderated IS NOT TRUE
                                        AND f.item_is_deleted IS NOT TRUE
                                        AND f.item_is_moderated IS NOT TRUE
                                        AND f.parent_is_deleted IS NOT TRUE
                                        AND f.parent_is_moderated IS NOT TRUE
                                        AND (
                                                f.microcosm_id IN (SELECT microcosm_id FROM m)
                                             OR hp.profile_id = u.for_profile_id
                                            )
                                      GROUP BY u.update_type_id
                                              ,u.item_type_id
                                              ,u.item_id
                                     )
                           UNION
                          -- 8;'new_item';'When a new item is created in a microcosm you are watching'
                          SELECT u.update_id
                                ,u.for_profile_id
                                ,u.update_type_id
                                ,u.item_type_id
                                ,u.item_id
                                ,u.created_by
                                ,u.created
                                ,$1 AS site_id
                            FROM updates u
                           WHERE update_id IN (
                                     SELECT MAX(u.update_id)
                                       FROM updates u
                                            JOIN flags f ON f.item_type_id = u.item_type_id
                                                        AND f.item_id = u.item_id
                                                        AND f.microcosm_id IN (SELECT microcosm_id FROM m)
                                            JOIN watchers w ON w.profile_id = $2
                                                           AND w.item_type_id = 2
                                                           AND w.item_id = f.microcosm_id
                                            LEFT JOIN ignores i ON i.profile_id = $2
                                                               AND i.item_type_id = 3
                                                               AND i.item_id = u.created_by
                                      WHERE u.for_profile_id = $2
                                        AND i.profile_id IS NULL
                                        AND u.update_type_id = 8
                                        AND f.microcosm_is_deleted IS NOT TRUE
                                        AND f.microcosm_is_moderated IS NOT TRUE
                                        AND f.item_is_deleted IS NOT TRUE
                                        AND f.item_is_moderated IS NOT TRUE
                                        AND f.parent_is_deleted IS NOT TRUE
                                        AND f.parent_is_moderated IS NOT TRUE
                                      GROUP BY u.item_type_id, u.item_id
                                 )
                          ) AS rollup ON rollup.item_type_id = f.item_type_id
                                     AND rollup.item_id = f.item_id
           ORDER BY created DESC
           LIMIT $3
          OFFSET $4
          ) final_rollup`

	rows, err := db.Query(sqlQuery, siteId, profileId, limit, offset)
	if err != nil {
		glog.Errorf(
			"db.Query(%d, %d, %d, %d) %+v",
			profileId,
			siteId,
			limit,
			offset,
			err,
		)
		return []UpdateType{}, 0, 0, http.StatusInternalServerError,
			errors.New("Database query failed")
	}
	defer rows.Close()

	var total int64
	ems := []UpdateType{}
	for rows.Next() {
		var unread bool
		m := UpdateType{}
		err = rows.Scan(
			&total,
			&m.Id,
			&m.ForProfileId,
			&m.UpdateTypeId,
			&m.ItemTypeId,
			&m.ItemId,
			&m.Meta.CreatedById,
			&m.Meta.Created,
			&m.SiteId,
			&unread,
		)
		if err != nil {
			glog.Errorf("rows.Scan() %+v", err)
			return []UpdateType{}, 0, 0, http.StatusInternalServerError,
				errors.New("Row parsing error")
		}

		itemType, err := h.GetItemTypeFromInt(m.ItemTypeId)
		if err != nil {
			glog.Errorf("h.GetItemTypeFromInt(%d) %+v", m.ItemTypeId, err)
			return []UpdateType{}, 0, 0, http.StatusInternalServerError, err
		}
		m.ItemType = itemType

		m.Meta.Flags.Unread = unread

		ems = append(ems, m)
	}
	err = rows.Err()
	if err != nil {
		glog.Errorf("rows.Err() %+v", err)
		return []UpdateType{}, 0, 0, http.StatusInternalServerError,
			errors.New("Error fetching rows")
	}
	rows.Close()

	pages := h.GetPageCount(total, limit)
	maxOffset := h.GetMaxOffset(total, limit)

	if offset > maxOffset {
		glog.Infoln("offset > maxOffset")
		return []UpdateType{}, 0, 0, http.StatusBadRequest,
			errors.New(
				fmt.Sprintf("not enough records, "+
					"offset (%d) would return an empty page.", offset),
			)
	}

	// Get the first round of summaries
	var wg1 sync.WaitGroup
	chan1 := make(chan SummaryContainerRequest)
	defer close(chan1)

	seq := 0
	for i := 0; i < len(ems); i++ {
		go HandleSummaryContainerRequest(
			siteId,
			h.ItemTypes[h.ItemTypeProfile],
			ems[i].Meta.CreatedById,
			ems[i].ForProfileId,
			seq,
			chan1,
		)
		wg1.Add(1)
		seq++

		go HandleSummaryContainerRequest(
			siteId,
			ems[i].ItemTypeId,
			ems[i].ItemId,
			ems[i].ForProfileId,
			seq,
			chan1,
		)
		wg1.Add(1)
		seq++

		updateType, status, err := GetUpdateType(ems[i].UpdateTypeId)
		if err != nil {
			return []UpdateType{}, 0, 0, status, err
		}
		ems[i].UpdateType = updateType.Title
	}

	resps := []SummaryContainerRequest{}
	for i := 0; i < seq; i++ {
		resp := <-chan1
		wg1.Done()

		resps = append(resps, resp)
	}
	wg1.Wait()

	for _, resp := range resps {
		if resp.Err != nil {
			return []UpdateType{}, 0, 0, resp.Status, resp.Err
		}
	}

	sort.Sort(SummaryContainerRequestsBySeq(resps))

	// Insert the first round of summaries, and get the summaries for the
	// comments
	var wg2 sync.WaitGroup
	chan2 := make(chan SummaryContainerRequest)
	defer close(chan2)

	seq = 0
	parentSeq := 0
	for i := 0; i < len(ems); i++ {

		ems[i].Meta.CreatedBy = resps[seq].Item.Summary
		seq++

		ems[i].Item = resps[seq].Item.Summary
		seq++

		if ems[i].ItemTypeId == h.ItemTypes[h.ItemTypeComment] {
			comment := ems[i].Item.(CommentSummaryType)

			go HandleSummaryContainerRequest(
				siteId,
				comment.ItemTypeId,
				comment.ItemId,
				ems[i].ForProfileId,
				seq,
				chan2,
			)
			parentSeq++
			wg2.Add(1)
		}
	}

	parentResps := []SummaryContainerRequest{}
	for i := 0; i < parentSeq; i++ {
		resp := <-chan2
		wg2.Done()
		parentResps = append(parentResps, resp)
	}
	wg2.Wait()

	for _, resp := range parentResps {
		if resp.Err != nil {
			return []UpdateType{}, 0, 0, resp.Status, resp.Err
		}
	}

	sort.Sort(SummaryContainerRequestsBySeq(parentResps))

	// Insert the comment summaries, and get the summaries of the items the
	// comments are attached to
	var wg3 sync.WaitGroup
	chan3 := make(chan SummaryContainerRequest)
	defer close(chan3)

	parentSeq = 0
	commentItemSeq := 0
	for i := 0; i < len(ems); i++ {

		if ems[i].ItemTypeId == h.ItemTypes[h.ItemTypeComment] {
			comment := ems[i].Item.(CommentSummaryType)

			go HandleSummaryContainerRequest(
				siteId,
				comment.ItemTypeId,
				comment.ItemId,
				ems[i].ForProfileId,
				commentItemSeq,
				chan3,
			)
			parentSeq++
			commentItemSeq++
			wg3.Add(1)

			ems[i].ParentItemTypeId = comment.ItemTypeId
			parentItemType, err := h.GetMapStringFromInt(
				h.ItemTypes,
				comment.ItemTypeId,
			)
			if err != nil {
				return []UpdateType{}, 0, 0, http.StatusInternalServerError, err
			}
			ems[i].ParentItemType = parentItemType
			ems[i].ParentItemId = comment.ItemId
		}
	}

	commentResps := []SummaryContainerRequest{}
	for i := 0; i < commentItemSeq; i++ {
		resp := <-chan3
		wg3.Done()
		commentResps = append(commentResps, resp)
	}
	wg3.Wait()

	for _, resp := range commentResps {
		if resp.Err != nil {
			return []UpdateType{}, 0, 0, resp.Status, resp.Err
		}
	}

	sort.Sort(SummaryContainerRequestsBySeq(commentResps))

	commentItemSeq = 0
	for i := 0; i < len(ems); i++ {
		if ems[i].ItemTypeId == h.ItemTypes[h.ItemTypeComment] {
			ems[i].ParentItem = commentResps[commentItemSeq].Item.Summary
			commentItemSeq++
		}
	}

	return ems, total, pages, http.StatusOK, nil
}
