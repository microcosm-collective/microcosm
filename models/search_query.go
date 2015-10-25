package models

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
	h "github.com/microcosm-cc/microcosm/helpers"
)

// SearchQuery encapsulates all of the meta parts of a search over the
// microcosm content
type SearchQuery struct {
	URL       url.URL    `json:"-"`
	URLValues url.Values `json:"-"`

	// Implemented in search
	Query             string    `json:"q,omitempty"`
	InTitle           bool      `json:"inTitle,omitempty"`
	Hashtags          []string  `json:"hashtags,omitempty"`
	MicrocosmIDsQuery []int64   `json:"forumId,omitempty"`
	MicrocosmIDs      []int64   `json:"-"`
	ItemTypesQuery    []string  `json:"type,omitempty"`
	ItemTypeIDs       []int64   `json:"-"`
	ItemIDsQuery      []int64   `json:"id,omitempty"`
	ItemIDs           []int64   `json:"-"`
	ProfileID         int64     `json:"authorId,omitempty"`
	Following         bool      `json:"following,omitempty"`
	Since             string    `json:"since,omitempty"`
	SinceTime         time.Time `json:"-"`
	Until             string    `json:"until,omitempty"`
	UntilTime         time.Time `json:"-"`
	EventAfter        string    `json:"eventAfter,omitempty"`
	EventAfterTime    time.Time `json:"-"`
	EventBefore       string    `json:"eventBefore,omitempty"`
	EventBeforeTime   time.Time `json:"-"`
	Attendee          bool      `json:"attendee,omitempty"`
	Sort              string    `json:"sort,omitempty"`
	Limit             int64     `json:"-"`
	Offset            int64     `json:"-"`

	// Not yet implement in search
	Lat         float64 `json:"lat,omitempty"`
	Lon         float64 `json:"lon,omitempty"`
	Radius      int64   `json:"radius,omitempty"`
	North       float64 `json:"north,omitempty"`
	East        float64 `json:"east,omitempty"`
	South       float64 `json:"south,omitempty"`
	West        float64 `json:"west,omitempty"`
	ProfileName string  `json:"author,omitempty"`

	Ignored    string   `json:"ignored,omitempty"`
	IgnoredArr []string `json:"-"`
	Searched   string   `json:"searched,omitempty"`

	Valid bool `json:"-"`
}

// GetSearchQueryFromURL fetches and parses a query and returns a SearchQuery
func GetSearchQueryFromURL(requestURL url.URL) SearchQuery {

	sq := SearchQuery{
		URL:       requestURL,
		URLValues: requestURL.Query(),
	}

	sq.ParseFullQueryString()
	sq.ParseSingleQueryValue()
	sq.Validate()

	return sq
}

// ParseFullQueryString takes ?q=term&type=conversation
// And makes it
// 	q = term
// 	type = conversation
// Within the sq object
func (sq *SearchQuery) ParseFullQueryString() {

	// Get the named values first
	sq.Query = sq.URLValues.Get("q")

	for k, v := range sq.URLValues {
		if k == "id" {
			for _, t := range v {
				i, err := strconv.ParseInt(t, 10, 64)
				if err != nil {
					sq.IgnoredArr = append(
						sq.IgnoredArr,
						fmt.Sprintf("id=%s", t),
					)
				} else {
					var found bool
					for _, it := range sq.ItemIDs {
						if it == i {
							found = true
							break
						}
					}
					if !found {
						sq.ItemIDs = append(sq.ItemIDs, i)
					}
				}
			}
		}

		if k == "forumId" {
			for _, t := range v {
				i, err := strconv.ParseInt(t, 10, 64)
				if err != nil {
					sq.IgnoredArr = append(
						sq.IgnoredArr,
						fmt.Sprintf("forumId=%s", t),
					)
				} else {
					var found bool
					for _, it := range sq.MicrocosmIDs {
						if it == i {
							found = true
							break
						}
					}
					if !found {
						sq.MicrocosmIDs = append(sq.MicrocosmIDs, i)
					}
				}
			}
		}

		if k == "type" {
			for _, t := range v {
				itemTypeID := h.ItemTypes[t]

				if itemTypeID == 0 {
					sq.IgnoredArr = append(
						sq.IgnoredArr,
						fmt.Sprintf("type=%s", t),
					)
				} else {
					// Prevent duplicates
					var found bool
					for _, it := range sq.ItemTypeIDs {
						if it == itemTypeID {
							found = true
							break
						}
					}

					if !found {
						sq.ItemTypeIDs = append(sq.ItemTypeIDs, itemTypeID)
					}
				}
			}
		}
	}

	dateTimes := []string{"since", "until", "eventAfter", "eventBefore"}
	for _, key := range dateTimes {
		sq.ParseDateTime(key, sq.URLValues.Get(key), "")
	}

	ints := []string{"radius", "authorId"}
	for _, key := range ints {
		sq.ParseInt(key, sq.URLValues.Get(key), "")
	}

	floats := []string{"lat", "lon", "north", "east", "south", "west"}
	for _, key := range floats {
		sq.ParseFloat(key, sq.URLValues.Get(key), "")
	}

	bools := []string{"attendee", "following", "inTitle"}
	for _, key := range bools {
		sq.ParseBool(key, sq.URLValues.Get(key), "")
	}

	sq.Sort = strings.ToLower(sq.URLValues.Get("sort"))

	sq.ProfileName = sq.URLValues.Get("author")
}

// ParseSingleQueryValue takes the value of sq.Query which came from the
// querystring 'q' and sees whether there are things like type:conversation
// and if so will populate sq.* accordingly
func (sq *SearchQuery) ParseSingleQueryValue() {

	if sq.Query == "" {
		return
	}

	frags := strings.Split(sq.Query, " ")

	var query []string
	for _, frag := range frags {

		if strings.Contains(frag, ":") {

			kv := strings.Split(frag, ":")

			if len(kv) != 2 || kv[0] == "" || kv[1] == "" {
				query = append(query, frag)
				continue
			}

			key := strings.ToLower(kv[0])
			value := kv[1]

			switch strings.ToLower(key) {
			case "id":
				i, err := strconv.ParseInt(value, 10, 64)
				if err != nil {
					sq.IgnoredArr = append(sq.IgnoredArr, frag)
				} else {
					var found bool
					for _, t := range sq.ItemIDs {
						if t == i {
							found = true
							break
						}
					}
					if !found {
						sq.ItemIDs = append(sq.ItemIDs, i)
					}
				}
			case "forumid":
				i, err := strconv.ParseInt(value, 10, 64)
				if err != nil {
					sq.IgnoredArr = append(sq.IgnoredArr, frag)
				} else {
					var found bool
					for _, t := range sq.MicrocosmIDs {
						if t == i {
							found = true
							break
						}
					}
					if !found {
						sq.MicrocosmIDs = append(sq.MicrocosmIDs, i)
					}
				}
			case "type":
				// itemTypes
				itemType := value
				itemTypeID := h.ItemTypes[itemType]

				if itemTypeID == 0 {
					sq.IgnoredArr = append(sq.IgnoredArr, frag)
				} else {
					var found bool
					for _, t := range sq.ItemTypeIDs {
						if t == itemTypeID {
							found = true
							break
						}
					}

					if !found {
						sq.ItemTypeIDs = append(sq.ItemTypeIDs, itemTypeID)
					}
				}

			case "since", "until", "eventafter", "eventbefore":
				sq.ParseDateTime(key, value, frag)
			case "lat", "lon", "north", "east", "south", "west":
				sq.ParseFloat(key, value, frag)
			case "radius", "authorid":
				sq.ParseInt(key, value, frag)
			case "attendee", "following", "intitle":
				sq.ParseBool(key, value, frag)
			case "author":
				sq.ProfileName = value
			case "sort":
				sq.Sort = strings.ToLower(value)

			default:
				query = append(query, frag)
			}

		} else {
			query = append(query, frag)
		}
	}

	sq.Query = strings.Join(query, " ")

	// Extract hashtags
	// regHashtags is defined in hashtags.go and used to find hashtags in
	// Markdown text but also works in this scenario
	sq.Hashtags = regHashtags.FindAllString(sq.Query, -1)
}

// ParseDateTime parses a string containing a potential datetime arg
func (sq *SearchQuery) ParseDateTime(key string, value string, frag string) {
	if key == "" || value == "" {
		return
	}

	if frag == "" {
		frag = fmt.Sprintf("%s=%s", key, value)
	}

	// dates and times
	i, err := strconv.Atoi(value)
	if err == nil {
		// is it a count of days?
		switch strings.ToLower(key) {
		case "since":
			sq.Since = value
			sq.SinceTime = time.Now().AddDate(0, 0, i)
		case "until":
			sq.Until = value
			sq.UntilTime = time.Now().AddDate(0, 0, i)
		case "eventafter":
			sq.EventAfter = value
			sq.EventAfterTime = time.Now().AddDate(0, 0, i)
		case "eventbefore":
			sq.EventBefore = value
			sq.EventBeforeTime = time.Now().AddDate(0, 0, i)
		default:
			sq.IgnoredArr = append(sq.IgnoredArr, frag)
		}
	} else {
		k, err := time.Parse("2006-01-02", value)
		if err == nil {
			switch strings.ToLower(key) {
			case "since":
				sq.Since = value
				sq.SinceTime = k
			case "until":
				sq.Until = value
				sq.UntilTime = k
			case "eventafter":
				sq.EventAfter = value
				sq.EventAfterTime = k
			case "eventbefore":
				sq.EventBefore = value
				sq.EventBeforeTime = k
			default:
				sq.IgnoredArr = append(sq.IgnoredArr, frag)
			}
		} else {
			k, err = time.Parse("2006-01-02T15:04", value)
			if err == nil {
				switch strings.ToLower(key) {
				case "since":
					sq.Since = value
					sq.SinceTime = k
				case "until":
					sq.Until = value
					sq.UntilTime = k
				case "eventafter":
					sq.EventAfter = value
					sq.EventAfterTime = k
				case "eventbefore":
					sq.EventBefore = value
					sq.EventBeforeTime = k
				default:
					sq.IgnoredArr = append(sq.IgnoredArr, frag)
				}
			} else {
				sq.IgnoredArr = append(sq.IgnoredArr, frag)
			}
		}
	}
}

// ParseFloat parses a float arg
func (sq *SearchQuery) ParseFloat(key string, value string, frag string) {
	if key == "" || value == "" {
		return
	}

	if frag == "" {
		frag = fmt.Sprintf("%s=%s", key, value)
	}

	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		sq.IgnoredArr = append(sq.IgnoredArr, frag)
		return
	}

	switch strings.ToLower(key) {
	case "lat":
		sq.Lat = f
	case "lon":
		sq.Lon = f
	case "north":
		sq.North = f
	case "east":
		sq.East = f
	case "south":
		sq.South = f
	case "west":
		sq.West = f
	default:
		sq.IgnoredArr = append(sq.IgnoredArr, frag)
	}
}

// ParseInt parses an integer arg
func (sq *SearchQuery) ParseInt(key string, value string, frag string) {
	if key == "" || value == "" {
		return
	}

	if frag == "" {
		frag = fmt.Sprintf("%s=%s", key, value)
	}

	i, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		sq.IgnoredArr = append(sq.IgnoredArr, frag)
		return
	}

	switch strings.ToLower(key) {
	case "radius":
		sq.Radius = i
	case "forumid":
		sq.MicrocosmIDs = append(sq.MicrocosmIDs, i)
	case "authorid":
		sq.ProfileID = i
	default:
		sq.IgnoredArr = append(sq.IgnoredArr, frag)
	}
}

// ParseBool parses a boolean arg
func (sq *SearchQuery) ParseBool(key string, value string, frag string) {
	if key == "" || value == "" {
		return
	}

	if frag == "" {
		frag = fmt.Sprintf("%s=%s", key, value)
	}

	b, err := strconv.ParseBool(value)
	if err != nil {
		sq.IgnoredArr = append(sq.IgnoredArr, frag)
		return
	}

	switch strings.ToLower(key) {
	case "attendee":
		sq.Attendee = b
	case "following":
		sq.Following = b
	case "intitle":
		sq.InTitle = b
	default:
		sq.IgnoredArr = append(sq.IgnoredArr, frag)
	}
}

// Validate returns true if the query is valid
func (sq *SearchQuery) Validate() {

	var valid bool

	if (sq.Lat == 0 && sq.Lon == 0 && sq.Radius == 0) ||
		(sq.Lat > 0 && sq.Lon > 0) {

		if sq.Lat > 0 && sq.Radius == 0 {
			sq.Radius = 5000 // Value is in meters, 5KM
		}
		// TODO: Implement geo search
		// valid = true
	} else {
		if sq.Lat > 0 {
			sq.IgnoredArr = append(
				sq.IgnoredArr,
				fmt.Sprintf("lat:%f", sq.Lat),
			)
			sq.Lat = 0
		}
		if sq.Lon > 0 {
			sq.IgnoredArr = append(
				sq.IgnoredArr,
				fmt.Sprintf("lon:%f", sq.Lon),
			)
			sq.Lon = 0
		}
		if sq.Radius > 0 {
			sq.IgnoredArr = append(
				sq.IgnoredArr,
				fmt.Sprintf("radius:%d", sq.Radius),
			)
			sq.Radius = 0
		}
	}

	if (sq.North == 0 && sq.East == 0 && sq.South == 0 && sq.West == 0) ||
		(sq.North > 0 && sq.East > 0 && sq.South > 0 && sq.West > 0) {

		// NESW overrides lat,lon + radius
		if sq.Lat > 0 {
			sq.IgnoredArr = append(
				sq.IgnoredArr,
				fmt.Sprintf("lat:%f", sq.Lat),
			)
			sq.Lat = 0
		}
		if sq.Lon > 0 {
			sq.IgnoredArr = append(
				sq.IgnoredArr,
				fmt.Sprintf("lon:%f", sq.Lon),
			)
			sq.Lon = 0
		}
		if sq.Radius > 0 {
			sq.IgnoredArr = append(
				sq.IgnoredArr,
				fmt.Sprintf("radius:%d", sq.Radius),
			)
			sq.Radius = 0
		}
		// TODO: Implement geo search
		// valid = true
	} else {
		if sq.North > 0 {
			sq.IgnoredArr = append(
				sq.IgnoredArr,
				fmt.Sprintf("north:%f", sq.North),
			)
			sq.North = 0
		}
		if sq.East > 0 {
			sq.IgnoredArr = append(
				sq.IgnoredArr,
				fmt.Sprintf("east:%f", sq.East),
			)
			sq.East = 0
		}
		if sq.South > 0 {
			sq.IgnoredArr = append(
				sq.IgnoredArr,
				fmt.Sprintf("south:%f", sq.South),
			)
			sq.South = 0
		}
		if sq.West > 0 {
			sq.IgnoredArr = append(
				sq.IgnoredArr,
				fmt.Sprintf("west:%f", sq.West),
			)
			sq.West = 0
		}
	}

	sq.Query = strings.TrimSpace(sq.Query)
	if sq.Query != "" {
		valid = true
	} else {
		sq.Query = ""
	}

	// TODO: Implement geo search
	// if sq.Lat > 0 {
	// 	searched = append(searched, fmt.Sprintf("lat:%f", sq.Lat))
	// }

	// if sq.Lon > 0 {
	// 	searched = append(searched, fmt.Sprintf("lon:%f", sq.Lon))
	// }

	// if sq.Radius > 0 {
	// 	searched = append(searched, fmt.Sprintf("radius:%f", sq.Radius))
	// }

	// if sq.North > 0 {
	// 	searched = append(searched, fmt.Sprintf("north:%f", sq.North))
	// }

	// if sq.East > 0 {
	// 	searched = append(searched, fmt.Sprintf("east:%f", sq.East))
	// }

	// if sq.South > 0 {
	// 	searched = append(searched, fmt.Sprintf("south:%f", sq.South))
	// }

	// if sq.West > 0 {
	// 	searched = append(searched, fmt.Sprintf("west:%f", sq.West))
	// }

	if len(sq.ItemTypeIDs) > 0 {
		valid = true
	}

	if !sq.EventAfterTime.IsZero() {
		if len(sq.ItemTypeIDs) != 1 ||
			sq.ItemTypeIDs[0] != h.ItemTypes[h.ItemTypeEvent] {

			sq.IgnoredArr = append(
				sq.IgnoredArr,
				fmt.Sprintf("eventAfter:%s", sq.EventAfter),
			)
			sq.EventAfterTime = time.Time{}
		}
	}

	if !sq.EventBeforeTime.IsZero() {
		if len(sq.ItemTypeIDs) != 1 ||
			sq.ItemTypeIDs[0] != h.ItemTypes[h.ItemTypeEvent] {

			sq.IgnoredArr = append(
				sq.IgnoredArr,
				fmt.Sprintf("eventBefore:%s", sq.EventBefore),
			)
			sq.EventBeforeTime = time.Time{}
		}
	}

	if strings.TrimSpace(sq.ProfileName) != "" {
		if sq.ProfileID == 0 {
			// TODO: get profile ID by search for profiles that exact match a username

			if sq.ProfileID > 0 {
				// valid = true
			} else {
				sq.IgnoredArr = append(
					sq.IgnoredArr,
					fmt.Sprintf("author:%s", sq.ProfileName),
				)
				sq.ProfileName = ""
			}
		}
	}

	if sq.ProfileID > 0 {
		valid = true
	}

	if sq.Attendee {
		// Events can be sorted by the date of the event
		if !(len(sq.ItemTypeIDs) == 1 &&
			sq.ItemTypeIDs[0] == h.ItemTypes[h.ItemTypeEvent]) {

			sq.IgnoredArr = append(sq.IgnoredArr, "attendee:true")
			sq.Attendee = false
		}
	}

	if len(sq.MicrocosmIDs) > 0 {
		// Expand microcosmIDs so that the array includes child forums too
		it := []string{}
		for _, microcosmID := range sq.MicrocosmIDs {
			it = append(it, strconv.FormatInt(microcosmID, 10))
		}
		microcosmIDs := `{` + strings.Join(it, `,`) + `}`

		// Retrieve resources
		db, err := h.GetConnection()
		if err != nil {
			glog.Errorf("h.GetConnection() %+v", err)
			return
		}

		rows, err := db.Query(`-- GetMicrocosmChildren
SELECT microcosm_id
  FROM microcosms
 WHERE path <@ (
           SELECT path
             FROM microcosms
            WHERE microcosm_id = ANY ($1::bigint[])
       );`,
			microcosmIDs,
		)
		if err != nil {
			glog.Errorf("db.Query(%s) %+v", microcosmIDs, err)
			return
		}
		defer rows.Close()

		// Get a list of the identifiers of the items to return
		ids := []int64{}
		for rows.Next() {
			var id int64
			err = rows.Scan(&id)
			if err != nil {
				glog.Errorf("rows.Scan() %+v", err)
				return
			}
			ids = append(ids, id)
		}
		err = rows.Err()
		if err != nil {
			glog.Errorf("rows.Err() %+v", err)
			return
		}
		rows.Close()

		sq.MicrocosmIDs = ids

		if len(sq.MicrocosmIDs) > 0 {
			valid = true
		}
	}

	// Build up our knowledge of what we're ignoring and what we are searching
	sq.Ignored = strings.Join(sq.IgnoredArr, " ")

	searched := []string{}
	if sq.Query != "" {
		searched = append(searched, sq.Query)
	}

	if len(sq.ItemTypeIDs) > 0 {
		for _, v := range sq.ItemTypeIDs {
			itemType, _ := h.GetMapStringFromInt(h.ItemTypes, v)
			sq.ItemTypesQuery = append(sq.ItemTypesQuery, itemType)
			searched = append(searched, fmt.Sprintf("type:%s", itemType))
		}
	}

	if len(sq.ItemIDs) > 0 {
		for _, v := range sq.ItemIDs {
			sq.ItemIDsQuery = append(sq.ItemIDsQuery, v)
			searched = append(searched, fmt.Sprintf("id:%d", v))
		}
	}

	if sq.InTitle {
		searched = append(searched, fmt.Sprintf("inTitle:%t", sq.InTitle))
	}

	if sq.Following {
		searched = append(searched, fmt.Sprintf("following:%t", sq.Following))
	}

	if !sq.SinceTime.IsZero() {
		searched = append(searched, fmt.Sprintf("since:%s", sq.Since))
	}

	if !sq.UntilTime.IsZero() {
		searched = append(searched, fmt.Sprintf("until:%s", sq.Until))
	}

	if !sq.EventAfterTime.IsZero() {
		searched = append(searched, fmt.Sprintf("eventAfter:%s", sq.EventAfter))
	}

	if !sq.EventBeforeTime.IsZero() {
		searched = append(searched, fmt.Sprintf("eventBefore:%s", sq.EventBefore))
	}

	if sq.Attendee {
		searched = append(searched, fmt.Sprintf("attendee:%t", sq.Attendee))
	}

	if len(sq.MicrocosmIDs) > 0 {
		for _, v := range sq.MicrocosmIDs {
			sq.MicrocosmIDsQuery = append(sq.MicrocosmIDsQuery, v)
			searched = append(searched, fmt.Sprintf("forumId:%d", v))
		}
	}

	if sq.ProfileID > 0 {
		searched = append(searched, fmt.Sprintf("authorId:%d", sq.ProfileID))
	}

	if sq.Sort != "" {
		searched = append(searched, fmt.Sprintf("sort:%s", sq.Sort))
	}

	sq.Searched = strings.Join(searched, " ")

	if valid {
		sq.Valid = true
	}

	return
}
