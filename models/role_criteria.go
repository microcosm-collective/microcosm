package models

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"

	h "github.com/microcosm-cc/microcosm/helpers"
)

const (
	// Every data type
	Equals    string = "eq"
	NotEquals string = "ne"

	// Numbers and Dates
	LessThan            string = "lt"
	LessThanOrEquals    string = "le"
	GreaterThanOrEquals string = "ge"
	GreaterThan         string = "gt"

	// Strings
	Substring    string = "substr"
	NotSubstring string = "nsubstr"
)

var (
	StringPredicates = []string{Equals, NotEquals, Substring, NotSubstring}
	Int64Predicates  = []string{
		Equals,
		NotEquals,
		LessThan,
		LessThanOrEquals,
		GreaterThanOrEquals,
		GreaterThan,
	}
	TimePredicates = []string{
		Equals,
		NotEquals,
		LessThan,
		LessThanOrEquals,
		GreaterThanOrEquals,
		GreaterThan,
	}
	BoolPredicates = []string{Equals, NotEquals}

	ProfileColumns = []ProfileColumn{
		ProfileColumn{
			Camel:      "id",
			Snake:      "profile_id",
			Type:       "int64",
			Predicates: Int64Predicates,
		},
		ProfileColumn{
			Camel:      "profileName",
			Snake:      "profile_name",
			Type:       "string",
			Predicates: StringPredicates,
		},
		ProfileColumn{
			Camel:      "gender",
			Snake:      "gender",
			Type:       "string",
			Predicates: StringPredicates,
		},
		ProfileColumn{
			Camel:      "email",
			Snake:      "email",
			Type:       "string",
			Predicates: StringPredicates,
		},
		ProfileColumn{
			Camel:      "itemCount",
			Snake:      "item_count",
			Type:       "int64",
			Predicates: Int64Predicates,
		},
		ProfileColumn{
			Camel:      "commentCount",
			Snake:      "comment_count",
			Type:       "int64",
			Predicates: Int64Predicates,
		},
		ProfileColumn{
			Camel:      "created",
			Snake:      "created",
			Type:       "time",
			Predicates: TimePredicates,
		},
		ProfileColumn{
			Camel:      "isBanned",
			Snake:      "is_banned",
			Type:       "bool",
			Predicates: BoolPredicates,
		},
	}
)

type ProfileColumn struct {
	Camel      string
	Snake      string
	Type       string
	Predicates []string
}

type RoleCriteriaType struct {
	RoleCriteria h.ArrayType    `json:"criteria"`
	Meta         h.CoreMetaType `json:"meta"`
}

type RoleCriterionType struct {
	Id                    int64          `json:"id,omitempty"`
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

type RoleCriterionRequest struct {
	Item   RoleCriterionType
	Err    error
	Status int
	Seq    int
}

type RoleCriterionRequestBySeq []RoleCriterionRequest

func (v RoleCriterionRequestBySeq) Len() int {
	return len(v)
}

func (v RoleCriterionRequestBySeq) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

func (v RoleCriterionRequestBySeq) Less(i, j int) bool {
	return v[i].Seq < v[j].Seq
}

func GetProfileColumnByCamel(key string) (bool, ProfileColumn) {
	for _, m := range ProfileColumns {
		if m.Camel == key {
			return true, m
		}
	}

	return false, ProfileColumn{}
}

func GetProfileColumnBySnake(key string) (bool, ProfileColumn) {
	for _, m := range ProfileColumns {
		if m.Snake == key {
			return true, m
		}
	}

	return false, ProfileColumn{}
}

func (m *ProfileColumn) ValidPredicate(predicate string) bool {
	for _, v := range m.Predicates {
		if v == predicate {
			return true
		}
	}

	return false
}

func (m *RoleCriterionType) GetLink(roleLink string) string {
	return fmt.Sprintf("%s/criteria/%d", roleLink, m.Id)
}

func (m *RoleCriterionType) Validate(exists bool) (int, error) {

	if exists && m.Id < 1 {
		return http.StatusBadRequest,
			errors.New("profile id needs to be a positive integer")
	}

	// coerce value
	switch m.Value.(type) {
	case int:
		m.ValueString = strconv.FormatInt(int64(m.Value.(int)), 10)
		m.Type = NUMBER
	case int32:
		m.ValueString = strconv.FormatInt(int64(m.Value.(int32)), 10)
		m.Type = NUMBER
	case int64:
		m.ValueString = strconv.FormatInt(m.Value.(int64), 10)
		m.Type = NUMBER
	case float32:
		m.ValueString =
			strconv.FormatFloat(float64(m.Value.(float32)), 'f', -1, 64)
		m.Type = NUMBER
	case float64:
		m.ValueString = strconv.FormatFloat(m.Value.(float64), 'f', -1, 64)
		m.Type = NUMBER
	case string:
		m.ValueString = strings.Trim(m.Value.(string), " ")
		m.Type = STRING
	case bool:
		if m.Value.(bool) {
			m.ValueString = "true"
		} else {
			m.ValueString = "false"
		}
		m.Type = BOOLEAN
	default:
		return http.StatusBadRequest,
			errors.New("The type of `value` cannot be determined or is invalid")
	}

	if m.Type == "" {
		return http.StatusBadRequest,
			errors.New("value cannot be null, empty or just whitespace")
	}

	// Validate profileColumn
	if strings.Trim(m.ProfileColumn, " ") != "" {

		exists, col :=
			GetProfileColumnByCamel(strings.Trim(m.ProfileColumn, " "))
		if !exists {
			return http.StatusBadRequest,
				errors.New("The profileColumn is not valid")
		}

		if !col.ValidPredicate(m.Predicate) {
			return http.StatusBadRequest,
				errors.New("The predicate for the profileColumn is not valid")
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
			case STRING:
				var found bool
				for _, v := range StringPredicates {
					if v == m.Predicate {
						found = true
					}
				}
				if !found {
					return http.StatusBadRequest,
						errors.New(
							"The predicate for the attribute type is not valid",
						)
				}
			case NUMBER:
				var found bool
				for _, v := range Int64Predicates {
					if v == m.Predicate {
						found = true
					}
				}
				if !found {
					return http.StatusBadRequest,
						errors.New(
							"The predicate for the attribute type is not valid",
						)
				}
			case BOOLEAN:
				var found bool
				for _, v := range BoolPredicates {
					if v == m.Predicate {
						found = true
					}
				}
				if !found {
					return http.StatusBadRequest,
						errors.New(
							"The predicate for the attribute type is not valid",
						)
				}
			default:
				return http.StatusBadRequest, errors.New("type is not valid")
			}
		} else {
			return http.StatusBadRequest,
				errors.New(
					"Either profileColumn or attrKey MUST be supplied",
				)
		}
	}

	return http.StatusOK, nil
}

func (m *RoleCriterionType) Insert(roleId int64) (int, error) {

	status, err := m.Validate(false)
	if err != nil {
		return status, err
	}

	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	var insertId int64
	err = tx.QueryRow(`
INSERT INTO criteria (
       role_id, or_group, profile_column, key, type,
       predicate, value
) VALUES (
       $1, $2, $3, $4, $5,
       $6, $7
) RETURNING criteria_id`,
		roleId,
		m.OrGroup,
		m.ProfileColumnNullable,
		m.AttrKeyNullable,
		m.Type,

		m.Predicate,
		m.ValueString,
	).Scan(
		&insertId,
	)
	if err != nil {
		return http.StatusInternalServerError,
			errors.New(fmt.Sprintf("Error executing insert: %v", err.Error()))
	}
	m.Id = insertId

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			errors.New(fmt.Sprintf("Transaction failed: %v", err.Error()))
	}

	go PurgeCache(h.ItemTypes[h.ItemTypeRole], roleId)

	return http.StatusOK, nil
}

func (m *RoleCriterionType) Update(roleId int64) (int, error) {

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
		m.Id,
		m.OrGroup,
		m.ProfileColumnNullable,
		m.AttrKeyNullable,
		m.Type,

		m.Predicate,
		m.ValueString,
	)
	if err != nil {
		return http.StatusInternalServerError,
			errors.New(fmt.Sprintf("Error executing insert: %v", err.Error()))
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			errors.New(fmt.Sprintf("Transaction failed: %v", err.Error()))
	}

	go PurgeCache(h.ItemTypes[h.ItemTypeRole], roleId)

	return http.StatusOK, nil
}

func DeleteManyRoleCriteria(roleId int64, ems []RoleCriterionType) (int, error) {
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
			roleId,
		)
		if err != nil {
			return http.StatusInternalServerError,
				errors.New(fmt.Sprintf("Error executing delete: %+v", err))
		}
	} else {
		// Delete specific profiles
		for _, m := range ems {
			status, err := m.delete(tx, roleId)
			if err != nil {
				return status, err
			}
		}
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			errors.New(fmt.Sprintf("Transaction failed: %v", err.Error()))
	}

	go PurgeCache(h.ItemTypes[h.ItemTypeRole], roleId)

	return http.StatusOK, nil
}

func (m *RoleCriterionType) Delete(roleId int64) (int, error) {
	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	status, err := m.delete(tx, roleId)
	if err != nil {
		return status, err
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			errors.New(fmt.Sprintf("Transaction failed: %v", err.Error()))
	}

	go PurgeCache(h.ItemTypes[h.ItemTypeRole], roleId)

	return http.StatusOK, nil
}

func (m *RoleCriterionType) delete(tx *sql.Tx, roleId int64) (int, error) {

	// Only inserts once, cannot break the primary key
	_, err := tx.Exec(`
DELETE FROM criteria
 WHERE role_id = $1
   AND criteria_id = $2`,
		roleId,
		m.Id,
	)
	if err != nil {
		return http.StatusInternalServerError,
			errors.New(fmt.Sprintf("Error executing delete: %v", err.Error()))
	}

	return http.StatusOK, nil
}

func HandleRoleCriterionRequest(
	id int64,
	roleId int64,
	seq int,
	out chan<- RoleCriterionRequest,
) {

	item, status, err := GetRoleCriterion(id, roleId)

	response := RoleCriterionRequest{
		Item:   item,
		Status: status,
		Err:    err,
		Seq:    seq,
	}
	out <- response
}

func GetRoleCriterion(id int64, roleId int64) (RoleCriterionType, int, error) {

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
		roleId,
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
			errors.New("criterion not found")
	} else if err != nil {
		return RoleCriterionType{}, http.StatusInternalServerError,
			errors.New(
				fmt.Sprintf("Database query failed: %v", err.Error()),
			)
	}

	m.Id = id
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
	case STRING:
		m.Value = m.ValueString
	case DATE:
		s, _ := strconv.ParseFloat(m.ValueString, 64)
		m.Value = s
	case NUMBER:
		s, _ := strconv.ParseFloat(m.ValueString, 64)
		m.Value = s
	case BOOLEAN:
		s, _ := strconv.ParseBool(m.ValueString)
		m.Value = s
	default:
		return RoleCriterionType{}, http.StatusInternalServerError,
			errors.New("Type was not one of string|date|number|boolean")
	}

	return m, http.StatusOK, nil
}

func GetRoleCriteria(
	roleId int64,
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
		roleId,
		limit,
		offset,
	)
	if err != nil {
		return []RoleCriterionType{}, 0, 0, http.StatusInternalServerError,
			errors.New(
				fmt.Sprintf("Database query failed: %v", err.Error()),
			)
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
				errors.New(
					fmt.Sprintf("Row parsing error: %v", err.Error()),
				)
		}

		ids = append(ids, id)
	}
	err = rows.Err()
	if err != nil {
		return []RoleCriterionType{}, 0, 0, http.StatusInternalServerError,
			errors.New(
				fmt.Sprintf("Error fetching rows: %v", err.Error()),
			)
	}
	rows.Close()

	// Make a request for each identifier
	var wg1 sync.WaitGroup
	req := make(chan RoleCriterionRequest)
	defer close(req)

	for seq, id := range ids {
		go HandleRoleCriterionRequest(id, roleId, seq, req)
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
		return []RoleCriterionType{}, 0, 0, http.StatusBadRequest, errors.New(
			fmt.Sprintf(
				"not enough records, offset (%d) would return an empty page.",
				offset,
			),
		)
	}

	return ems, total, pages, http.StatusOK, nil
}
