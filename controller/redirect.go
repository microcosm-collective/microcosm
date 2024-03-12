package controller

import (
	"fmt"
	"net/http"

	"github.com/microcosm-cc/microcosm/models"
	"github.com/microcosm-cc/microcosm/redirector"
)

// RedirectHandler is a web handler
func RedirectHandler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeEmptyContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}
	ctl := RedirectController{}

	method := c.GetHTTPMethod()
	switch method {
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
