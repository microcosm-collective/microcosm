package controller

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/grafana/pyroscope-go"
	h "github.com/microcosm-collective/microcosm/helpers"
	"github.com/microcosm-collective/microcosm/models"
)

// WatchersHandler is a web handler
func WatchersHandler(w http.ResponseWriter, r *http.Request) {
	path := "/watchers"
	pyroscope.TagWrapper(context.Background(), pyroscope.Labels("path", path), func(ctx context.Context) {

		c, status, err := models.MakeContext(r, w)
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}
		ctl := WatchersController{}

		method := c.GetHTTPMethod()
		switch method {
		case "OPTIONS":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				c.RespondWithOptions([]string{"OPTIONS", "GET", "PUT", "POST"})
			})
			return
		case "GET":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				ctl.ReadMany(c)
			})
		case "PUT":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				ctl.Create(c)
			})
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

// WatchersController is a web controller
type WatchersController struct{}

// ReadMany handles GET for the collection
func (ctl *WatchersController) ReadMany(c *models.Context) {

	if c.Auth.ProfileID < 1 {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}

	query := c.Request.URL.Query()

	limit, offset, status, err := h.GetLimitAndOffset(query)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	ems, total, pages, status, err := models.GetProfileWatchers(
		c.Auth.ProfileID,
		c.Site.ID,
		limit,
		offset,
	)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	// Construct the response
	thisLink := h.GetLinkToThisPage(*c.Request.URL, offset, limit, total)

	m := models.WatchersType{}
	m.Watchers = h.ConstructArray(
		ems,
		h.APITypeWatcher,
		total,
		limit,
		offset,
		pages,
		c.Request.URL,
	)
	m.Meta.Links =
		[]h.LinkType{
			{Rel: "self", Href: thisLink.String()},
		}

	c.RespondWithData(m)

}

// WatcherType allows a watcher to be marshaled from form data
type WatcherType struct {
	models.WatcherType

	UpdateTypeID int64 `json:"updateTypeId"`
}

// Create handles POST
func (ctl *WatchersController) Create(c *models.Context) {
	// Fill from POST data
	m := WatcherType{}
	err := c.Fill(&m)

	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("The post data is invalid: %v", err.Error()),
			http.StatusBadRequest,
		)
		return
	}

	itemType := strings.ToLower(m.ItemType)
	if itemType != "" {
		if _, exists := h.ItemTypes[itemType]; !exists {
			c.RespondWithErrorMessage(
				"watcher could not be saved: Item type not found",
				http.StatusBadRequest,
			)
			return
		}

		m.ItemTypeID = h.ItemTypes[itemType]
	} else {
		c.RespondWithErrorMessage(
			"no itemType supplied, cannot create a watcher",
			http.StatusBadRequest,
		)
		return
	}

	sendEmail, status, err := models.RegisterWatcher(
		c.Auth.ProfileID,
		m.UpdateTypeID,
		m.ItemID,
		m.ItemTypeID,
		c.Site.ID,
	)
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("Watcher could not be registered: %v", err.Error()),
			status,
		)
		return
	}

	c.RespondWithData(sendEmail)
}
