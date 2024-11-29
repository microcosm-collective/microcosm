package controller

import (
	"context"
	"fmt"
	"net/http"

	h "git.dee.kitchen/buro9/microcosm/helpers"
	"git.dee.kitchen/buro9/microcosm/models"
	"github.com/grafana/pyroscope-go"
)

// IgnoredHandler is a web handler
func IgnoredHandler(w http.ResponseWriter, r *http.Request) {
	path := "/ignored"
	pyroscope.TagWrapper(context.Background(), pyroscope.Labels("path", path), func(ctx context.Context) {
		c, status, err := models.MakeContext(r, w)
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}
		ctl := IgnoredController{}

		method := c.GetHTTPMethod()
		switch method {
		case "OPTIONS":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				c.RespondWithOptions([]string{"OPTIONS", "HEAD", "GET", "PUT", "DELETE"})
			})
			return
		case "HEAD":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				ctl.ReadMany(c)
			})
		case "GET":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				ctl.ReadMany(c)
			})
		case "PUT":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				ctl.Update(c)
			})
		case "DELETE":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				ctl.Delete(c)
			})
		default:
			c.RespondWithStatus(http.StatusMethodNotAllowed)
			return
		}
	})
}

// IgnoredController is a web controller
type IgnoredController struct{}

// ReadMany handles GET
func (ctl *IgnoredController) ReadMany(c *models.Context) {
	if c.Auth.ProfileID < 1 {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}

	limit, offset, status, err := h.GetLimitAndOffset(c.Request.URL.Query())
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	ems, total, pages, status, err := models.GetIgnored(
		c.Site.ID,
		c.Auth.ProfileID,
		limit,
		offset,
	)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	thisLink := h.GetLinkToThisPage(*c.Request.URL, offset, limit, total)
	m := models.IgnoredType{}
	m.Ignored = h.ConstructArray(
		ems,
		`/api/v1/ignored`,
		total,
		limit,
		offset,
		pages,
		c.Request.URL,
	)
	m.Meta.Links =
		[]h.LinkType{
			{
				Rel:  "self",
				Href: thisLink.String(),
			},
		}

	c.RespondWithData(m)
}

// Update handles PUT
func (ctl *IgnoredController) Update(c *models.Context) {
	m := models.IgnoreType{}
	err := c.Fill(&m)
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("The post data is invalid: %v", err.Error()),
			http.StatusBadRequest,
		)
		return
	}

	if c.Auth.ProfileID < 1 {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}

	m.ProfileID = c.Auth.ProfileID
	status, err := m.Update()
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	c.RespondWithOK()
}

// Delete handles DELETE
func (ctl *IgnoredController) Delete(c *models.Context) {
	m := models.IgnoreType{}
	err := c.Fill(&m)
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("The post data is invalid: %v", err.Error()),
			http.StatusBadRequest,
		)
		return
	}

	if c.Auth.ProfileID < 1 {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}

	m.ProfileID = c.Auth.ProfileID
	status, err := m.Delete()
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	c.RespondWithOK()
}
