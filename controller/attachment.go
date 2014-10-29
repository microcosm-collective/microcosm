package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	h "github.com/microcosm-cc/microcosm/helpers"
	"github.com/microcosm-cc/microcosm/models"
)

func AttachmentHandler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	ctl := AttachmentController{}

	switch c.GetHttpMethod() {
	case "OPTIONS":
		c.RespondWithOptions([]string{"OPTIONS", "DELETE"})
		return
	case "DELETE":
		ctl.Delete(c)
	default:
		c.RespondWithStatus(http.StatusMethodNotAllowed)
		return
	}
}

type AttachmentController struct{}

func (ctl *AttachmentController) Delete(c *models.Context) {

	itemTypeId, itemId, perms, status, err := ParseItemInfo(c)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	if !perms.IsSiteOwner && !perms.IsModerator && perms.IsOwner {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}

	fileHash := c.RouteVars["fileHash"]
	if fileHash == "" {
		c.RespondWithErrorMessage(
			fmt.Sprintf("The supplied file hash cannot be zero characters: %s", c.RouteVars["fileHash"]),
			http.StatusBadRequest,
		)
		return
	}

	metadata, status, err := models.GetMetadata(fileHash)
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

	status, err = models.DeleteAttachment(itemTypeId, itemId, fileHash)
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("Could not remove attachment: %v", err.Error()),
			status,
		)
		return
	}

	// Update attach count on attachment_meta
	metadata.AttachCount = metadata.AttachCount - 1
	status, err = metadata.Update()
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	c.RespondWithOK()
}

func ParseItemInfo(c *models.Context) (int64, int64, models.PermissionType, int, error) {

	var itemTypeId int64
	var itemId int64

	if c.RouteVars["profile_id"] != "" {

		profileId, err := strconv.ParseInt(c.RouteVars["profile_id"], 10, 64)
		if err != nil {
			return 0, 0, models.PermissionType{}, http.StatusBadRequest,
				errors.New(fmt.Sprintf(
					"The supplied profile ID ('%s') is not a number.",
					c.RouteVars["profile_id"],
				))
		}
		_, status, err := models.GetProfileSummary(c.Site.Id, profileId)
		if err != nil {
			if status == http.StatusNotFound {
				return 0, 0, models.PermissionType{}, http.StatusBadRequest,
					errors.New(fmt.Sprintf(
						"Profile with ID ('%d') does not exist.", profileId,
					))
			} else {
				return 0, 0, models.PermissionType{}, http.StatusBadRequest, err
			}
		}

		itemId = profileId
		itemTypeId = h.ItemTypes[h.ItemTypeProfile]

	} else if c.RouteVars["comment_id"] != "" {

		commentId, err := strconv.ParseInt(c.RouteVars["comment_id"], 10, 64)
		if err != nil {
			return 0, 0, models.PermissionType{}, http.StatusBadRequest,
				errors.New(fmt.Sprintf(
					"The supplied comment ID ('%s') is not a number.",
					c.RouteVars["comment_id"],
				))
		}
		_, status, err := models.GetCommentSummary(c.Site.Id, commentId)
		if err != nil {
			if status == http.StatusNotFound {
				return 0, 0, models.PermissionType{}, http.StatusBadRequest,
					errors.New(fmt.Sprintf(
						"Comment with ID ('%d') does not exist.", commentId,
					))
			} else {
				return 0, 0, models.PermissionType{}, http.StatusBadRequest, err
			}
		}

		itemId = commentId
		itemTypeId = h.ItemTypes[h.ItemTypeComment]

	} else {
		return 0, 0, models.PermissionType{}, http.StatusBadRequest,
			errors.New("You must supply a profile_id or comment_id as a RouteVar")
	}

	perms := models.GetPermission(
		models.MakeAuthorisationContext(
			c, 0, itemTypeId, itemId),
	)

	return itemTypeId, itemId, perms, http.StatusOK, nil
}
