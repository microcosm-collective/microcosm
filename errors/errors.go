package errors

/*
* Error codes are intended to convey detailed errors internally and to clients.
* These should be combined with the appropriate HTTP status code, but are not
* intended to supercede correct HTTP responses. Therefore there is no error code
* for "not found" because HTTP 404 is sufficient. But there is an error code for
* "deleted" which should be served with HTTP 404 status code.
*
* Error codes are grouped under HTTP status code, with some item-specific errors
* defined below these. These should be return with HTTP 400 unless otherwise
* stated.
*
 */

const (

	// HTTP 400 Bad Request.
	// Content-type is not accepted (e.g. text/xml).
	BadContentType ErrCode = 1
	// Content does not match Content-Type or unmarshalling error.
	InvalidContent ErrCode = 2
	// A parameter was not of the expected type.
	UnexpectedType ErrCode = 3
	// A parameter was outside the expected range.
	OutOfRange ErrCode = 4

	// HTTP 401 Unauthorized.
	// Requested an action not permitted for guests.
	LoginRequired ErrCode = 5

	// HTTP 403 Forbidden.
	// Authenication.
	ExpiredToken ErrCode = 6
	InvalidToken ErrCode = 7
	// Authorisation.
	NotAdmin ErrCode = 8
	// Values from GetPermission.
	NoRead   ErrCode = 9
	NoCreate ErrCode = 10
	NoUpdate ErrCode = 11
	NoDelete ErrCode = 12
	// User has been banned.
	UserBanned ErrCode = 13

	// HTTP 404 Not Found.
	SiteNotFound ErrCode = 14
	// Entity deleted flag is true.
	Deleted ErrCode = 15

	// HTTP 429 Too Many Requests.
	ExceededQuota ErrCode = 16

	// Commenting-specific errors.
	Closed ErrCode = 17

	// Events-specific errors.
	EventRSVPFull    ErrCode = 18
	EventRSVPExpired ErrCode = 19

	// File attachments.
	// HTTP 413 Request Entity Too Large.
	FileTooLarge ErrCode = 20
	// Image dimensions are invalid.
	InvalidDimensions = 21
	// Creating an attachment with non-existent file metadata.
	MetadataNotPresent = 22
	// Item to which attachment is being attached does not exist.
	AttachmentItemNotFound = 23

	// HTTP 500 Internal Server Error.
	// Search query timed out.
	SearchTimeout = 24
)

// MicrocosmError implements the Error interface.
type MicrocosmError struct {
	SiteId       int64   `json:"siteId"`
	ProfileId    int64   `json:"profileId,omitempty"`
	Function     string  `json:"-"`
	ErrorCode    ErrCode `json:"errorCode"`
	ErrorMessage string  `json:"errorDetail"`
}

type ErrCode uint8

func (e MicrocosmError) Error() string {
	return e.ErrorMessage
}

func New(siteId int64, profileId int64, function string, errCode ErrCode, errMessage string) error {
	return &MicrocosmError{
		SiteId:       siteId,
		ProfileId:    profileId,
		Function:     function,
		ErrorCode:    errCode,
		ErrorMessage: errMessage,
	}
}
