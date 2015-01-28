package controller

import (
	"fmt"
	"net/http"

	h "github.com/microcosm-cc/microcosm/helpers"
	"github.com/microcosm-cc/microcosm/models"
)

func UpdateOptionHandler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	ctl := UpdateOptionController{}

	switch c.GetHttpMethod() {
	case "OPTIONS":
		c.RespondWithOptions([]string{"OPTIONS", "HEAD", "GET", "PUT"})
		return
	case "HEAD":
		ctl.Read(c)
	case "GET":
		ctl.Read(c)
	case "PUT":
		ctl.Update(c)
	default:
		c.RespondWithStatus(http.StatusMethodNotAllowed)
		return
	}
}

type UpdateOptionController struct{}

func (ctl *UpdateOptionController) Read(c *models.Context) {

	if c.Auth.ProfileId < 1 {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}

	_, _, itemId, status, err := c.GetItemTypeAndItemId()
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	m, status, err := models.GetUpdateOptionByUpdateType(c.Auth.ProfileId, itemId)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	c.RespondWithData(m)
}

func (ctl *UpdateOptionController) Update(c *models.Context) {

	_, _, itemId, status, err := c.GetItemTypeAndItemId()
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	var exists bool

	m, status, err := models.GetUpdateOptionByUpdateType(c.Auth.ProfileId, itemId)
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
	m.UpdateTypeId = itemId
	m.ProfileId = c.Auth.ProfileId

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
			h.ApiTypeUpdateOptionType,
			m.UpdateTypeId,
		),
	)

}
