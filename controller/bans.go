package controller

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/golang/glog"
	"github.com/grafana/pyroscope-go"

	h "github.com/microcosm-collective/microcosm/helpers"
	"github.com/microcosm-collective/microcosm/models"
)

// BansController is a web controller
type BansController struct{}

// BansHandler is a web handler
func BansHandler(w http.ResponseWriter, r *http.Request) {
	path := "/bans"
	pyroscope.TagWrapper(context.Background(), pyroscope.Labels("path", path), func(ctx context.Context) {
		c, status, err := models.MakeContext(r, w)
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}
		ctl := BansController{}

		method := c.GetHTTPMethod()
		switch method {
		case "OPTIONS":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				c.RespondWithOptions([]string{"OPTIONS", "GET", "HEAD", "POST"})
			})
			return
		case "GET":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				ctl.ReadMany(c)
			})
		case "HEAD":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				ctl.ReadMany(c)
			})
		case "POST":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				ctl.Create(c)
			})
		default:
			c.RespondWithStatus(http.StatusMethodNotAllowed)
			return
		}
	})
}

// BanController is a web controller
type BanController struct{}

// BanHandler is a web handler
func BanHandler(w http.ResponseWriter, r *http.Request) {
	path := "/bans/{id}"
	pyroscope.TagWrapper(context.Background(), pyroscope.Labels("path", path), func(ctx context.Context) {
		c, status, err := models.MakeContext(r, w)
		if err != nil {
			c.RespondWithErrorDetail(err, status)
			return
		}
		ctl := BanController{}

		method := c.GetHTTPMethod()
		switch method {
		case "OPTIONS":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				c.RespondWithOptions([]string{"OPTIONS", "GET", "HEAD", "PUT", "DELETE"})
			})
			return
		case "GET":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				ctl.Read(c)
			})
		case "HEAD":
			pyroscope.TagWrapper(ctx, pyroscope.Labels("method", method), func(context.Context) {
				ctl.Read(c)
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

// ReadMany handles GET
func (ctl *BansController) ReadMany(c *models.Context) {
	if !canManageBans(c) {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}

	limit, offset, status, err := h.GetLimitAndOffset(c.Request.URL.Query())
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	ems, total, pages, status, err := models.GetBans(c.Site.ID, limit, offset)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	m := models.BansType{}
	m.Bans = h.ConstructArray(
		ems,
		h.APITypeBan,
		total,
		limit,
		offset,
		pages,
		c.Request.URL,
	)

	c.RespondWithData(m)
}

// Create handles POST
func (ctl *BansController) Create(c *models.Context) {
	if !canManageBans(c) {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}

	m := models.BanType{}
	err := c.Fill(&m)
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("The post data is invalid: %v", err.Error()),
			http.StatusBadRequest,
		)
		return
	}

	status, err := m.Insert(c.Site.ID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	logBanAction(c, "create", m, models.BanType{})

	c.RespondWithSeeOther(m.GetLink())
}

// Read handles GET
func (ctl *BanController) Read(c *models.Context) {
	if !canManageBans(c) {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}

	banID, status, err := getBanID(c)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	m, status, err := models.GetBan(c.Site.ID, banID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	c.RespondWithData(m)
}

// Update handles PUT
func (ctl *BanController) Update(c *models.Context) {
	if !canManageBans(c) {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}

	banID, status, err := getBanID(c)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	old, status, err := models.GetBan(c.Site.ID, banID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	m := old
	err = c.Fill(&m)
	if err != nil {
		c.RespondWithErrorMessage(
			fmt.Sprintf("The post data is invalid: %v", err.Error()),
			http.StatusBadRequest,
		)
		return
	}

	m.ID = old.ID
	m.SiteID = old.SiteID
	if m.UserID != old.UserID {
		c.RespondWithErrorMessage("userId cannot be changed", http.StatusBadRequest)
		return
	}

	status, err = m.Update(c.Site.ID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	logBanAction(c, "update", m, old)

	c.RespondWithSeeOther(m.GetLink())
}

// Delete handles DELETE
func (ctl *BanController) Delete(c *models.Context) {
	if !canManageBans(c) {
		c.RespondWithErrorMessage(h.NoAuthMessage, http.StatusForbidden)
		return
	}

	banID, status, err := getBanID(c)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	m, status, err := models.GetBan(c.Site.ID, banID)
	if err != nil {
		if status == http.StatusNotFound {
			c.RespondWithOK()
			return
		}

		c.RespondWithErrorDetail(err, status)
		return
	}

	status, err = m.Delete(c.Site.ID)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	logBanAction(c, "delete", models.BanType{}, m)

	c.RespondWithOK()
}

func canManageBans(c *models.Context) bool {
	rootMicrocosmID := models.GetRootMicrocosmID(c.Site.ID)
	if rootMicrocosmID == 0 {
		return false
	}

	perms := models.GetPermission(
		models.MakeAuthorisationContext(
			c,
			rootMicrocosmID,
			h.ItemTypes[h.ItemTypeMicrocosm],
			rootMicrocosmID,
		),
	)

	return c.Auth.IsSiteOwner || perms.IsModerator
}

func getBanID(c *models.Context) (int64, int, error) {
	banID, err := strconv.ParseInt(c.RouteVars["ban_id"], 10, 64)
	if err != nil {
		return 0, http.StatusBadRequest, fmt.Errorf("ban_id in URL is not a number")
	}

	return banID, http.StatusOK, nil
}

func logBanAction(c *models.Context, action string, ban models.BanType, old models.BanType) {
	glog.Infof(
		"ban_%s site_id=%d actor_profile_id=%d actor_user_id=%d ip=%s ban_id=%d user_id=%d expires=%s old_expires=%s display_reason=%q old_display_reason=%q admin_reason=%q old_admin_reason=%q",
		action,
		c.Site.ID,
		c.Auth.ProfileID,
		c.Auth.UserID,
		c.IP.String(),
		firstNonZero(ban.ID, old.ID),
		firstNonZero(ban.UserID, old.UserID),
		formatBanExpires(ban.Expires),
		formatBanExpires(old.Expires),
		ban.DisplayReason,
		old.DisplayReason,
		ban.AdminReason,
		old.AdminReason,
	)
}

func firstNonZero(a int64, b int64) int64 {
	if a != 0 {
		return a
	}

	return b
}

func formatBanExpires(expires *time.Time) string {
	if expires == nil {
		return ""
	}

	return expires.Format(time.RFC3339)
}
