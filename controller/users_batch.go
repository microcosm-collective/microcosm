package controller

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/mail"
	"strconv"
	"strings"

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

	switch c.GetHTTPMethod() {
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
		dat, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			c.RespondWithErrorMessage(
				fmt.Sprintf("The CSV could not be read from the request: %s", err.Error()),
				http.StatusBadRequest,
			)
			return
		}

		data := string(dat)
		data = strings.Replace(data, "\r\n", "\n", -1)
		data = strings.Replace(data, "\r", "\n", -1)

		reader := csv.NewReader(bytes.NewBufferString(data))

		rows, err := reader.ReadAll()
		if err != nil {
			c.RespondWithErrorMessage(
				fmt.Sprintf("That was not a CSV file: %s", err.Error()),
				http.StatusBadRequest,
			)
			return
		}

		for _, row := range rows {
			if len(row) < 2 {
				c.RespondWithErrorMessage(
					"Each line in the CSV file must have at least 2 fields: email,integer",
					http.StatusBadRequest,
				)
				return
			}

			if row[0] == "" || row[1] == "" {
				continue
			}

			email, err := mail.ParseAddress(row[0])
			if err != nil {
				c.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
				return
			}

			isMember, err := strconv.ParseBool(row[1])
			if err != nil {
				c.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
				return
			}

			var m models.UserMembership
			m.Email = email.Address
			m.IsMember = isMember

			ems = append(ems, m)
		}
	default:
		c.RespondWithErrorMessage(
			"Only application/json or text/csv can be POST'd",
			http.StatusBadRequest,
		)
		return
	}

	if len(ems) == 0 {
		c.RespondWithErrorMessage(
			"Input empty, no rows to process",
			http.StatusBadRequest,
		)
		return
	}

	status, err := models.ManageUsers(c.Site, ems)
	if err != nil {
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
