package controller

import (
	"net/http"

	"git.dee.kitchen/buro9/microcosm/models"
)

// SearchController is a web controller
type SearchController struct{}

// SearchHandler is a web handler
func SearchHandler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}
	ctl := SearchController{}

	method := c.GetHTTPMethod()
	switch method {
	case "OPTIONS":
		c.RespondWithOptions([]string{"OPTIONS", "HEAD", "GET"})
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
