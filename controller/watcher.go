package controller

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/golang/glog"

	h "github.com/microcosm-cc/microcosm/helpers"
	"github.com/microcosm-cc/microcosm/models"
)

func WatcherHandler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	ctl := WatcherController{}

	switch c.GetHttpMethod() {
	case "PATCH":
		ctl.Update(c)
	case "OPTIONS":
		c.RespondWithOptions([]string{"OPTIONS", "DELETE", "PATCH"})
		return
	case "DELETE":
		ctl.Delete(c)
	default:
		c.RespondWithStatus(http.StatusMethodNotAllowed)
		return
	}
}

type WatcherController struct{}

func (ctl *WatcherController) Delete(c *models.Context) {

	_, _, itemId, status, err := c.GetItemTypeAndItemId()
	if itemId != 0 {
		m, status, err := models.GetWatcher(itemId, c.Site.ID)
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}

		// Check ownership
		if c.Auth.ProfileId != m.ProfileID {
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

	itemId, itemType, status, err := h.GetItemAndItemType(c.Request.URL.Query())

	if _, exists := h.ItemTypes[itemType]; !exists {
		c.RespondWithErrorMessage(
			fmt.Sprintf("Watcher could not be deleted: Item type not found"),
			http.StatusBadRequest,
		)
		return
	} else {
		m.ItemTypeID = h.ItemTypes[itemType]
	}

	m.ID, _, _, _, status, err = models.GetWatcherAndIgnoreStatus(
		m.ItemTypeID,
		itemId,
		c.Auth.ProfileId,
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
				fmt.Sprintf("Watcher could not be saved: Item type not found"),
				http.StatusBadRequest,
			)
			return
		} else {
			m.ItemTypeID = h.ItemTypes[itemType]
		}
	}

	var status int
	// watcher must exist to be updated
	// Also the returned watcher ID belongs to the authed person by definition
	// - no need to check later
	m.ID, _, _, _, status, err = models.GetWatcherAndIgnoreStatus(
		m.ItemTypeID,
		m.ItemID,
		c.Auth.ProfileId,
	)
	if err != nil {
		glog.Error(err)
		c.RespondWithErrorDetail(err, status)
		return
	}

	// To update we only need id, SendEmail and SendSMS
	status, err = m.Update()
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
