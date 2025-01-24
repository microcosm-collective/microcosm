package models

import (
	"database/sql"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/lib/pq"

	c "github.com/microcosm-collective/microcosm/cache"
	h "github.com/microcosm-collective/microcosm/helpers"
)

// RSVPStates is a map of reply types, the numerical order is implicitly
// important (it's the sort field)
var RSVPStates = map[string]int64{
	"yes":     1,
	"maybe":   2,
	"invited": 3,
	"no":      4,
}

// AttendeesType is a collection of attendees
type AttendeesType struct {
	Attendees h.ArrayType    `json:"attendees"`
	Meta      h.CoreMetaType `json:"meta"`
}

// AttendeeType is an attendee on an event
type AttendeeType struct {
	ID        int64       `json:"-"`
	EventID   int64       `json:"-"`
	ProfileID int64       `json:"profileId,omitempty"`
	Profile   interface{} `json:"profile"`
	RSVPID    int64       `json:"-"`
	RSVP      string      `json:"rsvp"`
	RSVPd     pq.NullTime `json:"-"`
	RSVPdOn   string      `json:"rsvpdOn,omitempty"`

	Meta h.DefaultNoFlagsMetaType `json:"meta"`
}

// AttendeeRequest is a request for an attendee
type AttendeeRequest struct {
	Item   AttendeeType
	Err    error
	Status int
	Seq    int
}

// AttendeeRequestBySeq is a collection of requests for attendees
type AttendeeRequestBySeq []AttendeeRequest

// Len is the length of the collection
func (v AttendeeRequestBySeq) Len() int { return len(v) }

// Swap exchanges two items in the collection
func (v AttendeeRequestBySeq) Swap(i, j int) { v[i], v[j] = v[j], v[i] }

// Less determines if an item is less then another by sequence
func (v AttendeeRequestBySeq) Less(i, j int) bool { return v[i].Seq < v[j].Seq }

// Validate returns true if the attendee is valid
func (m *AttendeeType) Validate(tx *sql.Tx) (int, error) {
	if m.ProfileID <= 0 {
		glog.Infoln("m.ProfileId <= 0")
		return http.StatusBadRequest,
			fmt.Errorf("you must specify the attendees Profile ID")
	}

	if strings.Trim(m.RSVP, " ") == "" {
		m.RSVP = "invited"
	}

	if _, inList := RSVPStates[m.RSVP]; !inList {
		glog.Infoln("inList := RSVPStates[m.RSVP]; !inList")
		return http.StatusBadRequest,
			fmt.Errorf("you must specify a valid rsvp value " +
				"('invited', 'yes', 'maybe', or 'no')")
	}

	if m.RSVP == "yes" {
		//check to see if event is full

		var spaces, rsvpLimit int64
		err := tx.QueryRow(`
SELECT rsvp_spaces
      ,rsvp_limit
  FROM events
 WHERE event_id = $1`,
			m.EventID,
		).Scan(
			&spaces,
			&rsvpLimit,
		)
		if err != nil {
			glog.Errorf("tx.QueryRow(%d).Scan() %+v", m.EventID, err)
			return http.StatusInternalServerError,
				fmt.Errorf("error fetching row")
		}

		if spaces <= 0 && rsvpLimit != 0 {
			glog.Infoln("spaces <= 0 && rsvpLimit != 0")
			return http.StatusBadRequest, fmt.Errorf("event is full")
		}
	}

	m.RSVPd = m.Meta.EditedNullable
	m.RSVPID = RSVPStates[m.RSVP]

	return http.StatusOK, nil
}

// Hydrate populates a partially populated struct
func (m *AttendeeType) Hydrate(siteID int64) (int, error) {

	profile, status, err := GetProfileSummary(siteID, m.ProfileID)
	if err != nil {
		glog.Errorf(
			"GetProfileSummary(%d, %d) %+v",
			siteID,
			m.ProfileID,
			err,
		)
		return status, err
	}
	m.Profile = profile

	profile, status, err = GetProfileSummary(siteID, m.Meta.CreatedByID)
	if err != nil {
		glog.Errorf(
			"GetProfileSummary(%d, %d) %+v",
			siteID,
			m.Meta.CreatedByID,
			err,
		)
		return status, err
	}
	m.Meta.CreatedBy = profile

	if m.Meta.EditedByNullable.Valid {
		profile, status, err :=
			GetProfileSummary(siteID, m.Meta.EditedByNullable.Int64)
		if err != nil {
			glog.Errorf(
				"GetProfileSummary(%d, %d) %+v",
				siteID,
				m.Meta.EditedByNullable.Int64,
				err,
			)
			return status, err
		}
		m.Meta.EditedBy = profile
	}

	return http.StatusOK, nil
}

// UpdateManyAttendees updates many attendees
func UpdateManyAttendees(siteID int64, ems []AttendeeType) (int, error) {
	event, status, err := GetEvent(siteID, ems[0].EventID, 0)
	if err != nil {
		glog.Errorf("GetEvent(%d, %d, 0) %+v", siteID, ems[0].EventID, err)
		return status, err
	}

	tx, err := h.GetTransaction()
	if err != nil {
		glog.Errorf("h.GetTransaction() %+v", err)
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	for _, m := range ems {
		status, err = m.upsert(tx)
		if err != nil {
			glog.Errorf("m.upsert(tx) %+v", err)
			return status, err
		}
	}

	status, err = event.UpdateAttendees(tx)
	if err != nil {
		glog.Errorf("event.UpdateAttendees(tx) %+v", err)
		return status, err
	}

	err = tx.Commit()
	if err != nil {
		glog.Errorf("tx.Commit() %+v", err)
		return http.StatusInternalServerError, fmt.Errorf("transaction failed")
	}

	go PurgeCache(h.ItemTypes[h.ItemTypeEvent], event.ID)

	return http.StatusOK, nil
}

// Update updates an attendee
func (m *AttendeeType) Update(siteID int64) (int, error) {
	event, status, err := GetEvent(siteID, m.EventID, 0)
	if err != nil {
		glog.Errorf("GetEvent(%d, %d, 0) %+v", siteID, m.EventID, err)
		return status, err
	}

	// Update resource
	tx, err := h.GetTransaction()
	if err != nil {
		glog.Errorf("h.GetTransaction() %+v", err)
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	status, err = m.upsert(tx)
	if err != nil {
		glog.Errorf("m.upsert(tx) %+v", err)
		return status, err
	}

	status, err = event.UpdateAttendees(tx)
	if err != nil {
		glog.Errorf("event.UpdateAttendees(tx) %+v", err)
		return status, err
	}

	err = tx.Commit()
	if err != nil {
		glog.Errorf("tx.Commit() %+v", err)
		return http.StatusInternalServerError, fmt.Errorf("transaction failed")
	}

	go PurgeCache(h.ItemTypes[h.ItemTypeEvent], m.EventID)

	return http.StatusOK, nil
}

func (m *AttendeeType) upsert(tx *sql.Tx) (int, error) {
	status, err := m.Validate(tx)
	if err != nil {
		glog.Errorf("m.Validate(tx) %+v", err)
		return status, err
	}

	err = tx.QueryRow(`
	UPDATE attendees
	   SET state_id = $3,
	       state_date = $4,
	       edited = $5,
	       edited_by = $6,
	       edit_reason = $7
	 WHERE profile_id = $1
	   AND event_id = $2
 RETURNING attendee_id`,
		m.ProfileID,
		m.EventID,
		m.RSVPID,
		m.RSVPd,
		m.Meta.EditedNullable,
		m.Meta.EditedByNullable,
		m.Meta.EditReason,
	).Scan(
		&m.ID,
	)
	if err == nil {
		glog.Infof(
			"Set attendee %d as attending = '%s' to event %d",
			m.ProfileID,
			m.RSVP,
			m.EventID,
		)
		go PurgeCache(h.ItemTypes[h.ItemTypeAttendee], m.ID)

		return http.StatusOK, nil

	} else if err != sql.ErrNoRows {

		glog.Errorf("tx.QueryRow(...).Scan() %+v", err)
		return http.StatusInternalServerError,
			fmt.Errorf("error updating data and returning ID")
	}

	_, err = tx.Exec(`
INSERT INTO attendees (
    event_id, profile_id, created, created_by, state_id,
    state_date
) VALUES (
    $1, $2, $3, $4, $5,
    $6
)`,
		m.EventID,
		m.ProfileID,
		m.Meta.Created,
		m.Meta.CreatedByID,
		m.RSVPID,
		m.RSVPd,
	)
	if err != nil {
		glog.Errorf("tx.Exec(...) %+v", err)
		return http.StatusInternalServerError,
			fmt.Errorf("error executing insert")
	}

	go PurgeCache(h.ItemTypes[h.ItemTypeAttendee], m.ID)
	glog.Infof(
		"Set attendee %d as attending = '%s' to event %d",
		m.ProfileID,
		m.RSVP,
		m.EventID,
	)

	return http.StatusOK, nil
}

// Delete removes an attendee
func (m *AttendeeType) Delete(siteID int64) (int, error) {
	event, status, err := GetEvent(siteID, m.EventID, 0)
	if err != nil {
		glog.Errorf("GetEvent(%d, %d, 0) %+v", siteID, m.EventID, err)
		return status, err
	}

	// Connect to DB
	tx, err := h.GetTransaction()
	if err != nil {
		glog.Errorf("h.GetTransaction() %+v", err)
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
DELETE FROM attendees
 WHERE attendee_id = $1`,
		m.ID,
	)
	if err != nil {
		glog.Errorf("tx.Exec(%d) %+v", m.ID, err)
		return http.StatusInternalServerError, fmt.Errorf("delete failed")
	}

	status, err = event.UpdateAttendees(tx)
	if err != nil {
		glog.Errorf("event.UpdateAttendees(tx) %+v", err)
		return status, err
	}

	err = tx.Commit()
	if err != nil {
		glog.Errorf("tx.Commit() %+v", err)
		return http.StatusInternalServerError, fmt.Errorf("transaction failed")
	}

	go PurgeCache(h.ItemTypes[h.ItemTypeAttendee], m.ID)
	go PurgeCache(h.ItemTypes[h.ItemTypeEvent], m.EventID)

	return http.StatusOK, nil
}

// GetAttendeeID returns the attendee id of a profile
func GetAttendeeID(eventID int64, profileID int64) (int64, int, error) {
	// Open db connection and retrieve resource
	db, err := h.GetConnection()
	if err != nil {
		glog.Errorf("h.GetConnection() %+v", err)
		return 0, http.StatusInternalServerError, err
	}

	var attendeeID int64

	err = db.QueryRow(`
SELECT attendee_id
  FROM attendees
 WHERE event_id = $1
   AND profile_id = $2`,
		eventID,
		profileID,
	).Scan(
		&attendeeID,
	)
	if err == sql.ErrNoRows {
		return 0, http.StatusNotFound, fmt.Errorf("attendee not found")

	} else if err != nil {
		glog.Errorf("db.QueryRow(%d, %d) %+v", eventID, profileID, err)
		return 0, http.StatusInternalServerError,
			fmt.Errorf("database query failed")
	}

	return attendeeID, http.StatusOK, nil
}

// HandleAttendeeRequest fetches an attendee for a request
func HandleAttendeeRequest(
	siteID int64,
	id int64,
	seq int,
	out chan<- AttendeeRequest,
) {
	item, status, err := GetAttendee(siteID, id)
	response := AttendeeRequest{
		Item:   item,
		Status: status,
		Err:    err,
		Seq:    seq,
	}
	out <- response
}

// GetAttendee returns an attendee
func GetAttendee(siteID int64, id int64) (AttendeeType, int, error) {
	// Get from cache if it's available
	mcKey := fmt.Sprintf(mcAttendeeKeys[c.CacheDetail], id)
	if val, ok := c.Get(mcKey, AttendeeType{}); ok {
		m := val.(AttendeeType)
		m.Hydrate(siteID)
		return m, 0, nil
	}

	// Open db connection and retrieve resource
	db, err := h.GetConnection()
	if err != nil {
		glog.Errorf("h.GetConnection() %+v", err)
		return AttendeeType{}, http.StatusInternalServerError, err
	}

	var m AttendeeType
	err = db.QueryRow(`
SELECT attendee_id
      ,event_id
      ,profile_id
      ,created
      ,created_by
      ,edited
      ,edited_by
      ,edit_reason
      ,state_id
      ,state_date
 FROM attendees
WHERE attendee_id = $1`,
		id,
	).Scan(
		&m.ID,
		&m.EventID,
		&m.ProfileID,
		&m.Meta.Created,
		&m.Meta.CreatedByID,
		&m.Meta.EditedNullable,
		&m.Meta.EditedByNullable,
		&m.Meta.EditReasonNullable,
		&m.RSVPID,
		&m.RSVPd,
	)
	if err == sql.ErrNoRows {
		return AttendeeType{}, http.StatusNotFound,
			fmt.Errorf("resource with ID %d not found", id)
	} else if err != nil {
		glog.Errorf("db.QueryRow(%d) %+v", id, err)
		return AttendeeType{}, http.StatusInternalServerError,
			fmt.Errorf("database query failed")
	}

	if m.Meta.EditReasonNullable.Valid {
		m.Meta.EditReason = m.Meta.EditReasonNullable.String
	}

	if m.Meta.EditedNullable.Valid {
		m.Meta.Edited = m.Meta.EditedNullable.Time.Format(time.RFC3339Nano)
	}

	if m.RSVPd.Valid {
		m.RSVPdOn = m.RSVPd.Time.Format(time.RFC3339Nano)
	}

	m.RSVP, err = h.GetMapStringFromInt(RSVPStates, m.RSVPID)
	if err != nil {
		return AttendeeType{}, http.StatusInternalServerError, err
	}

	m.Meta.Links = []h.LinkType{
		h.GetExtendedLink(
			"self",
			"",
			h.ItemTypeAttendee,
			m.EventID,
			m.ProfileID,
		),
		h.GetLink("profile", "", h.ItemTypeProfile, m.ProfileID),
		h.GetLink("event", "", h.ItemTypeEvent, m.EventID),
	}

	// Update cache
	c.Set(mcKey, m, mcTTL)
	m.Hydrate(siteID)

	return m, http.StatusOK, nil
}

// GetAttendees returns a collection of attendees
func GetAttendees(
	siteID int64,
	eventID int64,
	limit int64,
	offset int64,
	attending bool,
) (
	[]AttendeeType,
	int64,
	int64,
	int,
	error,
) {
	// Retrieve resources
	db, err := h.GetConnection()
	if err != nil {
		return []AttendeeType{}, 0, 0, http.StatusInternalServerError, err
	}

	var where string
	if attending {
		where += `
		 AND state_id = 1`
	}

	rows, err := db.Query(
		fmt.Sprintf(`
SELECT COUNT(*) OVER() AS total
      ,attendee_id
  FROM attendees
 WHERE event_id = $1%s
 ORDER BY state_id ASC, state_date ASC
 LIMIT $2
OFFSET $3`,
			where,
		),
		eventID,
		limit,
		offset,
	)
	if err != nil {
		return []AttendeeType{}, 0, 0, http.StatusInternalServerError,
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
			return []AttendeeType{}, 0, 0, http.StatusInternalServerError,
				fmt.Errorf("row parsing error: %v", err.Error())
		}

		ids = append(ids, id)
	}
	err = rows.Err()
	if err != nil {
		return []AttendeeType{}, 0, 0, http.StatusInternalServerError,
			fmt.Errorf("error fetching rows: %v", err.Error())
	}
	rows.Close()

	// Make a request for each identifier
	var wg1 sync.WaitGroup
	req := make(chan AttendeeRequest)
	defer close(req)

	for seq, id := range ids {
		go HandleAttendeeRequest(siteID, id, seq, req)
		wg1.Add(1)
	}

	// Receive the responses and check for errors
	resps := []AttendeeRequest{}
	for i := 0; i < len(ids); i++ {
		resp := <-req
		wg1.Done()
		resps = append(resps, resp)
	}
	wg1.Wait()

	for _, resp := range resps {
		if resp.Err != nil {
			return []AttendeeType{}, 0, 0, resp.Status, resp.Err
		}
	}

	// Sort them
	sort.Sort(AttendeeRequestBySeq(resps))

	// Extract the values
	ems := []AttendeeType{}
	for _, resp := range resps {
		ems = append(ems, resp.Item)
	}

	pages := h.GetPageCount(total, limit)
	maxOffset := h.GetMaxOffset(total, limit)

	if offset > maxOffset {
		return []AttendeeType{}, 0, 0, http.StatusBadRequest,
			fmt.Errorf(
				"not enough records, offset (%d) would return an empty page",
				offset,
			)
	}

	return ems, total, pages, http.StatusOK, nil
}

// GetAttendeesCSV returns the attendees details as a CSV file
func GetAttendeesCSV(
	siteID int64,
	eventID int64,
	profileID int64,
) (
	string,
	int,
	error,
) {
	_, status, err := GetEvent(siteID, eventID, profileID)
	if err != nil {
		return "", status, err
	}

	// Retrieve resources
	db, err := h.GetConnection()
	if err != nil {
		return "", http.StatusInternalServerError, err
	}

	rows, err := db.Query(`-- GetAttendeesCSV
SELECT p.profile_name
      ,p.profile_id
      ,u.email
      ,a.state_date
  FROM attendees a
       JOIN profiles p ON p.profile_id = a.profile_id
       JOIN users u ON u.user_id = p.user_id
 WHERE event_id = $1
   AND a.state_id = 1
 ORDER BY 1;`,
		eventID,
	)
	if err != nil {
		return "", http.StatusInternalServerError,
			fmt.Errorf("database query failed: %v", err.Error())
	}
	defer rows.Close()

	csv := "name,id,email,date\r\n"

	// Get a list of the identifiers of the items to return
	for rows.Next() {
		var (
			name  string
			id    int64
			email string
			date  time.Time
		)
		err = rows.Scan(
			&name,
			&id,
			&email,
			&date,
		)
		if err != nil {
			return "", http.StatusInternalServerError,
				fmt.Errorf("row parsing error: %v", err.Error())
		}

		csv += fmt.Sprintf("\"%s\",%d,\"%s\",\"%s\"\r\n", name, id, email, date.Format(time.RFC3339))
	}
	err = rows.Err()
	if err != nil {
		return "", http.StatusInternalServerError,
			fmt.Errorf("error fetching rows: %v", err.Error())
	}
	rows.Close()

	return csv, http.StatusOK, nil
}
