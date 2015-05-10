package controller

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	h "github.com/microcosm-cc/microcosm/helpers"
	"github.com/microcosm-cc/microcosm/models"
)

type NewCommentController struct{}

func NewCommentHandler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	ctl := NewCommentController{}

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

func (ctl *NewCommentController) Read(c *models.Context) {

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

	query := c.Request.URL.Query()
	limit, _, status, err := h.GetLimitAndOffset(query)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	offset, commentId, _, err := models.GetLatestComments(c.Site.ID, itemType, itemId, c.Auth.ProfileId, limit)
	if err != nil {
		//Go to to the first page
		parsed, _ := url.Parse(c.Request.URL.String())
		values := parsed.Query()

		values.Del("offset")

		if values.Get("limit") == fmt.Sprintf("%d", h.DefaultQueryLimit) {
			values.Del("limit")
		}

		location := fmt.Sprintf(
			"%s/%d",
			h.ItemTypesToApiItem[itemType],
			itemId,
		)
		parsed.RawQuery = values.Encode()
		parsed.Path = location
		c.RespondWithLocation(parsed.String())
		return
	}

	//construct redirect
	parsed, _ := url.Parse(c.Request.URL.String())
	values := parsed.Query()

	values.Del("offset")
	if offset != h.DefaultQueryOffset {
		values.Set("offset", strconv.FormatInt(offset, 10))
	}

	values.Del("limit")
	if limit != h.DefaultQueryLimit {
		values.Set("limit", strconv.FormatInt(limit, 10))
	}

	values.Del("comment_id")
	values.Set("comment_id", strconv.FormatInt(commentId, 10))

	location := fmt.Sprintf(
		"%s/%d",
		h.ItemTypesToApiItem[itemType],
		itemId,
	)
	parsed.RawQuery = values.Encode()
	parsed.Path = location

	c.RespondWithLocation(parsed.String())
}
