package controller

import (
	"context"
	"net/http"

	"git.dee.kitchen/buro9/microcosm/models"
	"github.com/grafana/pyroscope-go"
)

// SearchController is a web controller
type SearchController struct{}

// SearchHandler is a web handler
func SearchHandler(w http.ResponseWriter, r *http.Request) {
	path := "/search"
	pyroscope.TagWrapper(context.Background(), pyroscope.Labels("path", path), func(ctx context.Context) {

		c, status, err := models.MakeContext(r, w)
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}
		ctl := SearchController{}

		method := c.GetHTTPMethod()
		switch method {
		case "OPTIONS":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				c.RespondWithOptions([]string{"OPTIONS", "HEAD", "GET"})
			})
			return
		case "GET":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				ctl.Read(c)
			})
		case "HEAD":
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
func (ctl *SearchController) Read(c *models.Context) {
	results, status, err := models.Search(
		c.Site.ID,
		*c.Request.URL,
		c.Auth.ProfileID,
	)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	c.RespondWithData(results)
}
