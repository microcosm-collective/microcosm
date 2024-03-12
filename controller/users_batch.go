package controller

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"github.com/microcosm-cc/microcosm/models"
)

// UsersBatchController is a web controller
type UsersBatchController struct{}

// UsersBatchHandler is a web handler
func UsersBatchHandler(w http.ResponseWriter, r *http.Request) {
	c, status, err := models.MakeContext(r, w)
	if err != nil {
		c.RespondWithErrorMessage(err.Error(), status)
		return
	}
	ctl := UsersBatchController{}

	method := c.GetHTTPMethod()
	switch method {
	case "OPTIONS":
		c.RespondWithOptions([]string{"OPTIONS", "POST"})
		return
	case "POST":
		ctl.Manage(c)
	default:
		c.RespondWithStatus(http.StatusMethodNotAllowed)
		return
	}
}

// Manage allows the management of users by a site admin
func (ctl *UsersBatchController) Manage(c *models.Context) {
	if !c.Auth.IsSiteOwner {
		c.RespondWithErrorMessage(
			"Only a site owner can batch manage users",
			http.StatusForbidden,
		)
		return
	}

	var ems []models.UserMembership

	switch getContentType(c.Request) {
	case "application/json":
		if err := c.Fill(&ems); err != nil {
			c.RespondWithErrorMessage(
				fmt.Sprintf("The post data is invalid: %s", err.Error()),
				http.StatusBadRequest,
			)
			return
		}
	case "text/csv":
		dat, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.RespondWithErrorMessage(
				fmt.Sprintf("The CSV could not be read from the request: %s", err.Error()),
				http.StatusBadRequest,
			)
			return
		}

		data := string(bytes.Trim(dat, "\xef\xbb\xbf"))
		data = strings.Replace(data, "\r\n", "\n", -1)
		data = strings.Replace(data, "\r", "\n", -1)

		glog.Warningf("CSV received for site %d (%s): %s", c.Site.ID, c.Site.SubdomainKey, data)

		reader := csv.NewReader(bytes.NewBufferString(data))

		rows, err := reader.ReadAll()
		if err != nil {
			glog.Errorf("That was not a CSV file: %s", err.Error())
			c.RespondWithErrorMessage(
				fmt.Sprintf("That was not a CSV file: %s", err.Error()),
				http.StatusBadRequest,
			)
			return
		}

		for _, row := range rows {
			if len(row) < 2 {
				glog.Errorf("Each line in the CSV file must have at least 2 fields: email,integer")
				c.RespondWithErrorMessage(
					"Each line in the CSV file must have at least 2 fields: email,integer",
					http.StatusBadRequest,
				)
				return
			}

			if row[0] == "" || row[1] == "" {
				glog.Warningf("Invalid (empty) values: %s %s", row[0], row[1])
				continue
			}

			email, err := mail.ParseAddress(strings.TrimSpace(row[0]))
			if err != nil {
				glog.Errorf("Not an email: %s %s", row[0], err.Error())
				c.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
				return
			}

			isMember, err := strconv.ParseBool(strings.TrimSpace(row[1]))
			if err != nil {
				glog.Errorf("Not a bool: %s %s", row[1], err.Error())
				c.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
				return
			}

			var m models.UserMembership
			m.Email = email.Address
			m.IsMember = isMember

			ems = append(ems, m)
		}
	default:
		glog.Errorf("Only application/json or text/csv can be POST'd")
		c.RespondWithErrorMessage(
			"Only application/json or text/csv can be POST'd",
			http.StatusBadRequest,
		)
		return
	}

	if len(ems) == 0 {
		glog.Errorf("Input empty, no rows to process")
		c.RespondWithErrorMessage(
			"Input empty, no rows to process",
			http.StatusBadRequest,
		)
		return
	}

	status, err := models.ManageUsers(c.Site, ems)
	if err != nil {
		glog.Errorf("models.ManageUsers(%d, ems): %s", c.Site.ID, err.Error())
		c.RespondWithErrorMessage(err.Error(), status)
		return
	}

	c.RespondWithOK()
}

func getContentType(r *http.Request) string {
	ct := r.Header.Get("Content-Type")
	if strings.TrimSpace(ct) == "" {
		return "application/x-www-form-urlencoded"
	}

	// Ignore anything after a ; (like charset) and return the content-type
	return strings.ToLower(strings.Split(ct, ";")[0])
}
