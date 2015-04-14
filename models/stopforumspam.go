package models

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/golang/glog"
)

type sFSResponse struct {
	Success int64 `json:"success"`
	Email   struct {
		Normalized string  `json:"normalized"`
		LastSeen   int64   `json:"lastseen"`
		Frequency  int64   `json:"frequency"`
		Appears    int64   `json:"appears"`
		Confidence float64 `json:"confidence"`
	}
}

// IsSpammer takes an email address and returns true if the email has ever
// been used to spam forums.
func IsSpammer(email string) bool {
	// http://api.stopforumspam.org/api?email=jhonteddywilson45@gmail.com&f=json&unix
	// {"success":1,"email":{"lastseen":1423394226,"frequency":1,"appears":1,"confidence":16.34}}

	u, _ := url.Parse(`http://api.stopforumspam.org/api`)
	q := u.Query()
	q.Add(`f`, `json`)    // JSON response
	q.Add(`unix`, ``)     // Unix time for dates
	q.Add(`email`, email) // The email to check
	u.RawQuery = q.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		glog.Errorf(`%+v`, err)

		// Default to false, as we are going to allow user registration
		// whenever Stop Forum Spam is down.
		return false
	} else if resp.StatusCode != http.StatusOK {
		glog.Errorf(`!200`)
		return false
	}

	defer resp.Body.Close()
	d := json.NewDecoder(resp.Body)

	var m sFSResponse
	if err := d.Decode(&m); err != io.EOF && err != nil {
		glog.Errorf(`%+v`, err)
		return false
	}

	// If a successful API query, and the email has been seen in the last year
	// and confidence is now above 0... then we have a spammer.
	if m.Success > 0 &&
		m.Email.LastSeen > time.Now().AddDate(-1, 0, 0).Unix() &&
		m.Email.Confidence > 0 {
		glog.Infof(`IsSpammer = true for %s : %+v `, email, m)
		return true
	}

	glog.Infof(`IsSpammer = false for %s`, email)
	return false
}
