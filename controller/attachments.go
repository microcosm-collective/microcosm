package controller

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	h "github.com/microcosm-cc/microcosm/helpers"
	"github.com/microcosm-cc/microcosm/models"
)

func AttachmentsHandler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	ctl := AttachmentsController{}

	switch c.GetHttpMethod() {
	case "OPTIONS":
		c.RespondWithOptions([]string{"OPTIONS", "POST", "HEAD", "GET"})
		return
	case "POST":
		ctl.Create(c)
	case "HEAD":
		ctl.ReadMany(c)
	case "GET":
		ctl.ReadMany(c)
	default:
		c.RespondWithStatus(http.StatusMethodNotAllowed)
		return
	}
}

type AttachmentsController struct{}

func (ctl *AttachmentsController) Create(c *models.Context) {

	attachment := models.AttachmentType{}

	err := c.Fill(&attachment)
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("The post data is invalid: %v", err.Error()),
			http.StatusBadRequest,
		)
		return
	}

	if attachment.FileHash == "" {
		c.RespondWithErrorMessage(
			"You must supply a file hash",
			http.StatusBadRequest,
		)
		return
	}

	// Check that the file hash has a corresponding attachment_meta record
	metadata, status, err := models.GetMetadata(attachment.FileHash)
	if err != nil {
		if status == http.StatusNotFound {
			c.RespondWithErrorMessage(
				fmt.Sprintf("File does not have a metadata record"),
				http.StatusBadRequest,
			)
			return
		} else {
			c.RespondWithErrorMessage(
				fmt.Sprintf("Could not retrieve metadata: %v", err.Error()),
				http.StatusBadRequest,
			)
			return
		}
	}

	attachment.AttachmentMetaId = metadata.AttachmentMetaId

	// Determine whether this is an attachment to a profile or comment, and if the
	// user is authorised to do so
	path_prefix := ""

	if c.RouteVars["profile_id"] != "" {

		profileId, err := strconv.ParseInt(c.RouteVars["profile_id"], 10, 64)
		if err != nil {
			c.RespondWithErrorMessage(
				fmt.Sprintf("The supplied profile ID ('%s') is not a number.", c.RouteVars["profile_id"]),
				http.StatusBadRequest,
			)
			return
		}
		_, status, err := models.GetProfileSummary(c.Site.ID, profileId)
		if err != nil {
			if status == http.StatusNotFound {
				c.RespondWithErrorMessage(
					fmt.Sprintf("Profile with ID ('%d') does not exist.", profileId),
					http.StatusBadRequest,
				)
				return
			} else {
				c.RespondWithErrorMessage(
					fmt.Sprintf("Could not retrieve profile: %v.", err.Error()),
					http.StatusInternalServerError,
				)
				return
			}
		}

		perms := models.GetPermission(
			models.MakeAuthorisationContext(
				c, 0, h.ItemTypes[h.ItemTypeProfile], profileId),
		)
		if !perms.CanCreate && !perms.CanUpdate {
			c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
			return
		}

		attachment.ItemId = profileId
		attachment.ItemTypeId = h.ItemTypes[h.ItemTypeProfile]
		path_prefix = h.ApiTypeProfile

	} else if c.RouteVars["comment_id"] != "" {

		commentId, err := strconv.ParseInt(c.RouteVars["comment_id"], 10, 64)
		if err != nil {
			c.RespondWithErrorMessage(
				fmt.Sprintf("The supplied comment ID ('%s') is not a number.", c.RouteVars["comment_id"]),
				http.StatusBadRequest,
			)
			return
		}

		_, status, err := models.GetCommentSummary(c.Site.ID, commentId)
		if err != nil {
			if status == http.StatusNotFound {
				c.RespondWithErrorMessage(
					fmt.Sprintf("Comment with ID ('%d') does not exist.", commentId),
					http.StatusBadRequest,
				)
				return
			} else {
				c.RespondWithErrorMessage(
					fmt.Sprintf("Could not retrieve comment: %v.", err.Error()),
					http.StatusInternalServerError,
				)
				return
			}
		}

		perms := models.GetPermission(
			models.MakeAuthorisationContext(
				c, 0, h.ItemTypes[h.ItemTypeComment], commentId),
		)
		if !perms.CanCreate && !perms.CanUpdate {
			c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
			return
		}

		if metadata.FileSize > 3145728 {
			c.RespondWithErrorMessage(fmt.Sprintf("File size must be under 3 megabytes"), http.StatusBadRequest)
			return
		}

		attachment.ItemId = commentId
		attachment.ItemTypeId = h.ItemTypes[h.ItemTypeComment]
		path_prefix = h.ApiTypeComment

	} else {
		c.RespondWithErrorMessage(
			"You must supply a profile_id or comment_id as a RouteVar",
			http.StatusBadRequest,
		)
		return
	}

	// Check that this file hasn't already been attached to this item
	oldattachment := models.AttachmentType{}
	oldattachment, status, err = models.GetAttachment(
		attachment.ItemTypeId,
		attachment.ItemId,
		attachment.FileHash,
		false,
	)

	if err != nil && status != http.StatusNotFound {
		c.RespondWithErrorMessage(
			fmt.Sprintf("An error occurred when checking the attachment: %v", err.Error()),
			http.StatusInternalServerError,
		)
		return
	}

	if status != http.StatusNotFound && attachment.ItemTypeId != h.ItemTypes[h.ItemTypeProfile] {
		c.RespondWithSeeOther(
			fmt.Sprintf("%s/%d/%s", path_prefix, oldattachment.ItemId, h.ApiTypeAttachment),
		)
		return
	}

	if status == http.StatusNotFound {
		// Update attach count on attachment_meta
		metadata.AttachCount += 1
		status, err = metadata.Update()
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}

		attachment.ProfileId = c.Auth.ProfileId
		attachment.Created = time.Now()

		status, err = attachment.Insert()
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}
	} else {
		//already exists, need to update it and pull back the attachmentId
		attachment = oldattachment
		attachment.Created = time.Now()
		status, err = attachment.Update()
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}
	}

	// If attaching to a profile, update the profile with new avatar URL
	if attachment.ItemTypeId == h.ItemTypes[h.ItemTypeProfile] {
		profile, _, err := models.GetProfile(c.Site.ID, attachment.ItemId)
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}
		filePath := metadata.FileHash
		if metadata.FileExt != "" {
			filePath += `.` + metadata.FileExt
		}
		profile.AvatarUrlNullable = sql.NullString{
			String: fmt.Sprintf("%s/%s", h.ApiTypeFile, filePath),
			Valid:  true,
		}
		profile.AvatarIdNullable = sql.NullInt64{
			Int64: attachment.AttachmentId,
			Valid: true,
		}
		status, err = profile.Update()
		if err != nil {
			c.RespondWithErrorMessage(
				fmt.Sprintf("Could not update profile with avatar: %v", err.Error()),
				status,
			)
			return
		}
	}

	c.RespondWithSeeOther(
		fmt.Sprintf("%s/%d/%s", path_prefix, attachment.ItemId, h.ApiTypeAttachment),
	)
}

func (ctl *AttachmentsController) ReadMany(c *models.Context) {

	itemTypeId, itemId, perms, status, err := ParseItemInfo(c)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	if !perms.CanRead {
		c.RespondWithErrorMessage(
			h.NoAuthMessage,
			http.StatusForbidden,
		)
		return
	}

	query := c.Request.URL.Query()

	limit, offset, status, err := h.GetLimitAndOffset(query)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	attachments, total, pages, status, err := models.GetAttachments(itemTypeId, itemId, limit, offset)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	thisLink := h.GetLinkToThisPage(*c.Request.URL, offset, limit, total)

	m := models.AttachmentsType{}
	m.Attachments = h.ConstructArray(
		attachments,
		h.ApiTypeAttachment,
		total,
		limit,
		offset,
		pages,
		c.Request.URL,
	)
	m.Meta.Links =
		[]h.LinkType{
			h.LinkType{Rel: "self", Href: thisLink.String()},
		}
	m.Meta.Permissions = perms

	c.RespondWithData(m)
}
