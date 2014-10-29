package models

import (
	"database/sql"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/golang/glog"

	h "github.com/microcosm-cc/microcosm/helpers"
)

type MenuType struct {
	SiteId   int64
	Href     string
	Text     string
	Title    sql.NullString
	Sequence int
}

func (m *MenuType) Validate() (int, error) {

	// SiteId
	if m.SiteId == 0 {
		return http.StatusBadRequest, errors.New("SiteId is a required field")
	}

	// Href
	m.Href = strings.Trim(m.Href, " ")
	if m.Href == "" {
		return http.StatusBadRequest, errors.New("Href is a required field")
	}

	u, err := url.Parse(m.Href)
	if err != nil {
		return http.StatusBadRequest, errors.New("Href is not a valid URL")
	}
	m.Href = u.String()

	// Text
	m.Text = strings.Trim(m.Text, " ")
	if m.Href == "" {
		return http.StatusBadRequest, errors.New("Text is a required field")
	}

	m.Text = SanitiseText(m.Text)
	m.Text = strings.Trim(m.Text, " ")
	if m.Href == "" {
		return http.StatusBadRequest, errors.New("Text is a required field")
	}

	// Title
	if m.Title.Valid {
		m.Title.String = strings.Trim(m.Title.String, " ")
		m.Title.String = SanitiseText(m.Title.String)
		m.Title.String = strings.Trim(m.Title.String, " ")
		if m.Title.String == "" {
			m.Title.Valid = false
		}
	}

	if m.Sequence > 10 {
		return http.StatusBadRequest,
			errors.New("Menus are limited to 10 links")
	}

	return http.StatusOK, nil
}

func UpdateMenu(siteId int64, ems []h.LinkType) (int, error) {
	if len(ems) == 0 {
		return http.StatusBadRequest,
			errors.New("A menu without links is not a menu that is of any use." +
				" Have you tried DELETE?")
	}

	// We bring in []LinkType but we need []MenuType
	menu := []MenuType{}
	for ii, link := range ems {
		m := MenuType{}
		m.SiteId = siteId
		m.Href = link.Href
		m.Text = link.Text
		if link.Title != "" {
			m.Title.String = link.Title
			m.Title.Valid = true
		}
		m.Sequence = ii

		status, err := m.Validate()
		if err != nil {
			return status, err
		}

		menu = append(menu, m)
	}

	// Let's do it
	tx, err := h.GetTransaction()
	if err != nil {
		glog.Errorf("h.GetTransaction() %+v", err)
		return http.StatusInternalServerError, errors.New("Could not get transaction")
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
DELETE FROM menus
 WHERE site_id = $1`,
		siteId,
	)
	if err != nil {
		glog.Errorf("stmt1.Exec(%d) %+v", siteId, err)
		return http.StatusInternalServerError, errors.New("Failed to delete existing menu")
	}

	for _, m := range menu {

		_, err = tx.Exec(`
INSERT INTO menus (
    site_id
   ,href
   ,title
   ,"text"
   ,sequence
) VALUES (
    $1
   ,$2
   ,$3
   ,$4
   ,$5
)`,
			m.SiteId,
			m.Href,
			m.Title,
			m.Text,
			m.Sequence,
		)
		if err != nil {
			glog.Errorf(
				"stmt2.Exec(%d, %s, %v, %s, %d) %+v",
				m.SiteId,
				m.Href,
				m.Title,
				m.Text,
				m.Sequence,
				err,
			)
			return http.StatusInternalServerError,
				errors.New("Failed to delete existing menu")
		}
	}

	err = tx.Commit()
	if err != nil {
		glog.Errorf("tx.Commit() %+v", err)
		return http.StatusInternalServerError, errors.New("Transaction failed")
	}

	go PurgeCache(h.ItemTypes[h.ItemTypeSite], siteId)

	return http.StatusOK, nil
}

func DeleteMenu(siteId int64) (int, error) {
	tx, err := h.GetTransaction()
	if err != nil {
		glog.Errorf("h.GetTransaction() %+v", err)
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
DELETE FROM menus
 WHERE site_id = $1`,
		siteId,
	)
	if err != nil {
		glog.Errorf("stmt1.Exec(%d) %+v", siteId, err)
		return http.StatusInternalServerError,
			errors.New("Failed to delete existing menu")
	}

	err = tx.Commit()
	if err != nil {
		glog.Errorf("tx.Commit() %+v", err)
		return http.StatusInternalServerError,
			errors.New("Transaction failed")
	}

	go PurgeCache(h.ItemTypes[h.ItemTypeSite], siteId)

	return http.StatusOK, nil
}

func GetMenu(siteId int64) ([]h.LinkType, int, error) {
	db, err := h.GetConnection()
	if err != nil {
		return []h.LinkType{}, http.StatusInternalServerError, err
	}

	rows, err := db.Query(`
SELECT href
      ,title
      ,"text"
  FROM menus
 WHERE site_id = $1
 ORDER BY sequence ASC`,
		siteId,
	)
	if err != nil {
		glog.Errorf("tx.Query(%d) %+v", siteId, err)
		return []h.LinkType{}, http.StatusInternalServerError,
			errors.New("Database query failed")
	}
	defer rows.Close()

	ems := []h.LinkType{}
	for rows.Next() {
		m := h.LinkType{}
		s := sql.NullString{}
		err = rows.Scan(
			&m.Href,
			&s,
			&m.Text,
		)
		if err != nil {
			glog.Errorf("rows.Scan() %+v", err)
			return []h.LinkType{}, http.StatusInternalServerError,
				errors.New("Row parsing error")
		}

		if s.Valid {
			m.Title = s.String
		}

		ems = append(ems, m)
	}
	err = rows.Err()
	if err != nil {
		glog.Errorf("rows.Err() %+v", err)
		return []h.LinkType{}, http.StatusInternalServerError,
			errors.New("Error fetching rows")
	}
	rows.Close()

	return ems, http.StatusOK, nil
}
