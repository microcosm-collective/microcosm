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

// HuddlesController is a web controller
type HuddlesController struct{}

// HuddlesHandler is a web handler
func HuddlesHandler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	ctl := HuddlesController{}

	switch c.GetHTTPMethod() {
	case "OPTIONS":
		c.RespondWithOptions([]string{"OPTIONS", "GET", "HEAD", "POST"})
		return
	case "GET":
		ctl.ReadMany(c)
	case "HEAD":
		ctl.ReadMany(c)
	case "POST":
		ctl.Create(c)
	default:
		c.RespondWithStatus(http.StatusMethodNotAllowed)
		return
	}
}

// ReadMany handles GET for the collection
func (ctl *HuddlesController) ReadMany(c *models.Context) {
	// NOTE: Auth check skipped, permissions are enforced by limiting the scope
	// of the underlying queries in the model to only show huddles you are a
	// participant in

	// Fetch query string args if any exist
	limit, offset, status, err := h.GetLimitAndOffset(c.Request.URL.Query())
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	ems, total, pages, status, err := models.GetHuddles(
		c.Site.ID,
		c.Auth.ProfileID,
		limit,
		offset,
		(strings.ToLower(c.Request.URL.Query().Get("unread")) == "true"),
	)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	// Construct the response
	thisLink := h.GetLinkToThisPage(*c.Request.URL, offset, limit, total)

	m := models.HuddlesType{}
	m.Huddles = h.ConstructArray(
		ems,
		h.APITypeHuddle,
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

// Create handles POST
func (ctl *HuddlesController) Create(c *models.Context) {
	// Validate inputs
	m := models.HuddleType{}

	err := c.Fill(&m)
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("The post data is invalid: %v", err.Error()),
			http.StatusBadRequest,
		)
		return
	}

	// Start : Authorisation
	perms := models.GetPermission(
		models.MakeAuthorisationContext(
			c, 0, h.ItemTypes[h.ItemTypeHuddle], 0), // TODO: Erm, is this right?
	)
	if !perms.CanCreate {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}
	// End : Authorisation

	// Populate where applicable from auth and context
	m.Meta.CreatedByID = c.Auth.ProfileID
	m.Meta.Created = time.Now()

	status, err := m.Insert(c.Site.ID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	audit.Create(
		c.Site.ID,
		h.ItemTypes[h.ItemTypeHuddle],
		m.ID,
		c.Auth.ProfileID,
		time.Now(),
		c.IP,
	)

	c.RespondWithSeeOther(
		fmt.Sprintf(
			"%s/%d",
			h.APITypeHuddle,
			m.ID,
		),
	)
}
