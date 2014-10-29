package controller

import (
	"fmt"
	"net/http"

	h "github.com/microcosm-cc/microcosm/helpers"
	"github.com/microcosm-cc/microcosm/models"
)

func LegalsHandler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	switch c.GetHttpMethod() {
	case "OPTIONS":
		c.RespondWithOptions([]string{"OPTIONS", "GET"})
		return
	case "GET":
		if c.IsRootSite() {
			// Root site
			c.RespondWithData(
				h.LinkArrayType{Links: []h.LinkType{
					h.LinkType{Rel: "api", Href: "/api/v1/legal/service"},
				}},
			)
			return
		} else {
			// A customer site
			c.RespondWithData(
				h.LinkArrayType{Links: []h.LinkType{
					h.LinkType{Rel: "cookies", Href: "/api/v1/legal/cookies"},
					h.LinkType{Rel: "privacy", Href: "/api/v1/legal/privacy"},
					h.LinkType{Rel: "terms", Href: "/api/v1/legal/terms"},
				}},
			)
			return
		}
	default:
		c.RespondWithStatus(http.StatusMethodNotAllowed)
		return
	}
}

func LegalHandler(w http.ResponseWriter, r *http.Request) {

	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	ctl := LegalController{}

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

type LegalController struct{}

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
