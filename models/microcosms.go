package models

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/lib/pq"

	c "github.com/microcosm-cc/microcosm/cache"
	h "github.com/microcosm-cc/microcosm/helpers"
)

// MicrocosmCore describes the ancestor microcosms this item belongs to
type MicrocosmCore struct {
	ID               int64 `json:"id"`
	ParentID         int64 `json:"parentId,omitempty"`
	parentIDNullable sql.NullInt64
	Breadcrumb       *[]MicrocosmLinkType `json:"breadcrumb,omitempty"`
	SiteID           int64                `json:"siteId,omitempty"`
	Visibility       string               `json:"visibility"`
	Title            string               `json:"title"`
	Description      string               `json:"description"`
	LogoURL          string               `json:"logoUrl"`
	LogoURLNullable  sql.NullString       `json:"-"`
	ItemTypes        []string             `json:"itemTypes"`
}

// MicrocosmsType is a collection of microcosms
type MicrocosmsType struct {
	Microcosms h.ArrayType    `json:"microcosms"`
	Meta       h.CoreMetaType `json:"meta"`
}

// MicrocosmSummaryType is a summary of a microcosm
type MicrocosmSummaryType struct {
	MicrocosmCore

	Moderators   []int64 `json:"moderators"`
	ItemCount    int64   `json:"totalItems"`
	CommentCount int64   `json:"totalComments"`

	MRU interface{} `json:"mostRecentUpdate,omitempty"`

	Meta h.SummaryMetaType `json:"meta"`
}

// MicrocosmType is a microcosm
type MicrocosmType struct {
	MicrocosmCore

	RemoveLogo bool `json:"removeLogo,omitempty"`

	OwnedByID int64 `json:"-"`

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

// MicrocosmLinkType is a link
type MicrocosmLinkType struct {
	Rel              string `json:"rel,omitempty"` // REST
	Href             string `json:"href"`
	Title            string `json:"title,omitempty"`
	ID               int64  `json:"id"`
	Level            int64  `json:"level,omitempty"`
	ParentID         int64  `json:"parentId,omitempty"`
	parentIDNullable sql.NullInt64
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
	m.Title = CleanSentence(m.Title)
	m.Description = string(SanitiseHTML([]byte(CleanBlockText(m.Description))))

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

	if strings.Trim(m.Description, " ") == "" {
		return http.StatusBadRequest,
			fmt.Errorf("Description is a required field")
	}

	if m.ParentID > 0 {
		m.parentIDNullable.Valid = true
		m.parentIDNullable.Int64 = m.ParentID
	}

	if strings.TrimSpace(m.LogoURL) != "" {
		// Ensure that this is in fact a URL and not some malicious input
		u, err := url.Parse(m.LogoURL)
		if err != nil {
			return http.StatusBadRequest, err
		}
		m.LogoURLNullable.Valid = true
		m.LogoURLNullable.String = u.String()
	}

	if m.RemoveLogo {
		m.LogoURLNullable = sql.NullString{}
	}

	return http.StatusOK, nil
}

// Hydrate populates a partially populated microcosm struct
func (m *MicrocosmType) Hydrate(
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

	status, err = m.FetchBreadcrumb()
	if err != nil {
		return status, err
	}

	return http.StatusOK, nil
}

// Hydrate populates a partially populated struct
func (m *MicrocosmSummaryType) Hydrate(
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

	status, err = m.FetchBreadcrumb()
	if err != nil {
		return status, err
	}

	return http.StatusOK, nil
}

// FetchBreadcrumb will determine the ancestor microcosms and establish the
// breadcrumb trail
func (m *MicrocosmCore) FetchBreadcrumb() (int, error) {
	parents, status, err := getMicrocosmParents(m.ID)
	if err != nil {
		return status, err
	}
	breadcrumb := []MicrocosmLinkType{}
	for _, parent := range parents {
		if parent.ID == m.ID || parent.Level == 1 {
			continue
		}
		breadcrumb = append(breadcrumb, parent)
	}
	if len(breadcrumb) > 0 {
		m.Breadcrumb = &breadcrumb
	}
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
    $6, $7, $8, $9, $10
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
		m.LogoURLNullable,
	).Scan(
		&insertID,
	)
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("Error inserting data and returning ID: %+v", err)
	}
	m.ID = insertID

	purgeCache := false
	status, err := updateMicrocosmPaths(tx, m.SiteID, purgeCache)
	if err != nil {
		return status, err
	}

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
       parent_id = $10,
       logo_url = $11
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
		m.LogoURLNullable,
	)
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("Update failed: %v", err.Error())
	}

	purgeCache := true
	updateMicrocosmPaths(tx, m.SiteID, purgeCache)

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

		m.Hydrate(siteID, profileID)

		return m, 0, nil
	}

	// Retrieve resource
	db, err := h.GetConnection()
	if err != nil {
		return MicrocosmType{}, http.StatusInternalServerError, err
	}

	// TODO(buro9): admins and mods could see this with isDeleted=true in the
	// querystring
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
       logo_url,
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
		&m.LogoURLNullable,
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

	if m.LogoURLNullable.Valid {
		m.LogoURL = m.LogoURLNullable.String
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

	m.Hydrate(siteID, profileID)
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

		m.Hydrate(siteID, profileID)

		return m, http.StatusOK, nil
	}

	// Retrieve resource
	db, err := h.GetConnection()
	if err != nil {
		glog.Error(err)
		return MicrocosmSummaryType{}, http.StatusInternalServerError, err
	}

	// TODO(buro9): admins and mods could see this with isDeleted=true in the
	// querystring
	var (
		m         MicrocosmSummaryType
		itemTypes string
	)
	err = db.QueryRow(`--GetMicrocosmSummary
SELECT m.microcosm_id
      ,m.parent_id
      ,m.site_id
      ,m.visibility
      ,m.title
      ,m.description
      ,m.logo_url
      ,m.created
      ,m.created_by
      ,m.is_sticky
      ,m.is_open
      ,m.is_deleted
      ,m.is_moderated
      ,m.is_visible
      ,(SELECT SUM(item_count)
          FROM microcosms
         WHERE path <@ m.path
           AND is_deleted IS NOT TRUE
           AND is_moderated IS NOT TRUE) AS item_count
      ,(SELECT SUM(comment_count)
          FROM microcosms
         WHERE path <@ m.path
           AND is_deleted IS NOT TRUE
           AND is_moderated IS NOT TRUE) AS comment_count
      ,m.item_types
  FROM microcosms m
 WHERE m.site_id = $1
   AND m.microcosm_id = $2
   AND m.is_deleted IS NOT TRUE
   AND m.is_moderated IS NOT TRUE`,
		siteID,
		id,
	).Scan(
		&m.ID,
		&m.parentIDNullable,
		&m.SiteID,
		&m.Visibility,
		&m.Title,
		&m.Description,
		&m.LogoURLNullable,
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

	if m.LogoURLNullable.Valid {
		m.LogoURL = m.LogoURLNullable.String
	}

	mru, status, err := GetMostRecentItem(siteID, m.ID, profileID)
	if err != nil {
		glog.Error(err)
		return MicrocosmSummaryType{}, status, fmt.Errorf("Row parsing error")
	}
	if mru.Valid {
		m.MRU = mru
	}

	if m.ParentID > 0 && m.ParentID != GetRootMicrocosmID(siteID) {
		m.Meta.Links =
			[]h.LinkType{
				h.GetLink("self", "", h.ItemTypeMicrocosm, m.ID),
				h.GetLink("parent", GetMicrocosmTitle(m.ParentID), h.ItemTypeMicrocosm, m.ParentID),
				h.GetLink("site", "", h.ItemTypeSite, m.SiteID),
			}
	} else {
		m.Meta.Links =
			[]h.LinkType{
				h.GetLink("self", "", h.ItemTypeMicrocosm, m.ID),
				h.GetLink("site", "", h.ItemTypeSite, m.SiteID),
			}

	}

	// Update cache
	c.Set(mcKey, m, mcTTL)

	m.Hydrate(siteID, profileID)

	return m, http.StatusOK, nil
}

// GetRootMicrocosmID returns the ID of the microcosm that is the root microcosm
// that ultimately is the ancestor of all microcosms
func GetRootMicrocosmID(id int64) int64 {
	// Get from cache if it's available
	mcKey := fmt.Sprintf(mcSiteKeys[c.CacheRootID], id)
	if val, ok := c.GetInt64(mcKey); ok {
		return val
	}

	// Retrieve resource
	db, err := h.GetConnection()
	if err != nil {
		glog.Errorf("h.GetConmection() %+v", err)
		return 0
	}

	var rootID int64
	err = db.QueryRow(`--GetRootMicrocosmID
SELECT microcosm_id
  FROM microcosms
 WHERE site_id = $1
   AND parent_id IS NULL`,
		id,
	).Scan(
		&rootID,
	)
	if err != nil {
		glog.Errorf("db.QueryRow(%d).Scan(&rootID) %+v", id, err)
		return 0
	}

	// Update cache
	c.SetInt64(mcKey, rootID, mcTTL)

	return rootID
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
	_, err := tx.Exec(`--Increment Microcosm Item Count
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
	_, err := tx.Exec(`--Decrement Microcosm Item Count
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

// GetRootMicrocosm fetches the root microcosm and it's contents
func GetRootMicrocosm(
	siteID int64,
	profileID int64,
	limit int64,
	offset int64,
) (
	[]SummaryContainer,
	int64,
	int64,
	int,
	error,
) {
	rootMicrocosmID := GetRootMicrocosmID(siteID)
	if rootMicrocosmID == 0 {
		return []SummaryContainer{}, 0, 0,
			http.StatusInternalServerError, fmt.Errorf("Root microcosm must exist")
	}

	return GetAllItems(siteID, rootMicrocosmID, profileID, limit, offset)
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

// updateMicrocosmPaths will update the parent paths for all microcosms, and if
// this resulted in any rows in the database being updated then this will also
// purge those from cache so that the updates are reflected
func updateMicrocosmPaths(tx *sql.Tx, siteId int64, cachePurge bool) (int, error) {
	res, err := tx.Exec(`-- Update Microcosm Paths
WITH RECURSIVE parent_microcosms AS (
    SELECT microcosm_id
          ,CAST(microcosm_id AS VARCHAR) AS path
      FROM microcosms
     WHERE parent_id IS NULL
       AND site_id = $1
     UNION ALL
    SELECT c.microcosm_id
          ,p.path || '.' || CAST(c.microcosm_id AS VARCHAR)
      FROM microcosms c
      JOIN parent_microcosms p ON c.parent_id = p.microcosm_id
)
UPDATE microcosms m
   SET path = CAST(p.path AS ltree)
  FROM parent_microcosms p
 WHERE m.microcosm_id = p.microcosm_id
   AND (
           m.path IS NULL
        OR CAST(m.path AS VARCHAR) <> p.path
       )`,
		siteId,
	)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	if cachePurge {
		rowsUpdated, err := res.RowsAffected()
		if err != nil {
			return http.StatusInternalServerError, err
		}

		if rowsUpdated > 0 {
			rows, err := tx.Query(`-- Get all Microcosms for Site
SELECT microcosm_id
  FROM microcosms
 WHERE site_id = $1
   AND is_deleted IS NOT TRUE
   AND is_moderated IS NOT TRUE`,
				siteId,
			)
			if err != nil {
				return http.StatusInternalServerError, err
			}

			ids := []int64{}
			for rows.Next() {
				var id int64
				err = rows.Scan(&id)
				if err != nil {
					return http.StatusInternalServerError, err
				}
				ids = append(ids, id)
			}
			err = rows.Err()
			if err != nil {
				return http.StatusInternalServerError, err
			}
			rows.Close()

			for _, id := range ids {
				PurgeCache(h.ItemTypes[h.ItemTypeMicrocosm], id)
			}
		}
	}

	return http.StatusOK, nil
}

func getMicrocosmParents(microcosmID int64) ([]MicrocosmLinkType, int, error) {
	mcKey := fmt.Sprintf(mcMicrocosmKeys[c.CacheBreadcrumb], microcosmID)
	if val, ok := c.Get(mcKey, []MicrocosmLinkType{}); ok {
		m := val.([]MicrocosmLinkType)
		return m, http.StatusOK, nil
	}

	// Retrieve resources
	db, err := h.GetConnection()
	if err != nil {
		return []MicrocosmLinkType{},
			http.StatusInternalServerError,
			err
	}

	rows, err := db.Query(`--GetMicrocosmParents
SELECT microcosm_id
      ,title
      ,NLEVEL(m.path) AS depth
  FROM (
           SELECT path
             FROM microcosms
            WHERE microcosm_id = $1
       ) im
  JOIN microcosms m ON m.path @> im.path
 ORDER BY m.path`,
		microcosmID,
	)
	if err != nil {
		glog.Error(err)
		return []MicrocosmLinkType{},
			http.StatusInternalServerError,
			fmt.Errorf("Database query failed: %v", err.Error())
	}
	defer rows.Close()

	links := []MicrocosmLinkType{}
	for rows.Next() {
		link := MicrocosmLinkType{}
		err = rows.Scan(
			&link.ID,
			&link.Title,
			&link.Level,
		)
		if err != nil {
			return []MicrocosmLinkType{},
				http.StatusInternalServerError,
				fmt.Errorf("Row parsing error: %v", err.Error())
		}

		link.Rel = "microcosm ancestor"
		if link.Level > 1 {
			link.Href =
				fmt.Sprintf("%s/%d", h.ItemTypesToAPIItem[h.ItemTypeMicrocosm], link.ID)
		} else {
			link.Href = h.ItemTypesToAPIItem[h.ItemTypeMicrocosm]
		}

		links = append(links, link)
	}
	err = rows.Err()
	if err != nil {
		return []MicrocosmLinkType{},
			http.StatusInternalServerError,
			fmt.Errorf("Error fetching rows: %v", err.Error())
	}
	rows.Close()

	c.Set(mcKey, links, mcTTL)

	return links, http.StatusOK, nil
}

func GetMicrocosmTree(siteID int64, profileID int64) ([]MicrocosmLinkType, int, error) {
	// Retrieve resources
	db, err := h.GetConnection()
	if err != nil {
		return []MicrocosmLinkType{},
			http.StatusInternalServerError,
			err
	}

	rows, err := db.Query(`--GetMicrocosmTree
WITH m AS (
    SELECT m.microcosm_id
          ,m.comment_count + m.item_count AS seq
      FROM microcosms m
      LEFT JOIN permissions_cache p ON p.site_id = m.site_id
                                   AND p.item_type_id = 2
                                   AND p.item_id = m.microcosm_id
                                   AND p.profile_id = $2
           LEFT JOIN ignores_expanded i ON i.profile_id = $2
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
SELECT microcosm_id
      ,title
      ,parent_id
      ,NLEVEL(path) AS depth
  FROM microcosms
 WHERE microcosm_id IN (SELECT microcosm_id FROM m)
 ORDER BY path`,
		siteID,
		profileID,
	)
	if err != nil {
		glog.Error(err)
		return []MicrocosmLinkType{},
			http.StatusInternalServerError,
			fmt.Errorf("Database query failed: %v", err.Error())
	}
	defer rows.Close()

	links := []MicrocosmLinkType{}
	for rows.Next() {
		link := MicrocosmLinkType{}
		err = rows.Scan(
			&link.ID,
			&link.Title,
			&link.parentIDNullable,
			&link.Level,
		)
		if err != nil {
			return []MicrocosmLinkType{},
				http.StatusInternalServerError,
				fmt.Errorf("Row parsing error: %v", err.Error())
		}

		if link.parentIDNullable.Valid {
			link.ParentID = link.parentIDNullable.Int64
		}

		link.Rel = "microcosm"
		if link.Level > 1 {
			link.Href =
				fmt.Sprintf("%s/%d", h.ItemTypesToAPIItem[h.ItemTypeMicrocosm], link.ID)
		} else {
			link.Href = h.ItemTypesToAPIItem[h.ItemTypeMicrocosm]
		}

		links = append(links, link)
	}
	err = rows.Err()
	if err != nil {
		return []MicrocosmLinkType{},
			http.StatusInternalServerError,
			fmt.Errorf("Error fetching rows: %v", err.Error())
	}
	rows.Close()

	return links, http.StatusOK, nil
}
