package redirector

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/golang/glog"

	h "github.com/microcosm-cc/microcosm/helpers"
	"github.com/microcosm-cc/microcosm/models"
)

// GetRedirect will return a link for a given short URL.
func GetRedirect(shortURL string) (models.Link, int, error) {

	db, err := h.GetConnection()
	if err != nil {
		glog.Errorf("h.GetConnection() %+v", err)
		return models.Link{}, http.StatusInternalServerError, err
	}

	sqlQuery := `
UPDATE links
   SET hits = hits + 1
 WHERE short_url = $1
RETURNING
       link_id,
       short_url,
       domain,
       url,
       inner_text,
       created,
       resolved_url,
       resolved,
       hits;`

	stmt, err := db.Prepare(sqlQuery)
	if err != nil {
		glog.Errorf("db.Prepare(`%s`) %+v", sqlQuery, err)
		return models.Link{}, http.StatusInternalServerError,
			errors.New("Could not prepare statement")
	}
	defer stmt.Close()

	rows, err := stmt.Query(shortURL)
	if err != nil {
		glog.Errorf("stmt.Query(%s) %+v", shortURL, err)
		return models.Link{}, http.StatusInternalServerError,
			errors.New("Database query failed")
	}
	defer rows.Close()

	var m models.Link
	for rows.Next() {
		m = models.Link{}
		err = rows.Scan(
			&m.Id,
			&m.ShortUrl,
			&m.Domain,
			&m.Url,
			&m.Text,
			&m.Created,
			&m.ResolvedUrl,
			&m.Resolved,
			&m.Hits,
		)
		if err != nil {
			glog.Errorf("rows.Scan() %+v", err)
			return models.Link{}, http.StatusInternalServerError,
				errors.New("Row parsing error")
		}
	}
	err = rows.Err()
	if err != nil {
		glog.Errorf("rows.Err() %+v", err)
		return models.Link{}, http.StatusInternalServerError,
			errors.New("Error fetching rows")
	}

	if m.Id == 0 {
		glog.Errorf("m.Id == 0 for URL %s", shortURL)
		return models.Link{}, http.StatusNotFound,
			fmt.Errorf("URL %s%s not found", h.JumpUrl, shortURL)
	}

	if affiliateMayExist(m.Domain) {
		m.Url = getAffiliateLink(m)
	}

	//glog.Infof("Found models.link %s redirecting to %s", shortURL, m.Url)

	return m, http.StatusOK, nil
}
