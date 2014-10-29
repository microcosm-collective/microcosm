package helpers

import (
	"errors"
)

const (
	JumpUrl string = "http://microco.sm/out/"

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

var ItemTypesScoreable = map[string]int64{
	ItemTypeConversation: 6,
	ItemTypePoll:         7,
	ItemTypeArticle:      8,
	ItemTypeEvent:        9,
	ItemTypeQuestion:     10,
	ItemTypeClassified:   11,
	ItemTypeAlbum:        12,
}

const (
	ApiTypeActivity         string = "/api/v1/activity"
	ApiTypeAlbum            string = "/api/v1/albums"
	ApiTypeArticle          string = "/api/v1/articles"
	ApiTypeAttendee         string = "/api/v1/events/%d/attendees"
	ApiTypeAttachment       string = "attachments"
	ApiTypeAttribute        string = "/api/v1/%s/%d/attributes"
	ApiTypeAuth             string = "/api/v1/auth"
	ApiTypeClassified       string = "/api/v1/classifieds"
	ApiTypeComment          string = "/api/v1/comments"
	ApiTypeConversation     string = "/api/v1/conversations"
	ApiTypeEvent            string = "/api/v1/events"
	ApiTypeFile             string = "/api/v1/files"
	ApiTypeHuddle           string = "/api/v1/huddles"
	ApiTypeMicrocosm        string = "/api/v1/microcosms"
	ApiTypeQuestion         string = "/api/v1/questions"
	ApiTypePoll             string = "/api/v1/polls"
	ApiTypeProfile          string = "/api/v1/profiles"
	ApiTypeRole             string = "/api/v1/roles"
	ApiTypeSite             string = "/api/v1/sites"
	ApiTypeUpdate           string = "/api/v1/updates"
	ApiTypeUpdateOptionType string = "/api/v1/updates/preferences/%d"
	ApiTypeUser             string = "/api/v1/users"
	ApiTypeWatcher          string = "/api/v1/watchers"
	ApiTypeWhoAmI           string = "/api/v1/whoami"
)

var ItemTypesToApiItem = map[string]string{
	ItemTypeAttendee:         ApiTypeAttendee,
	ItemTypeActivity:         ApiTypeActivity,
	ItemTypeAlbum:            ApiTypeAlbum,
	ItemTypeArticle:          ApiTypeArticle,
	ItemTypeAttribute:        ApiTypeAttribute,
	ItemTypeAuth:             ApiTypeAuth,
	ItemTypeClassified:       ApiTypeClassified,
	ItemTypeComment:          ApiTypeComment,
	ItemTypeConversation:     ApiTypeConversation,
	ItemTypeEvent:            ApiTypeEvent,
	ItemTypeFile:             ApiTypeFile,
	ItemTypeHuddle:           ApiTypeHuddle,
	ItemTypeMicrocosm:        ApiTypeMicrocosm,
	ItemTypePoll:             ApiTypePoll,
	ItemTypeProfile:          ApiTypeProfile,
	ItemTypeQuestion:         ApiTypeQuestion,
	ItemTypeRole:             ApiTypeRole,
	ItemTypeSite:             ApiTypeSite,
	ItemTypeUpdate:           ApiTypeUpdate,
	ItemTypeUpdateOptionType: ApiTypeUpdateOptionType,
	ItemTypeUser:             ApiTypeUser,
	ItemTypeWatcher:          ApiTypeWatcher,
	ItemTypeWhoAmI:           ApiTypeWhoAmI,
}

const (
	UpdateTypeEventReminder      string = "event_reminder"
	UpdateTypeMentioned          string = "mentioned"
	UpdateTypeNewComment         string = "new_comment"
	UpdateTypeNewCommentInHuddle string = "new_comment_in_huddle"
	UpdateTypeNewEventAttendee   string = "new_attendee"
	UpdateTypeNewItem            string = "new_item"
	UpdateTypeNewPollVote        string = "new_vote"
	UpdateTypeReplyToComment     string = "reply_to_comment"
)

var UpdateTypes = map[string]int64{
	UpdateTypeNewComment:         1, // New comment on an item you're watching
	UpdateTypeReplyToComment:     2, // A reply to one of your comments
	UpdateTypeMentioned:          3, // Your username is mentioned
	UpdateTypeNewCommentInHuddle: 4, // New comment within a huddle
	UpdateTypeNewEventAttendee:   5, // RSVP to an event you're watching
	UpdateTypeNewPollVote:        6, // Vote on a poll you're watching
	UpdateTypeEventReminder:      7, // Reminder about an event you've RSVPd to
	UpdateTypeNewItem:            8, // New item created in microcosm you're watching
}

const (
	UpdateTextEventReminder      string = "You have an upcoming event"
	UpdateTextMentioned          string = "You were mentioned in a comment"
	UpdateTextMicrocosmActivity  string = "There is new activity in a microcosm you are subscribed to"
	UpdateTextNewComment         string = "There is a new comment in an item you are subscribed to"
	UpdateTextNewCommentInHuddle string = "There is a new comment in a huddle that you are part of"
	UpdateTextNewEventAttendee   string = "A user has RSVPed to an event you are subscribed to"
	UpdateTextNewItem            string = "There is a new item in a microcosm you are subscribed to"
	UpdateTextNewPollVote        string = "A vote has been cast in a poll you are subscribed to"
	UpdateTextReplyToComment     string = "A user has replied to your post"
)

var UpdateTexts = map[int64]string{
	1: UpdateTextNewComment,
	2: UpdateTextReplyToComment,
	3: UpdateTextMentioned,
	4: UpdateTextNewCommentInHuddle,
	5: UpdateTextNewEventAttendee,
	6: UpdateTextNewPollVote,
	7: UpdateTextEventReminder,
	8: UpdateTextNewItem,
	9: UpdateTextMicrocosmActivity,
}

func GetItemTypeFromInt(value int64) (string, error) {
	return GetMapStringFromInt(ItemTypes, value)
}

func GetMapStringFromInt(theMap map[string]int64, value int64) (string, error) {
	for k, v := range theMap {
		if v == value {
			return k, nil
		}
	}
	return "", errors.New("Item does not exist")
}
