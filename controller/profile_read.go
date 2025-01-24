package controller

import (
	"context"
	"fmt"
	"net/http"

	"github.com/golang/glog"
	"github.com/grafana/pyroscope-go"

	h "github.com/microcosm-collective/microcosm/helpers"
	"github.com/microcosm-collective/microcosm/models"
)

// ProfileReadHandler is a web handler
func ProfileReadHandler(w http.ResponseWriter, r *http.Request) {
	path := "/profiles/read"
	pyroscope.TagWrapper(context.Background(), pyroscope.Labels("path", path), func(ctx context.Context) {
		c, status, err := models.MakeContext(r, w)
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}
		ctl := ProfileReadController{}

		method := c.GetHTTPMethod()
		switch method {
		case "OPTIONS":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				c.RespondWithOptions([]string{"OPTIONS", "PUT"})
			})
			return
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

// ProfileReadController is a web controller
type ProfileReadController struct{}

// Update handles PUT
func (ctl *ProfileReadController) Update(c *models.Context) {
	if c.Auth.ProfileID == 0 {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}

	rs := models.ReadScopeType{}
	err := c.Fill(&rs)
	if err != nil {
		glog.Errorln(err.Error())
		c.RespondWithErrorMessage(
			fmt.Sprintf("The post data is invalid: %v", err.Error()),
			http.StatusBadRequest,
		)
		return
	}

	itemTypeID, ok := h.ItemTypes[rs.ItemType]
	if !ok {
		c.RespondWithErrorMessage("Unknown item type", http.StatusBadRequest)
		return
	}
	rs.ItemTypeID = itemTypeID

	status, err := models.MarkScopeAsRead(c.Auth.ProfileID, rs)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	c.RespondWithOK()
}
