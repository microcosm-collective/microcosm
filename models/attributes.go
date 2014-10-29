package models

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/lib/pq"

	h "github.com/microcosm-cc/microcosm/helpers"
)

const (
	STRING  string = "string"
	DATE    string = "date"
	NUMBER  string = "number"
	BOOLEAN string = "boolean"
)

var AttributeTypes = map[string]int64{
	STRING:  1,
	DATE:    2,
	NUMBER:  3,
	BOOLEAN: 4,
}

type AttributesType struct {
	Attributes h.ArrayType `json:"attributes"`
}

type AttributeType struct {
	Id      int64           `json:"-"`
	Key     string          `json:"key"`
	Type    string          `json:"-"`
	Value   interface{}     `json:"value"`
	String  sql.NullString  `json:"-"`
	Number  sql.NullFloat64 `json:"-"`
	Date    pq.NullTime     `json:"-"`
	Boolean sql.NullBool    `json:"-"`
}

type AttributeRequest struct {
	Item   AttributeType
	Err    error
	Status int
	Seq    int
}

type AttributeRequestBySeq []AttributeRequest

func (v AttributeRequestBySeq) Len() int           { return len(v) }
func (v AttributeRequestBySeq) Swap(i, j int)      { v[i], v[j] = v[j], v[i] }
func (v AttributeRequestBySeq) Less(i, j int) bool { return v[i].Seq < v[j].Seq }

func (m *AttributeType) Validate() (int, error) {

	m.Key = SanitiseText(m.Key)

	if strings.Trim(m.Key, " ") == "" {
		return http.StatusBadRequest,
			errors.New("Attribute key cannot be null or empty")
	}

	switch m.Value.(type) {
	case int:
		m.Number =
			sql.NullFloat64{Float64: float64(m.Value.(int)), Valid: true}
		m.Type = NUMBER
	case int32:
		m.Number =
			sql.NullFloat64{Float64: float64(m.Value.(int32)), Valid: true}
		m.Type = NUMBER
	case int64:
		m.Number =
			sql.NullFloat64{Float64: float64(m.Value.(int64)), Valid: true}
		m.Type = NUMBER
	case float32:
		m.Number =
			sql.NullFloat64{Float64: float64(m.Value.(float32)), Valid: true}
		m.Type = NUMBER
	case float64:
		m.Number =
			sql.NullFloat64{Float64: m.Value.(float64), Valid: true}
		m.Type = NUMBER
	case string:
		m.String =
			sql.NullString{
				String: strings.Trim(SanitiseText(m.Value.(string)), " "),
				Valid:  true,
			}
		m.Type = STRING
	case bool:
		m.Boolean =
			sql.NullBool{Bool: m.Value.(bool), Valid: true}
		m.Type = BOOLEAN
	default:
		return http.StatusBadRequest,
			errors.New("the type of value cannot be determined or is invalid")
	}

	switch m.Type {
	case STRING:
		if m.String.String == "" {
			return http.StatusBadRequest, errors.New("value is null or empty")
		}

		t, err := time.Parse("2006-01-02", m.String.String)
		if err == nil {
			m.Date = pq.NullTime{Time: t, Valid: true}
			m.Type = DATE
			m.String = sql.NullString{}
		}
	case NUMBER:
		if m.Number.Float64 == math.MaxFloat64 {
			return http.StatusBadRequest,
				errors.New("type = number, but number is null or empty")
		}
	}

	return http.StatusOK, nil
}

func UpdateManyAttributes(
	itemTypeId int64,
	itemId int64,
	ems []AttributeType,
) (
	int,
	error,
) {

	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	for _, m := range ems {
		status, err := m.upsert(tx, itemTypeId, itemId)
		if err != nil {
			return status, err
		}
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			errors.New(fmt.Sprintf("Transaction failed: %v", err.Error()))
	}

	return http.StatusOK, nil
}

func (m *AttributeType) Update(itemTypeId int64, itemId int64) (int, error) {
	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	status, err := m.upsert(tx, itemTypeId, itemId)
	if err != nil {
		return status, err
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			errors.New(fmt.Sprintf("Transaction failed: %v", err.Error()))
	}

	return http.StatusOK, nil
}

func (m *AttributeType) upsert(
	tx *sql.Tx,
	itemTypeId int64,
	itemId int64,
) (
	int,
	error,
) {

	status, err := m.Validate()
	if err != nil {
		return status, err
	}

	res, err := tx.Exec(`
UPDATE attribute_values
   SET value_type_id = $4
      ,string = $5
      ,date = $6
      ,"number" = $7
      ,"boolean" = $8
WHERE attribute_id IN (
      SELECT attribute_id
        FROM attribute_keys
       WHERE item_type_id = $1
         AND item_id = $2
         AND key = $3)`,
		itemTypeId,
		itemId,
		m.Key,
		AttributeTypes[m.Type],
		m.String,
		m.Date,
		m.Number,
		m.Boolean,
	)
	if err != nil {
		return http.StatusInternalServerError,
			errors.New(fmt.Sprintf("Error executing update: %v", err.Error()))
	}

	// If update was successful, we are done
	if rowsAffected, _ := res.RowsAffected(); rowsAffected > 0 {

		if itemTypeId == h.ItemTypes[h.ItemTypeProfile] {

			status, err = FlushRoleMembersCacheByProfileId(tx, itemId)
			if err != nil {
				return http.StatusInternalServerError,
					errors.New(
						fmt.Sprintf(
							"Error flushing role members cache: %+v",
							err,
						),
					)
			}
		}

		return http.StatusOK, nil
	}

	err = tx.QueryRow(`
INSERT INTO attribute_keys (
    item_type_id, item_id, key
) VALUES (
    $1, $2, $3
) RETURNING attribute_id`,
		itemTypeId,
		itemId,
		m.Key,
	).Scan(
		&m.Id,
	)
	if err != nil {
		return http.StatusInternalServerError,
			errors.New(
				fmt.Sprintf("Error inserting data and returning ID: %+v", err),
			)
	}

	res, err = tx.Exec(`
INSERT INTO attribute_values (
    attribute_id, value_type_id, string, date, "number",
    "boolean"
) VALUES (
    $1, $2, $3, $4, $5,
    $6
)`,
		m.Id,
		AttributeTypes[m.Type],
		m.String,
		m.Date,
		m.Number,
		m.Boolean,
	)
	if err != nil {
		return http.StatusInternalServerError,
			errors.New(fmt.Sprintf("Error executing insert: %v", err.Error()))
	}

	if itemTypeId == h.ItemTypes[h.ItemTypeProfile] {
		status, err = FlushRoleMembersCacheByProfileId(tx, itemId)
		if err != nil {
			return http.StatusInternalServerError,
				errors.New(
					fmt.Sprintf("Error flushing role members cache: %+v", err),
				)
		}
	}

	return http.StatusOK, nil
}

func DeleteManyAttributes(
	itemTypeId int64,
	itemId int64,
	ms []AttributeType,
) (
	int,
	error,
) {

	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	for _, m := range ms {

		var (
			attributeId int64
			status      int
		)

		if m.Id > 0 {
			attributeId = m.Id
		} else {
			attributeId, status, err = GetAttributeId(itemTypeId, itemId, m.Key)
			if err != nil {
				return status, err
			}
		}

		status, err := m.delete(tx, attributeId)
		if err != nil {
			return status, err
		}
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			errors.New(fmt.Sprintf("Transaction failed: %v", err.Error()))
	}

	return http.StatusOK, nil
}

func (m *AttributeType) Delete() (int, error) {
	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	status, err := m.delete(tx, m.Id)
	if err != nil {
		return status, err
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			errors.New(fmt.Sprintf("Transaction failed: %v", err.Error()))
	}

	return http.StatusOK, nil
}

func (m *AttributeType) delete(tx *sql.Tx, attributeId int64) (int, error) {

	_, err := tx.Exec(`
DELETE FROM attribute_values
 WHERE attribute_id = $1`,
		attributeId,
	)
	if err != nil {
		return http.StatusInternalServerError,
			errors.New(fmt.Sprintf("Failed to delete attribute value: %v", err))
	}

	_, err = tx.Exec(`
DELETE FROM attribute_keys
 WHERE attribute_id = $1`,
		attributeId,
	)
	if err != nil {
		return http.StatusInternalServerError,
			errors.New(fmt.Sprintf("Failed to delete attribute key: %+v", err))
	}

	return http.StatusOK, nil
}

func HandleAttributeRequest(id int64, seq int, out chan<- AttributeRequest) {
	item, status, err := GetAttribute(id)

	response := AttributeRequest{
		Item:   item,
		Status: status,
		Err:    err,
		Seq:    seq,
	}
	out <- response
}

func GetAttributeId(
	itemTypeId int64,
	itemId int64,
	key string,
) (
	int64,
	int,
	error,
) {

	db, err := h.GetConnection()
	if err != nil {
		return 0, http.StatusInternalServerError, err
	}

	var attrId int64

	err = db.QueryRow(`
SELECT attribute_id
  FROM attribute_keys
 WHERE item_type_id = $1
   AND item_id = $2
   AND key = $3
`,
		itemTypeId,
		itemId,
		key,
	).Scan(
		&attrId,
	)
	if err == sql.ErrNoRows {
		return attrId, http.StatusNotFound, errors.New("Attribute not found.")

	} else if err != nil {
		return attrId, http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Database query failed: %v", err.Error()),
		)
	}

	return attrId, http.StatusOK, nil
}

func GetAttribute(id int64) (AttributeType, int, error) {

	db, err := h.GetConnection()
	if err != nil {
		return AttributeType{}, http.StatusInternalServerError, err
	}

	var typeId int64

	m := AttributeType{Id: id}
	err = db.QueryRow(`
SELECT k.key
      ,v.value_type_id
      ,v.string
      ,v.date
      ,v."number"
      ,v."boolean"
  FROM attribute_keys k,
       attribute_values v
 WHERE k.attribute_id = v.attribute_id
   AND k.attribute_id = $1`,
		id,
	).Scan(
		&m.Key,
		&typeId,
		&m.String,
		&m.Date,
		&m.Number,
		&m.Boolean,
	)
	if err == sql.ErrNoRows {
		return AttributeType{}, http.StatusNotFound, errors.New(
			fmt.Sprintf("Attribute not found: %v", err.Error()),
		)
	} else if err != nil {
		return AttributeType{}, http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Database query failed: %v", err.Error()),
		)
	}

	typeStr, err := h.GetMapStringFromInt(AttributeTypes, typeId)
	if err != nil {
		return AttributeType{}, http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Type is not a valid attribute type: %v", err.Error()),
		)
	}
	m.Type = typeStr

	switch m.Type {
	case STRING:
		if m.String.Valid {
			m.Value = m.String.String
		} else {
			return AttributeType{}, http.StatusInternalServerError,
				errors.New("Type is string, but value is invalid")
		}
	case DATE:
		if m.Date.Valid {
			m.Value = m.Date.Time.Format("2006-01-02")
		} else {
			return AttributeType{}, http.StatusInternalServerError,
				errors.New("Type is date, but value is invalid")
		}
	case NUMBER:
		if m.Number.Valid {
			m.Value = m.Number.Float64
		} else {
			return AttributeType{}, http.StatusInternalServerError,
				errors.New("Type is number, but value is invalid")
		}
	case BOOLEAN:
		if m.Boolean.Valid {
			m.Value = m.Boolean.Bool
		} else {
			return AttributeType{}, http.StatusInternalServerError,
				errors.New("Type is boolean, but value is invalid")
		}
	default:
		return AttributeType{}, http.StatusInternalServerError,
			errors.New("Type was not one of string|date|number|boolean")
	}

	return m, http.StatusOK, nil
}

func GetAttributes(
	itemTypeId int64,
	itemId int64,
	limit int64,
	offset int64,
) (
	[]AttributeType,
	int64,
	int64,
	int,
	error,
) {

	// Retrieve resources
	db, err := h.GetConnection()
	if err != nil {
		return []AttributeType{}, 0, 0, http.StatusInternalServerError, err
	}

	rows, err := db.Query(`
SELECT COUNT(*) OVER() AS total
      ,attribute_id
  FROM attribute_keys
 WHERE item_type_id = $1
   AND item_id = $2
 ORDER BY key ASC
 LIMIT $3
OFFSET $4`,
		itemTypeId,
		itemId,
		limit,
		offset,
	)
	if err != nil {
		return []AttributeType{}, 0, 0, http.StatusInternalServerError, errors.New(
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
			return []AttributeType{}, 0, 0, http.StatusInternalServerError, errors.New(
				fmt.Sprintf("Row parsing error: %v", err.Error()),
			)
		}

		ids = append(ids, id)

	}
	err = rows.Err()
	if err != nil {
		return []AttributeType{}, 0, 0, http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Error fetching rows: %v", err.Error()),
		)
	}
	rows.Close()

	// Make a request for each identifier
	var wg1 sync.WaitGroup
	req := make(chan AttributeRequest)
	defer close(req)

	for seq, id := range ids {
		go HandleAttributeRequest(id, seq, req)
		wg1.Add(1)
	}

	// Receive the responses and check for errors
	resps := []AttributeRequest{}
	for i := 0; i < len(ids); i++ {
		resp := <-req
		wg1.Done()
		resps = append(resps, resp)
	}
	wg1.Wait()

	for _, resp := range resps {
		if resp.Err != nil {
			return []AttributeType{}, 0, 0, resp.Status, resp.Err
		}
	}

	// Sort them
	sort.Sort(AttributeRequestBySeq(resps))

	// Extract the values
	ems := []AttributeType{}
	for _, resp := range resps {
		ems = append(ems, resp.Item)
	}

	pages := h.GetPageCount(total, limit)
	maxOffset := h.GetMaxOffset(total, limit)

	if offset > maxOffset {
		return []AttributeType{}, 0, 0, http.StatusBadRequest, errors.New(
			fmt.Sprintf(
				"not enough records, offset (%d) would return an empty page.",
				offset,
			),
		)
	}

	return ems, total, pages, http.StatusOK, nil
}
