package models

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Usage encapsulates a request and the key metrics and info around the request
// such as the time spent serving it, who asked for it, the endpoint, etc
type Usage struct {
	Method        string
	URL           string
	EndPointURL   string
	UserAgent     string
	HTTPStatus    int
	IPAddr        string
	Host          string
	ContentLength int
	Created       string `json:"timestamp"`
	TimeSpent     int64
	AccessToken   string
	SiteID        int64
	UserID        int64
	ProfileID     int64
	Error         string
}

// For replacing resource IDs in URLS so they can be easily grouped
var regURLIDs = regexp.MustCompile(`/[0-9]+`)

const repURLIDs string = `/{id}`

var regJumpLink = regexp.MustCompile(`^(/out/)(.*)$`)

const repJumpLink string = `$1{id}`

// SendUsage is called at the end of processing a request and will record info
// about the request for analytics and error detection later
func SendUsage(
	c *Context,
	statusCode int,
	contentLength int,
	dur time.Duration,
	errors []string,
) {

	m := Usage{}

	m.Method = c.GetHTTPMethod()
	m.URL = c.Request.URL.String()

	// Remove querystring and replace IDs to allow grouping
	endPointURL := regURLIDs.ReplaceAllString(
		strings.Split(m.URL, "?")[0],
		repURLIDs,
	)
	if strings.Contains(endPointURL, "/out/") {
		endPointURL = regJumpLink.ReplaceAllString(endPointURL, repJumpLink)
	}
	m.EndPointURL = endPointURL

	// Only send first 4 chars
	if c.Auth.AccessToken.TokenValue != "" {
		m.AccessToken = c.Auth.AccessToken.TokenValue[:4]
	}

	// Only send last two sections of IP address
	if c.Request.Header.Get("X-Real-IP") != "" {
		if strings.Contains(c.Request.Header.Get("X-Real-IP"), ".") {
			// IPv4
			m.IPAddr = strings.Join(
				strings.Split(c.Request.Header.Get("X-Real-IP"), ".")[2:],
				".",
			)

		} else if strings.Contains(c.Request.Header.Get("X-Real-IP"), ":") {
			// IPv6
			ipv6Split := strings.Split(c.Request.Header.Get("X-Real-IP"), ":")
			m.IPAddr = strings.Join(ipv6Split[(len(ipv6Split)-2):], ":")
		}
	}

	m.UserAgent = c.Request.UserAgent()
	m.HTTPStatus = statusCode
	m.Host = c.Request.Host
	m.ContentLength = contentLength
	m.Created = time.Now().Format(time.RFC3339)
	m.TimeSpent = dur.Nanoseconds()
	m.SiteID = c.Site.ID
	m.UserID = c.Auth.UserID
	m.ProfileID = c.Auth.ProfileID

	if len(errors) > 0 {
		m.Error = strings.Join(errors, ", ")
	}

	fmt.Printf(`%v\n`, m)
}
