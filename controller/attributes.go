package controller

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/microcosm-cc/microcosm/audit"
	h "github.com/microcosm-cc/microcosm/helpers"
	"github.com/microcosm-cc/microcosm/models"
)

// AttributesController is a web controller
type AttributesController struct{}

// AttributesHandler is a web handler
func AttributesHandler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}
	ctl := AttributesController{}

	method := c.GetHTTPMethod()
	switch method {
	case "OPTIONS":
		c.RespondWithOptions([]string{"OPTIONS", "GET", "HEAD", "POST", "PUT", "DELETE"})
		return
	case "GET":
		ctl.ReadMany(c)
	case "HEAD":
		ctl.ReadMany(c)
	case "PUT":
		ctl.UpdateMany(c)
	case "DELETE":
		ctl.DeleteMany(c)
	default:
		c.RespondWithStatus(http.StatusMethodNotAllowed)
		return
	}
}

// ReadMany handles GET for a collection
func (ctl *AttributesController) ReadMany(c *models.Context) {
	_, itemTypeID, itemID, status, err := c.GetItemTypeAndItemID()
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	perms := models.GetPermission(models.MakeAuthorisationContext(c, 0, itemTypeID, itemID))
	if !perms.CanRead {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}

	// Fetch query string args if any exist
	limit, offset, status, err := h.GetLimitAndOffset(c.Request.URL.Query())
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	ems, total, pages, status, err := models.GetAttributes(itemTypeID, itemID, limit, offset)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	// Construct the response
	m := models.AttributesType{}
	m.Attributes = h.ConstructArray(
		ems,
		fmt.Sprintf(h.APITypeAttribute, c.RouteVars["type"], 0),
		total,
		limit,
		offset,
		pages,
		c.Request.URL,
	)

	c.RespondWithData(m)
}

// UpdateMany handles PUT for the collection
func (ctl *AttributesController) UpdateMany(c *models.Context) {
	_, itemTypeID, itemID, status, err := c.GetItemTypeAndItemID()
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	ems := []models.AttributeType{}

	err = c.Fill(&ems)
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("The post data is invalid: %v", err.Error()),
			http.StatusBadRequest,
		)
		return
	}

	for _, v := range ems {
		if strings.Trim(v.Key, " ") == "" {
			c.RespondWithErrorMessage(
				"key must be supplied with every attribute when updating multiple attributes",
				http.StatusBadRequest,
			)
			return
		}
	}

	perms := models.GetPermission(models.MakeAuthorisationContext(c, 0, itemTypeID, itemID))
	if !perms.CanUpdate {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}

	status, err = models.UpdateManyAttributes(itemTypeID, itemID, ems)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	for _, m := range ems {
		audit.Replace(
			c.Site.ID,
			h.ItemTypes[h.ItemTypeAttribute],
			m.ID,
			c.Auth.ProfileID,
			time.Now(),
			c.IP,
		)
	}

	c.RespondWithOK()
}

// DeleteMany handles DELETE for the collection
func (ctl *AttributesController) DeleteMany(c *models.Context) {
	_, itemTypeID, itemID, status, err := c.GetItemTypeAndItemID()
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	ems := []models.AttributeType{}

	err = c.Fill(&ems)
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("The post data is invalid: %v", err.Error()),
			http.StatusBadRequest,
		)
		return
	}

	for _, v := range ems {
		if strings.Trim(v.Key, " ") == "" {
			c.RespondWithErrorMessage(
				"key must be supplied with every attribute when deleting multiple attributes",
				http.StatusBadRequest,
			)
			return
		}
	}

	perms := models.GetPermission(models.MakeAuthorisationContext(c, 0, itemTypeID, itemID))
	if !perms.CanDelete {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}

	status, err = models.DeleteManyAttributes(itemTypeID, itemID, ems)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	for _, m := range ems {
		audit.Delete(
			c.Site.ID,
			h.ItemTypes[h.ItemTypeAttribute],
			m.ID,
			c.Auth.ProfileID,
			time.Now(),
			c.IP,
		)
	}

	c.RespondWithOK()
}
