package helpers

import (
	"net/url"
)

type ArrayType struct {
	Total     int64       `json:"total"`
	Limit     int64       `json:"limit"`
	Offset    int64       `json:"offset"`
	MaxOffset int64       `json:"maxOffset"`
	Pages     int64       `json:"totalPages"`
	Page      int64       `json:"page"`
	Links     []LinkType  `json:"links,omitempty"`
	Type      string      `json:"type"`
	Items     interface{} `json:"items"`
}

func ConstructArray(
	resources interface{},
	contentType string,
	total int64,
	limit int64,
	offset int64,
	pages int64,
	requestUrl *url.URL,
) ArrayType {

	if requestUrl != nil {
		return ArrayType{
			Total:     total,
			Limit:     limit,
			Offset:    offset,
			MaxOffset: GetMaxOffset(total, limit),
			Pages:     pages,
			Page:      GetCurrentPage(offset, limit),
			Links:     GetArrayLinks(*requestUrl, offset, limit, total),
			Type:      contentType,
			Items:     resources,
		}
	} else {
		return ArrayType{
			Total:     total,
			Limit:     limit,
			Offset:    offset,
			MaxOffset: GetMaxOffset(total, limit),
			Pages:     pages,
			Page:      GetCurrentPage(offset, limit),
			Type:      contentType,
			Items:     resources,
		}
	}
}

func GetCurrentPage(offset int64, limit int64) int64 {
	if limit == 0 {
		return 0
	}

	return (offset + limit) / limit
}
