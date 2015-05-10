package models

import (
	"database/sql"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type SearchResults struct {
	Query     SearchQuery `json:"query"`
	TimeTaken int64       `json:"timeTakenInMs,omitempty"`
	Results   interface{} `json:"results,omitempty"`
}

type SearchResult struct {
	ItemType         string        `json:"itemType"`
	ItemTypeId       int64         `json:"-"`
	ItemId           int64         `json:"-"`
	Item             interface{}   `json:"item"`
	ParentItemType   string        `json:"parentItemType,omitempty"`
	ParentItemTypeId sql.NullInt64 `json:"-"`
	ParentItemId     sql.NullInt64 `json:"-"`
	ParentItem       interface{}   `json:"parentItem,omitempty"`
	Unread           bool          `json:"unread"`

	// TODO(buro9): Remove rank
	Rank         float64   `json:"rank"`
	LastModified time.Time `json:"lastModified"`
	Highlight    string    `json:"highlight"`
}

func Search(
	siteId int64,
	searchUrl url.URL,
	profileId int64,
) (
	SearchResults,
	int,
	error,
) {

	// Parse the search options and determine what kind of search that we will
	// be performing.
	m := SearchResults{
		Query: GetSearchQueryFromURL(searchUrl),
	}

	if !m.Query.Valid {
		return m, http.StatusOK, nil
	}

	if strings.Trim(m.Query.Query, " ") != "" {
		return searchFullText(siteId, searchUrl, profileId, m)
	} else {
		return searchMetaData(siteId, searchUrl, profileId, m)
	}

}
