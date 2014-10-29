package helpers

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

const (
	DefaultQueryLimit  int64 = 25
	DefaultQueryOffset int64 = 0
)

type LinkArrayType struct {
	Links []LinkType `json:"links"`
}

type LinkType struct {
	Rel   string `json:"rel,omitempty"` // REST
	Href  string `json:"href"`
	Title string `json:"title,omitempty"`
	Text  string `json:"text,omitempty"` // HTML
}

func GetLimitAndOffset(query url.Values) (int64, int64, int, error) {
	var (
		limit  int64
		offset int64
	)

	limit = DefaultQueryLimit
	if query.Get("limit") != "" {
		inLimit, err := strconv.ParseInt(query.Get("limit"), 10, 64)
		if err != nil {
			return 0, 0, http.StatusBadRequest, errors.New(
				fmt.Sprintf("limit (%s) is not a number.", query.Get("limit")),
			)
		}

		if inLimit < 1 {
			return 0, 0, http.StatusBadRequest, errors.New(
				fmt.Sprintf("limit (%d) cannot be zero or negative.", inLimit),
			)
		}

		if inLimit%5 != 0 {
			return 0, 0, http.StatusBadRequest, errors.New(
				fmt.Sprintf("limit (%d) must be a multiple of 5.", inLimit),
			)
		}

		if inLimit > 250 {
			return 0, 0, http.StatusBadRequest, errors.New(
				fmt.Sprintf("limit (%d) cannot exceed 100.", inLimit),
			)
		}

		limit = inLimit
	}

	offset = DefaultQueryOffset
	if query.Get("offset") != "" {
		inOffset, err := strconv.ParseInt(query.Get("offset"), 10, 64)
		if err != nil {
			return 0, 0, http.StatusBadRequest, errors.New(
				fmt.Sprintf("offset (%s) is not a number.", query.Get("offset")),
			)
		}

		if inOffset < 0 {
			return 0, 0, http.StatusBadRequest, errors.New(
				fmt.Sprintf("offset (%d) cannot be negative.", inOffset),
			)
		}

		if inOffset%limit != 0 {
			return 0, 0, http.StatusBadRequest, errors.New(
				fmt.Sprintf("offset (%d) must be a multiple of limit (%d) or zero.", inOffset, limit),
			)
		}

		offset = inOffset
	}

	return limit, offset, http.StatusOK, nil
}

func GetItemAndItemType(query url.Values) (int64, string, int, error) {
	var (
		itemId   int64
		itemType string
	)

	if query.Get("itemId") != "" {
		inItemId, err := strconv.ParseInt(query.Get("itemId"), 10, 64)
		if err != nil {
			return 0, "", http.StatusBadRequest, errors.New(
				fmt.Sprintf("itemId (%s) is not a number.", query.Get("itemId")),
			)
		}

		itemId = inItemId
	}

	if query.Get("itemType") != "" {
		inItemType := query.Get("itemType")

		itemType = inItemType
	}

	return itemId, itemType, http.StatusOK, nil
}

func GetAttending(query url.Values) (bool, int, error) {
	var (
		isAttending bool
	)

	if query.Get("isAttending") != "" {
		inAttending, err := strconv.ParseBool(query.Get("isAttending"))
		if err != nil {
			return false, http.StatusBadRequest, errors.New(
				fmt.Sprintf("isAttending (%s) is not a boolean.", query.Get("isAttending")),
			)
		}

		isAttending = inAttending
	}

	return isAttending, http.StatusOK, nil
}

func AttendanceStatus(query url.Values) (string, int, error) {
	var (
		status string
	)

	if query.Get("status") != "" {
		inStatus := query.Get("status")

		status = inStatus
	}

	return status, http.StatusOK, nil
}

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

func GetMaxOffset(total int64, limit int64) int64 {

	return ((total - 1) / limit) * limit
}

func getLinkToFirstPage(requestUrl url.URL, offset int64, limit int64, total int64) LinkType {

	offset = 0
	q := requestUrl.Query()
	q.Del("offset")
	requestUrl.RawQuery = q.Encode()

	return LinkType{
		Rel:   "first",
		Href:  requestUrl.String(),
		Title: getPageNumberAsTitle(offset, limit),
	}
}

func getLinkToPrevPage(requestUrl url.URL, offset int64, limit int64, total int64) LinkType {

	q := requestUrl.Query()
	if offset-limit > 0 {
		offset = offset - limit
		q.Set("offset", strconv.FormatInt(offset, 10))
	} else {
		offset = 0
		q.Del("offset")
	}
	requestUrl.RawQuery = q.Encode()

	return LinkType{
		Rel:   "prev",
		Href:  requestUrl.String(),
		Title: getPageNumberAsTitle(offset, limit),
	}
}

func getLinkToThisPage(requestUrl url.URL, offset int64, limit int64, total int64) LinkType {

	link := GetLinkToThisPage(requestUrl, offset, limit, total)

	return LinkType{
		Rel:   "self",
		Href:  link.String(),
		Title: getPageNumberAsTitle(offset, limit),
	}
}

func GetLinkToThisPage(requestUrl url.URL, offset int64, limit int64, total int64) url.URL {

	if offset == 0 {
		q := requestUrl.Query()
		q.Del("offset")
		requestUrl.RawQuery = q.Encode()
	}

	return requestUrl
}

func getLinkToNextPage(requestUrl url.URL, offset int64, limit int64, total int64) LinkType {
	maxOffset := GetMaxOffset(total, limit)

	if offset+limit <= maxOffset {
		offset = offset + limit
		q := requestUrl.Query()
		q.Set("offset", strconv.FormatInt(offset, 10))
		requestUrl.RawQuery = q.Encode()
	} else if offset == 0 {
		offset = 0
		q := requestUrl.Query()
		q.Del("offset")
		requestUrl.RawQuery = q.Encode()
	}

	return LinkType{
		Rel:   "next",
		Href:  requestUrl.String(),
		Title: getPageNumberAsTitle(offset, limit),
	}
}

func getLinkToLastPage(requestUrl url.URL, offset int64, limit int64, total int64) LinkType {
	maxOffset := GetMaxOffset(total, limit)

	q := requestUrl.Query()
	if maxOffset > 0 {
		offset = maxOffset
		q.Set("offset", strconv.FormatInt(offset, 10))
	} else {
		q.Del("offset")
	}
	requestUrl.RawQuery = q.Encode()

	return LinkType{
		Rel:   "last",
		Href:  requestUrl.String(),
		Title: getPageNumberAsTitle(offset, limit),
	}
}

func getPageNumberAsTitle(offset int64, limit int64) string {
	if offset == DefaultQueryOffset {
		return "1"
	} else {
		return strconv.FormatInt(offset/limit+1, 10)
	}
}

func GetArrayLinks(requestUrl url.URL, offset int64, limit int64, total int64) []LinkType {

	if limit == 0 {
		limit = DefaultQueryLimit
	}

	var arrayLinks []LinkType

	if offset > limit {
		arrayLinks = append(arrayLinks, getLinkToFirstPage(requestUrl, offset, limit, total))
	}

	if offset > 0 {
		arrayLinks = append(arrayLinks, getLinkToPrevPage(requestUrl, offset, limit, total))
	}

	arrayLinks = append(arrayLinks, getLinkToThisPage(requestUrl, offset, limit, total))

	if offset < GetMaxOffset(total, limit) {
		arrayLinks = append(arrayLinks, getLinkToNextPage(requestUrl, offset, limit, total))
	}

	if offset+limit < GetMaxOffset(total, limit) {
		arrayLinks = append(arrayLinks, getLinkToLastPage(requestUrl, offset, limit, total))
	}

	return arrayLinks
}

func GetLink(rel string, title string, itemType string, itemId int64) LinkType {

	var href string
	if itemId > 0 {
		href = fmt.Sprintf("%s/%d", ItemTypesToApiItem[itemType], itemId)
	} else {
		href = ItemTypesToApiItem[itemType]
	}

	return LinkType{Rel: rel, Href: href, Title: title}
}

func GetExtendedLink(rel string, title string, itemType string, firstId int64, secondId int64) LinkType {
	href := fmt.Sprintf("%s/%d", fmt.Sprintf(ItemTypesToApiItem[itemType], firstId), secondId)

	return LinkType{Rel: rel, Href: href, Title: title}
}
