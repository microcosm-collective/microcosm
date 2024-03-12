package controller

import (
	"net/http"
	"net/url"
	"strconv"

	h "github.com/microcosm-cc/microcosm/helpers"
	"github.com/microcosm-cc/microcosm/models"
)

// CommentContextController is a web controller
type CommentContextController struct{}

// CommentContextHandler is a web handler
func CommentContextHandler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}
	ctl := CommentContextController{}

	method := c.GetHTTPMethod()
	switch method {
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

// Read handles GET
func (ctl *CommentContextController) Read(c *models.Context) {
	_, itemTypeID, itemID, status, err := c.GetItemTypeAndItemID()
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	// Start Authorisation
	perms := models.GetPermission(
		models.MakeAuthorisationContext(
			c, 0, itemTypeID, itemID),
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

	m, status, err := models.GetCommentSummary(c.Site.ID, itemID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	link, status, err := m.GetPageLink(limit, c.Auth.ProfileID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	pageURL, err := url.Parse(link.Href)
	if err != nil {
		c.RespondWithErrorMessage(err.Error(), http.StatusInternalServerError)
		return
	}

	queryString := pageURL.Query()
	queryString.Add("comment_id", strconv.FormatInt(m.ID, 10))
	pageURL.RawQuery = queryString.Encode()

	c.RespondWithLocation(pageURL.String())
}
