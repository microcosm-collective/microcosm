package errors

// Error codes are intended to convey detailed errors internally and to clients.
// These should be combined with the appropriate HTTP status code, but are not
// intended to supercede correct HTTP responses. Therefore there is no error code
// for "not found" because HTTP 404 is sufficient. But there is an error code for
// "deleted" which should be served with HTTP 404 status code.
//
// Error codes are grouped under HTTP status code, with some item-specific errors
// defined below these. These should be return with HTTP 400 unless otherwise
// stated.

// HTTP 400 Bad Request
const (
	// BadContentType means that the content-type is not accepted (e.g. text/xml)
	BadContentType ErrCode = 1

	// InvalidContent means that the content does not match Content-Type
	InvalidContent ErrCode = 2

	// UnexpectedType means a parameter was not of the expected type
	UnexpectedType ErrCode = 3

	// OutOfRange means a parameter was outside the expected range
	OutOfRange ErrCode = 4

	// Closed means that the event is closed.
	Closed ErrCode = 17

	// EventRSVPFull means that the event is full
	EventRSVPFull ErrCode = 18

	// EventRSVPExpired means that the event has expired
	EventRSVPExpired ErrCode = 19
)

// HTTP 401 Unauthorized
const (
	// LoginRequired means an action not permitted for guests
	LoginRequired ErrCode = 5
)

// HTTP 403 Forbidden
const (
	// ExpiredToken means a token has expired
	ExpiredToken ErrCode = 6

	// InvalidToken means a token is no longer (or was never) valid
	InvalidToken ErrCode = 7

	// NotAdmin means a given action requires administrator privileges
	NotAdmin ErrCode = 8

	// NoRead means one does not have read permissions
	NoRead ErrCode = 9

	// NoCreate means one does not have create permissions
	NoCreate ErrCode = 10

	// NoUpdate means one does not have update permissions
	NoUpdate ErrCode = 11

	// NoDelete means one does not have delete permissions
	NoDelete ErrCode = 12

	// UserBanned means that user has been banned
	UserBanned ErrCode = 13
)

// HTTP 404 Not Found
const (
	// SiteNotFound means that the site is not found
	SiteNotFound ErrCode = 14

	// Deleted means that the item has been deleted
	Deleted ErrCode = 15
)

// HTTP 429 Too Many Requests
const (
	// ExceededQuota means that the client has exceeded the rate limiting quota
	ExceededQuota ErrCode = 16
)

// HTTP 413 Request Entity Too Large
const (
	// FileTooLarge means that the uploaded file is too large to process
	FileTooLarge ErrCode = 20

	// InvalidDimensions means that the dimensions of an invalid are not valid
	InvalidDimensions = 21

	// MetadataNotPresent means an attachment with non-existent file metadata
	MetadataNotPresent = 22

	// AttachmentItemNotFound means an attachment on an item that does not exist
	AttachmentItemNotFound = 23
)

// HTTP 500 Internal Server Error
const (
	// SearchTimeout means that the search has timed out
	SearchTimeout = 24
)

// MicrocosmError implements the Error interface.
type MicrocosmError struct {
	SiteID       int64   `json:"siteId"`
	ProfileID    int64   `json:"profileId,omitempty"`
	Function     string  `json:"-"`
	ErrorCode    ErrCode `json:"errorCode"`
	ErrorMessage string  `json:"errorDetail"`
}

// ErrCode is a uint8
type ErrCode uint8

// Error returns a string of the error message
func (e MicrocosmError) Error() string {
	return e.ErrorMessage
}

// New returns a new error
func New(siteID int64, profileID int64, function string, errCode ErrCode, errMessage string) error {
	return &MicrocosmError{
		SiteID:       siteID,
		ProfileID:    profileID,
		Function:     function,
		ErrorCode:    errCode,
		ErrorMessage: errMessage,
	}
}
