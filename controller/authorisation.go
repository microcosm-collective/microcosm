package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	h "github.com/microcosm-cc/microcosm/helpers"
	"github.com/microcosm-cc/microcosm/models"
)

type PermissionController struct{}

func PermissionHandler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}

	ctl := PermissionController{}

	switch c.GetHttpMethod() {
	case "OPTIONS":
		c.RespondWithOptions([]string{"OPTIONS", "HEAD", "GET"})
		return
	case "HEAD":
		ctl.HandleRequest(c)
	case "GET":
		ctl.HandleRequest(c)
	default:
		c.RespondWithStatus(http.StatusMethodNotAllowed)
		return
	}
}

func (ctl *PermissionController) HandleRequest(c *models.Context) {
	ac, status, err := GetAuthContext(c)
	if err != nil {
		c.RespondWithErrorDetail(err, status)
		return
	}
	m := models.GetPermission(ac)

	// Add cache duration header
	// 30 = 30 seconds
	c.ResponseWriter.Header().Set("Cache-Control", fmt.Sprintf(`max-age=%d, public`, 30))

	c.RespondWithData(m)
}

func GetAuthContext(c *models.Context) (models.AuthContext, int, error) {

	query := c.Request.URL.Query()

	var microcosmId int64
	if query.Get("microcosmId") != "" {
		id, err := strconv.ParseInt(strings.Trim(query.Get("microcosmId"), " "), 10, 64)
		if err != nil || id < 0 {
			return models.AuthContext{}, http.StatusBadRequest, errors.New("microcosmId needs to be a positive integer")
		}
		microcosmId = id
	}

	var itemTypeId int64
	itemType := strings.ToLower(query.Get("itemType"))
	if itemType != "" {
		if _, exists := h.ItemTypes[itemType]; !exists {
			return models.AuthContext{}, http.StatusBadRequest, errors.New("You must specify a valid itemType")
		} else {
			itemTypeId = h.ItemTypes[itemType]
		}
	}

	var itemId int64
	if query.Get("itemId") != "" {
		id, err := strconv.ParseInt(strings.Trim(query.Get("itemId"), " "), 10, 64)
		if err != nil || id < 0 {
			return models.AuthContext{}, http.StatusBadRequest, errors.New("itemId needs to be a positive integer")
		}
		itemId = id
	}

	return models.MakeAuthorisationContext(c, microcosmId, itemTypeId, itemId), http.StatusOK, nil
}
