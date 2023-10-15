package helpers

import "fmt"

// JumpURL is the URL to use for external URL redirects
const JumpURL string = "http://microcosm.app/out/"

// Constants for all of the itemTypes
const (
	ItemTypeActivity         string = "activity"
	ItemTypeAlbum            string = "album"
	ItemTypeArticle          string = "article"
	ItemTypeAttendee         string = "attendee"
	ItemTypeAttachment       string = "attachment"
	ItemTypeAttribute        string = "attribute"
	ItemTypeAuth             string = "auth"
	ItemTypeClassified       string = "classified"
	ItemTypeComment          string = "comment"
	ItemTypeConversation     string = "conversation"
	ItemTypeEvent            string = "event"
	ItemTypeFile             string = "file"
	ItemTypeHuddle           string = "huddle"
	ItemTypeMicrocosm        string = "microcosm"
	ItemTypePoll             string = "poll"
	ItemTypeProfile          string = "profile"
	ItemTypeQuestion         string = "question"
	ItemTypeRole             string = "role"
	ItemTypeSite             string = "site"
	ItemTypeUpdate           string = "update"
	ItemTypeUpdateOptionType string = "update_type"
	ItemTypeUser             string = "user"
	ItemTypeWatcher          string = "watcher"
	ItemTypeWhoAmI           string = "whoami"
)

// ItemTypes is a list of all itemTypes
var ItemTypes = map[string]int64{
	ItemTypeSite:             1,
	ItemTypeMicrocosm:        2,
	ItemTypeProfile:          3,
	ItemTypeComment:          4,
	ItemTypeHuddle:           5,
	ItemTypeConversation:     6,
	ItemTypePoll:             7,
	ItemTypeArticle:          8,
	ItemTypeEvent:            9,
	ItemTypeQuestion:         10,
	ItemTypeClassified:       11,
	ItemTypeAlbum:            12,
	ItemTypeAttendee:         13,
	ItemTypeUser:             14,
	ItemTypeAttribute:        15,
	ItemTypeUpdate:           16,
	ItemTypeRole:             17,
	ItemTypeUpdateOptionType: 18,
	ItemTypeWatcher:          19,
	ItemTypeAuth:             20,
	ItemTypeAttachment:       21,
}

// ItemTypesCommentable is a list of the itemTypes that can have comments on them
var ItemTypesCommentable = map[string]int64{
	ItemTypeProfile:      3,
	ItemTypeHuddle:       5,
	ItemTypeConversation: 6,
	ItemTypePoll:         7,
	ItemTypeArticle:      8,
	ItemTypeEvent:        9,
	ItemTypeQuestion:     10,
	ItemTypeClassified:   11,
	ItemTypeAlbum:        12,
}

// ItemTypesScoreable is a list of the itemTypes that will be scored to determine
// trending items
var ItemTypesScoreable = map[string]int64{
	ItemTypeConversation: 6,
	ItemTypePoll:         7,
	ItemTypeArticle:      8,
	ItemTypeEvent:        9,
	ItemTypeQuestion:     10,
	ItemTypeClassified:   11,
	ItemTypeAlbum:        12,
}

// List of APITypes
const (
	APITypeActivity         string = "/api/v1/activity"
	APITypeAlbum            string = "/api/v1/albums"
	APITypeArticle          string = "/api/v1/articles"
	APITypeAttendee         string = "/api/v1/events/%d/attendees"
	APITypeAttachment       string = "attachments"
	APITypeAttribute        string = "/api/v1/%s/%d/attributes"
	APITypeAuth             string = "/api/v1/auth"
	APITypeClassified       string = "/api/v1/classifieds"
	APITypeComment          string = "/api/v1/comments"
	APITypeConversation     string = "/api/v1/conversations"
	APITypeEvent            string = "/api/v1/events"
	APITypeFile             string = "/api/v1/files"
	APITypeHuddle           string = "/api/v1/huddles"
	APITypeMicrocosm        string = "/api/v1/microcosms"
	APITypeQuestion         string = "/api/v1/questions"
	APITypePoll             string = "/api/v1/polls"
	APITypeProfile          string = "/api/v1/profiles"
	APITypeRole             string = "/api/v1/roles"
	APITypeSite             string = "/api/v1/sites"
	APITypeUpdate           string = "/api/v1/updates"
	APITypeUpdateOptionType string = "/api/v1/updates/preferences/%d"
	APITypeUser             string = "/api/v1/users"
	APITypeWatcher          string = "/api/v1/watchers"
	APITypeWhoAmI           string = "/api/v1/whoami"
)

// ItemTypesToAPIItem maps from the itemType to the APIType
var ItemTypesToAPIItem = map[string]string{
	ItemTypeAttendee:         APITypeAttendee,
	ItemTypeActivity:         APITypeActivity,
	ItemTypeAlbum:            APITypeAlbum,
	ItemTypeArticle:          APITypeArticle,
	ItemTypeAttribute:        APITypeAttribute,
	ItemTypeAuth:             APITypeAuth,
	ItemTypeClassified:       APITypeClassified,
	ItemTypeComment:          APITypeComment,
	ItemTypeConversation:     APITypeConversation,
	ItemTypeEvent:            APITypeEvent,
	ItemTypeFile:             APITypeFile,
	ItemTypeHuddle:           APITypeHuddle,
	ItemTypeMicrocosm:        APITypeMicrocosm,
	ItemTypePoll:             APITypePoll,
	ItemTypeProfile:          APITypeProfile,
	ItemTypeQuestion:         APITypeQuestion,
	ItemTypeRole:             APITypeRole,
	ItemTypeSite:             APITypeSite,
	ItemTypeUpdate:           APITypeUpdate,
	ItemTypeUpdateOptionType: APITypeUpdateOptionType,
	ItemTypeUser:             APITypeUser,
	ItemTypeWatcher:          APITypeWatcher,
	ItemTypeWhoAmI:           APITypeWhoAmI,
}

// List of update_types
const (
	UpdateTypeNewComment         string = "new_comment"
	UpdateTypeReplyToComment     string = "reply_to_comment"
	UpdateTypeMentioned          string = "mentioned"
	UpdateTypeNewCommentInHuddle string = "new_comment_in_huddle"
	UpdateTypeNewEventAttendee   string = "new_attendee"
	UpdateTypeNewPollVote        string = "new_vote"
	UpdateTypeEventReminder      string = "event_reminder"
	UpdateTypeNewItem            string = "new_item"
	UpdateTypeNewUser            string = "new_user"
)

// UpdateTypes is a map of update_types
var UpdateTypes = map[string]int64{
	UpdateTypeNewComment:         1, // New comment on an item you're watching
	UpdateTypeReplyToComment:     2, // A reply to one of your comments
	UpdateTypeMentioned:          3, // Your username is mentioned
	UpdateTypeNewCommentInHuddle: 4, // New comment within a huddle
	UpdateTypeNewEventAttendee:   5, // RSVP to an event you're watching
	UpdateTypeNewPollVote:        6, // Vote on a poll you're watching
	UpdateTypeEventReminder:      7, // Reminder about an event you've RSVPd to
	UpdateTypeNewItem:            8, // New item created in microcosm you're watching
	UpdateTypeNewUser:            9, // New user on a site whose profiles you are watching
}

// List of texts for update_types
const (
	UpdateTextNewComment         string = "There is a new comment in an item you are subscribed to"
	UpdateTextReplyToComment     string = "A user has replied to your post"
	UpdateTextMentioned          string = "You were mentioned in a comment"
	UpdateTextNewCommentInHuddle string = "There is a new comment in a huddle that you are part of"
	UpdateTextNewEventAttendee   string = "A user has RSVPed to an event you are subscribed to"
	UpdateTextNewPollVote        string = "A vote has been cast in a poll you are subscribed to"
	UpdateTextEventReminder      string = "You have an upcoming event"
	UpdateTextNewItem            string = "There is a new item in a microcosm you are subscribed to"
	UpdateTextNewUser            string = "There is a new user and you are watching all profiles"
)

// UpdateTexts is a map of email titles for a given range of update_types
var UpdateTexts = map[int64]string{
	1: UpdateTextNewComment,
	2: UpdateTextReplyToComment,
	3: UpdateTextMentioned,
	4: UpdateTextNewCommentInHuddle,
	5: UpdateTextNewEventAttendee,
	6: UpdateTextNewPollVote,
	7: UpdateTextEventReminder,
	8: UpdateTextNewItem,
	9: UpdateTextNewUser,
}

// GetItemTypeFromInt returns the string itemType for the integer itemTypeID
func GetItemTypeFromInt(value int64) (string, error) {
	return GetMapStringFromInt(ItemTypes, value)
}

// GetMapStringFromInt for a given map[string]int64 return the value of the
// key where the value matches
func GetMapStringFromInt(theMap map[string]int64, value int64) (string, error) {
	for k, v := range theMap {
		if v == value {
			return k, nil
		}
	}
	return "", fmt.Errorf("item does not exist")
}
