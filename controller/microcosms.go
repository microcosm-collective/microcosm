package controller

import (
	"fmt"
	"net/http"
	"time"

	"github.com/microcosm-cc/microcosm/audit"
	h "github.com/microcosm-cc/microcosm/helpers"
	"github.com/microcosm-cc/microcosm/models"
)

func MicrocosmsHandler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	ctl := MicrocosmsController{}

	switch c.GetHttpMethod() {
	case "OPTIONS":
		c.RespondWithOptions([]string{"OPTIONS", "POST", "HEAD", "GET"})
		return
	case "POST":
		ctl.Create(c)
	case "HEAD":
		ctl.ReadMany(c)
	case "GET":
		ctl.ReadMany(c)
	default:
		c.RespondWithStatus(http.StatusMethodNotAllowed)
		return
	}
}

type MicrocosmsController struct{}

func (ctl *MicrocosmsController) Create(c *models.Context) {

	// Validate inputs
	m := models.MicrocosmType{}
	err := c.Fill(&m)
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("The post data is invalid: %v", err.Error()),
			http.StatusBadRequest,
		)
		return
	}

	// Start : Authorisation
	perms := models.GetPermission(models.MakeAuthorisationContext(c, 0, h.ItemTypes[h.ItemTypeSite], c.Site.Id))
	if !perms.CanCreate {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}
	// End : Authorisation

	// Populate where applicable from auth and context
	m.SiteId = c.Site.Id
	m.Meta.CreatedById = c.Auth.ProfileId
	m.Meta.Created = time.Now()
	m.OwnedById = c.Auth.ProfileId

	status, err := m.Insert()
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	audit.Create(
		c.Site.Id,
		h.ItemTypes[h.ItemTypeMicrocosm],
		m.Id,
		c.Auth.ProfileId,
		time.Now(),
		c.IP,
	)

	c.RespondWithSeeOther(
		fmt.Sprintf(
			"%s/%d",
			h.ApiTypeMicrocosm,
			m.Id,
		),
	)
}

func (ctl *MicrocosmsController) ReadMany(c *models.Context) {

	perms := models.GetPermission(
		models.MakeAuthorisationContext(c, 0, h.ItemTypes[h.ItemTypeSite], c.Site.Id),
	)

	// Fetch query string args if any exist
	limit, offset, status, err := h.GetLimitAndOffset(c.Request.URL.Query())
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	// Fetch list of microcosms
	ems, total, pages, status, err := models.GetMicrocosms(c.Site.Id, c.Auth.ProfileId, limit, offset)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	// Construct the response
	thisLink := h.GetLinkToThisPage(*c.Request.URL, offset, limit, total)
	m := models.MicrocosmsType{}
	m.Microcosms = h.ConstructArray(
		ems,
		h.ApiTypeMicrocosm,
		total,
		limit,
		offset,
		pages,
		c.Request.URL,
	)
	m.Meta.Links = []h.LinkType{
		h.LinkType{Rel: "self", Href: thisLink.String()},
	}
	m.Meta.Permissions = perms

	c.RespondWithData(m)
}
