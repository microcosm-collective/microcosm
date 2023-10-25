package controller

import (
	"context"
	"net/http"

	"github.com/grafana/pyroscope-go"
	"github.com/microcosm-cc/microcosm/models"
)

// MicrocosmsTreeHandler is a web handler
func MicrocosmsTreeHandler(w http.ResponseWriter, r *http.Request) {
	path := "/microcosms/tree"
	pyroscope.TagWrapper(context.Background(), pyroscope.Labels("path", path), func(context.Context) {
		c, status, err := models.MakeContext(r, w)
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}

		ctl := MicrocosmsTreeController{}

		method := c.GetHTTPMethod()
		switch method {
		case "OPTIONS":
			pyroscope.TagWrapper(context.Background(), pyroscope.Labels("method", method), func(context.Context) {
				c.RespondWithOptions([]string{"OPTIONS", "POST", "HEAD", "GET"})
			})
			return
		case "HEAD":
			pyroscope.TagWrapper(context.Background(), pyroscope.Labels("method", method), func(context.Context) {
				ctl.ReadMany(c)
			})
		case "GET":
			pyroscope.TagWrapper(context.Background(), pyroscope.Labels("method", method), func(context.Context) {
				ctl.ReadMany(c)
			})
		default:
			c.RespondWithStatus(http.StatusMethodNotAllowed)
			return
		}
	})
}

// MicrocosmsTreeController is a web controller
type MicrocosmsTreeController struct{}

// ReadMany handles GET
func (ctl *MicrocosmsTreeController) ReadMany(c *models.Context) {
	// Get Microcosm Tree
	m, status, err := models.GetMicrocosmTree(
		c.Site.ID,
		c.Auth.ProfileID,
	)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	c.RespondWithData(m)
}
