package controller

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	h "git.dee.kitchen/buro9/microcosm/helpers"
	"git.dee.kitchen/buro9/microcosm/models"
	"github.com/grafana/pyroscope-go"
)

// LastCommentController is a web controller
type LastCommentController struct{}

// LastCommentHandler is a web handler
func LastCommentHandler(w http.ResponseWriter, r *http.Request) {
	path := "/*/lastcomment"
	pyroscope.TagWrapper(context.Background(), pyroscope.Labels("path", path), func(ctx context.Context) {
		c, status, err := models.MakeContext(r, w)
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}
		ctl := LastCommentController{}

		method := c.GetHTTPMethod()
		switch method {
		case "OPTIONS":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				c.RespondWithOptions([]string{"OPTIONS", "GET"})
			})
			return
		case "GET":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				ctl.Read(c)
			})
		default:
			c.RespondWithStatus(http.StatusMethodNotAllowed)
			return
		}
	})
}

// Read handles GET
func (ctl *LastCommentController) Read(c *models.Context) {
	itemType, itemTypeID, itemID, status, err := c.GetItemTypeAndItemID()
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	perms := models.GetPermission(models.MakeAuthorisationContext(c, 0, itemTypeID, itemID))
	if !perms.CanRead {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}

	parsed := c.Request.URL
	query := parsed.Query()
	limit, _, status, err := h.GetLimitAndOffset(query)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	query.Del("limit")
	if limit != h.DefaultQueryLimit {
		query.Set("limit", strconv.FormatInt(limit, 10))
	}

	lastComment, status, err := models.GetLastComment(itemTypeID, itemID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	location := fmt.Sprintf(
		"%s/%d",
		h.ItemTypesToAPIItem[itemType],
		itemID,
	)
	parsed.Path = location

	// Construct location of the last comment on the item.
	if lastComment.Valid {
		_, _, offset, _, err := models.GetPageNumber(
			lastComment.ID,
			limit,
			c.Auth.ProfileID,
		)
		if err != nil {
			query.Del("offset")
			if offset != h.DefaultQueryOffset {
				query.Set("offset", strconv.FormatInt(offset, 10))
			}
			query.Del("comment_id")
			query.Set("comment_id", strconv.FormatInt(lastComment.ID, 10))

			parsed.RawQuery = query.Encode()
			c.RespondWithLocation(parsed.String())
			return
		}
	}

	parsed.RawQuery = query.Encode()
	c.RespondWithLocation(parsed.String())
}
