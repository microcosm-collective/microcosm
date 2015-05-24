package controller

import (
	"net/http"
	"net/url"
	"strconv"

	h "github.com/microcosm-cc/microcosm/helpers"
	"github.com/microcosm-cc/microcosm/models"
)

type CommentContextController struct{}

func CommentContextHandler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	ctl := CommentContextController{}

	switch c.GetHTTPMethod() {
	case "OPTIONS":
		c.RespondWithOptions([]string{"OPTIONS", "GET", "HEAD"})
		return
	case "GET":
		ctl.Read(c)
	case "HEAD":
		ctl.Read(c)
	default:
		c.RespondWithStatus(http.StatusMethodNotAllowed)
		return
	}
}

// Redirects to the comment within the context of the thing it is attached to
func (ctl *CommentContextController) Read(c *models.Context) {
	_, itemTypeId, itemId, status, err := c.GetItemTypeAndItemID()
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	// Start Authorisation
	perms := models.GetPermission(
		models.MakeAuthorisationContext(
			c, 0, itemTypeId, itemId),
	)
	if !perms.CanRead {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}
	// End Authorisation

	limit, _, status, err := h.GetLimitAndOffset(c.Request.URL.Query())
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	m, status, err := models.GetCommentSummary(c.Site.ID, itemId)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	link, status, err := m.GetPageLink(limit, c.Auth.ProfileID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	pageUrl, err := url.Parse(link.Href)
	if err != nil {
		c.RespondWithErrorMessage(err.Error(), http.StatusInternalServerError)
		return
	}

	queryString := pageUrl.Query()
	queryString.Add("comment_id", strconv.FormatInt(m.Id, 10))
	pageUrl.RawQuery = queryString.Encode()

	c.RespondWithLocation(pageUrl.String())
}
