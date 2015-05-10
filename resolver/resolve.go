package resolver

import (
	"net/http"
	"net/url"

	h "github.com/microcosm-cc/microcosm/helpers"
)

// Redirect encapsulates both the old URL and it's new destination
type Redirect struct {
	Origin     Origin     `json:"-"`
	RawURL     string     `json:"url"`
	ParsedURL  *url.URL   `json:"-"`
	Status     int        `json:"status"`
	URL        h.LinkType `json:"redirect"`
	ItemType   string     `json:"itemType,omitempty"`
	ItemTypeID int64      `json:"-"`
	ItemID     int64      `json:"itemId,omitempty"`
	Offset     int64      `json:"offset,omitempty"`
	Action     string     `json:"action,omitempty"`
	Search     string     `json:"search,omitempty"`
}

const (
	// ActionNewComment covers the query string arg newcomment
	ActionNewComment string = "newcomment"

	// ActionCommentInContext covers the query string arg incontext
	ActionCommentInContext string = "incontext"

	// ActionSearch covers the query string arg search
	ActionSearch string = "search"

	// ActionWhoIsOnline covers the query string arg online
	ActionWhoIsOnline string = "online"
)

// Resolve takes a URL and attempts to find a suitable new URL for the old one.
func Resolve(siteID int64, rawURL string, profileID int64) Redirect {

	redirect := Redirect{
		RawURL: rawURL,
	}

	origin := getOrigin(siteID)
	if origin == nil {
		redirect.Status = http.StatusNotFound
		return redirect
	}
	redirect.Origin = *origin

	u, err := url.Parse(rawURL)
	if err != nil {
		redirect.Status = http.StatusNotFound
		return redirect
	}
	redirect.ParsedURL = u

	switch origin.Product {
	case "vbulletin":
		return resolveVbulletinURL(redirect, profileID)
	default:
		redirect.Status = http.StatusNotFound
		return redirect
	}
}
