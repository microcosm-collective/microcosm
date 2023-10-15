package models

import (
	"database/sql"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"

	h "github.com/microcosm-cc/microcosm/helpers"
)

const (
	// All data types have these predicates

	// Equals predicate
	Equals string = "eq"

	// NotEquals predicate
	NotEquals string = "ne"

	// These apply only to Numbers and Dates

	// LessThan predicate
	LessThan string = "lt"

	// LessThanOrEquals predicate
	LessThanOrEquals string = "le"

	// GreaterThanOrEquals predicate
	GreaterThanOrEquals string = "ge"

	// GreaterThan predicate
	GreaterThan string = "gt"

	// These apply to strings...

	// Substring predicate
	Substring string = "substr"

	// NotSubstring predicate
	NotSubstring string = "nsubstr"
)

var (
	// StringPredicates handles the valid predicates for strings
	StringPredicates = []string{Equals, NotEquals, Substring, NotSubstring}

	// Int64Predicates handles the valid predicates for ints
	Int64Predicates = []string{
		Equals,
		NotEquals,
		LessThan,
		LessThanOrEquals,
		GreaterThanOrEquals,
		GreaterThan,
	}

	// TimePredicates handles the valid predicates for ints
	TimePredicates = []string{
		Equals,
		NotEquals,
		LessThan,
		LessThanOrEquals,
		GreaterThanOrEquals,
		GreaterThan,
	}

	// BoolPredicates handles the valid predicates for ints
	BoolPredicates = []string{Equals, NotEquals}

	// ProfileColumns is a hard-coded list of columns on the profiles table on
	// the database that we can filter against
	ProfileColumns = []ProfileColumn{
		{
			Camel:      "id",
			Snake:      "profile_id",
			Type:       "int64",
			Predicates: Int64Predicates,
		},
		{
			Camel:      "profileName",
			Snake:      "profile_name",
			Type:       "string",
			Predicates: StringPredicates,
		},
		{
			Camel:      "gender",
			Snake:      "gender",
			Type:       "string",
			Predicates: StringPredicates,
		},
		{
			Camel:      "email",
			Snake:      "email",
			Type:       "string",
			Predicates: StringPredicates,
		},
		{
			Camel:      "itemCount",
			Snake:      "item_count",
			Type:       "int64",
			Predicates: Int64Predicates,
		},
		{
			Camel:      "commentCount",
			Snake:      "comment_count",
			Type:       "int64",
			Predicates: Int64Predicates,
		},
		{
			Camel:      "created",
			Snake:      "created",
			Type:       "time",
			Predicates: TimePredicates,
		},
		{
			Camel:      "isBanned",
			Snake:      "is_banned",
			Type:       "bool",
			Predicates: BoolPredicates,
		},
	}
)

// ProfileColumn describes a column that can be matched on the profiles table
// for filtering and finding profiles
type ProfileColumn struct {
	Camel      string
	Snake      string
	Type       string
	Predicates []string
}

// RoleCriteriaType describes a collection of criterion
type RoleCriteriaType struct {
	RoleCriteria h.ArrayType    `json:"criteria"`
	Meta         h.CoreMetaType `json:"meta"`
}

// RoleCriterionType describes one criterion
type RoleCriterionType struct {
	ID                    int64          `json:"id,omitempty"`
	OrGroup               int64          `json:"orGroup"`
	ProfileColumn         string         `json:"profileColumn,omitempty"`
	ProfileColumnNullable sql.NullString `json:"-"`
	AttrKey               string         `json:"attrKey,omitempty"`
	AttrKeyNullable       sql.NullString `json:"-"`
	Predicate             string         `json:"predicate"`
	Value                 interface{}    `json:"value,omitempty"`
	ValueString           string         `json:"-"`
	Type                  string         `json:"-"`
}

// RoleCriterionRequest describes a channel envelope for a role criterion
type RoleCriterionRequest struct {
	Item   RoleCriterionType
	Err    error
	Status int
	Seq    int
}

// RoleCriterionRequestBySeq describes an ordered array of requests
type RoleCriterionRequestBySeq []RoleCriterionRequest

// Len returns the len of the array
func (v RoleCriterionRequestBySeq) Len() int {
	return len(v)
}

// Swap exchanges two items in the array by ordinal position
func (v RoleCriterionRequestBySeq) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

// Less determines which item is considered lesser than another in the array
func (v RoleCriterionRequestBySeq) Less(i, j int) bool {
	return v[i].Seq < v[j].Seq
}

// GetProfileColumnByCamel returns a profile column given the camel name
func GetProfileColumnByCamel(key string) (bool, ProfileColumn) {
	for _, m := range ProfileColumns {
		if m.Camel == key {
			return true, m
		}
	}

	return false, ProfileColumn{}
}

// GetProfileColumnBySnake returns a profile column given the snake name
func GetProfileColumnBySnake(key string) (bool, ProfileColumn) {
	for _, m := range ProfileColumns {
		if m.Snake == key {
			return true, m
		}
	}

	return false, ProfileColumn{}
}

// ValidPredicate returns true if the predicate is valid
func (m *ProfileColumn) ValidPredicate(predicate string) bool {
	for _, v := range m.Predicates {
		if v == predicate {
			return true
		}
	}

	return false
}

// GetLink returns a link to this criterion
func (m *RoleCriterionType) GetLink(roleLink string) string {
	return fmt.Sprintf("%s/criteria/%d", roleLink, m.ID)
}

// Validate returns true if the whole criterion is valid
func (m *RoleCriterionType) Validate(exists bool) (int, error) {

	if exists && m.ID < 1 {
		return http.StatusBadRequest,
			fmt.Errorf("profile id needs to be a positive integer")
	}

	// coerce value
	switch m.Value.(type) {
	case int:
		m.ValueString = strconv.FormatInt(int64(m.Value.(int)), 10)
		m.Type = tNumber
	case int32:
		m.ValueString = strconv.FormatInt(int64(m.Value.(int32)), 10)
		m.Type = tNumber
	case int64:
		m.ValueString = strconv.FormatInt(m.Value.(int64), 10)
		m.Type = tNumber
	case float32:
		m.ValueString =
			strconv.FormatFloat(float64(m.Value.(float32)), 'f', -1, 64)
		m.Type = tNumber
	case float64:
		m.ValueString = strconv.FormatFloat(m.Value.(float64), 'f', -1, 64)
		m.Type = tNumber
	case string:
		m.ValueString = strings.Trim(m.Value.(string), " ")
		m.Type = tString
	case bool:
		if m.Value.(bool) {
			m.ValueString = "true"
		} else {
			m.ValueString = "false"
		}
		m.Type = tBoolean
	default:
		return http.StatusBadRequest,
			fmt.Errorf("the type of `value` cannot be determined or is invalid")
	}

	if m.Type == "" {
		return http.StatusBadRequest,
			fmt.Errorf("value cannot be null, empty or just whitespace")
	}

	// Validate profileColumn
	if strings.Trim(m.ProfileColumn, " ") != "" {

		exists, col :=
			GetProfileColumnByCamel(strings.Trim(m.ProfileColumn, " "))
		if !exists {
			return http.StatusBadRequest,
				fmt.Errorf("the profileColumn is not valid")
		}

		if !col.ValidPredicate(m.Predicate) {
			return http.StatusBadRequest,
				fmt.Errorf("the predicate for the profileColumn is not valid")
		}

		m.ProfileColumnNullable = sql.NullString{
			String: col.Snake,
			Valid:  true,
		}
	} else {

		// Validate attribute
		if strings.Trim(m.AttrKey, " ") != "" {
			m.AttrKeyNullable = sql.NullString{
				String: strings.Trim(m.AttrKey, " "),
				Valid:  true,
			}

			switch m.Type {
			case tString:
				var found bool
				for _, v := range StringPredicates {
					if v == m.Predicate {
						found = true
					}
				}
				if !found {
					return http.StatusBadRequest,
						fmt.Errorf(
							"the predicate for the attribute type is not valid",
						)
				}
			case tNumber:
				var found bool
				for _, v := range Int64Predicates {
					if v == m.Predicate {
						found = true
					}
				}
				if !found {
					return http.StatusBadRequest,
						fmt.Errorf(
							"the predicate for the attribute type is not valid",
						)
				}
			case tBoolean:
				var found bool
				for _, v := range BoolPredicates {
					if v == m.Predicate {
						found = true
					}
				}
				if !found {
					return http.StatusBadRequest,
						fmt.Errorf(
							"the predicate for the attribute type is not valid",
						)
				}
			default:
				return http.StatusBadRequest, fmt.Errorf("type is not valid")
			}
		} else {
			return http.StatusBadRequest,
				fmt.Errorf(
					"either profileColumn or attrKey MUST be supplied",
				)
		}
	}

	return http.StatusOK, nil
}

// Insert saves the criterion to the database
func (m *RoleCriterionType) Insert(roleID int64) (int, error) {

	status, err := m.Validate(false)
	if err != nil {
		return status, err
	}

	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	var insertID int64
	err = tx.QueryRow(`
INSERT INTO criteria (
       role_id, or_group, profile_column, key, type,
       predicate, value
) VALUES (
       $1, $2, $3, $4, $5,
       $6, $7
) RETURNING criteria_id`,
		roleID,
		m.OrGroup,
		m.ProfileColumnNullable,
		m.AttrKeyNullable,
		m.Type,

		m.Predicate,
		m.ValueString,
	).Scan(
		&insertID,
	)
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("error executing insert: %v", err.Error())
	}
	m.ID = insertID

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("transaction failed: %v", err.Error())
	}

	go PurgeCache(h.ItemTypes[h.ItemTypeRole], roleID)

	return http.StatusOK, nil
}

// Update saves changes to the criterion to the database
func (m *RoleCriterionType) Update(roleID int64) (int, error) {

	status, err := m.Validate(true)
	if err != nil {
		return status, err
	}

	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
UPDATE criteria
   SET or_group = $2
      ,profile_column = $3
      ,key = $4
      ,type = $5
      ,predicate = $6
      ,value = $7
 WHERE criteria_id = $1`,
		m.ID,
		m.OrGroup,
		m.ProfileColumnNullable,
		m.AttrKeyNullable,
		m.Type,

		m.Predicate,
		m.ValueString,
	)
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("error executing insert: %v", err.Error())
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("transaction failed: %v", err.Error())
	}

	go PurgeCache(h.ItemTypes[h.ItemTypeRole], roleID)

	return http.StatusOK, nil
}

// DeleteManyRoleCriteria deletes all matching criterion from the database
func DeleteManyRoleCriteria(roleID int64, ems []RoleCriterionType) (int, error) {
	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	if len(ems) == 0 {
		// Delete all role criteria
		_, err = tx.Exec(`
DELETE FROM criteria
 WHERE role_id = $1`,
			roleID,
		)
		if err != nil {
			return http.StatusInternalServerError,
				fmt.Errorf("error executing delete: %+v", err)
		}
	} else {
		// Delete specific profiles
		for _, m := range ems {
			status, err := m.delete(tx, roleID)
			if err != nil {
				return status, err
			}
		}
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("transaction failed: %v", err.Error())
	}

	go PurgeCache(h.ItemTypes[h.ItemTypeRole], roleID)

	return http.StatusOK, nil
}

// Delete removes a single criterion from the database
func (m *RoleCriterionType) Delete(roleID int64) (int, error) {
	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	status, err := m.delete(tx, roleID)
	if err != nil {
		return status, err
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("transaction failed: %v", err.Error())
	}

	go PurgeCache(h.ItemTypes[h.ItemTypeRole], roleID)

	return http.StatusOK, nil
}

// delete physically deletes a criterion from the database
func (m *RoleCriterionType) delete(tx *sql.Tx, roleID int64) (int, error) {

	// Only inserts once, cannot break the primary key
	_, err := tx.Exec(`
DELETE FROM criteria
 WHERE role_id = $1
   AND criteria_id = $2`,
		roleID,
		m.ID,
	)
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("error executing delete: %v", err.Error())
	}

	return http.StatusOK, nil
}

// HandleRoleCriterionRequest wraps a request
func HandleRoleCriterionRequest(
	id int64,
	roleID int64,
	seq int,
	out chan<- RoleCriterionRequest,
) {

	item, status, err := GetRoleCriterion(id, roleID)

	response := RoleCriterionRequest{
		Item:   item,
		Status: status,
		Err:    err,
		Seq:    seq,
	}
	out <- response
}

// GetRoleCriterion fetches a single criterion from the database
func GetRoleCriterion(id int64, roleID int64) (RoleCriterionType, int, error) {

	// Retrieve resources
	db, err := h.GetConnection()
	if err != nil {
		return RoleCriterionType{}, http.StatusInternalServerError, err
	}

	var m RoleCriterionType
	err = db.QueryRow(`
SELECT or_group
      ,profile_column
      ,key
      ,type
      ,predicate
      ,value
  FROM criteria
 WHERE criteria_id = $1
   AND role_id = $2`,
		id,
		roleID,
	).Scan(
		&m.OrGroup,
		&m.ProfileColumnNullable,
		&m.AttrKeyNullable,
		&m.Type,
		&m.Predicate,
		&m.ValueString,
	)
	if err == sql.ErrNoRows {
		return RoleCriterionType{}, http.StatusNotFound,
			fmt.Errorf("criterion not found")
	} else if err != nil {
		return RoleCriterionType{}, http.StatusInternalServerError,
			fmt.Errorf("database query failed: %v", err.Error())
	}

	m.ID = id
	if m.ProfileColumnNullable.Valid {
		ok, col := GetProfileColumnBySnake(m.ProfileColumnNullable.String)
		if ok {
			m.ProfileColumn = col.Camel
		} else {
			m.ProfileColumn = m.ProfileColumnNullable.String
		}
	}
	if m.AttrKeyNullable.Valid {
		m.AttrKey = m.AttrKeyNullable.String
	}

	switch m.Type {
	case tString:
		m.Value = m.ValueString
	case tDate:
		s, _ := strconv.ParseFloat(m.ValueString, 64)
		m.Value = s
	case tNumber:
		s, _ := strconv.ParseFloat(m.ValueString, 64)
		m.Value = s
	case tBoolean:
		s, _ := strconv.ParseBool(m.ValueString)
		m.Value = s
	default:
		return RoleCriterionType{}, http.StatusInternalServerError,
			fmt.Errorf("type was not one of string|date|number|boolean")
	}

	return m, http.StatusOK, nil
}

// GetRoleCriteria fetches a criteria
func GetRoleCriteria(
	roleID int64,
	limit int64,
	offset int64,
) (
	[]RoleCriterionType,
	int64,
	int64,
	int,
	error,
) {

	// Retrieve resources
	db, err := h.GetConnection()
	if err != nil {
		return []RoleCriterionType{}, 0, 0, http.StatusInternalServerError, err
	}

	rows, err := db.Query(`
SELECT COUNT(*) OVER() AS total
      ,criteria_id
  FROM criteria
 WHERE role_id = $1
 ORDER BY or_group ASC, profile_column ASC, key ASC
 LIMIT $2
OFFSET $3`,
		roleID,
		limit,
		offset,
	)
	if err != nil {
		return []RoleCriterionType{}, 0, 0, http.StatusInternalServerError,
			fmt.Errorf("database query failed: %v", err.Error())
	}
	defer rows.Close()

	// Get a list of the identifiers of the items to return
	var total int64
	ids := []int64{}
	for rows.Next() {
		var id int64
		err = rows.Scan(
			&total,
			&id,
		)
		if err != nil {
			return []RoleCriterionType{}, 0, 0, http.StatusInternalServerError,
				fmt.Errorf("row parsing error: %v", err.Error())
		}

		ids = append(ids, id)
	}
	err = rows.Err()
	if err != nil {
		return []RoleCriterionType{}, 0, 0, http.StatusInternalServerError,
			fmt.Errorf("error fetching rows: %v", err.Error())
	}
	rows.Close()

	// Make a request for each identifier
	var wg1 sync.WaitGroup
	req := make(chan RoleCriterionRequest)
	defer close(req)

	for seq, id := range ids {
		go HandleRoleCriterionRequest(id, roleID, seq, req)
		wg1.Add(1)
	}

	// Receive the responses and check for errors
	resps := []RoleCriterionRequest{}
	for i := 0; i < len(ids); i++ {
		resp := <-req
		wg1.Done()
		resps = append(resps, resp)
	}
	wg1.Wait()

	for _, resp := range resps {
		if resp.Err != nil {
			return []RoleCriterionType{}, 0, 0, resp.Status, resp.Err
		}
	}

	// Sort them
	sort.Sort(RoleCriterionRequestBySeq(resps))

	// Extract the values
	ems := []RoleCriterionType{}
	for _, resp := range resps {
		m := resp.Item
		ems = append(ems, m)
	}

	pages := h.GetPageCount(total, limit)
	maxOffset := h.GetMaxOffset(total, limit)

	if offset > maxOffset {
		return []RoleCriterionType{}, 0, 0, http.StatusBadRequest,
			fmt.Errorf(
				"not enough records, offset (%d) would return an empty page",
				offset,
			)
	}

	return ems, total, pages, http.StatusOK, nil
}
