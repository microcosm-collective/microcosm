package helpers

import (
	"database/sql"
	"time"

	"github.com/lib/pq"
)

// StatType describes a statistic
type StatType struct {
	Metric string `json:"metric"`
	Value  int64  `json:"value"`
}

// AttachmentType describes an attachment
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

// ThumbnailType describes a thumbnail of an image attachment
type ThumbnailType struct {
	Href     string `json:"href"`
	MimeType string `json:"mimetype"`
	Bytes    int    `json:"bytes"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}

// DefaultMetaType is used by single items
type DefaultMetaType struct {
	CreatedType
	EditedType
	ExtendedMetaType
}

// SummaryMetaType is used by summary of single items
type SummaryMetaType struct {
	CreatedType
	ExtendedMetaType
}

// DefaultNoFlagsMetaType is used by single items
type DefaultNoFlagsMetaType struct {
	CreatedType
	EditedType
	CoreMetaType
}

// DefaultReducedFlagsMetaType is used by single items
type DefaultReducedFlagsMetaType struct {
	CreatedType
	EditedType
	SimpleMetaType
}

// CoreMetaType is used implicitly by all meta types, and explicitly by all
// collections
type CoreMetaType struct {
	Stats       []StatType  `json:"stats,omitempty"`
	Links       []LinkType  `json:"links,omitempty"`
	Permissions interface{} `json:"permissions,omitempty"`
}

// CreatedMetaType for items (such as Alerts) that cannot be edited but have a
// creator
type CreatedMetaType struct {
	CreatedType
	ExtendedMetaType
}

// SimpleMetaType is used explicitly by comments
type SimpleMetaType struct {
	Flags SimpleFlagsType `json:"flags,omitempty"`
	CoreMetaType
}

// ExtendedMetaType is used implicitly by all meta types, and explicitly by all
// collections
type ExtendedMetaType struct {
	Flags FlagsType `json:"flags,omitempty"`
	CoreMetaType
}

// CreatedType describes an author/creator
type CreatedType struct {
	Created     time.Time   `json:"created"`
	CreatedByID int64       `json:"-"`
	CreatedBy   interface{} `json:"createdBy"`
}

// EditedType describes edited meta data
type EditedType struct {
	EditedNullable     pq.NullTime    `json:"-"`
	Edited             string         `json:"edited,omitempty"`
	EditedByNullable   sql.NullInt64  `json:"-"`
	EditedBy           interface{}    `json:"editedBy,omitempty"`
	EditReasonNullable sql.NullString `json:"-"`
	EditReason         string         `json:"editReason,omitempty"`
}

// SimpleFlagsType describes simple flags
type SimpleFlagsType struct {
	Deleted   bool `json:"deleted"`
	Moderated bool `json:"moderated"`
	Visible   bool `json:"visible"`
}

// FlagsType describes the common flags
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
	SendSMS   interface{} `json:"sendSMS,omitempty"`
	Attending interface{} `json:"attending,omitempty"`
}

// SetVisible determines whether the item should be visible and if so sets the
// visible flag
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
