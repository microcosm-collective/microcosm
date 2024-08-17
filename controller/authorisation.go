package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	h "git.dee.kitchen/buro9/microcosm/helpers"
	"git.dee.kitchen/buro9/microcosm/models"
)

// PermissionController is a web controller
type PermissionController struct{}

// PermissionHandler is a web handler
func PermissionHandler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}
	ctl := PermissionController{}

	method := c.GetHTTPMethod()
	switch method {
	case "OPTIONS":
		c.RespondWithOptions([]string{"OPTIONS", "HEAD", "GET"})
		return
	case "HEAD":
		ctl.Read(c)
	case "GET":
		ctl.Read(c)
	default:
		c.RespondWithStatus(http.StatusMethodNotAllowed)
		return
	}
}

// Read handles GET
func (ctl *PermissionController) Read(c *models.Context) {
	ac, status, err := GetAuthContext(c)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}
	m := models.GetPermission(ac)

	c.RespondWithData(m)
}

// GetAuthContext returns the auth context for the current request
func GetAuthContext(c *models.Context) (models.AuthContext, int, error) {
	query := c.Request.URL.Query()

	var microcosmID int64
	if query.Get("microcosmId") != "" {
		id, err := strconv.ParseInt(strings.Trim(query.Get("microcosmId"), " "), 10, 64)
		if err != nil || id < 0 {
			return models.AuthContext{}, http.StatusBadRequest,
				fmt.Errorf("microcosmId needs to be a positive integer")
		}
		microcosmID = id
	}

	var itemTypeID int64
	itemType := strings.ToLower(query.Get("itemType"))
	if itemType != "" {
		if _, exists := h.ItemTypes[itemType]; !exists {
			return models.AuthContext{}, http.StatusBadRequest,
				fmt.Errorf("you must specify a valid itemType")
		}
		itemTypeID = h.ItemTypes[itemType]
	}

	var itemID int64
	if query.Get("itemId") != "" {
		id, err := strconv.ParseInt(strings.Trim(query.Get("itemId"), " "), 10, 64)
		if err != nil || id < 0 {
			return models.AuthContext{}, http.StatusBadRequest,
				fmt.Errorf("itemId needs to be a positive integer")
		}
		itemID = id
	}

	return models.MakeAuthorisationContext(c, microcosmID, itemTypeID, itemID), http.StatusOK, nil
}
