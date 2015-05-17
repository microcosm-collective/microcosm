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

// The numerical order is implicitly important (it's the sort field)
var RsvpStates = map[string]int64{
	"yes":     1,
	"maybe":   2,
	"invited": 3,
	"no":      4,
}

type AttendeesType struct {
	Attendees h.ArrayType    `json:"attendees"`
	Meta      h.CoreMetaType `json:"meta"`
}

type AttendeeType struct {
	Id        int64       `json:"-"`
	EventId   int64       `json:"-"`
	ProfileId int64       `json:"profileId,omitempty"`
	Profile   interface{} `json:"profile"`
	RSVPId    int64       `json:"-"`
	RSVP      string      `json:"rsvp"`
	RSVPd     pq.NullTime `json:"-"`
	RSVPdOn   string      `json:"rsvpdOn,omitempty"`

	Meta h.DefaultNoFlagsMetaType `json:"meta"`
}

type AttendeeRequest struct {
	Item   AttendeeType
	Err    error
	Status int
	Seq    int
}

type AttendeeRequestBySeq []AttendeeRequest

func (v AttendeeRequestBySeq) Len() int           { return len(v) }
func (v AttendeeRequestBySeq) Swap(i, j int)      { v[i], v[j] = v[j], v[i] }
func (v AttendeeRequestBySeq) Less(i, j int) bool { return v[i].Seq < v[j].Seq }

func (m *AttendeeType) Validate(tx *sql.Tx) (int, error) {

	if m.ProfileId <= 0 {
		glog.Infoln("m.ProfileId <= 0")
		return http.StatusBadRequest,
			errors.New("You must specify the attendees Profile ID")
	}

	if strings.Trim(m.RSVP, " ") == "" {
		m.RSVP = "invited"
	}

	if _, inList := RsvpStates[m.RSVP]; !inList {
		glog.Infoln("inList := RsvpStates[m.RSVP]; !inList")
		return http.StatusBadRequest,
			errors.New("You must specify a valid rsvp value " +
				"('invited', 'yes', 'maybe', or 'no')")
	}

	if m.RSVP == "yes" {
		//check to see if event is full

		var spaces, rsvp_limit int64
		err := tx.QueryRow(`
SELECT rsvp_spaces
      ,rsvp_limit
  FROM events
 WHERE event_id = $1`,
			m.EventId,
		).Scan(
			&spaces,
			&rsvp_limit,
		)
		if err != nil {
			glog.Errorf("tx.QueryRow(%d).Scan() %+v", m.EventId, err)
			return http.StatusInternalServerError,
				errors.New("Error fetching row")
		}

		if spaces <= 0 && rsvp_limit != 0 {
			glog.Infoln("spaces <= 0 && rsvp_limit != 0")
			return http.StatusBadRequest, errors.New("Event is full")
		}
	}

	m.RSVPd = m.Meta.EditedNullable
	m.RSVPId = RsvpStates[m.RSVP]

	return http.StatusOK, nil
}

func (m *AttendeeType) FetchProfileSummaries(siteId int64) (int, error) {

	profile, status, err := GetProfileSummary(siteId, m.ProfileId)
	if err != nil {
		glog.Errorf(
			"GetProfileSummary(%d, %d) %+v",
			siteId,
			m.ProfileId,
			err,
		)
		return status, err
	}
	m.Profile = profile

	profile, status, err = GetProfileSummary(siteId, m.Meta.CreatedById)
	if err != nil {
		glog.Errorf(
			"GetProfileSummary(%d, %d) %+v",
			siteId,
			m.Meta.CreatedById,
			err,
		)
		return status, err
	}
	m.Meta.CreatedBy = profile

	if m.Meta.EditedByNullable.Valid {
		profile, status, err :=
			GetProfileSummary(siteId, m.Meta.EditedByNullable.Int64)
		if err != nil {
			glog.Errorf(
				"GetProfileSummary(%d, %d) %+v",
				siteId,
				m.Meta.EditedByNullable.Int64,
				err,
			)
			return status, err
		}
		m.Meta.EditedBy = profile
	}

	return http.StatusOK, nil
}

func UpdateManyAttendees(siteId int64, ems []AttendeeType) (int, error) {
	event, status, err := GetEvent(siteId, ems[0].EventId, 0)
	if err != nil {
		glog.Errorf("GetEvent(%d, %d, 0) %+v", siteId, ems[0].EventId, err)
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
		return http.StatusInternalServerError, errors.New("Transaction failed")
	}

	go PurgeCache(h.ItemTypes[h.ItemTypeEvent], event.ID)

	return http.StatusOK, nil
}

func (m *AttendeeType) Update(siteId int64) (int, error) {
	event, status, err := GetEvent(siteId, m.EventId, 0)
	if err != nil {
		glog.Errorf("GetEvent(%d, %d, 0) %+v", siteId, m.EventId, err)
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
		return http.StatusInternalServerError, errors.New("Transaction failed")
	}

	go PurgeCache(h.ItemTypes[h.ItemTypeEvent], m.EventId)

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
		m.ProfileId,
		m.EventId,
		m.RSVPId,
		m.RSVPd,
		m.Meta.EditedNullable,
		m.Meta.EditedByNullable,
		m.Meta.EditReason,
	).Scan(
		&m.Id,
	)
	if err == nil {

		glog.Infof(
			"Set attendee %d as attending = '%s' to event %d",
			m.ProfileId,
			m.RSVP,
			m.EventId,
		)
		go PurgeCache(h.ItemTypes[h.ItemTypeAttendee], m.Id)

		return http.StatusOK, nil

	} else if err != sql.ErrNoRows {

		glog.Errorf("tx.QueryRow(...).Scan() %+v", err)
		return http.StatusInternalServerError,
			errors.New("Error updating data and returning ID")
	}

	_, err = tx.Exec(`
INSERT INTO attendees (
    event_id, profile_id, created, created_by, state_id,
    state_date
) VALUES (
    $1, $2, $3, $4, $5,
    $6
)`,
		m.EventId,
		m.ProfileId,
		m.Meta.Created,
		m.Meta.CreatedById,
		m.RSVPId,
		m.RSVPd,
	)
	if err != nil {
		glog.Errorf("tx.Exec(...) %+v", err)
		return http.StatusInternalServerError,
			errors.New("Error executing insert")
	}

	go PurgeCache(h.ItemTypes[h.ItemTypeAttendee], m.Id)
	glog.Infof(
		"Set attendee %d as attending = '%s' to event %d",
		m.ProfileId,
		m.RSVP,
		m.EventId,
	)

	return http.StatusOK, nil
}

func (m *AttendeeType) Delete(siteId int64) (int, error) {

	event, status, err := GetEvent(siteId, m.EventId, 0)
	if err != nil {
		glog.Errorf("GetEvent(%d, %d, 0) %+v", siteId, m.EventId, err)
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
		m.Id,
	)
	if err != nil {
		glog.Errorf("tx.Exec(%d) %+v", m.Id, err)
		return http.StatusInternalServerError, errors.New("Delete failed")
	}

	status, err = event.UpdateAttendees(tx)
	if err != nil {
		glog.Errorf("event.UpdateAttendees(tx) %+v", err)
		return status, err
	}

	err = tx.Commit()
	if err != nil {
		glog.Errorf("tx.Commit() %+v", err)
		return http.StatusInternalServerError, errors.New("Transaction failed")
	}

	go PurgeCache(h.ItemTypes[h.ItemTypeAttendee], m.Id)
	go PurgeCache(h.ItemTypes[h.ItemTypeEvent], m.EventId)

	return http.StatusOK, nil
}

func GetAttendeeId(eventId int64, profileId int64) (int64, int, error) {

	// Open db connection and retrieve resource
	db, err := h.GetConnection()
	if err != nil {
		glog.Errorf("h.GetConnection() %+v", err)
		return 0, http.StatusInternalServerError, err
	}

	var attendeeId int64

	err = db.QueryRow(`
SELECT attendee_id
  FROM attendees
 WHERE event_id = $1
   AND profile_id = $2`,
		eventId,
		profileId,
	).Scan(
		&attendeeId,
	)
	if err == sql.ErrNoRows {
		return 0, http.StatusNotFound, errors.New("attendee not found")

	} else if err != nil {
		glog.Errorf("db.QueryRow(%d, %d) %+v", eventId, profileId, err)
		return 0, http.StatusInternalServerError,
			errors.New("Database query failed")
	}

	return attendeeId, http.StatusOK, nil
}

func HandleAttendeeRequest(
	siteId int64,
	id int64,
	seq int,
	out chan<- AttendeeRequest,
) {
	item, status, err := GetAttendee(siteId, id)

	response := AttendeeRequest{
		Item:   item,
		Status: status,
		Err:    err,
		Seq:    seq,
	}
	out <- response
}

func GetAttendee(siteId int64, id int64) (AttendeeType, int, error) {

	// Get from cache if it's available
	mcKey := fmt.Sprintf(mcAttendeeKeys[c.CacheDetail], id)
	if val, ok := c.CacheGet(mcKey, AttendeeType{}); ok {
		m := val.(AttendeeType)
		m.FetchProfileSummaries(siteId)
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
		&m.Id,
		&m.EventId,
		&m.ProfileId,
		&m.Meta.Created,
		&m.Meta.CreatedById,
		&m.Meta.EditedNullable,
		&m.Meta.EditedByNullable,
		&m.Meta.EditReasonNullable,
		&m.RSVPId,
		&m.RSVPd,
	)
	if err == sql.ErrNoRows {
		return AttendeeType{}, http.StatusNotFound, errors.New(
			fmt.Sprintf("Resource with ID %d not found", id),
		)
	} else if err != nil {
		glog.Errorf("db.QueryRow(%d) %+v", id, err)
		return AttendeeType{}, http.StatusInternalServerError,
			errors.New("Database query failed")
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

	m.RSVP, err = h.GetMapStringFromInt(RsvpStates, m.RSVPId)
	if err != nil {
		return AttendeeType{}, http.StatusInternalServerError, err
	}

	m.Meta.Links = []h.LinkType{
		h.GetExtendedLink(
			"self",
			"",
			h.ItemTypeAttendee,
			m.EventId,
			m.ProfileId,
		),
		h.GetLink("profile", "", h.ItemTypeProfile, m.ProfileId),
		h.GetLink("event", "", h.ItemTypeEvent, m.EventId),
	}

	// Update cache
	c.CacheSet(mcKey, m, mcTTL)
	m.FetchProfileSummaries(siteId)

	return m, http.StatusOK, nil
}

func GetAttendees(
	siteId int64,
	eventId int64,
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
		eventId,
		limit,
		offset,
	)
	if err != nil {
		return []AttendeeType{}, 0, 0, http.StatusInternalServerError,
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
			return []AttendeeType{}, 0, 0, http.StatusInternalServerError,
				errors.New(
					fmt.Sprintf("Row parsing error: %v", err.Error()),
				)
		}

		ids = append(ids, id)
	}
	err = rows.Err()
	if err != nil {
		return []AttendeeType{}, 0, 0, http.StatusInternalServerError,
			errors.New(
				fmt.Sprintf("Error fetching rows: %v", err.Error()),
			)
	}
	rows.Close()

	// Make a request for each identifier
	var wg1 sync.WaitGroup
	req := make(chan AttendeeRequest)
	defer close(req)

	for seq, id := range ids {
		go HandleAttendeeRequest(siteId, id, seq, req)
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
		return []AttendeeType{}, 0, 0, http.StatusBadRequest, errors.New(
			fmt.Sprintf(
				"not enough records, offset (%d) would return an empty page.",
				offset,
			),
		)
	}

	return ems, total, pages, http.StatusOK, nil
}

func GetAttendeesCSV(
	siteId int64,
	eventId int64,
	profileId int64,
) (
	string,
	int,
	error,
) {
	_, status, err := GetEvent(siteId, eventId, profileId)
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
		eventId,
	)
	if err != nil {
		return "", http.StatusInternalServerError,
			errors.New(
				fmt.Sprintf("Database query failed: %v", err.Error()),
			)
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
				errors.New(
					fmt.Sprintf("Row parsing error: %v", err.Error()),
				)
		}

		csv += fmt.Sprintf("\"%s\",%d,\"%s\",\"%s\"\r\n", name, id, email, date.Format(time.RFC3339))
	}
	err = rows.Err()
	if err != nil {
		return "", http.StatusInternalServerError,
			errors.New(
				fmt.Sprintf("Error fetching rows: %v", err.Error()),
			)
	}
	rows.Close()

	return csv, http.StatusOK, nil
}
