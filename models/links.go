package models

import (
	"database/sql"
	"time"

	"github.com/lib/pq"
)

// Link describes a link within user generated content
type Link struct {
	ID          int64
	Rand        string
	ShortURL    string
	Domain      string
	URL         string
	Text        string
	Created     time.Time
	ResolvedURL sql.NullString
	Resolved    pq.NullTime
	Hits        int64
}
