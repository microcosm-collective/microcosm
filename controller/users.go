package controller

import (
	"fmt"
	"net/http"
	"time"

	"github.com/microcosm-cc/microcosm/audit"
	h "github.com/microcosm-cc/microcosm/helpers"
	"github.com/microcosm-cc/microcosm/models"
)

type UsersController struct{}

func UsersHandler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	ctl := UsersController{}

	switch c.GetHttpMethod() {
	case "OPTIONS":
		c.RespondWithOptions([]string{"OPTIONS", "POST"})
		return
	case "POST":
		ctl.Create(c)
	default:
		c.RespondWithStatus(http.StatusMethodNotAllowed)
		return
	}
}

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

			user, status, err = models.CreateUserByEmailAddress(m.Email)
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
				c.Auth.ProfileId,
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
			c.Auth.ProfileId,
			time.Now(),
			c.IP,
		)

		profiles = append(profiles, profile)
	}

	c.RespondWithData(profiles)
}
