package models

import (
	"errors"
	"net/http"

	"github.com/golang/glog"

	h "github.com/microcosm-cc/microcosm/helpers"
)

type SummaryContainer struct {
	ItemTypeId int64       `json:"-"`
	ItemType   string      `json:"itemType"`
	ItemId     int64       `json:"-"`
	Summary    interface{} `json:"item"`
	Valid      bool        `json:"-"`
}

type SummaryContainerRequest struct {
	Item   SummaryContainer
	Err    error
	Status int
	Seq    int
}

type SummaryContainerRequestsBySeq []SummaryContainerRequest

func (v SummaryContainerRequestsBySeq) Len() int {
	return len(v)
}

func (v SummaryContainerRequestsBySeq) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

func (v SummaryContainerRequestsBySeq) Less(i, j int) bool {
	return v[i].Seq < v[j].Seq
}

func HandleSummaryContainerRequest(
	siteId int64,
	itemTypeId int64,
	itemId int64,
	profileId int64,
	seq int,
	out chan<- SummaryContainerRequest,
) {

	item, status, err := GetSummaryContainer(
		siteId,
		itemTypeId,
		itemId,
		profileId,
	)

	response := SummaryContainerRequest{
		Item:   item,
		Status: status,
		Err:    err,
		Seq:    seq,
	}
	out <- response
}

func GetSummaryContainer(
	siteId int64,
	itemTypeId int64,
	itemId int64,
	profileId int64,
) (
	SummaryContainer,
	int,
	error,
) {

	summary, status, err := GetSummary(siteId, itemTypeId, itemId, profileId)
	if err != nil {
		return SummaryContainer{}, status, err
	}

	item := SummaryContainer{}
	item.ItemTypeId = itemTypeId

	itemType, _ := h.GetMapStringFromInt(h.ItemTypes, itemTypeId)
	item.ItemType = itemType

	item.ItemId = itemId
	item.Summary = summary
	item.Valid = true

	return item, http.StatusOK, nil
}

// Fetches the smallest and most cacheable representation of a thing, usually
// the result of Get<Item>Summary
//
// In this context, the 4th arg (profileId) is for the person asking for the summary
// That is needed for strongly permission items such as huddles.
func GetSummary(
	siteId int64,
	itemTypeId int64,
	itemId int64,
	profileId int64,
) (
	interface{},
	int,
	error,
) {

	if itemId == 0 {
		glog.Errorf(
			"GetSummary(%d, %d, %d, %d) Item not found",
			siteId,
			itemTypeId,
			itemId,
			profileId,
		)
		return nil, http.StatusNotFound, errors.New("Item not found")
	}

	switch itemTypeId {

	case h.ItemTypes[h.ItemTypeAlbum]:

	case h.ItemTypes[h.ItemTypeArticle]:

	case h.ItemTypes[h.ItemTypeAttendee]:
		summary, status, err := GetProfileSummary(siteId, itemId)
		if err != nil {
			glog.Errorf(
				"GetProfileSummary(%d, %d) %+v",
				siteId,
				itemId,
				err,
			)
		}
		return summary, status, err

	case h.ItemTypes[h.ItemTypeClassified]:

	case h.ItemTypes[h.ItemTypeComment]:
		summary, status, err := GetCommentSummary(siteId, itemId)
		if err != nil {
			glog.Errorf(
				"GetCommentSummary(%d, %d) %+v",
				siteId,
				itemId,
				err,
			)
		}
		return summary, status, err

	case h.ItemTypes[h.ItemTypeConversation]:
		summary, status, err := GetConversationSummary(
			siteId,
			itemId,
			profileId,
		)
		if err != nil {
			glog.Errorf(
				"GetConversationSummary(%d, %d, %d) %+v",
				siteId,
				itemId,
				profileId,
				err,
			)
		}
		return summary, status, err

	case h.ItemTypes[h.ItemTypeEvent]:
		summary, status, err := GetEventSummary(siteId, itemId, profileId)
		if err != nil {
			glog.Errorf(
				"GetEventSummary(%d, %d, %d) %+v",
				siteId,
				itemId,
				profileId,
				err,
			)
		}
		return summary, status, err

	case h.ItemTypes[h.ItemTypeHuddle]:
		summary, status, err := GetHuddleSummary(siteId, profileId, itemId)
		if err != nil {
			glog.Errorf(
				"GetHuddleSummary(%d, %d, %d) %+v",
				siteId,
				profileId,
				itemId,
				err,
			)
		}
		return summary, status, err

	case h.ItemTypes[h.ItemTypeMicrocosm]:
		summary, status, err := GetMicrocosmSummary(siteId, itemId, profileId)
		if err != nil {
			glog.Errorf(
				"GetMicrocosmSummary(%d, %d, %d) %+v",
				siteId,
				itemId,
				profileId,
				err,
			)
		}
		return summary, status, err

	case h.ItemTypes[h.ItemTypePoll]:
		summary, status, err := GetPollSummary(siteId, itemId, profileId)
		if err != nil {
			glog.Errorf(
				"GetPollSummary(%d, %d, %d) %+v",
				siteId,
				itemId,
				profileId,
				err,
			)
		}
		return summary, status, err

	case h.ItemTypes[h.ItemTypeProfile]:
		summary, status, err := GetProfileSummary(siteId, itemId)
		if err != nil && status != http.StatusNotFound {
			glog.Errorf(
				"GetProfileSummary(%d, %d) %+v",
				siteId,
				itemId,
				err,
			)
		}
		return summary, status, err

	case h.ItemTypes[h.ItemTypeQuestion]:

	case h.ItemTypes[h.ItemTypeSite]:
		summary, status, err := GetSite(siteId)
		if err != nil {
			glog.Errorf("GetSite(%d) %+v", siteId, err)
		}
		return summary, status, err

	default:

	}

	return nil, http.StatusInternalServerError,
		errors.New("GetSummary() not yet implemented")
}

// Fetches a title of a thing, or something that can be used as a title. Will
// not trim or otherwise shorten the title. In the case of profile this returns
// the profileName, etc.
func GetTitle(
	siteId int64,
	itemTypeId int64,
	itemId int64,
	profileId int64,
) (
	string,
	int,
	error,
) {

	switch itemTypeId {
	case h.ItemTypes[h.ItemTypeMicrocosm]:
		return GetMicrocosmTitle(itemId), http.StatusOK, nil
	case h.ItemTypes[h.ItemTypeHuddle]:
		return GetHuddleTitle(itemId), http.StatusOK, nil
	default:
	}

	summary, status, err := GetSummary(siteId, itemTypeId, itemId, profileId)
	if err != nil {
		glog.Errorf(
			"GetSummary(%d, %d, %d, %d) %+v",
			siteId,
			itemTypeId,
			itemId,
			profileId,
			err,
		)
		return "", status, err
	}

	switch summary.(type) {
	case ConversationSummaryType:
		return summary.(ConversationSummaryType).Title, http.StatusOK, nil

	case EventSummaryType:
		return summary.(EventSummaryType).Title, http.StatusOK, nil

	case PollSummaryType:
		return summary.(PollSummaryType).Title, http.StatusOK, nil

	case ProfileSummaryType:
		return summary.(ProfileSummaryType).ProfileName, http.StatusOK, nil

	case SiteType:
		return summary.(SiteType).Title, http.StatusOK, nil

	default:
	}

	return "", http.StatusNotImplemented,
		errors.New("GetTitle is not implemented for this item")
}
