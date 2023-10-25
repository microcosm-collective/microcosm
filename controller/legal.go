package controller

import (
	"context"
	"fmt"
	"net/http"

	"github.com/grafana/pyroscope-go"
	h "github.com/microcosm-cc/microcosm/helpers"
	"github.com/microcosm-cc/microcosm/models"
)

// LegalsHandler is a web handler
func LegalsHandler(w http.ResponseWriter, r *http.Request) {
	path := "/legal"
	pyroscope.TagWrapper(context.Background(), pyroscope.Labels("path", path), func(context.Context) {
		c, status, err := models.MakeContext(r, w)
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}

		method := c.GetHTTPMethod()
		switch method {
		case "OPTIONS":
			pyroscope.TagWrapper(context.Background(), pyroscope.Labels("method", method), func(context.Context) {
				c.RespondWithOptions([]string{"OPTIONS", "GET"})
			})
			return
		case "GET":
			pyroscope.TagWrapper(context.Background(), pyroscope.Labels("method", method), func(context.Context) {
				if c.IsRootSite() {
					// Root site
					c.RespondWithData(
						h.LinkArrayType{Links: []h.LinkType{
							{Rel: "api", Href: "/api/v1/legal/service"},
						}},
					)
					return
				}

				// A customer site
				c.RespondWithData(
					h.LinkArrayType{Links: []h.LinkType{
						{Rel: "cookies", Href: "/api/v1/legal/cookies"},
						{Rel: "privacy", Href: "/api/v1/legal/privacy"},
						{Rel: "terms", Href: "/api/v1/legal/terms"},
					}},
				)
			})
			return
		default:
			c.RespondWithStatus(http.StatusMethodNotAllowed)
			return
		}
	})
}

// LegalHandler is a web handler
func LegalHandler(w http.ResponseWriter, r *http.Request) {
	path := "/legal/{id}"
	pyroscope.TagWrapper(context.Background(), pyroscope.Labels("path", path), func(context.Context) {
		c, status, err := models.MakeContext(r, w)
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}

		ctl := LegalController{}

		method := c.GetHTTPMethod()
		switch method {
		case "OPTIONS":
			pyroscope.TagWrapper(context.Background(), pyroscope.Labels("method", method), func(context.Context) {
				c.RespondWithOptions([]string{"OPTIONS", "GET"})
			})
		case "GET":
			pyroscope.TagWrapper(context.Background(), pyroscope.Labels("method", method), func(context.Context) {
				ctl.Read(c)
			})
		default:
			c.RespondWithStatus(http.StatusMethodNotAllowed)
			return
		}
	})
}

// LegalController is a web controller
type LegalController struct{}

// Read handles GET
func (ctl *LegalController) Read(c *models.Context) {
	document := c.RouteVars["document"]
	if document == "" {
		c.RespondWithErrorMessage(
			fmt.Sprintf("The legal document does not exist: %s", c.RouteVars["document"]),
			http.StatusBadRequest,
		)
		return
	}

	m, status, err := models.GetLegalDocument(c.Site, document)
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("Could not retrieve legal document: %v", err.Error()),
			status,
		)
		return
	}

	c.RespondWithData(m)
}
