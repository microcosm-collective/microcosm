package models

import (
	"database/sql"
	"fmt"
	"net/http"
	"text/template"

	"github.com/golang/glog"

	c "github.com/microcosm-cc/microcosm/cache"
	h "github.com/microcosm-cc/microcosm/helpers"
)

// UpdateTypesType is an unwieldy name, but it convenes to how all other
// table structs are named (the table is `UpdateType`)
type UpdateTypesType struct {
	ID            int64  `json:"id"`
	Title         string `json:"title"`
	Description   string `json:"description"`
	EmailSubject  string `json:"emailSubject"`
	EmailBodyText string `json:"emailBodyText"`
	EmailBodyHTML string `json:"emailBodyHtml"`
}

// GetUpdateType retrieves an email update template for a given update type
func GetUpdateType(updateTypeID int64) (UpdateTypesType, int, error) {

	// Try fetching from cache
	mcKey := fmt.Sprintf(mcUpdateTypeKeys[c.CacheDetail], updateTypeID)
	if val, ok := c.Get(mcKey, UpdateTypesType{}); ok {
		m := val.(UpdateTypesType)
		return m, http.StatusOK, nil
	}

	db, err := h.GetConnection()
	if err != nil {
		return UpdateTypesType{}, http.StatusInternalServerError, err
	}

	var m UpdateTypesType
	err = db.QueryRow(`
SELECT update_type_id
      ,title
      ,description
      ,email_subject
      ,email_body_text
      ,email_body_html
  FROM update_types
 WHERE update_type_id = $1`,
		updateTypeID,
	).Scan(
		&m.ID,
		&m.Title,
		&m.Description,
		&m.EmailSubject,
		&m.EmailBodyText,
		&m.EmailBodyHTML,
	)
	if err == sql.ErrNoRows {
		return UpdateTypesType{}, http.StatusNotFound,
			fmt.Errorf("resource with update type ID %d not found", updateTypeID)
	} else if err != nil {
		return UpdateTypesType{}, http.StatusInternalServerError,
			fmt.Errorf("database query failed: %v", err.Error())
	}

	c.Set(mcKey, m, mcTTL)

	return m, http.StatusOK, nil
}

// GetEmailTemplates returns the email templates for an update type
func (m *UpdateTypesType) GetEmailTemplates() (
	*template.Template,
	*template.Template,
	*template.Template,
	int,
	error,
) {

	var (
		subjectTemplate  *template.Template
		bodyTextTemplate *template.Template
		bodyHTMLTemplate *template.Template
		err              error
	)

	subjectTemplate, err =
		template.New("email_subject").Parse(m.EmailSubject)
	if err != nil {
		glog.Errorf("Could not Subject get template: %+v", err)
		return subjectTemplate, bodyTextTemplate, bodyHTMLTemplate,
			http.StatusInternalServerError, err
	}

	bodyTextTemplate, err =
		template.New("email_body_text").Parse(m.EmailBodyText)
	if err != nil {
		glog.Errorf("Could not Text get template: %+v", err)
		return subjectTemplate, bodyTextTemplate, bodyHTMLTemplate,
			http.StatusInternalServerError, err
	}

	bodyHTMLTemplate, err =
		template.New("email_body_html").Parse(m.EmailBodyHTML)
	if err != nil {
		glog.Errorf("Could not HTML get template: %+v", err)
		return subjectTemplate, bodyTextTemplate, bodyHTMLTemplate,
			http.StatusInternalServerError, err
	}

	return subjectTemplate,
		bodyTextTemplate,
		bodyHTMLTemplate,
		http.StatusOK,
		nil
}
