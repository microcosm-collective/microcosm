package controller

import (
	"database/sql"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/microcosm-cc/microcosm/audit"
	h "github.com/microcosm-cc/microcosm/helpers"
	"github.com/microcosm-cc/microcosm/models"
)

type AttributeController struct{}

func AttributeHandler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	ctl := AttributeController{}

	switch c.GetHttpMethod() {
	case "OPTIONS":
		c.RespondWithOptions([]string{"OPTIONS", "GET", "HEAD", "POST", "PUT", "DELETE"})
		return
	case "GET":
		ctl.Read(c)
	case "HEAD":
		ctl.Read(c)
	case "PUT":
		ctl.Update(c)
	case "DELETE":
		ctl.Delete(c)
	default:
		c.RespondWithStatus(http.StatusMethodNotAllowed)
		return
	}
}

// Returns a single attribute
func (ctl *AttributeController) Read(c *models.Context) {
	_, itemTypeId, itemId, status, err := c.GetItemTypeAndItemId()
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	perms := models.GetPermission(models.MakeAuthorisationContext(c, 0, itemTypeId, itemId))
	if !perms.CanRead {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}

	attributeId, status, err := models.GetAttributeId(itemTypeId, itemId, c.RouteVars["key"])
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	m, status, err := models.GetAttribute(attributeId)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	c.RespondWithData(m)
}

// Updates a single attribute
func (ctl *AttributeController) Update(c *models.Context) {
	_, itemTypeId, itemId, status, err := c.GetItemTypeAndItemId()
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	key := c.RouteVars["key"]

	// Exception from normal model is that we don't attempt to fetch before
	// we update. We will be doing an upsert (update or insert) rather than
	// a pure update. As such, the item may not exist before we update it and
	// we allow that to be resolved later. This works in this case as the data
	// structure is simple and we don't care about extended metadata
	m := models.AttributeType{}
	m.Key = key
	m.Number = sql.NullFloat64{Float64: math.MaxFloat64}

	err = c.Fill(&m)
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("The post data is invalid: %v", err.Error()),
			http.StatusBadRequest,
		)
		return
	}

	perms := models.GetPermission(models.MakeAuthorisationContext(c, 0, itemTypeId, itemId))
	if !perms.CanUpdate {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}

	status, err = m.Update(itemTypeId, itemId)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	audit.Replace(
		c.Site.Id,
		h.ItemTypes[h.ItemTypeAttribute],
		m.Id,
		c.Auth.ProfileId,
		time.Now(),
		c.IP,
	)

	c.RespondWithSeeOther(
		fmt.Sprintf(
			"%s/%s",
			fmt.Sprintf(h.ApiTypeAttribute, c.RouteVars["type"], itemId),
			key,
		),
	)
}

// Deletes a single attribute
func (ctl *AttributeController) Delete(c *models.Context) {
	_, itemTypeId, itemId, status, err := c.GetItemTypeAndItemId()
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	key := c.RouteVars["key"]

	m := models.AttributeType{}
	m.Key = key

	attributeId, status, err := models.GetAttributeId(itemTypeId, itemId, m.Key)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	m.Id = attributeId

	status, err = m.Delete()
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	audit.Delete(
		c.Site.Id,
		h.ItemTypes[h.ItemTypeAttribute],
		m.Id,
		c.Auth.ProfileId,
		time.Now(),
		c.IP,
	)

	c.RespondWithOK()
}
