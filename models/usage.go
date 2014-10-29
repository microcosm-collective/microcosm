package models

import (
	"encoding/json"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/xtgo/uuid"

	conf "github.com/microcosm-cc/microcosm/config"
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

var elasticSearchConnString string

func init() {
	elasticSearchConnString = fmt.Sprintf(
		"%s:%d",
		conf.CONFIG_STRING[conf.KEY_ELASTICSEARCH_HOST],
		conf.CONFIG_INT64[conf.KEY_ELASTICSEARCH_PORT],
	)
}

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

	m.Method = c.GetHttpMethod()
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
	m.SiteID = c.Site.Id
	m.UserID = c.Auth.UserId
	m.ProfileID = c.Auth.ProfileId

	if len(errors) > 0 {
		m.Error = strings.Join(errors, ", ")
	}

	m.Send()
}

// Send usage data using ElasticSearch's bulk load UDP API
func (m *Usage) Send() {
	conn, err := net.Dial("udp", elasticSearchConnString)
	if err != nil {
		glog.Warningf("Couldn't dial: %s", err.Error())
	}
	defer conn.Close()

	coord, err := json.Marshal(map[string]interface{}{
		"index": map[string]string{
			"_index": "microcosm",
			"_type":  "log",
			"_id":    uuid.NewRandom().String(),
		},
	})
	if err != nil {
		glog.Warningf("Failed to marshal log coord: %s", err.Error())
	}
	usage, err := json.Marshal(m)
	if err != nil {
		glog.Warningf("Failed to marshal usage: %s", err.Error())
	}

	payload := fmt.Sprintf("%s\n%s\n", string(coord), string(usage))

	_, err = conn.Write([]byte(payload))
	if err != nil {
		glog.Warningf("Couldn't write usage to conn: %s", err.Error())
	}
}
