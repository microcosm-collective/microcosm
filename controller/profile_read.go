package controller

import (
	"fmt"
	"net/http"

	"github.com/golang/glog"

	h "github.com/microcosm-cc/microcosm/helpers"
	"github.com/microcosm-cc/microcosm/models"
)

func ProfileReadHandler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	ctl := ProfileReadController{}

	switch c.GetHttpMethod() {
	case "OPTIONS":
		c.RespondWithOptions([]string{"OPTIONS", "PUT"})
		return
	case "PUT":
		ctl.Update(c)
	default:
		c.RespondWithStatus(http.StatusMethodNotAllowed)
		return
	}
}

type ProfileReadController struct{}

func (ctl *ProfileReadController) Update(c *models.Context) {

	if c.Auth.ProfileId == 0 {
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

	itemTypeId, ok := h.ItemTypes[rs.ItemType]
	if !ok {
		c.RespondWithErrorMessage("Unknown item type", http.StatusBadRequest)
		return
	}
	rs.ItemTypeID = itemTypeId

	status, err := models.MarkScopeAsRead(c.Auth.ProfileId, rs)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	c.RespondWithOK()
}
