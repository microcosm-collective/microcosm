package controller

import (
	"context"
	"fmt"
	"net/http"

	"github.com/grafana/pyroscope-go"
	"github.com/microcosm-collective/microcosm/models"
	"github.com/microcosm-collective/microcosm/redirector"
)

// RedirectHandler is a web handler
func RedirectHandler(w http.ResponseWriter, r *http.Request) {
	path := "/out/{id}"
	pyroscope.TagWrapper(context.Background(), pyroscope.Labels("path", path), func(ctx context.Context) {
		c, status, err := models.MakeEmptyContext(r, w)
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}
		ctl := RedirectController{}

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

// RedirectController is a web controller
type RedirectController struct{}

// Read handles GET
func (ctl *RedirectController) Read(c *models.Context) {
	redirect, status, err := redirector.GetRedirect(c.RouteVars["short_url"])
	if err != nil {
		if status == http.StatusNotFound {
			c.RespondWithErrorMessage(
				fmt.Sprintf("%v", err.Error()),
				http.StatusNotFound,
			)
			return
		}

		c.RespondWithErrorMessage(
			fmt.Sprintf("Could not retrieve redirect: %v", err.Error()),
			http.StatusInternalServerError,
		)
		return
	}

	c.RespondWithLocation(redirect.URL)
}
