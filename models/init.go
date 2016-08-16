package models

import (
	"encoding/gob"

	h "github.com/microcosm-cc/microcosm/helpers"
)

func init() {
	// Required by the cache stuff
	gob.Register([]AttributeType{})
	gob.Register([]h.StatType{})
	gob.Register([]MicrocosmLinkType{})
	gob.Register(AccessTokenType{})
	gob.Register(AttendeeType{})
	gob.Register(CommentSummaryType{})
	gob.Register(ConversationSummaryType{})
	gob.Register(ConversationType{})
	gob.Register(EventSummaryType{})
	gob.Register(EventType{})
	gob.Register(HuddleSummaryType{})
	gob.Register(HuddleType{})
	gob.Register(Item{})
	gob.Register(LastComment{})
	gob.Register(MicrocosmSummaryType{})
	gob.Register(MicrocosmType{})
	gob.Register(PollSummaryType{})
	gob.Register(PollType{})
	gob.Register(ProfileOptionType{})
	gob.Register(ProfileSummaryType{})
	gob.Register(ProfileType{})
	gob.Register(RoleType{})
	gob.Register(SiteType{})
	gob.Register(SummaryContainer{})
	gob.Register(UpdateTypesType{})
	gob.Register(UpdateType{})
	gob.Register(UserType{})
	gob.Register(WatcherType{})
}
