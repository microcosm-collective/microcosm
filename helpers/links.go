package helpers

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

const (
	// DefaultQueryLimit defines the default number of items per page for APIs
	DefaultQueryLimit int64 = 25

	// DefaultQueryOffset defines the default offset for API responses
	DefaultQueryOffset int64 = 0
)

// LinkArrayType is a collection of links
type LinkArrayType struct {
	Links []LinkType `json:"links"`
}

// LinkType is a link
type LinkType struct {
	Rel   string `json:"rel,omitempty"` // REST
	Href  string `json:"href"`
	Title string `json:"title,omitempty"`
	Text  string `json:"text,omitempty"` // HTML
}

// GetLimitAndOffset returns the Limit and Offset for a given request querystring
func GetLimitAndOffset(query url.Values) (int64, int64, int, error) {
	var (
		limit  int64
		offset int64
	)

	limit = DefaultQueryLimit
	if query.Get("limit") != "" {
		inLimit, err := strconv.ParseInt(query.Get("limit"), 10, 64)
		if err != nil {
			return 0, 0, http.StatusBadRequest,
				fmt.Errorf("limit (%s) is not a number", query.Get("limit"))
		}

		if inLimit < 1 {
			return 0, 0, http.StatusBadRequest,
				fmt.Errorf("limit (%d) cannot be zero or negative", inLimit)
		}

		if inLimit%5 != 0 {
			return 0, 0, http.StatusBadRequest,
				fmt.Errorf("limit (%d) must be a multiple of 5", inLimit)
		}

		if inLimit > 250 {
			return 0, 0, http.StatusBadRequest,
				fmt.Errorf("limit (%d) cannot exceed 100", inLimit)
		}

		limit = inLimit
	}

	offset = DefaultQueryOffset
	if query.Get("offset") != "" {
		inOffset, err := strconv.ParseInt(query.Get("offset"), 10, 64)
		if err != nil {
			return 0, 0, http.StatusBadRequest,
				fmt.Errorf("offset (%s) is not a number", query.Get("offset"))
		}

		if inOffset < 0 {
			return 0, 0, http.StatusBadRequest,
				fmt.Errorf("offset (%d) cannot be negative", inOffset)
		}

		if inOffset%limit != 0 {
			return 0, 0, http.StatusBadRequest,
				fmt.Errorf(
					"offset (%d) must be a multiple of limit (%d) or zero",
					inOffset,
					limit,
				)
		}

		offset = inOffset
	}

	return limit, offset, http.StatusOK, nil
}

// GetItemAndItemType returns the item type and id for a given request query
func GetItemAndItemType(query url.Values) (int64, string, int, error) {
	var (
		itemID   int64
		itemType string
	)

	if query.Get("itemId") != "" {
		inItemID, err := strconv.ParseInt(query.Get("itemId"), 10, 64)
		if err != nil {
			return 0, "", http.StatusBadRequest,
				fmt.Errorf("itemId (%s) is not a number", query.Get("itemId"))
		}

		itemID = inItemID
	}

	if query.Get("itemType") != "" {
		inItemType := query.Get("itemType")

		itemType = inItemType
	}

	return itemID, itemType, http.StatusOK, nil
}

// GetAttending returns isAttending for a given request query
func GetAttending(query url.Values) (bool, int, error) {
	var isAttending bool

	if query.Get("isAttending") != "" {
		inAttending, err := strconv.ParseBool(query.Get("isAttending"))
		if err != nil {
			return false, http.StatusBadRequest,
				fmt.Errorf(
					"isAttending (%s) is not a boolean",
					query.Get("isAttending"),
				)
		}

		isAttending = inAttending
	}

	return isAttending, http.StatusOK, nil
}

// AttendanceStatus returns status for the given request query
func AttendanceStatus(query url.Values) (string, int, error) {
	var status string

	if query.Get("status") != "" {
		inStatus := query.Get("status")

		status = inStatus
	}

	return status, http.StatusOK, nil
}

// GetPageCount returns the number of pages for a given total and items per
// page
func GetPageCount(total int64, limit int64) int64 {
	if limit == 0 {
		limit = DefaultQueryLimit
	}

	pages := total / limit

	if total%limit > 0 {
		pages++
	}

	return pages
}

// GetMaxOffset returns the maximum possible offset for a given number of
// pages and limit per page
func GetMaxOffset(total int64, limit int64) int64 {
	return ((total - 1) / limit) * limit
}

func getLinkToFirstPage(
	requestURL url.URL,
	offset int64,
	limit int64,
	total int64,
) LinkType {
	offset = 0
	q := requestURL.Query()
	q.Del("offset")
	requestURL.RawQuery = q.Encode()

	return LinkType{
		Rel:   "first",
		Href:  requestURL.String(),
		Title: getPageNumberAsTitle(offset, limit),
	}
}

func getLinkToPrevPage(
	requestURL url.URL,
	offset int64,
	limit int64,
	total int64,
) LinkType {
	q := requestURL.Query()
	if offset-limit > 0 {
		offset = offset - limit
		q.Set("offset", strconv.FormatInt(offset, 10))
	} else {
		offset = 0
		q.Del("offset")
	}
	requestURL.RawQuery = q.Encode()

	return LinkType{
		Rel:   "prev",
		Href:  requestURL.String(),
		Title: getPageNumberAsTitle(offset, limit),
	}
}

func getLinkToThisPage(
	requestURL url.URL,
	offset int64,
	limit int64,
	total int64,
) LinkType {
	pageLink := GetLinkToThisPage(requestURL, offset, limit, total)

	return LinkType{
		Rel:   "self",
		Href:  pageLink.String(),
		Title: getPageNumberAsTitle(offset, limit),
	}
}

// GetLinkToThisPage returns a link to the current page
func GetLinkToThisPage(
	requestURL url.URL,
	offset int64,
	limit int64,
	total int64,
) url.URL {
	if offset == 0 {
		q := requestURL.Query()
		q.Del("offset")
		requestURL.RawQuery = q.Encode()
	}

	return requestURL
}

func getLinkToNextPage(
	requestURL url.URL,
	offset int64,
	limit int64,
	total int64,
) LinkType {
	maxOffset := GetMaxOffset(total, limit)

	if offset+limit <= maxOffset {
		offset = offset + limit
		q := requestURL.Query()
		q.Set("offset", strconv.FormatInt(offset, 10))
		requestURL.RawQuery = q.Encode()
	} else if offset == 0 {
		offset = 0
		q := requestURL.Query()
		q.Del("offset")
		requestURL.RawQuery = q.Encode()
	}

	return LinkType{
		Rel:   "next",
		Href:  requestURL.String(),
		Title: getPageNumberAsTitle(offset, limit),
	}
}

func getLinkToLastPage(
	requestURL url.URL,
	offset int64,
	limit int64,
	total int64,
) LinkType {
	maxOffset := GetMaxOffset(total, limit)

	q := requestURL.Query()
	if maxOffset > 0 {
		offset = maxOffset
		q.Set("offset", strconv.FormatInt(offset, 10))
	} else {
		q.Del("offset")
	}
	requestURL.RawQuery = q.Encode()

	return LinkType{
		Rel:   "last",
		Href:  requestURL.String(),
		Title: getPageNumberAsTitle(offset, limit),
	}
}

func getPageNumberAsTitle(offset int64, limit int64) string {
	if offset == DefaultQueryOffset {
		return "1"
	}
	return strconv.FormatInt(offset/limit+1, 10)
}

// GetArrayLinks returns a collection of valid links for navigating a
// collection of items
func GetArrayLinks(
	requestURL url.URL,
	offset int64,
	limit int64,
	total int64,
) []LinkType {
	if limit == 0 {
		limit = DefaultQueryLimit
	}

	var arrayLinks []LinkType

	if offset > limit {
		arrayLinks = append(arrayLinks, getLinkToFirstPage(requestURL, offset, limit, total))
	}

	if offset > 0 {
		arrayLinks = append(arrayLinks, getLinkToPrevPage(requestURL, offset, limit, total))
	}

	arrayLinks = append(arrayLinks, getLinkToThisPage(requestURL, offset, limit, total))

	if offset < GetMaxOffset(total, limit) {
		arrayLinks = append(arrayLinks, getLinkToNextPage(requestURL, offset, limit, total))
	}

	if offset+limit < GetMaxOffset(total, limit) {
		arrayLinks = append(arrayLinks, getLinkToLastPage(requestURL, offset, limit, total))
	}

	return arrayLinks
}

// GetLink returns a link to an item
func GetLink(rel string, title string, itemType string, itemID int64) LinkType {

	var href string
	if itemID > 0 {
		href = fmt.Sprintf("%s/%d", ItemTypesToAPIItem[itemType], itemID)
	} else {
		href = ItemTypesToAPIItem[itemType]
	}

	return LinkType{Rel: rel, Href: href, Title: title}
}

// GetExtendedLink returns a link for child items
func GetExtendedLink(
	rel string,
	title string,
	itemType string,
	firstID int64,
	secondID int64,
) LinkType {
	// Link to item
	href := fmt.Sprintf(ItemTypesToAPIItem[itemType], firstID)

	// Link to child
	href = fmt.Sprintf("%s/%d", href, secondID)

	return LinkType{Rel: rel, Href: href, Title: title}
}
