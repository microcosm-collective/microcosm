package controller

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/golang/glog"

	"github.com/microcosm-cc/microcosm/audit"
	h "github.com/microcosm-cc/microcosm/helpers"
	"github.com/microcosm-cc/microcosm/models"
)

type SitesController struct{}

func SitesHandler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}
	ctl := SitesController{}

	switch c.GetHTTPMethod() {
	case "OPTIONS":
		c.RespondWithOptions([]string{"OPTIONS", "POST"})
		return
	case "POST":
		ctl.Create(c)
	case "GET":
		ctl.ReadMany(c)
	default:
		c.RespondWithStatus(http.StatusMethodNotAllowed)
		return
	}
}

func (ctl *SitesController) ReadMany(c *models.Context) {

	limit, offset, status, err := h.GetLimitAndOffset(c.Request.URL.Query())
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	var userId int64

	// Passing ?filter=owned returns sites owned by logged-in user.
	if c.Request.FormValue("filter") == "owned" {
		if c.Auth.UserID == 0 {
			c.RespondWithErrorMessage(
				fmt.Sprintf("You must be logged in to list your own sites"),
				http.StatusForbidden,
			)
			return
		} else {
			userId = c.Auth.UserID
		}
	}

	// Passing 0 as userID means return all sites.
	ems, total, pages, status, err := models.GetSites(userId, limit, offset)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	thisLink := h.GetLinkToThisPage(*c.Request.URL, offset, limit, total)

	m := models.SitesType{}
	m.Sites = h.ConstructArray(
		ems,
		h.APITypeSite,
		total,
		limit,
		offset,
		pages,
		c.Request.URL,
	)
	m.Meta.Links =
		[]h.LinkType{
			h.LinkType{Rel: "self", Href: thisLink.String()},
		}

	c.RespondWithData(m)
}

func (ctl *SitesController) Create(c *models.Context) {

	if c.Auth.ProfileID < 1 {
		c.RespondWithErrorMessage(
			fmt.Sprintf("You must be logged in to the root site create a site"),
			http.StatusForbidden,
		)
		return
	}

	if c.Site.ID != 1 {
		c.RespondWithErrorMessage(
			fmt.Sprintf("Sites can only be created from the root site"),
			http.StatusBadRequest,
		)
		return
	}

	bytes, _ := httputil.DumpRequest(c.Request, true)
	glog.Infoln(string(bytes))

	m := models.SiteType{}
	// Default theme
	m.ThemeID = 1
	err := c.Fill(&m)
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("The post data is invalid: %v", err.Error()),
			http.StatusBadRequest,
		)
		return
	}

	// Fetch full user info from context user ID
	user, status, err := models.GetUser(c.Auth.UserID)
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("Could not retrieve use associated with ID %d", c.Auth.UserID),
			status,
		)
		return
	}

	site, _, status, err := models.CreateOwnedSite(m, user)
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("Error creating site: %s", err.Error()),
			http.StatusInternalServerError,
		)
		return
	}

	audit.Create(
		c.Site.ID,
		h.ItemTypes[h.ItemTypeSite],
		site.ID,
		c.Auth.ProfileID,
		time.Now(),
		c.IP,
	)

	c.RespondWithSeeOther(fmt.Sprintf("%s/%d", h.APITypeSite, site.ID))
}
