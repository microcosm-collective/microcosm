package controller

import (
	"net/http"

	h "github.com/microcosm-cc/microcosm/helpers"
	"github.com/microcosm-cc/microcosm/models"
)

type TrendingController struct{}

func TrendingHandler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}
	ctl := TrendingController{}
	switch c.GetHttpMethod() {
	case "OPTIONS":
		c.RespondWithOptions([]string{"OPTIONS", "GET", "HEAD"})
		return
	case "GET":
		ctl.ReadMany(c)
	case "HEAD":
		ctl.ReadMany(c)
	default:
		c.RespondWithStatus(http.StatusMethodNotAllowed)
		return
	}
}

func (ctl *TrendingController) ReadMany(c *models.Context) {

	limit, offset, status, err := h.GetLimitAndOffset(c.Request.URL.Query())
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	trending, total, pages, status, err := models.GetTrending(c.Site.Id, c.Auth.ProfileId, limit, offset)
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
			h.LinkType{Rel: "self", Href: thisLink.String()},
		}

	c.ResponseWriter.Header().Set("Cache-Control", "no-cache, max-age=0")
	c.RespondWithData(response)
}
