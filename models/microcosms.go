package models

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/lib/pq"

	c "github.com/microcosm-cc/microcosm/cache"
	h "github.com/microcosm-cc/microcosm/helpers"
)

type MicrocosmsType struct {
	Microcosms h.ArrayType    `json:"microcosms"`
	Meta       h.CoreMetaType `json:"meta"`
}

type MicrocosmSummaryType struct {
	Id           int64   `json:"id"`
	SiteId       int64   `json:"siteId,omitempty"`
	Visibility   string  `json:"visibility"`
	Title        string  `json:"title"`
	Description  string  `json:"description"`
	Moderators   []int64 `json:"moderators"`
	ItemCount    int64   `json:"totalItems"`
	CommentCount int64   `json:"totalComments"`

	MRU interface{} `json:"mostRecentUpdate,omitempty"`

	Meta h.SummaryMetaType `json:"meta"`
}

type MicrocosmType struct {
	Id          int64  `json:"id"`
	SiteId      int64  `json:"siteId,omitempty"`
	Visibility  string `json:"visibility"`
	Title       string `json:"title"`
	Description string `json:"description"`
	OwnedById   int64  `json:"-"`

	Moderators []int64 `json:"moderators"`

	Items h.ArrayType       `json:"items"`
	Meta  h.DefaultMetaType `json:"meta"`
}

type MicrocosmSummaryRequest struct {
	Item   MicrocosmSummaryType
	Err    error
	Status int
	Seq    int
}

type MicrocosmSummaryRequestBySeq []MicrocosmSummaryRequest

func (v MicrocosmSummaryRequestBySeq) Len() int {
	return len(v)
}

func (v MicrocosmSummaryRequestBySeq) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

func (v MicrocosmSummaryRequestBySeq) Less(i, j int) bool {
	return v[i].Seq < v[j].Seq
}

func (m *MicrocosmType) Validate(exists bool, isImport bool) (int, error) {

	m.Title = SanitiseText(m.Title)
	m.Description = string(SanitiseHTML([]byte(m.Description)))

	if exists && !isImport {
		if strings.Trim(m.Meta.EditReason, " ") == "" ||
			len(m.Meta.EditReason) == 0 {

			return http.StatusBadRequest,
				errors.New("You must provide a reason for the update")
		}
	}

	if !(m.Visibility == "public" ||
		m.Visibility == "protected" ||
		m.Visibility == "private") {

		m.Visibility = "public"
	}

	if strings.Trim(m.Title, " ") == "" {
		return http.StatusBadRequest, errors.New("Title is a required field")
	}

	m.Title = ShoutToWhisper(m.Title)

	if strings.Trim(m.Description, " ") == "" {
		return http.StatusBadRequest,
			errors.New("Description is a required field")
	}

	m.Description = ShoutToWhisper(m.Description)

	return http.StatusOK, nil
}

func (m *MicrocosmType) FetchSummaries(
	siteId int64,
	profileId int64,
) (
	int,
	error,
) {

	profile, status, err := GetProfileSummary(siteId, m.Meta.CreatedById)
	if err != nil {
		return status, err
	}
	m.Meta.CreatedBy = profile

	if m.Meta.EditedByNullable.Valid {
		profile, status, err :=
			GetProfileSummary(siteId, m.Meta.EditedByNullable.Int64)
		if err != nil {
			return status, err
		}
		m.Meta.EditedBy = profile
	}

	db, err := h.GetConnection()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	var unread bool
	err = db.QueryRow(
		`SELECT has_unread(2, $1, $2)`,
		m.Id,
		profileId,
	).Scan(
		&unread,
	)
	if err != nil {
		glog.Errorf("db.QueryRow() %+v", err)
		return http.StatusInternalServerError, err
	}
	m.Meta.Flags.Unread = unread

	return http.StatusOK, nil
}

func (m *MicrocosmSummaryType) FetchProfileSummaries(
	siteId int64,
) (
	int,
	error,
) {

	profile, status, err := GetProfileSummary(siteId, m.Meta.CreatedById)
	if err != nil {
		return status, err
	}
	m.Meta.CreatedBy = profile

	return http.StatusOK, nil
}

func (m *MicrocosmType) Insert() (int, error) {
	status, err := m.Validate(false, false)
	if err != nil {
		return status, err
	}

	return m.insert()
}

func (m *MicrocosmType) Import() (int, error) {
	status, err := m.Validate(true, true)
	if err != nil {
		return status, err
	}

	return m.insert()
}

func (m *MicrocosmType) insert() (int, error) {

	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	var insertId int64
	err = tx.QueryRow(`-- Create Microcosm
INSERT INTO microcosms (
    site_id, visibility, title, description, created,
    created_by, owned_by
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7
) RETURNING microcosm_id`,
		m.SiteId,
		m.Visibility,
		m.Title,
		m.Description,
		m.Meta.Created,
		m.Meta.CreatedById,
		m.OwnedById,
	).Scan(
		&insertId,
	)
	if err != nil {
		return http.StatusInternalServerError,
			errors.New(
				fmt.Sprintf("Error inserting data and returning ID: %+v", err),
			)
	}
	m.Id = insertId

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Transaction failed: %v", err.Error()),
		)
	}

	PurgeCache(h.ItemTypes[h.ItemTypeMicrocosm], m.Id)

	return http.StatusOK, nil
}

func (m *MicrocosmType) Update() (int, error) {

	status, err := m.Validate(true, false)
	if err != nil {
		return status, err
	}

	// Update resource
	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`-- Update Microcosm
UPDATE microcosms
   SET site_id = $2,
       visibility = $3,
       title = $4,
       description = $5,
       edited = $6,
       edited_by = $7,
       edit_reason = $8
 WHERE microcosm_id = $1`,
		m.Id,
		m.SiteId,
		m.Visibility,
		m.Title,
		m.Description,
		m.Meta.EditedNullable,
		m.Meta.EditedByNullable,
		m.Meta.EditReason,
	)
	if err != nil {
		return http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Update failed: %v", err.Error()),
		)
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Transaction failed: %v", err.Error()),
		)
	}

	PurgeCache(h.ItemTypes[h.ItemTypeMicrocosm], m.Id)

	return http.StatusOK, nil
}

func (m *MicrocosmType) Patch(
	ac AuthContext,
	patches []h.PatchType,
) (
	int,
	error,
) {

	// Update resource
	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	for _, patch := range patches {

		m.Meta.EditedNullable = pq.NullTime{Time: time.Now(), Valid: true}
		m.Meta.EditedByNullable = sql.NullInt64{Int64: ac.ProfileId, Valid: true}

		var column string
		patch.ScanRawValue()
		switch patch.Path {
		case "/meta/flags/sticky":
			column = "is_sticky"
			m.Meta.Flags.Sticky = patch.Bool.Bool
			m.Meta.EditReason =
				fmt.Sprintf("Set sticky to %t", m.Meta.Flags.Sticky)
		case "/meta/flags/open":
			column = "is_open"
			m.Meta.Flags.Open = patch.Bool.Bool
			m.Meta.EditReason =
				fmt.Sprintf("Set open to %t", m.Meta.Flags.Open)
		case "/meta/flags/deleted":
			column = "is_deleted"
			m.Meta.Flags.Deleted = patch.Bool.Bool
			m.Meta.EditReason =
				fmt.Sprintf("Set delete to %t", m.Meta.Flags.Deleted)
		case "/meta/flags/moderated":
			column = "is_moderated"
			m.Meta.Flags.Moderated = patch.Bool.Bool
			m.Meta.EditReason =
				fmt.Sprintf("Set moderated to %t", m.Meta.Flags.Moderated)
		default:
			return http.StatusBadRequest,
				errors.New("Unsupported path in patch replace operation")
		}

		m.Meta.Flags.SetVisible()

		_, err = tx.Exec(`-- Update Microcosm Flags
UPDATE microcosms
   SET `+column+` = $2
      ,is_visible = $3
      ,edited = $4
      ,edited_by = $5
      ,edit_reason = $6
 WHERE microcosm_id = $1`,
			m.Id,
			patch.Bool.Bool,
			m.Meta.Flags.Visible,
			m.Meta.EditedNullable,
			m.Meta.EditedByNullable,
			m.Meta.EditReason,
		)
		if err != nil {
			return http.StatusInternalServerError, errors.New(
				fmt.Sprintf("Update failed: %v", err.Error()),
			)
		}
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Transaction failed: %v", err.Error()),
		)
	}

	PurgeCache(h.ItemTypes[h.ItemTypeMicrocosm], m.Id)

	return http.StatusOK, nil
}

func (m *MicrocosmType) Delete() (int, error) {

	// Delete resource
	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`-- Delete Microcosm
UPDATE microcosms
   SET is_deleted = true
 WHERE microcosm_id = $1`,
		m.Id,
	)
	if err != nil {
		tx.Rollback()
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

	PurgeCache(h.ItemTypes[h.ItemTypeMicrocosm], m.Id)

	return http.StatusOK, nil
}

func GetMicrocosm(
	siteId int64,
	id int64,
	profileId int64,
) (
	MicrocosmType,
	int,
	error,
) {

	if id == 0 {
		return MicrocosmType{}, http.StatusNotFound,
			errors.New("Microcosm not found")
	}

	// Get from cache if it's available
	mcKey := fmt.Sprintf(mcMicrocosmKeys[c.CacheDetail], id)
	if val, ok := c.CacheGet(mcKey, MicrocosmType{}); ok {

		m := val.(MicrocosmType)
		if m.SiteId != siteId {
			return MicrocosmType{}, http.StatusNotFound, errors.New("Not found")
		}

		m.FetchSummaries(siteId, profileId)

		return m, 0, nil
	}

	// Retrieve resource
	db, err := h.GetConnection()
	if err != nil {
		return MicrocosmType{}, http.StatusInternalServerError, err
	}

	// TODO(buro9): admins and mods could see this with isDeleted=true in the querystring
	var m MicrocosmType
	err = db.QueryRow(`--GetMicrocosm
SELECT microcosm_id,
       site_id,
       visibility,
       title,
       description,
       created,
       created_by,
       edited,
       edited_by,
       edit_reason,
       is_sticky,
       is_open,
       is_deleted,
       is_moderated,
       is_visible
  FROM microcosms
 WHERE site_id = $1
   AND microcosm_id = $2
   AND is_deleted IS NOT TRUE
   AND is_moderated IS NOT TRUE`,
		siteId,
		id,
	).Scan(
		&m.Id,
		&m.SiteId,
		&m.Visibility,
		&m.Title,
		&m.Description,
		&m.Meta.Created,
		&m.Meta.CreatedById,
		&m.Meta.EditedNullable,
		&m.Meta.EditedByNullable,
		&m.Meta.EditReasonNullable,
		&m.Meta.Flags.Sticky,
		&m.Meta.Flags.Open,
		&m.Meta.Flags.Deleted,
		&m.Meta.Flags.Moderated,
		&m.Meta.Flags.Visible,
	)
	if err == sql.ErrNoRows {
		return MicrocosmType{}, http.StatusNotFound, errors.New(
			fmt.Sprintf("Resource with ID %d not found", id),
		)
	} else if err != nil {
		return MicrocosmType{}, http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Database query failed: %v", err.Error()),
		)
	}

	if m.Meta.EditReasonNullable.Valid {
		m.Meta.EditReason = m.Meta.EditReasonNullable.String
	}
	if m.Meta.EditedNullable.Valid {
		m.Meta.Edited =
			m.Meta.EditedNullable.Time.Format(time.RFC3339Nano)
	}
	m.Meta.Links =
		[]h.LinkType{
			h.GetLink("self", "", h.ItemTypeMicrocosm, m.Id),
			h.GetLink("site", "", h.ItemTypeSite, m.SiteId),
		}

	// Update cache
	c.CacheSet(mcKey, m, mcTtl)

	m.FetchSummaries(siteId, profileId)
	return m, http.StatusOK, nil
}

func HandleMicrocosmSummaryRequest(
	siteId int64,
	id int64,
	profileId int64,
	seq int,
	out chan<- MicrocosmSummaryRequest,
) {

	item, status, err := GetMicrocosmSummary(siteId, id, profileId)

	response := MicrocosmSummaryRequest{
		Item:   item,
		Status: status,
		Err:    err,
		Seq:    seq,
	}
	out <- response
}

func GetMicrocosmSummary(
	siteId int64,
	id int64,
	profileId int64,
) (
	MicrocosmSummaryType,
	int,
	error,
) {

	if id == 0 {
		return MicrocosmSummaryType{}, http.StatusNotFound,
			errors.New("Microcosm not found")
	}

	// Get from cache if it's available
	mcKey := fmt.Sprintf(mcMicrocosmKeys[c.CacheSummary], id)
	if val, ok := c.CacheGet(mcKey, MicrocosmSummaryType{}); ok {

		m := val.(MicrocosmSummaryType)

		if m.SiteId != siteId {
			return MicrocosmSummaryType{}, http.StatusNotFound,
				errors.New("Not found")
		}

		m.FetchProfileSummaries(siteId)

		return m, http.StatusOK, nil
	}

	// Retrieve resource
	db, err := h.GetConnection()
	if err != nil {
		glog.Error(err)
		return MicrocosmSummaryType{}, http.StatusInternalServerError, err
	}

	// TODO(buro9): admins and mods could see this with isDeleted=true in the querystring
	var m MicrocosmSummaryType
	err = db.QueryRow(`--GetMicrocosmSummary
SELECT microcosm_id
      ,site_id
      ,visibility
      ,title
      ,description
      ,created
      ,created_by
      ,is_sticky
      ,is_open
      ,is_deleted
      ,is_moderated
      ,is_visible
      ,item_count
      ,comment_count
  FROM microcosms
 WHERE site_id = $1
   AND microcosm_id = $2
   AND is_deleted IS NOT TRUE
   AND is_moderated IS NOT TRUE`,
		siteId,
		id,
	).Scan(
		&m.Id,
		&m.SiteId,
		&m.Visibility,
		&m.Title,
		&m.Description,
		&m.Meta.Created,
		&m.Meta.CreatedById,
		&m.Meta.Flags.Sticky,
		&m.Meta.Flags.Open,
		&m.Meta.Flags.Deleted,
		&m.Meta.Flags.Moderated,
		&m.Meta.Flags.Visible,
		&m.ItemCount,
		&m.CommentCount,
	)
	if err == sql.ErrNoRows {
		glog.Warning(err)
		return MicrocosmSummaryType{},
			http.StatusNotFound,
			errors.New(fmt.Sprintf("Microcosm with ID %d not found", id))

	} else if err != nil {
		glog.Error(err)
		return MicrocosmSummaryType{},
			http.StatusInternalServerError,
			errors.New("Database query failed")
	}

	mru, status, err := GetMostRecentItem(siteId, m.Id, profileId)
	if err != nil {
		glog.Error(err)
		return MicrocosmSummaryType{}, status, errors.New("Row parsing error")
	}
	if mru.Valid {
		m.MRU = mru
	}
	m.Meta.Links =
		[]h.LinkType{
			h.GetLink("self", "", h.ItemTypeMicrocosm, m.Id),
			h.GetLink("site", "", h.ItemTypeSite, m.SiteId),
		}

	// Update cache
	c.CacheSet(mcKey, m, mcTtl)

	m.FetchProfileSummaries(siteId)

	return m, http.StatusOK, nil
}

func GetMicrocosmTitle(id int64) string {

	// Get from cache if it's available
	mcKey := fmt.Sprintf(mcMicrocosmKeys[c.CacheTitle], id)
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
	err = db.QueryRow(`--GetMicrocosmTitle
SELECT title
  FROM microcosms
 WHERE microcosm_id = $1`,
		id,
	).Scan(
		&title,
	)
	if err != nil {
		glog.Errorf("db.QueryRow(%d).Scan(&title) %+v", id, err)
		return ""
	}

	// Update cache
	c.CacheSetString(mcKey, title, mcTtl)

	return title
}

func GetMicrocosmIdForItem(itemTypeId int64, itemId int64) int64 {
	db, err := h.GetConnection()
	if err != nil {
		glog.Error(err)
		return 0
	}

	var microcosmId int64
	err = db.QueryRow(`--GetMicrocosmIdForItem
SELECT microcosm_id
  FROM flags
 WHERE item_type_id = $1
   AND item_id = $2`,
		itemTypeId,
		itemId,
	).Scan(
		&microcosmId,
	)
	if err != nil {
		glog.Error(err)
		return 0
	}

	return microcosmId
}

func IncrementMicrocosmItemCount(tx *sql.Tx, microcosmId int64) error {
	_, err := tx.Exec(`--Update Microcosm Item Count
UPDATE microcosms
   SET item_count = item_count + 1
 WHERE microcosm_id = $1`,
		microcosmId,
	)
	if err != nil {
		glog.Error(err)
		return err
	}

	return nil
}

func DecrementMicrocosmItemCount(tx *sql.Tx, microcosmId int64) error {
	_, err := tx.Exec(`--Update Microcosm Item Count
UPDATE microcosms
   SET item_count = item_count - 1
 WHERE microcosm_id = $1`,
		microcosmId,
	)
	if err != nil {
		glog.Error(err)
		return err
	}

	return nil
}

func GetMicrocosms(
	siteId int64,
	profileId int64,
	limit int64,
	offset int64,
) (
	[]MicrocosmSummaryType,
	int64,
	int64,
	int,
	error,
) {

	// Retrieve resources
	db, err := h.GetConnection()
	if err != nil {
		glog.Errorf("h.GetConnection() %+v", err)
		return []MicrocosmSummaryType{}, 0, 0,
			http.StatusInternalServerError, err
	}

	rows, err := db.Query(`--GetMicrocosms
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
SELECT (SELECT COUNT(*) FROM m) AS total
      ,microcosm_id
      ,has_unread(2, microcosm_id, $2)
  FROM (
           SELECT microcosm_id
             FROM microcosms
            WHERE microcosm_id IN (SELECT microcosm_id FROM m)
            ORDER BY is_sticky DESC
                    ,comment_count DESC
                    ,item_count DESC
                    ,created ASC
            LIMIT $3
           OFFSET $4
       ) r`,
		siteId,
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
		return []MicrocosmSummaryType{}, 0, 0,
			http.StatusInternalServerError,
			errors.New("Database query failed")
	}
	defer rows.Close()

	// Get a list of the identifiers of the items to return
	var total int64
	ids := []int64{}
	unread := map[int64]bool{}
	for rows.Next() {
		var (
			id        int64
			hasUnread bool
		)
		err = rows.Scan(
			&total,
			&id,
			&hasUnread,
		)
		if err != nil {
			glog.Errorf("rows.Scan() %+v", err)
			return []MicrocosmSummaryType{}, 0, 0,
				http.StatusInternalServerError,
				errors.New("Row parsing error")
		}

		unread[id] = hasUnread
		ids = append(ids, id)
	}
	err = rows.Err()
	if err != nil {
		glog.Errorf("rows.Err() %+v", err)
		return []MicrocosmSummaryType{}, 0, 0,
			http.StatusInternalServerError,
			errors.New("Error fetching rows")
	}
	rows.Close()

	// Make a request for each identifier
	var wg1 sync.WaitGroup
	req := make(chan MicrocosmSummaryRequest)
	defer close(req)

	for seq, id := range ids {
		go HandleMicrocosmSummaryRequest(siteId, id, profileId, seq, req)
		wg1.Add(1)
	}

	// Receive the responses and check for errors
	resps := []MicrocosmSummaryRequest{}
	for i := 0; i < len(ids); i++ {
		resp := <-req
		wg1.Done()
		resps = append(resps, resp)
	}
	wg1.Wait()

	for _, resp := range resps {
		if resp.Err != nil {
			return []MicrocosmSummaryType{}, 0, 0,
				http.StatusInternalServerError, resp.Err
		}
	}

	// Sort them
	sort.Sort(MicrocosmSummaryRequestBySeq(resps))

	// Extract the values
	ems := []MicrocosmSummaryType{}
	for _, resp := range resps {
		m := resp.Item
		m.Meta.Flags.Unread = unread[m.Id]
		ems = append(ems, m)
	}

	pages := h.GetPageCount(total, limit)
	maxOffset := h.GetMaxOffset(total, limit)

	if offset > maxOffset {
		return []MicrocosmSummaryType{}, 0, 0,
			http.StatusBadRequest,
			errors.New(fmt.Sprintf("not enough records, "+
				"offset (%d) would return an empty page.", offset))
	}

	return ems, total, pages, http.StatusOK, nil
}
