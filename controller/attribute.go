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

// AttributeController is a web controller
type AttributeController struct{}

// AttributeHandler is a web handler
func AttributeHandler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	ctl := AttributeController{}

	switch c.GetHTTPMethod() {
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

// Read handles GET
func (ctl *AttributeController) Read(c *models.Context) {
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

	attributeID, status, err := models.GetAttributeID(itemTypeID, itemID, c.RouteVars["key"])
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	m, status, err := models.GetAttribute(attributeID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	c.RespondWithData(m)
}

// Update handles PUT
func (ctl *AttributeController) Update(c *models.Context) {
	_, itemTypeID, itemID, status, err := c.GetItemTypeAndItemID()
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

	perms := models.GetPermission(models.MakeAuthorisationContext(c, 0, itemTypeID, itemID))
	if !perms.CanUpdate {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}

	status, err = m.Update(itemTypeID, itemID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	audit.Replace(
		c.Site.ID,
		h.ItemTypes[h.ItemTypeAttribute],
		m.ID,
		c.Auth.ProfileID,
		time.Now(),
		c.IP,
	)

	c.RespondWithSeeOther(
		fmt.Sprintf(
			"%s/%s",
			fmt.Sprintf(h.APITypeAttribute, c.RouteVars["type"], itemID),
			key,
		),
	)
}

// Delete handles DELETE
func (ctl *AttributeController) Delete(c *models.Context) {
	_, itemTypeID, itemID, status, err := c.GetItemTypeAndItemID()
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	key := c.RouteVars["key"]

	m := models.AttributeType{}
	m.Key = key

	attributeID, status, err := models.GetAttributeID(itemTypeID, itemID, m.Key)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	m.ID = attributeID

	status, err = m.Delete()
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	audit.Delete(
		c.Site.ID,
		h.ItemTypes[h.ItemTypeAttribute],
		m.ID,
		c.Auth.ProfileID,
		time.Now(),
		c.IP,
	)

	c.RespondWithOK()
}
