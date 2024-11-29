package controller

import (
	"context"
	"fmt"
	"net/http"

	h "git.dee.kitchen/buro9/microcosm/helpers"
	"git.dee.kitchen/buro9/microcosm/models"
	"github.com/grafana/pyroscope-go"
)

// WhoAmIHandler is the web handler
func WhoAmIHandler(w http.ResponseWriter, r *http.Request) {
	path := "/whoami"
	pyroscope.TagWrapper(context.Background(), pyroscope.Labels("path", path), func(ctx context.Context) {

		c, status, err := models.MakeContext(r, w)
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}
		ctl := WhoAmIController{}

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

// WhoAmIController is the web controller
type WhoAmIController struct{}

func (wc *WhoAmIController) Read(c *models.Context) {
	if c.Request.Method != "GET" {
		c.RespondWithNotImplemented()
		return
	}

	if c.Auth.UserID < 0 {
		c.RespondWithErrorMessage(
			"Bad access token supplied",
			http.StatusForbidden,
		)
		return
	}

	if c.Auth.UserID == 0 {
		c.RespondWithErrorMessage(
			"You must be authenticated to ask 'who am I?'",
			http.StatusForbidden,
		)
		return
	}

	perms := models.GetPermission(
		models.MakeAuthorisationContext(
			c, 0, h.ItemTypes[h.ItemTypeProfile], c.Auth.ProfileID),
	)

	m, status, err := models.GetProfile(c.Site.ID, c.Auth.ProfileID)
	if err != nil {
		if status == http.StatusNotFound {
			c.RespondWithErrorMessage(
				"You must create a user profile for this site at api/v1/profiles/",
				http.StatusNotFound,
			)
			return
		}
		c.RespondWithErrorMessage(
			fmt.Sprintf("Could not retrieve profile: %v", err.Error()),
			http.StatusInternalServerError,
		)
		return
	}
	m.Meta.Permissions = perms

	if c.Auth.ProfileID > 0 {
		m.GetUnreadHuddleCount()

		m.Email = models.GetProfileEmail(c.Site.ID, m.ID)

		// Get member status
		attrs, _, _, status, err := models.GetAttributes(
			h.ItemTypes[h.ItemTypeProfile],
			m.ID,
			h.DefaultQueryLimit,
			h.DefaultQueryOffset,
		)
		if err != nil {
			c.RespondWithErrorDetail(err, status)
		}
		for _, attr := range attrs {
			if attr.Key == "is_member" && attr.Boolean.Valid {
				m.Member = attr.Boolean.Bool
			}
		}
	}

	c.RespondWithData(m)
}
