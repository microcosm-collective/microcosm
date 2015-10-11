package models

import (
	"database/sql"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/lib/pq"

	c "github.com/microcosm-cc/microcosm/cache"
	h "github.com/microcosm-cc/microcosm/helpers"
)

// MicrocosmsType is a collection of microcosms
type MicrocosmsType struct {
	Microcosms h.ArrayType    `json:"microcosms"`
	Meta       h.CoreMetaType `json:"meta"`
}

// MicrocosmSummaryType is a summary of a microcosm
type MicrocosmSummaryType struct {
	ID               int64 `json:"id"`
	ParentID         int64 `json:"parentId,omitempty"`
	parentIDNullable sql.NullInt64
	SiteID           int64    `json:"siteId,omitempty"`
	Visibility       string   `json:"visibility"`
	Title            string   `json:"title"`
	Description      string   `json:"description"`
	Moderators       []int64  `json:"moderators"`
	ItemCount        int64    `json:"totalItems"`
	CommentCount     int64    `json:"totalComments"`
	ItemTypes        []string `json:"itemTypes"`

	MRU interface{} `json:"mostRecentUpdate,omitempty"`

	Meta h.SummaryMetaType `json:"meta"`
}

// MicrocosmType is a microcosm
type MicrocosmType struct {
	ID               int64 `json:"id"`
	ParentID         int64 `json:"parentId,omitempty"`
	parentIDNullable sql.NullInt64
	SiteID           int64    `json:"siteId,omitempty"`
	Visibility       string   `json:"visibility"`
	Title            string   `json:"title"`
	Description      string   `json:"description"`
	OwnedByID        int64    `json:"-"`
	ItemTypes        []string `json:"itemTypes"`

	Moderators []int64 `json:"moderators"`

	Items h.ArrayType       `json:"items"`
	Meta  h.DefaultMetaType `json:"meta"`
}

// MicrocosmSummaryRequest is an envelope for a microcosm summary
type MicrocosmSummaryRequest struct {
	Item   MicrocosmSummaryType
	Err    error
	Status int
	Seq    int
}

// MicrocosmSummaryRequestBySeq is a collection of requests
type MicrocosmSummaryRequestBySeq []MicrocosmSummaryRequest

// Len returns the length of the collection
func (v MicrocosmSummaryRequestBySeq) Len() int {
	return len(v)
}

// Swap changes the position of two items in the collection
func (v MicrocosmSummaryRequestBySeq) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

// Less determines which item is sequenced lower than the other
func (v MicrocosmSummaryRequestBySeq) Less(i, j int) bool {
	return v[i].Seq < v[j].Seq
}

// Validate returns true if the microcosm is valid
func (m *MicrocosmType) Validate(exists bool, isImport bool) (int, error) {
	m.Title = SanitiseText(m.Title)
	m.Description = string(SanitiseHTML([]byte(m.Description)))

	if exists && !isImport {
		if strings.Trim(m.Meta.EditReason, " ") == "" ||
			len(m.Meta.EditReason) == 0 {

			return http.StatusBadRequest,
				fmt.Errorf("You must provide a reason for the update")
		}
	}

	if !(m.Visibility == "public" ||
		m.Visibility == "protected" ||
		m.Visibility == "private") {

		m.Visibility = "public"
	}

	if strings.Trim(m.Title, " ") == "" {
		return http.StatusBadRequest, fmt.Errorf("Title is a required field")
	}

	m.Title = ShoutToWhisper(m.Title)

	if strings.Trim(m.Description, " ") == "" {
		return http.StatusBadRequest,
			fmt.Errorf("Description is a required field")
	}

	m.Description = ShoutToWhisper(m.Description)

	if m.ParentID > 0 {
		m.parentIDNullable.Valid = true
		m.parentIDNullable.Int64 = m.ParentID
	}

	return http.StatusOK, nil
}

// FetchSummaries populates a partially populated microcosm struct
func (m *MicrocosmType) FetchSummaries(
	siteID int64,
	profileID int64,
) (
	int,
	error,
) {

	profile, status, err := GetProfileSummary(siteID, m.Meta.CreatedByID)
	if err != nil {
		return status, err
	}
	m.Meta.CreatedBy = profile

	if m.Meta.EditedByNullable.Valid {
		profile, status, err :=
			GetProfileSummary(siteID, m.Meta.EditedByNullable.Int64)
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
		m.ID,
		profileID,
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

// FetchSummaries populates a partially populated struct
func (m *MicrocosmSummaryType) FetchSummaries(siteID int64) (int, error) {

	profile, status, err := GetProfileSummary(siteID, m.Meta.CreatedByID)
	if err != nil {
		return status, err
	}
	m.Meta.CreatedBy = profile

	return http.StatusOK, nil
}

// Insert saves the microcosm to the database
func (m *MicrocosmType) Insert() (int, error) {
	status, err := m.Validate(false, false)
	if err != nil {
		return status, err
	}

	return m.insert()
}

// Import saves the microcosm to the database
func (m *MicrocosmType) Import() (int, error) {
	status, err := m.Validate(true, true)
	if err != nil {
		return status, err
	}

	return m.insert()
}

// insert saves the microcosm to the database
func (m *MicrocosmType) insert() (int, error) {
	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	var insertID int64
	err = tx.QueryRow(`-- Create Microcosm
INSERT INTO microcosms (
    site_id, visibility, title, description, created,
    created_by, owned_by, item_types, parent_id
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9
) RETURNING microcosm_id`,
		m.SiteID,
		m.Visibility,
		m.Title,
		m.Description,
		m.Meta.Created,
		m.Meta.CreatedByID,
		m.OwnedByID,
		itemTypesToPSQLArray(m.ItemTypes),
		m.parentIDNullable,
	).Scan(
		&insertID,
	)
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("Error inserting data and returning ID: %+v", err)
	}
	m.ID = insertID

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("Transaction failed: %v", err.Error())
	}

	PurgeCache(h.ItemTypes[h.ItemTypeMicrocosm], m.ID)

	return http.StatusOK, nil
}

// Update saves changes to the microcosm
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
       edit_reason = $8,
       item_types = $9,
       parent_id = $10
 WHERE microcosm_id = $1`,
		m.ID,
		m.SiteID,
		m.Visibility,
		m.Title,
		m.Description,
		m.Meta.EditedNullable,
		m.Meta.EditedByNullable,
		m.Meta.EditReason,
		itemTypesToPSQLArray(m.ItemTypes),
		m.parentIDNullable,
	)
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("Update failed: %v", err.Error())
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("Transaction failed: %v", err.Error())
	}

	PurgeCache(h.ItemTypes[h.ItemTypeMicrocosm], m.ID)

	return http.StatusOK, nil
}

// Patch partially updates a microcosm
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
		m.Meta.EditedByNullable = sql.NullInt64{Int64: ac.ProfileID, Valid: true}

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
				fmt.Errorf("Unsupported path in patch replace operation")
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
			m.ID,
			patch.Bool.Bool,
			m.Meta.Flags.Visible,
			m.Meta.EditedNullable,
			m.Meta.EditedByNullable,
			m.Meta.EditReason,
		)
		if err != nil {
			return http.StatusInternalServerError,
				fmt.Errorf("Update failed: %v", err.Error())
		}
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("Transaction failed: %v", err.Error())
	}

	PurgeCache(h.ItemTypes[h.ItemTypeMicrocosm], m.ID)

	return http.StatusOK, nil
}

// Delete removes a microcosm from the database
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
		m.ID,
	)
	if err != nil {
		tx.Rollback()
		return http.StatusInternalServerError,
			fmt.Errorf("Delete failed: %v", err.Error())
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("Transaction failed: %v", err.Error())
	}

	PurgeCache(h.ItemTypes[h.ItemTypeMicrocosm], m.ID)

	return http.StatusOK, nil
}

// GetMicrocosm fetches a microcosm
func GetMicrocosm(
	siteID int64,
	id int64,
	profileID int64,
) (
	MicrocosmType,
	int,
	error,
) {

	if id == 0 {
		return MicrocosmType{}, http.StatusNotFound,
			fmt.Errorf("Microcosm not found")
	}

	// Get from cache if it's available
	mcKey := fmt.Sprintf(mcMicrocosmKeys[c.CacheDetail], id)
	if val, ok := c.Get(mcKey, MicrocosmType{}); ok {

		m := val.(MicrocosmType)
		if m.SiteID != siteID {
			return MicrocosmType{}, http.StatusNotFound, fmt.Errorf("Not found")
		}

		m.FetchSummaries(siteID, profileID)

		return m, 0, nil
	}

	// Retrieve resource
	db, err := h.GetConnection()
	if err != nil {
		return MicrocosmType{}, http.StatusInternalServerError, err
	}

	// TODO(buro9): admins and mods could see this with isDeleted=true in the querystring
	var (
		m         MicrocosmType
		itemTypes string
	)
	err = db.QueryRow(`--GetMicrocosm
SELECT microcosm_id,
       parent_id,
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
       is_visible,
       item_types
  FROM microcosms
 WHERE site_id = $1
   AND microcosm_id = $2
   AND is_deleted IS NOT TRUE
   AND is_moderated IS NOT TRUE`,
		siteID,
		id,
	).Scan(
		&m.ID,
		&m.parentIDNullable,
		&m.SiteID,
		&m.Visibility,
		&m.Title,
		&m.Description,
		&m.Meta.Created,
		&m.Meta.CreatedByID,
		&m.Meta.EditedNullable,
		&m.Meta.EditedByNullable,
		&m.Meta.EditReasonNullable,
		&m.Meta.Flags.Sticky,
		&m.Meta.Flags.Open,
		&m.Meta.Flags.Deleted,
		&m.Meta.Flags.Moderated,
		&m.Meta.Flags.Visible,
		&itemTypes,
	)
	if err == sql.ErrNoRows {
		return MicrocosmType{}, http.StatusNotFound,
			fmt.Errorf("Resource with ID %d not found", id)

	} else if err != nil {
		return MicrocosmType{}, http.StatusInternalServerError,
			fmt.Errorf("Database query failed: %v", err.Error())
	}

	m.ItemTypes = psqlArrayToItemTypes(itemTypes)

	if m.parentIDNullable.Valid {
		m.ParentID = m.parentIDNullable.Int64
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
			h.GetLink("self", "", h.ItemTypeMicrocosm, m.ID),
			h.GetLink("site", "", h.ItemTypeSite, m.SiteID),
		}

	// Update cache
	c.Set(mcKey, m, mcTTL)

	m.FetchSummaries(siteID, profileID)
	return m, http.StatusOK, nil
}

// HandleMicrocosmSummaryRequest provides a wrapper to fetching a summary
func HandleMicrocosmSummaryRequest(
	siteID int64,
	id int64,
	profileID int64,
	seq int,
	out chan<- MicrocosmSummaryRequest,
) {

	item, status, err := GetMicrocosmSummary(siteID, id, profileID)

	response := MicrocosmSummaryRequest{
		Item:   item,
		Status: status,
		Err:    err,
		Seq:    seq,
	}
	out <- response
}

// GetMicrocosmSummary fetches a summary of a microcosm
func GetMicrocosmSummary(
	siteID int64,
	id int64,
	profileID int64,
) (
	MicrocosmSummaryType,
	int,
	error,
) {

	if id == 0 {
		return MicrocosmSummaryType{}, http.StatusNotFound,
			fmt.Errorf("Microcosm not found")
	}

	// Get from cache if it's available
	mcKey := fmt.Sprintf(mcMicrocosmKeys[c.CacheSummary], id)
	if val, ok := c.Get(mcKey, MicrocosmSummaryType{}); ok {

		m := val.(MicrocosmSummaryType)

		if m.SiteID != siteID {
			return MicrocosmSummaryType{}, http.StatusNotFound,
				fmt.Errorf("Not found")
		}

		m.FetchSummaries(siteID)

		return m, http.StatusOK, nil
	}

	// Retrieve resource
	db, err := h.GetConnection()
	if err != nil {
		glog.Error(err)
		return MicrocosmSummaryType{}, http.StatusInternalServerError, err
	}

	// TODO(buro9): admins and mods could see this with isDeleted=true in the querystring
	var (
		m         MicrocosmSummaryType
		itemTypes string
	)
	err = db.QueryRow(`--GetMicrocosmSummary
SELECT microcosm_id
      ,parent_id
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
      ,item_types
  FROM microcosms
 WHERE site_id = $1
   AND microcosm_id = $2
   AND is_deleted IS NOT TRUE
   AND is_moderated IS NOT TRUE`,
		siteID,
		id,
	).Scan(
		&m.ID,
		&m.parentIDNullable,
		&m.SiteID,
		&m.Visibility,
		&m.Title,
		&m.Description,
		&m.Meta.Created,
		&m.Meta.CreatedByID,
		&m.Meta.Flags.Sticky,
		&m.Meta.Flags.Open,
		&m.Meta.Flags.Deleted,
		&m.Meta.Flags.Moderated,
		&m.Meta.Flags.Visible,
		&m.ItemCount,
		&m.CommentCount,
		&itemTypes,
	)
	if err == sql.ErrNoRows {
		glog.Warning(err)
		return MicrocosmSummaryType{},
			http.StatusNotFound,
			fmt.Errorf("Microcosm with ID %d not found", id)

	} else if err != nil {
		glog.Error(err)
		return MicrocosmSummaryType{},
			http.StatusInternalServerError,
			fmt.Errorf("Database query failed")
	}

	m.ItemTypes = psqlArrayToItemTypes(itemTypes)

	if m.parentIDNullable.Valid {
		m.ParentID = m.parentIDNullable.Int64
	}

	mru, status, err := GetMostRecentItem(siteID, m.ID, profileID)
	if err != nil {
		glog.Error(err)
		return MicrocosmSummaryType{}, status, fmt.Errorf("Row parsing error")
	}
	if mru.Valid {
		m.MRU = mru
	}
	m.Meta.Links =
		[]h.LinkType{
			h.GetLink("self", "", h.ItemTypeMicrocosm, m.ID),
			h.GetLink("site", "", h.ItemTypeSite, m.SiteID),
		}

	// Update cache
	c.Set(mcKey, m, mcTTL)

	m.FetchSummaries(siteID)

	return m, http.StatusOK, nil
}

// GetMicrocosmTitle provides a cheap way to retrieve the title
func GetMicrocosmTitle(id int64) string {

	// Get from cache if it's available
	mcKey := fmt.Sprintf(mcMicrocosmKeys[c.CacheTitle], id)
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
	c.SetString(mcKey, title, mcTTL)

	return title
}

// GetMicrocosmIDForItem provides a cheap way to fetch an id for an item
func GetMicrocosmIDForItem(itemTypeID int64, itemID int64) int64 {
	db, err := h.GetConnection()
	if err != nil {
		glog.Error(err)
		return 0
	}

	var microcosmID int64
	err = db.QueryRow(`--GetMicrocosmIdForItem
SELECT microcosm_id
  FROM flags
 WHERE item_type_id = $1
   AND item_id = $2`,
		itemTypeID,
		itemID,
	).Scan(
		&microcosmID,
	)
	if err != nil {
		glog.Error(err)
		return 0
	}

	return microcosmID
}

// IncrementMicrocosmItemCount adds an item to the microcosm
func IncrementMicrocosmItemCount(tx *sql.Tx, microcosmID int64) error {
	_, err := tx.Exec(`--Update Microcosm Item Count
UPDATE microcosms
   SET item_count = item_count + 1
 WHERE microcosm_id = $1`,
		microcosmID,
	)
	if err != nil {
		glog.Error(err)
		return err
	}

	return nil
}

// DecrementMicrocosmItemCount removes an item from the microcosm
func DecrementMicrocosmItemCount(tx *sql.Tx, microcosmID int64) error {
	_, err := tx.Exec(`--Update Microcosm Item Count
UPDATE microcosms
   SET item_count = item_count - 1
 WHERE microcosm_id = $1`,
		microcosmID,
	)
	if err != nil {
		glog.Error(err)
		return err
	}

	return nil
}

// GetRootMicrocosms fetches the collection of microcosms that have no parent
func GetRootMicrocosms(
	siteID int64,
	profileID int64,
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
       AND m.parent_id IS NULL
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
		siteID,
		profileID,
		limit,
		offset,
	)

	if err != nil {
		glog.Errorf(
			"db.Query(%d, %d, %d, %d) %+v",
			siteID,
			profileID,
			limit,
			offset,
			err,
		)
		return []MicrocosmSummaryType{}, 0, 0,
			http.StatusInternalServerError,
			fmt.Errorf("Database query failed")
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
				fmt.Errorf("Row parsing error")
		}

		unread[id] = hasUnread
		ids = append(ids, id)
	}
	err = rows.Err()
	if err != nil {
		glog.Errorf("rows.Err() %+v", err)
		return []MicrocosmSummaryType{}, 0, 0,
			http.StatusInternalServerError,
			fmt.Errorf("Error fetching rows")
	}
	rows.Close()

	// Make a request for each identifier
	var wg1 sync.WaitGroup
	req := make(chan MicrocosmSummaryRequest)
	defer close(req)

	for seq, id := range ids {
		go HandleMicrocosmSummaryRequest(siteID, id, profileID, seq, req)
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
		m.Meta.Flags.Unread = unread[m.ID]
		ems = append(ems, m)
	}

	pages := h.GetPageCount(total, limit)
	maxOffset := h.GetMaxOffset(total, limit)

	if offset > maxOffset {
		return []MicrocosmSummaryType{}, 0, 0,
			http.StatusBadRequest,
			fmt.Errorf(fmt.Sprintf("not enough records, "+
				"offset (%d) would return an empty page.", offset))
	}

	return ems, total, pages, http.StatusOK, nil
}

func itemTypesToPSQLArray(s []string) string {
	it := []string{}
	for _, v := range s {
		if val, ok := h.ItemTypes[v]; ok {
			it = append(it, strconv.FormatInt(val, 10))
		}
	}
	return `{` + strings.Join(it, `,`) + `}`
}

func psqlArrayToItemTypes(s string) []string {
	s = strings.TrimLeft(s, `{`)
	s = strings.TrimRight(s, `}`)

	itemTypes := []string{}
	for _, v := range strings.Split(s, `,`) {
		i, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			continue
		}
		str, err := h.GetItemTypeFromInt(i)
		if err != nil {
			continue
		}
		itemTypes = append(itemTypes, str)
	}

	return itemTypes
}
