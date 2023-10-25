package controller

import (
	"context"
	"net/http"
	"strconv"

	"github.com/grafana/pyroscope-go"
	"github.com/microcosm-cc/microcosm/models"
)

// SiteReservedController is a web controller
type SiteReservedController struct{}

// SiteReservedHandler is a web handler
func SiteReservedHandler(w http.ResponseWriter, r *http.Request) {
	path := "/reserved/{id}"
	pyroscope.TagWrapper(context.Background(), pyroscope.Labels("path", path), func(context.Context) {
		c, status, err := models.MakeContext(r, w)
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}
		ctl := SiteReservedController{}

		method := c.GetHTTPMethod()
		switch method {
		case "OPTIONS":
			pyroscope.TagWrapper(context.Background(), pyroscope.Labels("method", method), func(context.Context) {
				c.RespondWithOptions([]string{"OPTIONS", "HEAD", "GET"})
			})
			return
		case "GET":
			pyroscope.TagWrapper(context.Background(), pyroscope.Labels("method", method), func(context.Context) {
				ctl.Read(c)
			})
		case "HEAD":
			pyroscope.TagWrapper(context.Background(), pyroscope.Labels("method", method), func(context.Context) {
				ctl.Read(c)
			})
		default:
			c.RespondWithStatus(http.StatusMethodNotAllowed)
			return
		}
	})
}

// Read handles GET
func (ctl *SiteReservedController) Read(c *models.Context) {
	host, exists := c.RouteVars["subdomain"]
	if !exists {
		c.RespondWithErrorMessage("No subdomain query specified", http.StatusBadRequest)
		return
	}

	reserved, err := models.IsReservedSubdomain(host)
	if err != nil {
		c.RespondWithErrorMessage(err.Error(), http.StatusInternalServerError)
		return
	}

	var responseBody string
	if reserved {
		responseBody = `{"reserved":true}`
		c.ResponseWriter.Header().Set("Content-Length", strconv.Itoa(len(responseBody)))
	} else {
		responseBody = `{"reserved":false}`
		c.ResponseWriter.Header().Set("Content-Length", strconv.Itoa(len(responseBody)))
	}

	c.ResponseWriter.Header().Set("Content-Type", "application/json")
	c.ResponseWriter.Header().Set("Access-Control-Allow-Origin", "*")
	c.ResponseWriter.WriteHeader(http.StatusOK)
	c.ResponseWriter.Write([]byte(responseBody))
}
