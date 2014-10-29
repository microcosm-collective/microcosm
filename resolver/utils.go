package resolver

import (
	"strconv"
)

func atoi64(s string) int64 {
	i, _ := strconv.ParseInt(s, 10, 64)
	return i
}

func pageToOffset(page int64, perPage int64) int64 {
	if page > 1 {
		return (page - 1) * perPage
	}

	return page
}
