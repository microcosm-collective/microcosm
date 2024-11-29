package controller

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"git.dee.kitchen/buro9/microcosm/audit"
	h "git.dee.kitchen/buro9/microcosm/helpers"
	"git.dee.kitchen/buro9/microcosm/models"
	"github.com/grafana/pyroscope-go"
)

// NOTE: Access tokens are created in auth0.go
// This file really only handles the deletion of existing access_tokens

// AuthHandler is a web handler
func AuthHandler(w http.ResponseWriter, r *http.Request) {
	path := "/auth"
	pyroscope.TagWrapper(context.Background(), pyroscope.Labels("path", path), func(ctx context.Context) {
		c, status, err := models.MakeContext(r, w)
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}

		method := c.GetHTTPMethod()
		switch method {
		case "OPTIONS":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				c.RespondWithOptions([]string{"OPTIONS"})
			})
			return
		default:
			c.RespondWithStatus(http.StatusMethodNotAllowed)
			return
		}
	})
}

// AuthAccessTokenController is a web controller
type AuthAccessTokenController struct{}

// AuthAccessTokenHandler is a web handler
func AuthAccessTokenHandler(w http.ResponseWriter, r *http.Request) {
	path := "/auth/{id}"
	pyroscope.TagWrapper(context.Background(), pyroscope.Labels("path", path), func(ctx context.Context) {
		c, status, err := models.MakeContext(r, w)
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}
		ctl := AuthAccessTokenController{}

		method := c.GetHTTPMethod()
		switch method {
		case "OPTIONS":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				c.RespondWithOptions([]string{"OPTIONS", "HEAD", "GET", "DELETE"})
			})
			return
		case "HEAD":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				ctl.Read(c)
			})
		case "GET":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				ctl.Read(c)
			})
		case "DELETE":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				ctl.Delete(c)
			})
		default:
			c.RespondWithStatus(http.StatusMethodNotAllowed)
			return
		}
	})
}

// Read handles GET
func (ctl *AuthAccessTokenController) Read(c *models.Context) {
	// Extract access token from request and retrieve its metadata
	m, status, err := models.GetAccessToken(c.RouteVars["access_token"])
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("Error retrieving access token: %v", err.Error()),
			status,
		)
		return
	}
	c.RespondWithData(m)
}

// Delete handles DELETE
func (ctl *AuthAccessTokenController) Delete(c *models.Context) {
	auth_access_token := c.Auth.AccessToken.TokenValue
	path_access_token := c.RouteVars["access_token"]

	if auth_access_token == `` {
		c.RespondWithErrorMessage(
			`?access_token=${access_token} expected in query string as the access_token that authenticates the current request`,
			http.StatusBadRequest,
		)
		return
	}

	if path_access_token == `` {
		c.RespondWithErrorMessage(
			`/api/v1/auth/${access_token} expected in the URI as the access_token to delete`,
			http.StatusBadRequest,
		)
		return
	}

	if !strings.EqualFold(auth_access_token, path_access_token) {
		c.RespondWithErrorMessage(
			`/api/v1/auth/${access_token} and ?access_token=${access_token} must match as you can only delete the access_token for the currently authenticated session`,
			http.StatusBadRequest,
		)
		return
	}

	// Extract access token from request and delete its record
	m, status, err := models.GetAccessToken(path_access_token)
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("error retrieving access token: %v", err.Error()),
			status,
		)
		return
	}

	status, err = m.Delete()
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("error deleting access token: %v", err.Error()),
			status,
		)
		return
	}

	audit.Delete(
		c.Site.ID,
		h.ItemTypes[h.ItemTypeAuth],
		m.UserID,
		c.Auth.ProfileID,
		time.Now(),
		c.IP,
	)

	c.RespondWithOK()
}
