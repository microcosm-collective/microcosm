package controller

import (
	"context"
	"fmt"
	"net/http"

	"github.com/grafana/pyroscope-go"
	h "github.com/microcosm-collective/microcosm/helpers"
	"github.com/microcosm-collective/microcosm/models"
)

// UpdateOptionHandler is a web handler
func UpdateOptionHandler(w http.ResponseWriter, r *http.Request) {
	path := "/updates/preferences/{id}"
	pyroscope.TagWrapper(context.Background(), pyroscope.Labels("path", path), func(ctx context.Context) {
		c, status, err := models.MakeContext(r, w)
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}
		ctl := UpdateOptionController{}

		method := c.GetHTTPMethod()
		switch method {
		case "OPTIONS":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				c.RespondWithOptions([]string{"OPTIONS", "HEAD", "GET", "PUT"})
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
		default:
			c.RespondWithStatus(http.StatusMethodNotAllowed)
			return
		}
	})
}

// UpdateOptionController is a web controller
type UpdateOptionController struct{}

// Read handles GET
func (ctl *UpdateOptionController) Read(c *models.Context) {

	if c.Auth.ProfileID < 1 {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}

	_, _, itemID, status, err := c.GetItemTypeAndItemID()
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	m, status, err := models.GetUpdateOptionByUpdateType(c.Auth.ProfileID, itemID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	c.RespondWithData(m)
}

// Update handles PUT
func (ctl *UpdateOptionController) Update(c *models.Context) {
	_, _, itemID, status, err := c.GetItemTypeAndItemID()
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	var exists bool

	m, status, err := models.GetUpdateOptionByUpdateType(c.Auth.ProfileID, itemID)
	if err != nil && status != http.StatusNotFound {
		c.RespondWithErrorDetail(err, status)
		return
	}
	if status == http.StatusOK {
		exists = true
	}

	err = c.Fill(&m)
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("The post data is invalid: %v", err.Error()),
			http.StatusBadRequest,
		)
		return
	}

	// Profile ID cannot be changed
	m.UpdateTypeID = itemID
	m.ProfileID = c.Auth.ProfileID

	if exists {
		// Update
		status, err = m.Update()
	} else {
		// Create
		status, err = m.Insert()
	}
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	// Respond
	c.RespondWithSeeOther(
		fmt.Sprintf(
			h.APITypeUpdateOptionType,
			m.UpdateTypeID,
		),
	)
}
