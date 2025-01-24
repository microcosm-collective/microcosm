package controller

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/grafana/pyroscope-go"
	"github.com/microcosm-collective/microcosm/audit"
	h "github.com/microcosm-collective/microcosm/helpers"
	"github.com/microcosm-collective/microcosm/models"
)

// UsersController is a web controller
type UsersController struct{}

// UsersHandler is a web handler
func UsersHandler(w http.ResponseWriter, r *http.Request) {
	path := "/users"
	pyroscope.TagWrapper(context.Background(), pyroscope.Labels("path", path), func(ctx context.Context) {
		c, status, err := models.MakeContext(r, w)
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}
		ctl := UsersController{}

		method := c.GetHTTPMethod()
		switch method {
		case "OPTIONS":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				c.RespondWithOptions([]string{"OPTIONS", "POST"})
			})
			return
		case "POST":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				ctl.Create(c)
			})
		default:
			c.RespondWithStatus(http.StatusMethodNotAllowed)
			return
		}
	})
}

// Create handles POST
func (ctl *UsersController) Create(c *models.Context) {
	// Batch create users by email address
	ems := []models.UserType{}

	err := c.Fill(&ems)
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("The post data is invalid: %v", err.Error()),
			http.StatusBadRequest,
		)
		return
	}

	if !c.Auth.IsSiteOwner {
		c.RespondWithErrorMessage(
			"Only a site owner can batch create users",
			http.StatusForbidden,
		)
		return
	}

	for _, m := range ems {
		status, err := m.Validate(false)
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}
	}

	// We are good to go
	profiles := []models.ProfileType{}

	for _, m := range ems {
		// Retrieve user details by email address
		user, status, err := models.GetUserByEmailAddress(m.Email)
		if status == http.StatusNotFound {

			user, _, err = models.CreateUserByEmailAddress(m.Email)
			if err != nil {
				c.RespondWithErrorMessage(
					fmt.Sprintf("Couldn't create user: %v", err.Error()),
					http.StatusInternalServerError,
				)
				return
			}

			audit.Create(
				c.Site.ID,
				h.ItemTypes[h.ItemTypeUser],
				user.ID,
				c.Auth.ProfileID,
				time.Now(),
				c.IP,
			)

		} else if err != nil {
			c.RespondWithErrorMessage(
				fmt.Sprintf("Error retrieving user: %v", err.Error()),
				http.StatusInternalServerError,
			)
			return
		}

		// create a corresponding profile for this user
		profile, status, err := models.GetOrCreateProfile(c.Site, user)
		if err != nil {
			c.RespondWithErrorMessage(
				fmt.Sprintf("Failed to create profile with ID %d: %v", profile.ID, err.Error()),
				status,
			)
			return
		}

		audit.Create(
			c.Site.ID,
			h.ItemTypes[h.ItemTypeProfile],
			profile.ID,
			c.Auth.ProfileID,
			time.Now(),
			c.IP,
		)

		profiles = append(profiles, profile)
	}

	c.RespondWithData(profiles)
}
