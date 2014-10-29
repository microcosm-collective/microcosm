package helpers

import (
	"database/sql"
	"time"

	"github.com/lib/pq"
)

type StatType struct {
	Metric string `json:"metric"`
	Value  int64  `json:"value"`
}

type AttachmentType struct {
	Href      string        `json:"href"`
	Created   int           `json:"created"`
	CreatedBy string        `json:"createdBy"`
	MimeType  string        `json:"mimetypes"`
	Bytes     int           `json:"bytes"`
	Width     int           `json:"width"`
	Height    int           `json:"height"`
	Views     int           `json:"views"`
	Thumbnail ThumbnailType `json:"thumbnail,omitempty"`
}

type ThumbnailType struct {
	Href     string `json:"href"`
	MimeType string `json:"mimetype"`
	Bytes    int    `json:"bytes"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}

// Used by single items
type DefaultMetaType struct {
	CreatedType
	EditedType
	ExtendedMetaType
}

// Used by summary of single items
type SummaryMetaType struct {
	CreatedType
	ExtendedMetaType
}

// Used by single items
type DefaultNoFlagsMetaType struct {
	CreatedType
	EditedType
	CoreMetaType
}

// Used by single items
type DefaultReducedFlagsMetaType struct {
	CreatedType
	EditedType
	SimpleMetaType
}

// Used implicitly by all meta types, and explicitly by all collections
type CoreMetaType struct {
	Stats       []StatType  `json:"stats,omitempty"`
	Links       []LinkType  `json:"links,omitempty"`
	Permissions interface{} `json:"permissions,omitempty"`
}

// For items (such as Alerts) that cannot be edited but have a creator
type CreatedMetaType struct {
	CreatedType
	ExtendedMetaType
}

// Used explicitly by comments
type SimpleMetaType struct {
	Flags SimpleFlagsType `json:"flags,omitempty"`
	CoreMetaType
}

// Used implicitly by all meta types, and explicitly by all collections
type ExtendedMetaType struct {
	Flags FlagsType `json:"flags,omitempty"`
	CoreMetaType
}

type CreatedType struct {
	Created     time.Time   `json:"created"`
	CreatedById int64       `json:"-"`
	CreatedBy   interface{} `json:"createdBy"`
}
type EditedType struct {
	EditedNullable     pq.NullTime    `json:"-"`
	Edited             string         `json:"edited,omitempty"`
	EditedByNullable   sql.NullInt64  `json:"-"`
	EditedBy           interface{}    `json:"editedBy,omitempty"`
	EditReasonNullable sql.NullString `json:"-"`
	EditReason         string         `json:"editReason,omitempty"`
}

type SimpleFlagsType struct {
	Deleted   bool `json:"deleted"`
	Moderated bool `json:"moderated"`
	Visible   bool `json:"visible"`
}
type FlagsType struct {
	Sticky    interface{} `json:"sticky,omitempty"`
	Open      interface{} `json:"open,omitempty"`
	Deleted   interface{} `json:"deleted,omitempty"`
	Moderated interface{} `json:"moderated,omitempty"`
	Visible   interface{} `json:"visible,omitempty"`
	Unread    interface{} `json:"unread,omitempty"`
	Watched   interface{} `json:"watched,omitempty"`
	Ignored   interface{} `json:"ignored,omitempty"`
	SendEmail interface{} `json:"sendEmail,omitempty"`
	SendSms   interface{} `json:"sendSMS,omitempty"`
	Attending interface{} `json:"attending,omitempty"`
}

func (f *FlagsType) SetVisible() {
	switch f.Moderated.(type) {
	case bool:
		switch f.Deleted.(type) {
		case bool:
			f.Visible = !(f.Moderated.(bool) || f.Deleted.(bool))
		default:
			return
		}
	default:
		return
	}
}
