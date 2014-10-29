package models

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/golang/glog"

	h "github.com/microcosm-cc/microcosm/helpers"
)

type EmailMergeData struct {
	SiteTitle    string
	ProtoAndHost string
	ForProfile   ProfileSummaryType
	ForEmail     string
	ByProfile    ProfileSummaryType
	Subject      string
	ContextLink  string
	ContextText  string
	Body         string
}

// The only public interfaces to this dispatcher are the following methods
// which provide one interface per UpdateType.
//
// These methods then work out who should be notified of the update and how
// and will then call the notification methods

// Update Type #1 : New comment in an item you're watching
func SendUpdatesForNewCommentInItem(
	siteId int64,
	comment CommentSummaryType,
) (
	int,
	error,
) {

	updateType, status, err := GetUpdateType(
		h.UpdateTypes[h.UpdateTypeNewComment],
	)
	if err != nil {
		glog.Errorf("%s %+v", "GetUpdateType()", err)
		return status, err
	}

	// WHO GETS THE UPDATES?

	// Nobody watchers a comment, so we need to get the recipients for the item
	// the comment is attached to
	recipients, status, err := GetUpdateRecipients(
		siteId,
		comment.ItemTypeId,
		comment.ItemId,
		updateType.Id,
		comment.Meta.CreatedById,
	)
	if err != nil {
		glog.Errorf("%s %+v", "GetUpdateRecipients()", err)
		return status, err
	}

	// SEND UPDATES
	//
	// Freely acknowledging that we're going to loop the same thing many
	// times. But... we want all of the local updates to work even if the
	// emails fail, etc. So we'll do them primarily in the order of
	// delivery type rather than the order of recipients.

	if len(recipients) == 0 {
		glog.Info("No recipients to send updates to")
		return http.StatusOK, nil
	}

	///////////////////
	// LOCAL UPDATES //
	///////////////////
	tx, err := h.GetTransaction()
	if err != nil {
		glog.Errorf("%s %+v", "h.GetTransaction()", err)
		return http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Could not start transaction: %v", err.Error()),
		)
	}
	defer tx.Rollback()

	//glog.Info("Creating updates")
	sendEmails := false
	for _, recipient := range recipients {

		if !sendEmails &&
			recipient.SendEmail &&
			recipient.ForProfile.Id != comment.Meta.CreatedById {

			sendEmails = true
		}

		// Everyone gets an update
		var update = UpdateType{}
		update.SiteId = siteId
		update.UpdateTypeId = updateType.Id
		update.ForProfileId = recipient.ForProfile.Id
		update.ItemTypeId = h.ItemTypes[h.ItemTypeComment]
		update.ItemId = comment.Id
		update.Meta.CreatedById = comment.Meta.CreatedById
		status, err := update.insert(tx)
		if err != nil {
			glog.Errorf("%s %+v", "update.insert(tx)", err)
			return status, err
		}
	}
	err = tx.Commit()
	if err != nil {
		glog.Errorf("%s %+v", "tx.Commit()", err)
		return http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Transaction failed: %v", err.Error()),
		)
	} else {
		//glog.Info("Updates sent")
	}

	///////////////////
	// EMAIL UPDATES //
	///////////////////

	// For which we have a template in the form of a subject, HTML and text
	// body and we need to build up a data object to merge into the template
	if sendEmails {

		//glog.Info("Building email merge data")
		mergeData := EmailMergeData{}

		site, status, err := GetSite(siteId)
		if err != nil {
			glog.Errorf("%s %+v", "GetSite()", err)
			return status, err
		}
		mergeData.SiteTitle = site.Title
		mergeData.ProtoAndHost = site.GetUrl()

		mergeData.ContextLink = fmt.Sprintf(
			"%s/comments/%d/incontext/",
			mergeData.ProtoAndHost,
			comment.Id,
		)

		itemTitle, status, err := GetTitle(
			siteId,
			comment.ItemTypeId,
			comment.ItemId,
			0,
		)
		if err != nil {
			glog.Errorf("%s %+v", "GetTitle()", err)
			return status, err
		}
		mergeData.ContextText = itemTitle

		byProfile, status, err := GetProfileSummary(
			siteId,
			comment.Meta.CreatedById,
		)
		if err != nil {
			glog.Errorf("%s %+v", "GetProfileSummary()", err)
			return http.StatusInternalServerError, err
		}
		mergeData.ByProfile = byProfile

		mergeData.Body = comment.HTML

		// And the templates
		subjectTemplate, textTemplate, htmlTemplate, status, err :=
			updateType.GetEmailTemplates()
		if err != nil {
			glog.Errorf("%s %+v", "updateType.GetEmailTemplates()", err)
			return status, err
		}

		for _, recipient := range recipients {
			// Everyone who wants an email gets an email... except:
			// 1) the author
			// 2) anyone already emailed
			// 3) if this comment is a reply to another, the parent comment
			//    author as that is handled by the NewReplyToYourComment thing
			lastRead, status, err := GetLastReadTime(
				comment.ItemTypeId,
				comment.ItemId,
				recipient.ForProfile.Id,
			)
			if err != nil {
				glog.Errorf("%s %+v", "GetLastReadTime()", err)
				return status, err
			}

			var parentCommentCreatedById int64
			if comment.InReplyTo > 0 {

				parentComment, status, err := GetCommentSummary(
					siteId,
					comment.InReplyTo,
				)
				if err != nil {
					glog.Errorf("%s %+v", "GetComment()", err)
					return status, err
				}
				parentCommentCreatedById = parentComment.Meta.CreatedById
			}

			if recipient.SendEmail &&
				recipient.ForProfile.Id != comment.Meta.CreatedById &&
				(lastRead.After(recipient.LastNotified) ||
					recipient.LastNotified.IsZero()) &&
				recipient.ForProfile.Id != parentCommentCreatedById {

				// Personalisation of email
				mergeData.ForProfile = recipient.ForProfile

				user, status, err := GetUser(recipient.ForProfile.UserId)
				if err != nil {
					glog.Errorf("%s %+v", "GetUser()", err)
					return status, err
				}
				mergeData.ForEmail = user.Email

				status, err = MergeAndSendEmail(
					siteId,
					fmt.Sprintf(EMAIL_FROM, GetSiteTitle(siteId)),
					mergeData.ForEmail,
					subjectTemplate,
					textTemplate,
					htmlTemplate,
					mergeData,
				)
				if err != nil {
					glog.Errorf("%s %+v", "MergeAndSendEmail()", err)
				}

				recipient.Watcher.UpdateLastNotified()
			}
		}
	}

	/////////////////
	// SMS UPDATES //
	/////////////////
	for _, recipient := range recipients {
		// Everyone who wants an SMS, except the author, gets an SMS
		if recipient.SendSMS {
			// Send SMS
		}
	}

	return http.StatusOK, nil
}

// Update Type #2 : New reply to a comment that you made
func SendUpdatesForNewReplyToYourComment(
	siteId int64,
	comment CommentSummaryType,
) (
	int,
	error,
) {

	updateType, status, err := GetUpdateType(
		h.UpdateTypes[h.UpdateTypeReplyToComment],
	)
	if err != nil {
		glog.Errorf("%s %+v", "GetUpdateType()", err)
		return status, err
	}

	parentComment, status, err := GetCommentSummary(siteId, comment.InReplyTo)
	if err != nil {
		glog.Errorf("%s %+v", "GetComment()", err)
		return status, err
	}
	profileId := parentComment.Meta.CreatedById

	forProfile, status, err := GetProfileSummary(siteId, profileId)
	if err != nil {
		glog.Errorf("%s %+v", "GetProfileSummary()", err)
		return http.StatusInternalServerError, err
	}

	///////////////////
	// LOCAL UPDATES //
	///////////////////
	tx, err := h.GetTransaction()
	if err != nil {
		glog.Errorf("%s %+v", "h.GetTransaction()", err)
		return http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Could not start transaction: %v", err.Error()),
		)
	}
	defer tx.Rollback()

	//glog.Info("Creating update")

	// Everyone gets an update
	var update = UpdateType{}
	update.SiteId = siteId
	update.UpdateTypeId = updateType.Id
	update.ForProfileId = forProfile.Id
	update.ItemTypeId = h.ItemTypes[h.ItemTypeComment]
	update.ItemId = comment.Id
	update.Meta.CreatedById = comment.Meta.CreatedById
	status, err = update.insert(tx)
	if err != nil {
		glog.Errorf("%s %+v", "update.insert(tx)", err)
		return status, err
	}

	err = tx.Commit()
	if err != nil {
		glog.Errorf("%s %+v", "tx.Commit()", err)
		return http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Transaction failed: %v", err.Error()),
		)
	} else {
		//glog.Info("Update sent")
	}

	///////////////////
	// EMAIL UPDATES //
	///////////////////
	updateOptions, status, err := GetCommunicationOptions(
		siteId,
		profileId,
		updateType.Id,
		h.ItemTypes[h.ItemTypeComment],
		comment.Id,
	)
	if err != nil {
		glog.Errorf("%s %+v", "GetUpdateOptionForUpdateType()", err)
		return status, err
	}

	if updateOptions.SendEmail {

		glog.Info("Building email merge data")
		mergeData := EmailMergeData{}

		site, status, err := GetSite(siteId)
		if err != nil {
			glog.Errorf("%s %+v", "GetSite()", err)
			return status, err
		}
		mergeData.SiteTitle = site.Title
		mergeData.ProtoAndHost = site.GetUrl()

		mergeData.ContextLink = fmt.Sprintf(
			"%s/comments/%d/incontext/",
			mergeData.ProtoAndHost,
			comment.Id,
		)

		itemTitle, status, err := GetTitle(
			siteId,
			comment.ItemTypeId,
			comment.ItemId,
			0,
		)
		if err != nil {
			glog.Errorf("%s %+v", "GetTitle()", err)
			return status, err
		}
		mergeData.ContextText = itemTitle

		byProfile, status, err := GetProfileSummary(
			siteId,
			comment.Meta.CreatedById,
		)
		if err != nil {
			glog.Errorf("%s %+v", "GetProfileSummary()", err)
			return http.StatusInternalServerError, err
		}
		mergeData.ByProfile = byProfile

		mergeData.Body = comment.HTML

		// And the templates
		subjectTemplate, textTemplate, htmlTemplate, status, err :=
			updateType.GetEmailTemplates()
		if err != nil {
			glog.Errorf("%s %+v", "updateType.GetEmailTemplates()", err)
			return status, err
		}

		// Personalisation of email
		mergeData.ForProfile = forProfile

		user, status, err := GetUser(forProfile.UserId)
		if err != nil {
			glog.Errorf("%s %+v", "GetUser()", err)
			return status, err
		}
		mergeData.ForEmail = user.Email

		status, err = MergeAndSendEmail(
			siteId,
			fmt.Sprintf(EMAIL_FROM, GetSiteTitle(siteId)),
			mergeData.ForEmail,
			subjectTemplate,
			textTemplate,
			htmlTemplate,
			mergeData,
		)
		if err != nil {
			glog.Errorf("%s %+v", "MergeAndSendEmail()", err)
		}
	}

	/////////////////
	// SMS UPDATES //
	/////////////////
	if updateOptions.SendSMS {
	}

	return http.StatusOK, nil
}

// Update Type #3 : New mention in a comment
func SendUpdatesForNewMentionInComment(
	siteId int64,
	forProfileId int64,
	commentId int64,
) (
	int,
	error,
) {

	updateType, status, err := GetUpdateType(
		h.UpdateTypes[h.UpdateTypeMentioned],
	)
	if err != nil {
		glog.Errorf("%s %+v", "GetUpdateType()", err)
		return status, err
	}

	// Two things to note about this method and where it was called from...
	// 1) Update.upsert() has already created the entry in the updates table
	// 2) The revision of the comment, and maybe the comment itself, may not yet
	//    exist
	//
	// This is because this is called mid-transaction within mentions.go
	//
	// It's our job to send the email and/or SMS, but we need to sleep so that
	// the transaction finishes and everything is ready
	time.Sleep(5 * time.Second)

	updateOptions, status, err := GetCommunicationOptions(
		siteId,
		forProfileId,
		updateType.Id,
		h.ItemTypes[h.ItemTypeComment],
		commentId,
	)
	if err != nil {
		glog.Errorf("%s %+v", "GetUpdateOptionForUpdateType()", err)
		return status, err
	}

	///////////////////
	// EMAIL UPDATES //
	///////////////////
	if updateOptions.SendEmail {
		comment, status, err := GetCommentSummary(siteId, commentId)
		if err != nil {
			glog.Errorf("%s %+v", "GetComment()", err)
			return status, err
		}

		forProfile, status, err := GetProfileSummary(siteId, forProfileId)
		if err != nil {
			glog.Errorf("%s %+v", "GetProfileSummary()", err)
			return http.StatusInternalServerError, err
		}

		glog.Info("Building email merge data")
		mergeData := EmailMergeData{}

		site, status, err := GetSite(siteId)
		if err != nil {
			glog.Errorf("%s %+v", "GetSite()", err)
			return status, err
		}
		mergeData.SiteTitle = site.Title
		mergeData.ProtoAndHost = site.GetUrl()

		mergeData.ContextLink = fmt.Sprintf(
			"%s/comments/%d/incontext/",
			mergeData.ProtoAndHost,
			comment.Id,
		)

		itemTitle, status, err := GetTitle(
			siteId,
			comment.ItemTypeId,
			comment.ItemId,
			0,
		)
		if err != nil {
			glog.Errorf("%s %+v", "GetTitle()", err)
			return status, err
		}
		mergeData.ContextText = itemTitle

		byProfile, status, err := GetProfileSummary(
			siteId,
			comment.Meta.CreatedById,
		)
		if err != nil {
			glog.Errorf("%s %+v", "GetProfileSummary()", err)
			return http.StatusInternalServerError, err
		}
		mergeData.ByProfile = byProfile

		mergeData.Body = comment.HTML

		// And the templates
		subjectTemplate, textTemplate, htmlTemplate, status, err :=
			updateType.GetEmailTemplates()
		if err != nil {
			glog.Errorf("%s %+v", "updateType.GetEmailTemplates()", err)
			return status, err
		}

		// Personalisation of email
		mergeData.ForProfile = forProfile

		user, status, err := GetUser(forProfile.UserId)
		if err != nil {
			glog.Errorf("%s %+v", "GetUser()", err)
			return status, err
		}
		mergeData.ForEmail = user.Email

		status, err = MergeAndSendEmail(
			siteId,
			fmt.Sprintf(EMAIL_FROM, GetSiteTitle(siteId)),
			mergeData.ForEmail,
			subjectTemplate,
			textTemplate,
			htmlTemplate,
			mergeData,
		)
		if err != nil {
			glog.Errorf("%s %+v", "MergeAndSendEmail()", err)
		}

	}

	/////////////////
	// SMS UPDATES //
	/////////////////
	if updateOptions.SendSMS {

	}

	return http.StatusOK, nil
}

// Update Type #4 : New comment in a huddle you are participating in
//
// TODO(buro9): This is still based on the same code as new comment, but are
// there some special rules because it's a huddle? If so, they're not yet
// implemented.
func SendUpdatesForNewCommentInHuddle(
	siteId int64,
	comment CommentSummaryType,
) (
	int,
	error,
) {

	updateType, status, err := GetUpdateType(
		h.UpdateTypes[h.UpdateTypeNewCommentInHuddle],
	)
	if err != nil {
		glog.Errorf("%s %+v", "GetUpdateType()", err)
		return status, err
	}

	// WHO GETS THE UPDATES?

	// Nobody watchers a comment, so we need to get the recipients for the item
	// the comment is attached to
	recipients, status, err := GetUpdateRecipients(
		siteId,
		comment.ItemTypeId,
		comment.ItemId,
		updateType.Id,
		comment.Meta.CreatedById,
	)
	if err != nil {
		glog.Errorf("%s %+v", "GetUpdateRecipients()", err)
		return status, err
	}

	// SEND UPDATES
	//
	// Freely acknowledging that we're going to loop the same thing many
	// times. But... we want all of the local updates to work even if the
	// emails fail, etc. So we'll do them primarily in the order of
	// delivery type rather than the order of recipients.

	if len(recipients) == 0 {
		glog.Info("No recipients to send updates to")
		return http.StatusOK, nil
	}

	///////////////////
	// LOCAL UPDATES //
	///////////////////
	tx, err := h.GetTransaction()
	if err != nil {
		glog.Errorf("%s %+v", "h.GetTransaction()", err)
		return http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Could not start transaction: %v", err.Error()),
		)
	}
	defer tx.Rollback()

	//glog.Info("Creating updates")
	sendEmails := false
	for _, recipient := range recipients {

		if !sendEmails &&
			recipient.SendEmail &&
			recipient.ForProfile.Id != comment.Meta.CreatedById {

			sendEmails = true
		}

		// Everyone gets an update
		var update = UpdateType{}
		update.SiteId = siteId
		update.UpdateTypeId = updateType.Id
		update.ForProfileId = recipient.ForProfile.Id
		update.ItemTypeId = h.ItemTypes[h.ItemTypeComment]
		update.ItemId = comment.Id
		update.Meta.CreatedById = comment.Meta.CreatedById
		status, err := update.insert(tx)
		if err != nil {
			glog.Errorf("%s %+v", "update.insert(tx)", err)
			return status, err
		}
	}
	err = tx.Commit()
	if err != nil {
		glog.Errorf("%s %+v", "tx.Commit()", err)
		return http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Transaction failed: %v", err.Error()),
		)
	} else {
		//glog.Info("Updates sent")
	}

	///////////////////
	// EMAIL UPDATES //
	///////////////////

	// For which we have a template in the form of a subject, HTML and text
	// body and we need to build up a data object to merge into the template
	if sendEmails {

		glog.Info("Building email merge data")
		mergeData := EmailMergeData{}

		site, status, err := GetSite(siteId)
		if err != nil {
			glog.Errorf("%s %+v", "GetSite()", err)
			return status, err
		}
		mergeData.SiteTitle = site.Title
		mergeData.ProtoAndHost = site.GetUrl()

		mergeData.ContextLink = fmt.Sprintf(
			"%s/comments/%d/incontext/",
			mergeData.ProtoAndHost,
			comment.Id,
		)

		itemTitle, status, err := GetTitle(
			siteId,
			comment.ItemTypeId,
			comment.ItemId,
			0,
		)
		if err != nil {
			glog.Errorf("%s %+v", "GetTitle()", err)
			return status, err
		}
		mergeData.ContextText = itemTitle

		byProfile, status, err := GetProfileSummary(
			siteId,
			comment.Meta.CreatedById,
		)
		if err != nil {
			glog.Errorf("%s %+v", "GetProfileSummary()", err)
			return http.StatusInternalServerError, err
		}
		mergeData.ByProfile = byProfile

		mergeData.Body = comment.HTML

		// And the templates
		subjectTemplate, textTemplate, htmlTemplate, status, err :=
			updateType.GetEmailTemplates()
		if err != nil {
			glog.Errorf("%s %+v", "updateType.GetEmailTemplates()", err)
			return status, err
		}

		for _, recipient := range recipients {
			// Everyone who wants an email gets an email... except:
			// 1) the author
			// 2) anyone already emailed
			// 3) if this comment is a reply to another, the parent comment
			//    author as that is handled by the NewReplyToYourComment thing
			lastRead, status, err := GetLastReadTime(
				comment.ItemTypeId,
				comment.ItemId,
				recipient.ForProfile.Id,
			)
			if err != nil {
				glog.Errorf("%s %+v", "GetLastReadTime()", err)
				return status, err
			}

			var parentCommentCreatedById int64
			if comment.InReplyTo > 0 {

				parentComment, status, err := GetCommentSummary(
					siteId,
					comment.InReplyTo,
				)
				if err != nil {
					glog.Errorf("%s %+v", "GetComment()", err)
					return status, err
				}
				parentCommentCreatedById = parentComment.Meta.CreatedById
			}

			if recipient.SendEmail &&
				recipient.ForProfile.Id != comment.Meta.CreatedById &&
				(lastRead.After(recipient.LastNotified) ||
					recipient.LastNotified.IsZero()) &&
				recipient.ForProfile.Id != parentCommentCreatedById {

				// Personalisation of email
				mergeData.ForProfile = recipient.ForProfile

				user, status, err := GetUser(recipient.ForProfile.UserId)
				if err != nil {
					glog.Errorf("%s %+v", "GetUser()", err)
					return status, err
				}
				mergeData.ForEmail = user.Email

				status, err = MergeAndSendEmail(
					siteId,
					fmt.Sprintf(EMAIL_FROM, GetSiteTitle(siteId)),
					mergeData.ForEmail,
					subjectTemplate,
					textTemplate,
					htmlTemplate,
					mergeData,
				)
				if err != nil {
					glog.Errorf("%s %+v", "MergeAndSendEmail()", err)
				}

				recipient.Watcher.UpdateLastNotified()
			}
		}
	}

	/////////////////
	// SMS UPDATES //
	/////////////////
	for _, recipient := range recipients {
		// Everyone who wants an SMS, except the author, gets an SMS
		if recipient.SendSMS {
			// Send SMS
		}
	}

	return http.StatusOK, nil
}

// Update Type #5 : New attendee in an event
func SendUpdatesForNewAttendeeInAnEvent(
	siteId int64,
	attendee AttendeeType,
) (
	int,
	error,
) {

	updateType, status, err := GetUpdateType(
		h.UpdateTypes[h.UpdateTypeNewEventAttendee],
	)
	if err != nil {
		glog.Errorf("%s %+v", "GetUpdateType()", err)
		return status, err
	}

	// WHO GETS THE UPDATES?

	// Nobody watchers a comment, so we need to get the recipients for the item
	// the comment is attached to
	recipients, status, err := GetUpdateRecipients(
		siteId,
		h.ItemTypes[h.ItemTypeEvent],
		attendee.EventId,
		updateType.Id,
		attendee.ProfileId,
	)
	if err != nil {
		glog.Errorf("%s %+v", "GetUpdateRecipients()", err)
		return status, err
	}

	// SEND UPDATES
	//
	// Freely acknowledging that we're going to loop the same thing many
	// times. But... we want all of the local updates to work even if the
	// emails fail, etc. So we'll do them primarily in the order of
	// delivery type rather than the order of recipients.

	if len(recipients) == 0 {
		glog.Info("No recipients to send updates to")
		return http.StatusOK, nil
	}

	///////////////////
	// LOCAL UPDATES //
	///////////////////
	tx, err := h.GetTransaction()
	if err != nil {
		glog.Errorf("%s %+v", "h.GetTransaction()", err)
		return http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Could not start transaction: %v", err.Error()),
		)
	}
	defer tx.Rollback()

	glog.Info("Creating updates")
	sendEmails := false
	for _, recipient := range recipients {

		if !sendEmails &&
			recipient.SendEmail &&
			recipient.ForProfile.Id != attendee.ProfileId {

			sendEmails = true
		}

		// Everyone gets an update
		var update = UpdateType{}
		update.SiteId = siteId
		update.UpdateTypeId = updateType.Id
		update.ForProfileId = recipient.ForProfile.Id
		update.ItemTypeId = h.ItemTypes[h.ItemTypeEvent]
		update.ItemId = attendee.EventId
		update.Meta.CreatedById = attendee.ProfileId
		status, err := update.insert(tx)
		if err != nil {
			glog.Errorf("%s %+v", "update.insert(tx)", err)
			return status, err
		}
	}
	err = tx.Commit()
	if err != nil {
		glog.Errorf("%s %+v", "tx.Commit()", err)
		return http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Transaction failed: %v", err.Error()),
		)
	} else {
		glog.Info("Updates sent")
	}

	///////////////////
	// EMAIL UPDATES //
	///////////////////

	// For which we have a template in the form of a subject, HTML and text
	// body and we need to build up a data object to merge into the template
	if sendEmails {

		glog.Info("Building email merge data")
		mergeData := EmailMergeData{}

		site, status, err := GetSite(siteId)
		if err != nil {
			glog.Errorf("%s %+v", "GetSite()", err)
			return status, err
		}
		mergeData.SiteTitle = site.Title
		mergeData.ProtoAndHost = site.GetUrl()

		mergeData.ContextLink = fmt.Sprintf(
			"%s/events/%d/",
			mergeData.ProtoAndHost,
			attendee.EventId,
		)

		itemTitle, status, err := GetTitle(
			siteId,
			h.ItemTypes[h.ItemTypeEvent],
			attendee.EventId,
			0,
		)
		if err != nil {
			glog.Errorf("%s %+v", "GetTitle()", err)
			return status, err
		}
		mergeData.ContextText = itemTitle

		byProfile, status, err := GetProfileSummary(siteId, attendee.ProfileId)
		if err != nil {
			glog.Errorf("%s %+v", "GetProfileSummary()", err)
			return http.StatusInternalServerError, err
		}
		mergeData.ByProfile = byProfile

		// And the templates
		subjectTemplate, textTemplate, htmlTemplate, status, err :=
			updateType.GetEmailTemplates()
		if err != nil {
			glog.Errorf("%s %+v", "updateType.GetEmailTemplates()", err)
			return status, err
		}

		for _, recipient := range recipients {
			// Everyone who wants an email gets an email... except:
			// 1) the author
			// 2) anyone already emailed
			// 3) if this comment is a reply to another, the parent comment author
			//    as that is handled by the NewReplyToYourComment thing
			lastRead, status, err := GetLastReadTime(
				h.ItemTypes[h.ItemTypeEvent],
				attendee.EventId,
				recipient.ForProfile.Id,
			)
			if err != nil {
				glog.Errorf("%s %+v", "GetLastReadTime()", err)
				return status, err
			}

			if recipient.SendEmail &&
				recipient.ForProfile.Id != attendee.ProfileId &&
				(lastRead.After(recipient.LastNotified) ||
					recipient.LastNotified.IsZero()) {

				// Personalisation of email
				mergeData.ForProfile = recipient.ForProfile

				user, status, err := GetUser(recipient.ForProfile.UserId)
				if err != nil {
					glog.Errorf("%s %+v", "GetUser()", err)
					return status, err
				}
				mergeData.ForEmail = user.Email

				status, err = MergeAndSendEmail(
					siteId,
					fmt.Sprintf(EMAIL_FROM, GetSiteTitle(siteId)),
					mergeData.ForEmail,
					subjectTemplate,
					textTemplate,
					htmlTemplate,
					mergeData,
				)
				if err != nil {
					glog.Errorf("%s %+v", "MergeAndSendEmail()", err)
				}

				recipient.Watcher.UpdateLastNotified()
			}
		}
	}

	/////////////////
	// SMS UPDATES //
	/////////////////
	for _, recipient := range recipients {
		// Everyone who wants an SMS, except the author, gets an SMS
		if recipient.SendSMS {
			// Send SMS
		}
	}

	return http.StatusOK, nil
}

// Update Type #6 : New vote in a poll
func SendUpdatesForNewVoteInAPoll(siteId int64, poll *PollType) (int, error) {

	// TODO(buro9): Not yet implemented

	return http.StatusOK, nil
}

// Update Type #7 : Event reminder as event imminent
func SendUpdatesForEventReminder(siteId int64, event *EventType) (int, error) {

	// TODO(buro9): Not yet implemented but could be done

	return http.StatusOK, nil
}

// Update Type #8 : A new item in a Microcosm
func SendUpdatesForNewItemInAMicrocosm(
	siteId int64,
	item interface{},
) (
	int,
	error,
) {

	updateType, status, err := GetUpdateType(h.UpdateTypes[h.UpdateTypeNewItem])
	if err != nil {
		glog.Errorf("%s %+v", "GetUpdateType()", err)
		return status, err
	}

	var (
		itemTypeId   int64
		itemType     string
		itemId       int64
		createdById  int64
		conversation ConversationType
		event        EventType
		poll         PollType
	)

	switch item.(type) {
	case ConversationType:
		conversation = item.(ConversationType)
		itemTypeId = h.ItemTypes[h.ItemTypeConversation]
		itemType = h.ItemTypeConversation
		itemId = conversation.Id
		createdById = conversation.Meta.CreatedById

	case EventType:
		event = item.(EventType)
		itemTypeId = h.ItemTypes[h.ItemTypeEvent]
		itemType = h.ItemTypeEvent
		itemId = event.Id
		createdById = event.Meta.CreatedById

	case PollType:
		poll = item.(PollType)
		itemTypeId = h.ItemTypes[h.ItemTypePoll]
		itemType = h.ItemTypePoll
		itemId = poll.Id
		createdById = poll.Meta.CreatedById

	default:
		glog.Errorf("%s %+v", "type not known", item)
		return http.StatusExpectationFailed,
			errors.New("Type of item is mysterious")
	}

	// WHO GETS THE UPDATES?
	recipients, status, err := GetUpdateRecipients(
		siteId,
		itemTypeId,
		itemId,
		updateType.Id,
		createdById,
	)
	if err != nil {
		glog.Errorf("%s %+v", "GetUpdateRecipients()", err)
		return status, err
	}

	// SEND UPDATES
	if len(recipients) == 0 {
		glog.Info("No recipients to send updates to")
		return http.StatusOK, nil
	}

	///////////////////
	// LOCAL UPDATES //
	///////////////////
	tx, err := h.GetTransaction()
	if err != nil {
		glog.Errorf("%s %+v", "h.GetTransaction()", err)
		return http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Could not start transaction: %v", err.Error()),
		)
	}
	defer tx.Rollback()

	glog.Info("Creating updates")
	sendEmails := false
	for _, recipient := range recipients {

		if !sendEmails &&
			recipient.SendEmail &&
			recipient.ForProfile.Id != createdById {

			sendEmails = true
		}

		// Everyone gets an update
		var update = UpdateType{}
		update.SiteId = siteId
		update.UpdateTypeId = updateType.Id
		update.ForProfileId = recipient.ForProfile.Id
		update.ItemTypeId = itemTypeId
		update.ItemId = itemId
		update.Meta.CreatedById = createdById
		status, err := update.insert(tx)
		if err != nil {
			glog.Errorf("%s %+v", "update.insert(tx)", err)
			return status, err
		}
	}
	err = tx.Commit()
	if err != nil {
		glog.Errorf("%s %+v", "tx.Commit()", err)
		return http.StatusInternalServerError, errors.New(
			fmt.Sprintf("Transaction failed: %v", err.Error()),
		)
	} else {
		glog.Info("Updates sent")
	}

	///////////////////
	// EMAIL UPDATES //
	///////////////////

	// For which we have a template in the form of a subject, HTML and text
	// body and we need to build up a data object to merge into the template
	if sendEmails {

		glog.Info("Building email merge data")
		mergeData := EmailMergeData{}

		site, status, err := GetSite(siteId)
		if err != nil {
			glog.Errorf("%s %+v", "GetSite()", err)
			return status, err
		}
		mergeData.SiteTitle = site.Title
		mergeData.ProtoAndHost = site.GetUrl()

		mergeData.ContextLink = fmt.Sprintf(
			"%s/%ss/%d/",
			mergeData.ProtoAndHost,
			itemType,
			itemId,
		)

		itemTitle, status, err := GetTitle(siteId, itemTypeId, itemId, 0)
		if err != nil {
			glog.Errorf("%s %+v", "GetTitle()", err)
			return status, err
		}
		mergeData.ContextText = itemTitle

		byProfile, status, err := GetProfileSummary(siteId, createdById)
		if err != nil {
			glog.Errorf("%s %+v", "GetProfileSummary()", err)
			return http.StatusInternalServerError, err
		}
		mergeData.ByProfile = byProfile

		// And the templates
		subjectTemplate, textTemplate, htmlTemplate, status, err :=
			updateType.GetEmailTemplates()
		if err != nil {
			glog.Errorf("%s %+v", "updateType.GetEmailTemplates()", err)
			return status, err
		}

		for _, recipient := range recipients {
			// Everyone who wants an email gets an email... except:
			// 1) the author
			if recipient.SendEmail &&
				recipient.ForProfile.Id != createdById {

				// Personalisation of email
				mergeData.ForProfile = recipient.ForProfile

				user, status, err := GetUser(recipient.ForProfile.UserId)
				if err != nil {
					glog.Errorf("%s %+v", "GetUser()", err)
					return status, err
				}
				mergeData.ForEmail = user.Email

				status, err = MergeAndSendEmail(
					siteId,
					fmt.Sprintf(EMAIL_FROM, GetSiteTitle(siteId)),
					mergeData.ForEmail,
					subjectTemplate,
					textTemplate,
					htmlTemplate,
					mergeData,
				)
				if err != nil {
					glog.Errorf("%s %+v", "MergeAndSendEmail()", err)
				}

				recipient.Watcher.UpdateLastNotified()
			}
		}
	}

	/////////////////
	// SMS UPDATES //
	/////////////////
	for _, recipient := range recipients {
		// Everyone who wants an SMS, except the author, gets an SMS
		if recipient.SendSMS {
			// Send SMS
		}
	}

	return http.StatusOK, nil
}

// Update Type #9 : A new item in a Microcosm
func SendUpdatesForNewProfileOnSite(
	siteId int64,
	profileId int64,
) (
	int,
	error,
) {

	// Also send to watchers of the site

	// TODO(buro9): Not yet implemented but could be done

	return http.StatusOK, nil
}

// returns a user's update options if present, otherwise it returns
// the default preference for the given update type.
func GetCommunicationOptions(
	siteId int64,
	profileId int64,
	updateTypeId int64,
	itemTypeId int64,
	itemId int64,
) (
	UpdateOptionType,
	int,
	error,
) {

	_, status, err := GetProfileOptions(profileId)
	if err != nil {
		glog.Errorf("GetProfileOptions(%d) %+v", profileId, err)
		// Can't do anything here as the profile_id fkey constraint will fail
		return UpdateOptionType{}, status, errors.New("Insert of update options failed")
	}

	db, err := h.GetConnection()
	if err != nil {
		glog.Errorf("h.GetConnection() %+v", err)
		return UpdateOptionType{}, http.StatusInternalServerError, err
	}

	rows, err := db.Query(`
SELECT send_email
      ,send_sms
      ,description
  FROM get_communication_options($1, $2, $3, $4, $5)
       LEFT JOIN update_types a ON update_type_id = $5`,
		siteId,
		itemId,
		itemTypeId,
		profileId,
		updateTypeId,
	)
	if err != nil {
		glog.Errorf(
			"db.Query(%d, %d, %d, %d, %d) %+v",
			siteId,
			itemId,
			itemTypeId,
			profileId,
			updateTypeId,
			err,
		)
		return UpdateOptionType{},
			http.StatusInternalServerError,
			errors.New("Database query failed")
	}
	defer rows.Close()

	var m UpdateOptionType

	for rows.Next() {
		m = UpdateOptionType{}
		err = rows.Scan(
			&m.SendEmail,
			&m.SendSMS,
			&m.Description,
		)
		if err != nil {
			glog.Errorf("rows.Scan() %+v", err)
			return UpdateOptionType{},
				http.StatusInternalServerError,
				errors.New("Row parsing error")
		}
	}
	err = rows.Err()
	if err != nil {
		glog.Errorf("rows.Err() %+v", err)
		return UpdateOptionType{},
			http.StatusInternalServerError,
			errors.New("Error fetching rows")
	}
	rows.Close()

	m.ProfileId = int64(profileId)
	m.UpdateTypeId = int64(updateTypeId)

	return m, http.StatusOK, nil
}
