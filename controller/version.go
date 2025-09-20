package controller

import (
	"net/http"

	"github.com/microcosm-collective/microcosm/models"
)

var (
	// BuildVersion and BuildDate are set via ldflags during build
	BuildVersion = "development"
	BuildDate    = "unknown"
)

// VersionHandler is a web handler that returns build information
func VersionHandler(w http.ResponseWriter, r *http.Request) {
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
		shortVersion := BuildVersion
		if len(BuildVersion) > 8 {
			shortVersion = BuildVersion[:8]
		}
		version := map[string]string{
			"version": shortVersion,
			"date":    BuildDate,
		}
		c.RespondWithData(version)
		return
	default:
		c.RespondWithStatus(http.StatusMethodNotAllowed)
		return
	}
}