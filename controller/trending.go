package controller

import (
	"context"
	"net/http"

	"github.com/grafana/pyroscope-go"
	h "github.com/microcosm-cc/microcosm/helpers"
	"github.com/microcosm-cc/microcosm/models"
)

// TrendingController is a web controller
type TrendingController struct{}

// TrendingHandler is a web handler
func TrendingHandler(w http.ResponseWriter, r *http.Request) {
	path := "/trending"
	pyroscope.TagWrapper(context.Background(), pyroscope.Labels("path", path), func(context.Context) {
		c, status, err := models.MakeContext(r, w)
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}
		ctl := TrendingController{}

		method := c.GetHTTPMethod()
		switch method {
		case "OPTIONS":
			pyroscope.TagWrapper(context.Background(), pyroscope.Labels("method", method), func(context.Context) {
				c.RespondWithOptions([]string{"OPTIONS", "GET", "HEAD"})
			})
			return
		case "GET":
			pyroscope.TagWrapper(context.Background(), pyroscope.Labels("method", method), func(context.Context) {
				ctl.ReadMany(c)
			})
		case "HEAD":
			pyroscope.TagWrapper(context.Background(), pyroscope.Labels("method", method), func(context.Context) {
				ctl.ReadMany(c)
			})
		default:
			c.RespondWithStatus(http.StatusMethodNotAllowed)
			return
		}
	})
}

// ReadMany handles GET for the collection
func (ctl *TrendingController) ReadMany(c *models.Context) {
	limit, offset, status, err := h.GetLimitAndOffset(c.Request.URL.Query())
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	trending, total, pages, status, err := models.GetTrending(c.Site.ID, c.Auth.ProfileID, limit, offset)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	response := models.TrendingItems{}
	response.Items = h.ConstructArray(
		trending,
		"items",
		total,
		limit,
		offset,
		pages,
		c.Request.URL,
	)

	thisLink := h.GetLinkToThisPage(*c.Request.URL, offset, limit, total)
	response.Meta.Links =
		[]h.LinkType{
			{Rel: "self", Href: thisLink.String()},
		}

	c.RespondWithData(response)
}
