package models

import (
	"bytes"
	"encoding/json"
	"errors"
	"html"
	"net/http"
	"net/url"
	"strings"
	"text/template"

	"github.com/golang/glog"

	conf "github.com/microcosm-cc/microcosm/config"
)

const (
	EMAIL_FROM string = `%s <notify@microco.sm>`

	EMAIL_HTML_CONTAINER_HEADER string = `<!DOCTYPE html>
<meta charset="utf-8"><div>`

	EMAIL_HTML_CONTAINER_FOOTER string = `</div>`
)

type EmailType struct {
	From     string
	ReplyTo  string
	To       string
	Subject  string
	BodyText string
	BodyHTML string
}

func MergeAndSendEmail(
	siteId int64,
	from string,
	to string,
	subjectTemplate *template.Template,
	textTemplate *template.Template,
	htmlTemplate *template.Template,
	data interface{},
) (int, error) {

	// If we are not prod environment we really never want to send emails
	// by accident as we may be spamming people. This is by whitelist, if this
	// isn't the production environment then only @microcosm.cc recipients will
	// get the email.
	//
	// If you need to test emails to specific external email hosts then you
	// will need to consciously do so by white-listing them
	if conf.CONFIG_STRING[conf.KEY_ENVIRONMENT] != `prod` &&
		!strings.Contains(to, "@microcosm.cc") {

		glog.Infof("dev environment, skipping email to %s", to)
		return http.StatusOK, nil
	}

	var email = EmailType{}

	email.From = from

	email.To = to

	var emailSubject bytes.Buffer
	err := subjectTemplate.Execute(&emailSubject, data)
	if err != nil {
		glog.Errorf("%s %+v", "subjectTemplate.Execute()", err)
		return http.StatusInternalServerError, err
	}
	email.Subject = html.UnescapeString(emailSubject.String())

	var emailText bytes.Buffer
	err = textTemplate.Execute(&emailText, data)
	if err != nil {
		glog.Errorf("%s %+v", "textTemplate.Execute()", err)
		return http.StatusInternalServerError, err
	}
	email.BodyText = html.UnescapeString(emailText.String())

	var emailHTML bytes.Buffer
	err = htmlTemplate.Execute(&emailHTML, data)
	if err != nil {
		glog.Errorf("%s %+v", "htmlTemplate.Execute()", err)
		return http.StatusInternalServerError, err
	}
	email.BodyHTML = emailHTML.String()

	return email.Send(siteId)
}

//SendEmail uses mailgun to send an email and logs any errors.
func (m *EmailType) Send(siteId int64) (int, error) {

	if m.From == "" || m.To == "" {
		return http.StatusPreconditionFailed,
			errors.New("Cannot send an email without " +
				"both from: and to: email addresses")
	}

	if m.Subject == "" && m.BodyText == "" && m.BodyHTML == "" {
		return http.StatusPreconditionFailed,
			errors.New("Not willing to send a blank email")
	}

	formBody := url.Values{}
	formBody.Set("from", m.From)

	if m.ReplyTo != "" {
		formBody.Set("h:Reply-To", m.ReplyTo)
	}

	formBody.Set("to", m.To)
	formBody.Set("subject", m.Subject)
	formBody.Set("text", m.BodyText)
	formBody.Set(
		"html",
		EMAIL_HTML_CONTAINER_HEADER+
			AnchorRelativeUrls(siteId, m.BodyHTML)+
			EMAIL_HTML_CONTAINER_FOOTER,
	)

	req, err := http.NewRequest(
		"POST",
		conf.CONFIG_STRING[conf.KEY_MAILGUN_API_URL],
		strings.NewReader(formBody.Encode()),
	)

	req.Header.Set(
		"Content-Type",
		"application/x-www-form-urlencoded; charset=UTF-8",
	)

	req.SetBasicAuth("api", conf.CONFIG_STRING[conf.KEY_MAILGUN_API_KEY])

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		glog.Errorf("Failed to send email: %s", err.Error())
		return http.StatusInternalServerError, err
	}
	defer resp.Body.Close()

	type MailgunResp struct {
		Id      string `json:"id"`
		Message string `json:"message"`
	}

	parsedResp := MailgunResp{}
	err = json.NewDecoder(resp.Body).Decode(&parsedResp)
	if err != nil {
		glog.Errorf("Failed to read mailgun response: %s", err.Error())
		return http.StatusInternalServerError, err
	}

	glog.Infof("Success: %v %s", parsedResp.Id, parsedResp.Message)

	return http.StatusOK, nil
}

// Takes a HTML string that contains links like: href="/profiles/" and adds the
// given site's absolute URL so that it becomes:
// href="https://key.microco.sm/profiles/"
func AnchorRelativeUrls(siteId int64, bodyText string) string {
	site, _, err := GetSite(siteId)
	if err != nil {
		glog.Errorf("Failed to get site: %+v", err)
		return bodyText
	}

	siteUrl := site.GetURL()

	const (
		HREF_FIND string = `a href="/`
		SRC_FIND  string = `img src="/`
	)
	var (
		HREF_REPLACE string = `a href="` + siteUrl + `/`
		SRC_REPLACE  string = `img src="` + siteUrl + `/`
	)

	return strings.Replace(
		strings.Replace(
			bodyText,
			SRC_FIND,
			SRC_REPLACE,
			-1,
		),
		HREF_FIND,
		HREF_REPLACE,
		-1,
	)
}
