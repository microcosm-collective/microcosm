package models

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/golang/glog"

	h "github.com/microcosm-cc/microcosm/helpers"
)

// MenuType describes a menu item
type MenuType struct {
	SiteID   int64
	Href     string
	Text     string
	Title    sql.NullString
	Sequence int
}

// Validate returns true if the menu item is valid
func (m *MenuType) Validate() (int, error) {

	// SiteID
	if m.SiteID == 0 {
		return http.StatusBadRequest, fmt.Errorf("siteID is a required field")
	}

	// Href
	m.Href = strings.Trim(m.Href, " ")
	if m.Href == "" {
		return http.StatusBadRequest, fmt.Errorf("href is a required field")
	}

	u, err := url.Parse(m.Href)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("href is not a valid URL")
	}
	m.Href = u.String()

	// Text
	m.Text = strings.Trim(m.Text, " ")
	if m.Href == "" {
		return http.StatusBadRequest, fmt.Errorf("text is a required field")
	}

	preventShouting := false
	m.Text = CleanSentence(m.Text, preventShouting)
	m.Text = strings.Trim(m.Text, " ")
	if m.Href == "" {
		return http.StatusBadRequest, fmt.Errorf("text is a required field")
	}

	// Title
	if m.Title.Valid {
		m.Title.String = strings.Trim(m.Title.String, " ")
		m.Title.String = CleanSentence(m.Title.String, preventShouting)
		if m.Title.String == "" {
			m.Title.Valid = false
		}
	}

	if m.Sequence > 10 {
		return http.StatusBadRequest,
			fmt.Errorf("menus are limited to 10 links")
	}

	return http.StatusOK, nil
}

// UpdateMenu saves the full menu
func UpdateMenu(siteID int64, ems []h.LinkType) (int, error) {
	if len(ems) == 0 {
		return http.StatusBadRequest,
			fmt.Errorf("a menu without links is not a menu that is of any use." +
				" Have you tried DELETE?")
	}

	// We bring in []LinkType but we need []MenuType
	menu := []MenuType{}
	for ii, link := range ems {
		m := MenuType{}
		m.SiteID = siteID
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
		return http.StatusInternalServerError, fmt.Errorf("could not get transaction")
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
DELETE FROM menus
 WHERE site_id = $1`,
		siteID,
	)
	if err != nil {
		glog.Errorf("stmt1.Exec(%d) %+v", siteID, err)
		return http.StatusInternalServerError, fmt.Errorf("failed to delete existing menu")
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
			m.SiteID,
			m.Href,
			m.Title,
			m.Text,
			m.Sequence,
		)
		if err != nil {
			glog.Errorf(
				"stmt2.Exec(%d, %s, %v, %s, %d) %+v",
				m.SiteID,
				m.Href,
				m.Title,
				m.Text,
				m.Sequence,
				err,
			)
			return http.StatusInternalServerError,
				fmt.Errorf("failed to delete existing menu")
		}
	}

	err = tx.Commit()
	if err != nil {
		glog.Errorf("tx.Commit() %+v", err)
		return http.StatusInternalServerError, fmt.Errorf("transaction failed")
	}

	go PurgeCache(h.ItemTypes[h.ItemTypeSite], siteID)

	return http.StatusOK, nil
}

// DeleteMenu removes the entire menu
func DeleteMenu(siteID int64) (int, error) {
	tx, err := h.GetTransaction()
	if err != nil {
		glog.Errorf("h.GetTransaction() %+v", err)
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
DELETE FROM menus
 WHERE site_id = $1`,
		siteID,
	)
	if err != nil {
		glog.Errorf("stmt1.Exec(%d) %+v", siteID, err)
		return http.StatusInternalServerError,
			fmt.Errorf("failed to delete existing menu")
	}

	err = tx.Commit()
	if err != nil {
		glog.Errorf("tx.Commit() %+v", err)
		return http.StatusInternalServerError,
			fmt.Errorf("transaction failed")
	}

	go PurgeCache(h.ItemTypes[h.ItemTypeSite], siteID)

	return http.StatusOK, nil
}

// GetMenu returns a menu for a site
func GetMenu(siteID int64) ([]h.LinkType, int, error) {
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
		siteID,
	)
	if err != nil {
		glog.Errorf("tx.Query(%d) %+v", siteID, err)
		return []h.LinkType{}, http.StatusInternalServerError,
			fmt.Errorf("database query failed")
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
				fmt.Errorf("row parsing error")
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
			fmt.Errorf("error fetching rows")
	}
	rows.Close()

	return ems, http.StatusOK, nil
}
