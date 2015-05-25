package helpers

import (
	"net/url"
)

// ArrayType describes an array in JSON and how to paginate the collection
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

// ConstructArray returns an array
func ConstructArray(
	resources interface{},
	contentType string,
	total int64,
	limit int64,
	offset int64,
	pages int64,
	requestURL *url.URL,
) ArrayType {

	if requestURL != nil {
		return ArrayType{
			Total:     total,
			Limit:     limit,
			Offset:    offset,
			MaxOffset: GetMaxOffset(total, limit),
			Pages:     pages,
			Page:      GetCurrentPage(offset, limit),
			Links:     GetArrayLinks(*requestURL, offset, limit, total),
			Type:      contentType,
			Items:     resources,
		}
	}

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

// GetCurrentPage returns the current page for a given offset and limit value
func GetCurrentPage(offset int64, limit int64) int64 {
	if limit == 0 {
		return 0
	}

	return (offset + limit) / limit
}
