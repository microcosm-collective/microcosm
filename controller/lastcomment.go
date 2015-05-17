package controller

import (
	"fmt"
	"net/http"
	"strconv"

	h "github.com/microcosm-cc/microcosm/helpers"
	"github.com/microcosm-cc/microcosm/models"
)

type LastCommentController struct{}

func LastCommentHandler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	ctl := LastCommentController{}

	switch c.GetHttpMethod() {
	case "OPTIONS":
		c.RespondWithOptions([]string{"OPTIONS", "GET"})
		return
	case "GET":
		ctl.Read(c)
	default:
		c.RespondWithStatus(http.StatusMethodNotAllowed)
		return
	}
}

func (ctl *LastCommentController) Read(c *models.Context) {

	itemType, itemTypeId, itemId, status, err := c.GetItemTypeAndItemId()
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	perms := models.GetPermission(models.MakeAuthorisationContext(c, 0, itemTypeId, itemId))
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

	lastComment, status, err := models.GetLastComment(itemTypeId, itemId)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	location := fmt.Sprintf(
		"%s/%d",
		h.ItemTypesToApiItem[itemType],
		itemId,
	)
	parsed.Path = location

	// Construct location of the last comment on the item.
	if lastComment.Valid {
		_, _, offset, _, err := models.GetPageNumber(
			lastComment.ID,
			limit,
			c.Auth.ProfileId,
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
