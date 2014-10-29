package models

import (
	"database/sql"
	"time"

	"github.com/lib/pq"
)

type Link struct {
	Id          int64
	Rand        string
	ShortUrl    string
	Domain      string
	Url         string
	Text        string
	Created     time.Time
	ResolvedUrl sql.NullString
	Resolved    pq.NullTime
	Hits        int64
}
