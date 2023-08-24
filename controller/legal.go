package controller

import (
	"fmt"
	"net/http"

	h "github.com/microcosm-cc/microcosm/helpers"
	"github.com/microcosm-cc/microcosm/models"
)

// LegalsHandler is a web handler
func LegalsHandler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	switch c.GetHTTPMethod() {
	case "OPTIONS":
		c.RespondWithOptions([]string{"OPTIONS", "GET"})
		return
	case "GET":
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
		return
	default:
		c.RespondWithStatus(http.StatusMethodNotAllowed)
		return
	}
}

// LegalHandler is a web handler
func LegalHandler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	ctl := LegalController{}

	switch c.GetHTTPMethod() {
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
	return
}
