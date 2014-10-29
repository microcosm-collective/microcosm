package models

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"

	c "github.com/microcosm-cc/microcosm/cache"
	h "github.com/microcosm-cc/microcosm/helpers"
)

type RolesType struct {
	DefaultRoles bool           `json:"defaultRoles,omitempty"`
	Roles        h.ArrayType    `json:"roles"`
	Meta         h.CoreMetaType `json:"meta"`
}

type RoleType struct {
	Id                  int64         `json:"id"`
	Title               string        `json:"title"`
	SiteId              int64         `json:"siteId,omitempty"`
	MicrocosmId         int64         `json:"microcosmId,omitempty"`
	MicrocosmIdNullable sql.NullInt64 `json:"-"`

	IsModerator   bool `json:"moderator"`
	IsBanned      bool `json:"banned"`
	IncludeGuests bool `json:"includeGuests"`
	IncludeUsers  bool `json:"includeUsers"`

	CanCreate     bool `json:"create"`
	CanRead       bool `json:"read"`
	CanUpdate     bool `json:"update"`
	CanDelete     bool `json:"delete"`
	CanCloseOwn   bool `json:"closeOwn"`
	CanOpenOwn    bool `json:"openOwn"`
	CanReadOthers bool `json:"readOthers"`

	Meta RoleMetaType `json:"meta"`

	// These two are used by the importer and should not be JSON exported
	Criteria []RoleCriterionType `json:"-"`
	Profiles []RoleProfileType   `json:"-"`
}

type RoleSummaryType struct {
	RoleType

	Members h.ArrayType `json:"members"`
}

type RoleMetaType struct {
	h.CreatedType
	h.EditedType
	Links       []h.LinkType `json:"links,omitempty"`
	Permissions interface{}  `json:"permissions,omitempty"`
}

func (m *RoleType) GetLink() string {

	if m.MicrocosmId > 0 {
		return fmt.Sprintf(
			"%s/%d/roles/%d",
			h.ApiTypeMicrocosm,
			m.MicrocosmId,
			m.Id,
		)
	}

	return fmt.Sprintf("%s/%d", h.ApiTypeRole, m.Id)
}

func (m *RoleType) Validate(
	siteId int64,
	profileId int64,
	exists bool,
) (
	int,
	error,
) {

	m.Title = SanitiseText(m.Title)

	// Does the Microcosm specified exist on this site?
	if !exists && m.MicrocosmId > 0 {
		_, status, err := GetMicrocosmSummary(siteId, m.MicrocosmId, profileId)
		if err != nil {
			return status, err
		}
	}

	if exists {
		if m.Id < 1 {
			return http.StatusBadRequest,
				errors.New(
					fmt.Sprintf("ID ('%d') cannot be zero or negative.", m.Id),
				)
		}

		if strings.Trim(m.Meta.EditReason, " ") == "" ||
			len(m.Meta.EditReason) == 0 {

			m.Meta.EditReason = "No reason given"
		} else {
			m.Meta.EditReason = ShoutToWhisper(m.Meta.EditReason)
		}
	}

	if strings.Trim(m.Title, " ") == "" {
		return http.StatusBadRequest, errors.New("Title is a required field")
	}

	m.Title = ShoutToWhisper(m.Title)

	// Needs to be NULL if it's a default role
	if m.MicrocosmId > 0 {
		m.MicrocosmIdNullable = sql.NullInt64{Int64: m.MicrocosmId, Valid: true}
	}

	return http.StatusOK, nil
}

func (m *RoleType) Insert(siteId int64, profileId int64) (int, error) {

	status, err := m.Validate(siteId, profileId, false)
	if err != nil {
		return status, err
	}

	dupeKey := "dupe_" + h.Md5sum(
		strconv.FormatInt(m.MicrocosmId, 10)+
			m.Title+
			strconv.FormatInt(m.Meta.CreatedById, 10),
	)

	v, ok := c.CacheGetInt64(dupeKey)
	if ok {
		m.Id = v
		return http.StatusOK, nil
	}

	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	var insertId int64
	err = tx.QueryRow(`
INSERT INTO roles (
    title, site_id, microcosm_id, created, created_by,
    is_moderator_role, is_banned_role, can_read, can_create, can_update,
    can_delete, can_close_own, can_open_own, can_read_others, include_guests,
    include_users
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9, $10,
    $11, $12, $13, $14, $15,
    $16
) RETURNING role_id`,
		m.Title,
		m.SiteId,
		m.MicrocosmIdNullable,
		m.Meta.Created,
		m.Meta.CreatedById,

		m.IsModerator,
		m.IsBanned,
		m.CanRead,
		m.CanCreate,
		m.CanUpdate,

		m.CanDelete,
		m.CanCloseOwn,
		m.CanOpenOwn,
		m.CanReadOthers,
		m.IncludeGuests,

		m.IncludeUsers,
	).Scan(
		&insertId,
	)
	if err != nil {
		return http.StatusInternalServerError, errors.New(
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

	go PurgeCache(h.ItemTypes[h.ItemTypeRole], m.Id)

	return http.StatusOK, nil
}

func (m *RoleType) Update(siteId int64, profileId int64) (int, error) {

	status, err := m.Validate(siteId, profileId, true)
	if err != nil {
		return status, err
	}

	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
UPDATE roles
   SET title = $2
      ,edited = $3
      ,edited_by = $4
      ,edit_reason = $5
      ,is_moderator_role = $6
      ,is_banned_role = $7
      ,can_read = $8
      ,can_create =$9
      ,can_update = $10
      ,can_delete = $11
      ,can_close_own = $12
      ,can_open_own = $13
      ,can_read_others = $14
      ,include_guests = $15
      ,include_users = $16
 WHERE role_id = $1`,
		m.Id,
		m.Title,
		m.Meta.EditedNullable,
		m.Meta.EditedByNullable,
		m.Meta.EditReason,

		m.IsModerator,
		m.IsBanned,
		m.CanRead,
		m.CanCreate,
		m.CanUpdate,

		m.CanDelete,
		m.CanCloseOwn,
		m.CanOpenOwn,
		m.CanReadOthers,
		m.IncludeGuests,

		m.IncludeUsers,
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

	go PurgeCache(h.ItemTypes[h.ItemTypeRole], m.Id)

	return http.StatusOK, nil
}

func (m *RoleType) Patch(ac AuthContext, patches []h.PatchType) (int, error) {

	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	for _, patch := range patches {

		m.Meta.EditedNullable = pq.NullTime{Time: time.Now(), Valid: true}
		m.Meta.EditedByNullable =
			sql.NullInt64{Int64: ac.ProfileId, Valid: true}

		var column string
		patch.ScanRawValue()
		switch patch.Path {
		case "/moderator":
			column = "is_moderator_role"
			m.IsModerator = patch.Bool.Bool
			m.Meta.EditReason =
				fmt.Sprintf("Set %s to %t", patch.Path, m.IsModerator)
		case "/banned":
			column = "is_banned_role"
			m.IsBanned = patch.Bool.Bool
			m.Meta.EditReason =
				fmt.Sprintf("Set %s to %t", patch.Path, m.IsBanned)
		case "/read":
			column = "can_read"
			m.CanRead = patch.Bool.Bool
			m.Meta.EditReason =
				fmt.Sprintf("Set %s to %t", patch.Path, m.CanRead)
		case "/create":
			column = "can_create"
			m.CanCreate = patch.Bool.Bool
			m.Meta.EditReason =
				fmt.Sprintf("Set %s to %t", patch.Path, m.CanCreate)
		case "/update":
			column = "can_update"
			m.CanUpdate = patch.Bool.Bool
			m.Meta.EditReason =
				fmt.Sprintf("Set %s to %t", patch.Path, m.CanUpdate)
		case "/delete":
			column = "can_delete"
			m.CanDelete = patch.Bool.Bool
			m.Meta.EditReason =
				fmt.Sprintf("Set %s to %t", patch.Path, m.CanDelete)
		case "/closeOwn":
			column = "can_close_own"
			m.CanCloseOwn = patch.Bool.Bool
			m.Meta.EditReason =
				fmt.Sprintf("Set %s to %t", patch.Path, m.CanCloseOwn)
		case "/openOwn":
			column = "can_open_own"
			m.CanOpenOwn = patch.Bool.Bool
			m.Meta.EditReason =
				fmt.Sprintf("Set %s to %t", patch.Path, m.CanOpenOwn)
		case "/readOthers":
			column = "can_read_others"
			m.CanReadOthers = patch.Bool.Bool
			m.Meta.EditReason =
				fmt.Sprintf("Set %s to %t", patch.Path, m.CanReadOthers)
		case "/includeGuests":
			column = "include_guests"
			m.IncludeGuests = patch.Bool.Bool
			m.Meta.EditReason =
				fmt.Sprintf("Set %s to %t", patch.Path, m.IncludeGuests)
		case "/includeUsers":
			column = "include_users"
			m.IncludeUsers = patch.Bool.Bool
			m.Meta.EditReason =
				fmt.Sprintf("Set %s to %t", patch.Path, m.IncludeUsers)
		default:
			return http.StatusBadRequest,
				errors.New("Unsupported path in patch replace operation")
		}

		_, err = tx.Exec(`
UPDATE roles
   SET `+column+` = $2
      ,edited = $3
      ,edited_by = $4
      ,edit_reason = $5
 WHERE role_id = $1`,
			m.Id,
			patch.Bool.Bool,
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

	go PurgeCache(h.ItemTypes[h.ItemTypeRole], m.Id)

	return http.StatusOK, nil
}

func (m *RoleType) Delete() (int, error) {

	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
DELETE
  FROM role_members_cache
 WHERE role_id = $1`,
		m.Id,
	)
	if err != nil {
		return http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Delete failed: %v", err.Error()),
		)
	}

	_, err = tx.Exec(`
DELETE
  FROM role_profiles
 WHERE role_id = $1`,
		m.Id,
	)
	if err != nil {
		return http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Delete failed: %v", err.Error()),
		)
	}

	_, err = tx.Exec(`
DELETE
  FROM criteria
 WHERE role_id = $1`,
		m.Id,
	)
	if err != nil {
		return http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Delete failed: %v", err.Error()),
		)
	}

	_, err = tx.Exec(`
DELETE
  FROM roles
 WHERE role_id = $1`,
		m.Id,
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

	go PurgeCache(h.ItemTypes[h.ItemTypeRole], m.Id)

	return http.StatusOK, nil
}

func GetRole(
	siteId int64,
	microcosmId int64,
	roleId int64,
	profileId int64,
) (
	RoleType,
	int,
	error,
) {

	// Get from cache if it's available
	mcKey := fmt.Sprintf(mcRoleKeys[c.CacheDetail], roleId)
	if val, ok := c.CacheGet(mcKey, RoleType{}); ok {

		m := val.(RoleType)

		_, status, err := GetMicrocosmSummary(siteId, microcosmId, profileId)
		if err != nil {
			return RoleType{}, status, err
		}

		m.FetchProfileSummaries(siteId)

		return m, http.StatusOK, nil
	}

	// Retrieve resource
	db, err := h.GetConnection()
	if err != nil {
		return RoleType{}, http.StatusInternalServerError, err
	}

	m := RoleType{SiteId: siteId}
	err = db.QueryRow(`
SELECT role_id
      ,title
      ,created
      ,created_by
      ,edited

      ,edited_by
      ,edit_reason
      ,is_moderator_role
      ,is_banned_role
      ,include_guests

      ,include_users
      ,can_read
      ,can_create
      ,can_update
      ,can_delete

      ,can_close_own
      ,can_open_own
      ,can_read_others
      ,microcosm_id
  FROM roles
 WHERE site_id = $1
   AND role_id = $2`,
		siteId,
		roleId,
	).Scan(
		&m.Id,
		&m.Title,
		&m.Meta.Created,
		&m.Meta.CreatedById,
		&m.Meta.EditedNullable,

		&m.Meta.EditedByNullable,
		&m.Meta.EditReasonNullable,
		&m.IsModerator,
		&m.IsBanned,
		&m.IncludeGuests,

		&m.IncludeUsers,
		&m.CanRead,
		&m.CanCreate,
		&m.CanUpdate,
		&m.CanDelete,

		&m.CanCloseOwn,
		&m.CanOpenOwn,
		&m.CanReadOthers,
		&m.MicrocosmIdNullable,
	)
	if err == sql.ErrNoRows {
		return RoleType{}, http.StatusNotFound, errors.New(
			fmt.Sprintf("Role resource with ID %d not found", roleId),
		)
	} else if err != nil {
		return RoleType{}, http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Database query failed: %v", err.Error()),
		)
	}

	if m.MicrocosmIdNullable.Valid {
		m.MicrocosmId = m.MicrocosmIdNullable.Int64
	}

	if m.Meta.EditReasonNullable.Valid {
		m.Meta.EditReason = m.Meta.EditReasonNullable.String
	}

	if m.Meta.EditedNullable.Valid {
		m.Meta.Edited =
			m.Meta.EditedNullable.Time.Format(time.RFC3339Nano)
	}

	if m.MicrocosmId != 0 && m.MicrocosmId != microcosmId {
		return RoleType{}, http.StatusNotFound, errors.New(
			fmt.Sprintf("Valid role resource with ID %d not found", roleId),
		)
	}

	if m.MicrocosmId > 0 {
		m.Meta.Links =
			[]h.LinkType{
				h.GetLink("self", "", h.ItemTypeRole, m.Id),
				h.GetLink(
					"microcosm",
					GetMicrocosmTitle(m.MicrocosmId),
					h.ItemTypeMicrocosm,
					m.MicrocosmId,
				),
			}
	} else {
		m.Meta.Links =
			[]h.LinkType{
				h.GetLink("self", "", h.ItemTypeRole, m.Id),
			}
	}

	// Update cache
	c.CacheSet(mcKey, m, mcTtl)

	m.FetchProfileSummaries(siteId)
	return m, http.StatusOK, nil
}

func GetRoleSummary(
	siteId int64,
	microcosmId int64,
	roleId int64,
	profileId int64,
) (
	RoleSummaryType,
	int,
	error,
) {

	role, status, err := GetRole(siteId, microcosmId, roleId, profileId)
	if err != nil {
		return RoleSummaryType{}, status, err
	}

	roleSummary := RoleSummaryType{}
	roleSummary.Id = role.Id
	roleSummary.Title = role.Title
	roleSummary.SiteId = role.SiteId
	roleSummary.MicrocosmId = role.MicrocosmId
	roleSummary.IsModerator = role.IsModerator
	roleSummary.IsBanned = role.IsBanned
	roleSummary.IncludeGuests = role.IncludeGuests
	roleSummary.IncludeUsers = role.IncludeUsers
	roleSummary.CanCreate = role.CanCreate
	roleSummary.CanRead = role.CanRead
	roleSummary.CanUpdate = role.CanUpdate
	roleSummary.CanDelete = role.CanDelete
	roleSummary.CanCloseOwn = role.CanCloseOwn
	roleSummary.CanOpenOwn = role.CanOpenOwn
	roleSummary.CanReadOthers = role.CanReadOthers
	roleSummary.Meta = role.Meta

	// Fetch members
	limit := int64(5)
	offset := h.DefaultQueryOffset
	ems, total, pages, status, err := GetRoleMembers(
		siteId,
		roleId,
		limit,
		offset,
	)
	if err != nil {
		return RoleSummaryType{}, status, err
	}

	roleSummary.Members = h.ConstructArray(
		ems,
		h.ApiTypeProfile,
		total,
		limit,
		offset,
		pages,
		nil,
	)

	return roleSummary, http.StatusOK, nil
}

func (m *RoleType) FetchProfileSummaries(siteId int64) (int, error) {

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

	return http.StatusOK, nil
}

func GetRoles(
	siteId int64,
	microcosmId int64,
	profileId int64,
	limit int64,
	offset int64,
) (
	[]RoleSummaryType,
	int64,
	int64,
	int,
	error,
) {

	// Retrieve resources
	db, err := h.GetConnection()
	if err != nil {
		return []RoleSummaryType{}, 0, 0,
			http.StatusInternalServerError, err
	}

	rows, err := db.Query(`--GetRoles
WITH r AS (
    SELECT *
      FROM get_microcosm_roles($1,$2)
)
SELECT COUNT(*) OVER() AS total
      ,role_id
  FROM roles
 WHERE role_id IN (SELECT * FROM r)
 ORDER BY is_moderator_role DESC, is_banned_role, title
 LIMIT $3
OFFSET $4`,
		siteId,
		microcosmId,
		limit,
		offset,
	)
	if err != nil {
		return []RoleSummaryType{}, 0, 0, http.StatusInternalServerError,
			errors.New(
				fmt.Sprintf("Database query failed: %v", err.Error()),
			)
	}
	defer rows.Close()

	var (
		ems   []RoleSummaryType
		total int64
	)
	for rows.Next() {
		var id int64
		err = rows.Scan(
			&total,
			&id,
		)
		if err != nil {
			return []RoleSummaryType{}, 0, 0,
				http.StatusInternalServerError,
				errors.New(
					fmt.Sprintf("Row parsing error: %v", err.Error()),
				)
		}

		m, status, err := GetRoleSummary(siteId, microcosmId, id, profileId)
		if err != nil {
			return []RoleSummaryType{}, 0, 0, status, err
		}

		ems = append(ems, m)
	}
	err = rows.Err()
	if err != nil {
		return []RoleSummaryType{}, 0, 0, http.StatusInternalServerError,
			errors.New(
				fmt.Sprintf("Error fetching rows: %v", err.Error()),
			)
	}
	rows.Close()

	pages := h.GetPageCount(total, limit)
	maxOffset := h.GetMaxOffset(total, limit)

	if offset > maxOffset {
		return []RoleSummaryType{}, 0, 0, http.StatusBadRequest,
			errors.New(
				fmt.Sprintf("not enough records, "+
					"offset (%d) would return an empty page.", offset),
			)
	}

	return ems, total, pages, http.StatusOK, nil
}
