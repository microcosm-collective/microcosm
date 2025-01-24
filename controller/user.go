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

// UserController is a web controller
type UserController struct{}

// UserHandler is a web handler
func UserHandler(w http.ResponseWriter, r *http.Request) {
	path := "/users/{id}"
	pyroscope.TagWrapper(context.Background(), pyroscope.Labels("path", path), func(ctx context.Context) {
		c, status, err := models.MakeContext(r, w)
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}
		ctl := UserController{}

		method := c.GetHTTPMethod()
		switch method {
		case "OPTIONS":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				c.RespondWithOptions([]string{"OPTIONS", "HEAD", "GET", "PUT", "DELETE"})
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
		case "PUT":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				ctl.Update(c)
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
func (ctl *UserController) Read(c *models.Context) {
	_, itemTypeID, itemID, status, err := c.GetItemTypeAndItemID()
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	// Start Authorisation
	perms := models.GetPermission(
		models.MakeAuthorisationContext(
			c, 0, itemTypeID, itemID),
	)
	if !perms.CanRead {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}
	// End Authorisation

	m, status, err := models.GetUser(itemID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	if !models.UserIsOnSite(m.ID, c.Site.ID) {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}

	c.RespondWithData(m)
}

// Update handles PUT
func (ctl *UserController) Update(c *models.Context) {
	_, _, itemID, status, err := c.GetItemTypeAndItemID()
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	m, status, err := models.GetUser(itemID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	if !models.UserIsOnSite(m.ID, c.Site.ID) {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}

	err = c.Fill(&m)
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("The post data is invalid: %v", err.Error()),
			http.StatusBadRequest,
		)
		return
	}

	// Start Authorisation
	perms := models.GetPermission(
		models.MakeAuthorisationContext(
			c, 0, h.ItemTypes[h.ItemTypeProfile], itemID),
	)
	if !perms.CanUpdate {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}
	// End Authorisation

	status, err = m.Update()
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	audit.Replace(
		c.Site.ID,
		h.ItemTypes[h.ItemTypeProfile],
		itemID,
		c.Auth.ProfileID,
		time.Now(),
		c.IP,
	)

	c.RespondWithSeeOther(
		fmt.Sprintf(
			"%s/%d",
			h.APITypeUser,
			m.ID,
		),
	)
}

// Delete handles DELETE
func (ctl *UserController) Delete(c *models.Context) {
	_, _, itemID, status, err := c.GetItemTypeAndItemID()
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	m, status, err := models.GetUser(itemID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	if !models.UserIsOnSite(m.ID, c.Site.ID) {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}

	status, err = m.Delete()
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	audit.Delete(
		c.Site.ID,
		h.ItemTypes[h.ItemTypeUser],
		itemID,
		c.Auth.ProfileID,
		time.Now(),
		c.IP,
	)

	c.RespondWithOK()
}
