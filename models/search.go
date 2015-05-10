package models

import (
	"database/sql"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// SearchResults is a list of SearchResult
type SearchResults struct {
	Query     SearchQuery `json:"query"`
	TimeTaken int64       `json:"timeTakenInMs,omitempty"`
	Results   interface{} `json:"results,omitempty"`
}

// SearchResult is an encapsulation of a hit against a search
type SearchResult struct {
	ItemType         string        `json:"itemType"`
	ItemTypeID       int64         `json:"-"`
	ItemID           int64         `json:"-"`
	Item             interface{}   `json:"item"`
	ParentItemType   string        `json:"parentItemType,omitempty"`
	ParentItemTypeID sql.NullInt64 `json:"-"`
	ParentItemID     sql.NullInt64 `json:"-"`
	ParentItem       interface{}   `json:"parentItem,omitempty"`
	Unread           bool          `json:"unread"`

	// TODO(buro9): Remove rank
	Rank         float64   `json:"rank"`
	LastModified time.Time `json:"lastModified"`
	Highlight    string    `json:"highlight"`
}

// Search performs a search against the database
func Search(
	siteID int64,
	searchURL url.URL,
	profileID int64,
) (
	SearchResults,
	int,
	error,
) {

	// Parse the search options and determine what kind of search that we will
	// be performing.
	m := SearchResults{
		Query: GetSearchQueryFromURL(searchURL),
	}

	if !m.Query.Valid {
		return m, http.StatusOK, nil
	}

	if strings.Trim(m.Query.Query, " ") != "" {
		return searchFullText(siteID, searchURL, profileID, m)
	}

	return searchMetaData(siteID, searchURL, profileID, m)
}
