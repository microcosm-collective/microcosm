package models

import (
	"database/sql"
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
	tString  string = "string"
	tDate    string = "date"
	tNumber  string = "number"
	tBoolean string = "boolean"
)

// AttributeTypes is a map of valid types of attributes
var AttributeTypes = map[string]int64{
	tString:  1,
	tDate:    2,
	tNumber:  3,
	tBoolean: 4,
}

// AttributesType is a collection of attributes
type AttributesType struct {
	Attributes h.ArrayType `json:"attributes"`
}

// AttributeType is an attribute
type AttributeType struct {
	ID      int64           `json:"-"`
	Key     string          `json:"key"`
	Type    string          `json:"-"`
	Value   interface{}     `json:"value"`
	String  sql.NullString  `json:"-"`
	Number  sql.NullFloat64 `json:"-"`
	Date    pq.NullTime     `json:"-"`
	Boolean sql.NullBool    `json:"-"`
}

// AttributeRequest is a wrapper for a request for an attribute
type AttributeRequest struct {
	Item   AttributeType
	Err    error
	Status int
	Seq    int
}

// AttributeRequestBySeq is a collection of attribute requests
type AttributeRequestBySeq []AttributeRequest

// Len is the length of the collection
func (v AttributeRequestBySeq) Len() int { return len(v) }

// Swap exchanges two items in the collection
func (v AttributeRequestBySeq) Swap(i, j int) { v[i], v[j] = v[j], v[i] }

// Less determines if an attribute is less by sequence than another
func (v AttributeRequestBySeq) Less(i, j int) bool { return v[i].Seq < v[j].Seq }

// Validate returns true if the attribute is valid
func (m *AttributeType) Validate() (int, error) {
	m.Key = CleanWord(m.Key)

	if strings.Trim(m.Key, " ") == "" {
		return http.StatusBadRequest,
			fmt.Errorf("Attribute key cannot be null or empty")
	}

	switch m.Value.(type) {
	case int:
		m.Number =
			sql.NullFloat64{Float64: float64(m.Value.(int)), Valid: true}
		m.Type = tNumber
	case int32:
		m.Number =
			sql.NullFloat64{Float64: float64(m.Value.(int32)), Valid: true}
		m.Type = tNumber
	case int64:
		m.Number =
			sql.NullFloat64{Float64: float64(m.Value.(int64)), Valid: true}
		m.Type = tNumber
	case float32:
		m.Number =
			sql.NullFloat64{Float64: float64(m.Value.(float32)), Valid: true}
		m.Type = tNumber
	case float64:
		m.Number =
			sql.NullFloat64{Float64: m.Value.(float64), Valid: true}
		m.Type = tNumber
	case string:
		m.String =
			sql.NullString{
				String: strings.Trim(CleanSentence(m.Value.(string)), " "),
				Valid:  true,
			}
		m.Type = tString
	case bool:
		m.Boolean =
			sql.NullBool{Bool: m.Value.(bool), Valid: true}
		m.Type = tBoolean
	default:
		return http.StatusBadRequest,
			fmt.Errorf("the type of value cannot be determined or is invalid")
	}

	switch m.Type {
	case tString:
		if m.String.String == "" {
			return http.StatusBadRequest, fmt.Errorf("value is null or empty")
		}

		t, err := time.Parse("2006-01-02", m.String.String)
		if err == nil {
			m.Date = pq.NullTime{Time: t, Valid: true}
			m.Type = tDate
			m.String = sql.NullString{}
		}
	case tNumber:
		if m.Number.Float64 == math.MaxFloat64 {
			return http.StatusBadRequest,
				fmt.Errorf("type = number, but number is null or empty")
		}
	}

	return http.StatusOK, nil
}

// UpdateManyAttributes updates many attributes at once
func UpdateManyAttributes(
	itemTypeID int64,
	itemID int64,
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
		status, err := m.upsert(tx, itemTypeID, itemID)
		if err != nil {
			return status, err
		}
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("Transaction failed: %v", err.Error())
	}

	return http.StatusOK, nil
}

// Update updates a single attribute
func (m *AttributeType) Update(itemTypeID int64, itemID int64) (int, error) {
	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	status, err := m.upsert(tx, itemTypeID, itemID)
	if err != nil {
		return status, err
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("Transaction failed: %v", err.Error())
	}

	return http.StatusOK, nil
}

func (m *AttributeType) upsert(
	tx *sql.Tx,
	itemTypeID int64,
	itemID int64,
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
		itemTypeID,
		itemID,
		m.Key,
		AttributeTypes[m.Type],
		m.String,
		m.Date,
		m.Number,
		m.Boolean,
	)
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("Error executing update: %v", err.Error())
	}

	// If update was successful, we are done
	if rowsAffected, _ := res.RowsAffected(); rowsAffected > 0 {
		if itemTypeID == h.ItemTypes[h.ItemTypeProfile] {
			status, err = FlushRoleMembersCacheByProfileID(tx, itemID)
			if err != nil {
				return http.StatusInternalServerError,
					fmt.Errorf(
						"Error flushing role members cache: %+v",
						err,
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
		itemTypeID,
		itemID,
		m.Key,
	).Scan(
		&m.ID,
	)
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("Error inserting data and returning ID: %+v", err)
	}

	res, err = tx.Exec(`
INSERT INTO attribute_values (
    attribute_id, value_type_id, string, date, "number",
    "boolean"
) VALUES (
    $1, $2, $3, $4, $5,
    $6
)`,
		m.ID,
		AttributeTypes[m.Type],
		m.String,
		m.Date,
		m.Number,
		m.Boolean,
	)
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("Error executing insert: %v", err.Error())
	}

	if itemTypeID == h.ItemTypes[h.ItemTypeProfile] {
		status, err = FlushRoleMembersCacheByProfileID(tx, itemID)
		if err != nil {
			return http.StatusInternalServerError,
				fmt.Errorf("Error flushing role members cache: %+v", err)
		}
	}

	return http.StatusOK, nil
}

// DeleteManyAttributes removes many attributes
func DeleteManyAttributes(
	itemTypeID int64,
	itemID int64,
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
			attributeID int64
			status      int
		)

		if m.ID > 0 {
			attributeID = m.ID
		} else {
			attributeID, status, err = GetAttributeID(itemTypeID, itemID, m.Key)
			if err != nil {
				return status, err
			}
			m.ID = attributeID
		}

		status, err := m.delete(tx)
		if err != nil {
			return status, err
		}
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("Transaction failed: %v", err.Error())
	}

	return http.StatusOK, nil
}

// Delete removes a single attribute
func (m *AttributeType) Delete() (int, error) {
	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	status, err := m.delete(tx)
	if err != nil {
		return status, err
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("Transaction failed: %v", err.Error())
	}

	return http.StatusOK, nil
}

func (m *AttributeType) delete(tx *sql.Tx) (int, error) {
	_, err := tx.Exec(`
DELETE FROM attribute_values
 WHERE attribute_id = $1`,
		m.ID,
	)
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("Failed to delete attribute value: %v", err)
	}

	_, err = tx.Exec(`
DELETE FROM attribute_keys
 WHERE attribute_id = $1`,
		m.ID,
	)
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("Failed to delete attribute key: %+v", err)
	}

	return http.StatusOK, nil
}

// HandleAttributeRequest fetches an attribute to fulfil a request
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

// GetAttributeID fetches the id of an attribute
func GetAttributeID(
	itemTypeID int64,
	itemID int64,
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

	var attrID int64

	err = db.QueryRow(`
SELECT attribute_id
  FROM attribute_keys
 WHERE item_type_id = $1
   AND item_id = $2
   AND key = $3
`,
		itemTypeID,
		itemID,
		key,
	).Scan(
		&attrID,
	)
	if err == sql.ErrNoRows {
		return attrID, http.StatusNotFound, fmt.Errorf("Attribute not found.")

	} else if err != nil {
		return attrID, http.StatusInternalServerError,
			fmt.Errorf("Database query failed: %v", err.Error())
	}

	return attrID, http.StatusOK, nil
}

// GetAttribute returns an attribute
func GetAttribute(id int64) (AttributeType, int, error) {
	db, err := h.GetConnection()
	if err != nil {
		return AttributeType{}, http.StatusInternalServerError, err
	}

	var typeID int64

	m := AttributeType{ID: id}
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
		&typeID,
		&m.String,
		&m.Date,
		&m.Number,
		&m.Boolean,
	)
	if err == sql.ErrNoRows {
		return AttributeType{}, http.StatusNotFound,
			fmt.Errorf("Attribute not found: %v", err.Error())
	} else if err != nil {
		return AttributeType{}, http.StatusInternalServerError,
			fmt.Errorf("Database query failed: %v", err.Error())
	}

	typeStr, err := h.GetMapStringFromInt(AttributeTypes, typeID)
	if err != nil {
		return AttributeType{}, http.StatusInternalServerError,
			fmt.Errorf("Type is not a valid attribute type: %v", err.Error())
	}
	m.Type = typeStr

	switch m.Type {
	case tString:
		if m.String.Valid {
			m.Value = m.String.String
		} else {
			return AttributeType{}, http.StatusInternalServerError,
				fmt.Errorf("Type is string, but value is invalid")
		}
	case tDate:
		if m.Date.Valid {
			m.Value = m.Date.Time.Format("2006-01-02")
		} else {
			return AttributeType{}, http.StatusInternalServerError,
				fmt.Errorf("Type is date, but value is invalid")
		}
	case tNumber:
		if m.Number.Valid {
			m.Value = m.Number.Float64
		} else {
			return AttributeType{}, http.StatusInternalServerError,
				fmt.Errorf("Type is number, but value is invalid")
		}
	case tBoolean:
		if m.Boolean.Valid {
			m.Value = m.Boolean.Bool
		} else {
			return AttributeType{}, http.StatusInternalServerError,
				fmt.Errorf("Type is boolean, but value is invalid")
		}
	default:
		return AttributeType{}, http.StatusInternalServerError,
			fmt.Errorf("Type was not one of string|date|number|boolean")
	}

	return m, http.StatusOK, nil
}

// GetAttributes fetches a collection of attributes
func GetAttributes(
	itemTypeID int64,
	itemID int64,
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
		itemTypeID,
		itemID,
		limit,
		offset,
	)
	if err != nil {
		return []AttributeType{}, 0, 0, http.StatusInternalServerError,
			fmt.Errorf("Database query failed: %v", err.Error())
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
			return []AttributeType{}, 0, 0, http.StatusInternalServerError,
				fmt.Errorf("Row parsing error: %v", err.Error())
		}

		ids = append(ids, id)

	}
	err = rows.Err()
	if err != nil {
		return []AttributeType{}, 0, 0, http.StatusInternalServerError,
			fmt.Errorf("Error fetching rows: %v", err.Error())
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
		return []AttributeType{}, 0, 0, http.StatusBadRequest,
			fmt.Errorf(
				"not enough records, offset (%d) would return an empty page",
				offset,
			)
	}

	return ems, total, pages, http.StatusOK, nil
}
