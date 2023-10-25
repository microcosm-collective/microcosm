package controller

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/grafana/pyroscope-go"
	h "github.com/microcosm-cc/microcosm/helpers"
	"github.com/microcosm-cc/microcosm/models"
)

// AttachmentHandler is a web handler
func AttachmentHandler(w http.ResponseWriter, r *http.Request) {
	path := "/*/attachments/{fileHash}"
	pyroscope.TagWrapper(context.Background(), pyroscope.Labels("path", path), func(context.Context) {
		c, status, err := models.MakeContext(r, w)
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}

		ctl := AttachmentController{}

		method := c.GetHTTPMethod()
		switch method {
		case "OPTIONS":
			pyroscope.TagWrapper(context.Background(), pyroscope.Labels("method", method), func(context.Context) {
				c.RespondWithOptions([]string{"OPTIONS", "DELETE"})
			})
			return
		case "DELETE":
			pyroscope.TagWrapper(context.Background(), pyroscope.Labels("method", method), func(context.Context) {
				ctl.Delete(c)
			})
		default:
			c.RespondWithStatus(http.StatusMethodNotAllowed)
			return
		}
	})
}

// AttachmentController is a web controller
type AttachmentController struct{}

// Delete handles DELETE
func (ctl *AttachmentController) Delete(c *models.Context) {
	itemTypeID, itemID, perms, status, err := ParseItemInfo(c)
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
				"File does not have a metadata record",
				http.StatusBadRequest,
			)
			return
		}

		c.RespondWithErrorMessage(
			fmt.Sprintf("Could not retrieve metadata: %v", err.Error()),
			http.StatusBadRequest,
		)
		return
	}

	status, err = models.DeleteAttachment(itemTypeID, itemID, fileHash)
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

// ParseItemInfo determines what this attachment is attached to
func ParseItemInfo(c *models.Context) (int64, int64, models.PermissionType, int, error) {
	var itemTypeID int64
	var itemID int64

	if c.RouteVars["profile_id"] != "" {
		profileID, err := strconv.ParseInt(c.RouteVars["profile_id"], 10, 64)
		if err != nil {
			return 0, 0, models.PermissionType{}, http.StatusBadRequest,
				fmt.Errorf(
					"the supplied profile ID ('%s') is not a number",
					c.RouteVars["profile_id"],
				)
		}
		_, status, err := models.GetProfileSummary(c.Site.ID, profileID)
		if err != nil {
			if status == http.StatusNotFound {
				return 0, 0, models.PermissionType{}, http.StatusBadRequest,
					fmt.Errorf(
						"profile with ID ('%d') does not exist", profileID,
					)
			}

			return 0, 0, models.PermissionType{}, http.StatusBadRequest, err
		}

		itemID = profileID
		itemTypeID = h.ItemTypes[h.ItemTypeProfile]

	} else if c.RouteVars["comment_id"] != "" {

		commentID, err := strconv.ParseInt(c.RouteVars["comment_id"], 10, 64)
		if err != nil {
			return 0, 0, models.PermissionType{}, http.StatusBadRequest,
				fmt.Errorf(
					"the supplied comment ID ('%s') is not a number",
					c.RouteVars["comment_id"],
				)
		}
		_, status, err := models.GetCommentSummary(c.Site.ID, commentID)
		if err != nil {
			if status == http.StatusNotFound {
				return 0, 0, models.PermissionType{}, http.StatusBadRequest,
					fmt.Errorf(
						"comment with ID ('%d') does not exist", commentID,
					)
			}

			return 0, 0, models.PermissionType{}, http.StatusBadRequest, err
		}

		itemID = commentID
		itemTypeID = h.ItemTypes[h.ItemTypeComment]

	} else {
		return 0, 0, models.PermissionType{}, http.StatusBadRequest,
			fmt.Errorf("you must supply a profile_id or comment_id as a RouteVar")
	}

	perms := models.GetPermission(
		models.MakeAuthorisationContext(
			c, 0, itemTypeID, itemID),
	)

	return itemTypeID, itemID, perms, http.StatusOK, nil
}
