package controller

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang/glog"
	"github.com/grafana/pyroscope-go"

	h "github.com/microcosm-cc/microcosm/helpers"
	"github.com/microcosm-cc/microcosm/models"
)

// WatcherHandler is a web handler
func WatcherHandler(w http.ResponseWriter, r *http.Request) {
	path := "/watchers/{id}"
	pyroscope.TagWrapper(context.Background(), pyroscope.Labels("path", path), func(context.Context) {
		c, status, err := models.MakeContext(r, w)
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}

		ctl := WatcherController{}

		method := c.GetHTTPMethod()
		switch method {
		case "PATCH":
			pyroscope.TagWrapper(context.Background(), pyroscope.Labels("method", method), func(context.Context) {
				ctl.Update(c)
			})
		case "OPTIONS":
			pyroscope.TagWrapper(context.Background(), pyroscope.Labels("method", method), func(context.Context) {
				c.RespondWithOptions([]string{"OPTIONS", "DELETE", "PATCH"})
			})
			return
		case "DELETE":
			pyroscope.TagWrapper(context.Background(), pyroscope.Labels("method", method), func(context.Context) {
				ctl.Delete(c)
			})
		default:
			c.RespondWithStatus(http.StatusMethodNotAllowed)
			return
		}
	})
}

// WatcherController is a web controller
type WatcherController struct{}

// Delete handles DELETE
func (ctl *WatcherController) Delete(c *models.Context) {

	_, _, itemID, _, _ := c.GetItemTypeAndItemID()
	if itemID != 0 {
		m, status, err := models.GetWatcher(itemID, c.Site.ID)
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}

		// Check ownership
		if c.Auth.ProfileID != m.ProfileID {
			c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
			return
		}

		// Delete resource
		status, err = m.Delete()
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}

		c.RespondWithOK()
	}

	// Fill from query string
	m := models.WatcherType{}

	itemID, itemType, _, _ := h.GetItemAndItemType(c.Request.URL.Query())

	if _, exists := h.ItemTypes[itemType]; !exists {
		c.RespondWithErrorMessage(
			"watcher could not be deleted: Item type not found",
			http.StatusBadRequest,
		)
		return
	}

	m.ItemTypeID = h.ItemTypes[itemType]

	var status int
	var err error
	m.ID, _, _, _, status, err = models.GetWatcherAndIgnoreStatus(
		m.ItemTypeID,
		itemID,
		c.Auth.ProfileID,
	)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	// Get watcher item to delete it
	m, status, err = models.GetWatcher(m.ID, c.Site.ID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	// Delete resource
	status, err = m.Delete()
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	c.RespondWithOK()
}

// Update handles PUT
func (ctl *WatcherController) Update(c *models.Context) {
	m := models.WatcherType{}
	err := c.Fill(&m)

	if err != nil {
		glog.Warning(err)
		c.RespondWithErrorMessage(
			fmt.Sprintf("The post data is invalid: %v", err.Error()),
			http.StatusBadRequest,
		)
		return
	}

	itemType := strings.ToLower(m.ItemType)
	if itemType != "" {
		if _, exists := h.ItemTypes[itemType]; !exists {
			glog.Warning(err)
			c.RespondWithErrorMessage(
				"watcher could not be saved: Item type not found",
				http.StatusBadRequest,
			)
			return
		}

		m.ItemTypeID = h.ItemTypes[itemType]
	}

	var status int
	// watcher must exist to be updated
	// Also the returned watcher ID belongs to the authed person by definition
	// - no need to check later
	m.ID, _, _, _, status, err = models.GetWatcherAndIgnoreStatus(
		m.ItemTypeID,
		m.ItemID,
		c.Auth.ProfileID,
	)
	if err != nil {
		glog.Error(err)
		c.RespondWithErrorDetail(err, status)
		return
	}

	// To update we only need id, SendEmail and SendSMS
	_, err = m.Update()
	if err != nil {
		glog.Error(err)
		c.RespondWithErrorMessage(
			fmt.Sprintf("Could not update watcher: %v", err.Error()),
			http.StatusBadRequest,
		)
		return
	}

	// Respond
	c.RespondWithOK()
}
