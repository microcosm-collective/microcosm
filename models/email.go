package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	netmail "net/mail"
	"net/url"
	"strings"
	"text/template"

	"github.com/golang/glog"
	sendgrid "github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"

	conf "github.com/microcosm-cc/microcosm/config"
)

const (
	notificationEmail string = `<notify@microco.sm>`
	emailFrom         string = `%s ` + notificationEmail

	emailHTMLHeader string = `<!DOCTYPE html>
<meta charset="utf-8"><div>`

	emailHTMLFooter string = `</div>`
)

// EmailType describes an email
type EmailType struct {
	From     string
	ReplyTo  string
	To       string
	Subject  string
	BodyText string
	BodyHTML string
}

// MergeAndSendEmail creates both parts of an email from database stored
// templates and then merges the metadata and sends them.
func MergeAndSendEmail(
	siteID int64,
	from string,
	to string,
	subjectTemplate *template.Template,
	textTemplate *template.Template,
	htmlTemplate *template.Template,
	data interface{},
) (int, error) {
	// If we are not prod environment we really never want to send emails
	// by accident as we may be spamming people if the database hasn't been
	// sanitised (which it shoud). This is by whitelist, if this isn't the
	// production environment then only @microcosm.cc recipients will
	// get the email.
	//
	// If you need to test emails to specific external email hosts then you
	// will need to consciously do so by doing so outside of this code
	if conf.ConfigStrings[conf.Environment] != `prod` &&
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

	return email.Send(siteID)
}

// Send uses mailgun to send an email and logs any errors.
func (m *EmailType) Send(siteID int64) (int, error) {
	//m.From = fmt.Sprintf(emailFrom, GetSiteTitle(siteID))
	m.From = notificationEmail
	f, err := netmail.ParseAddress(m.From)
	if err != nil {
		return http.StatusPreconditionFailed, err
	}
	m.From = f.String()

	if m.From == "" || m.To == "" {
		return http.StatusPreconditionFailed,
			fmt.Errorf("Cannot send an email without " +
				"both from: and to: email addresses")
	}

	if m.Subject == "" && m.BodyText == "" && m.BodyHTML == "" {
		return http.StatusPreconditionFailed,
			fmt.Errorf("Not willing to send a blank email")
	}

	if sendGridAPIKey, ok := conf.ConfigStrings[conf.SendGridAPIKey]; ok {
		// SendGrid has priority
		sgm := mail.NewV3MailInit(
			&mail.Email{Name: GetSiteTitle(siteID), Address: m.From},
			m.Subject,
			&mail.Email{Address: m.To},
			mail.NewContent("text/plain", m.BodyText),
		)
		sgm.AddContent(
			mail.NewContent(
				"text/html",
				emailHTMLHeader+AnchorRelativeUrls(siteID, m.BodyHTML)+emailHTMLFooter,
			),
		)

		req := sendgrid.GetRequest(
			sendGridAPIKey,
			"/v3/mail/send",
			"https://api.sendgrid.com",
		)
		req.Method = "POST"
		req.Body = mail.GetRequestBody(sgm)
		resp, err := sendgrid.API(req)
		if err != nil {
			glog.Errorf("SendGrid: %s", err.Error())
			return http.StatusInternalServerError, err
		}

		glog.Infof("SendGrid: success %d %s %s", resp.StatusCode, m.To, resp.Body)

	} else if mailgunAPIKey, ok := conf.ConfigStrings[conf.MailgunAPIKey]; ok {
		// Then Mailgun
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
			emailHTMLHeader+AnchorRelativeUrls(siteID, m.BodyHTML)+emailHTMLFooter,
		)

		// EmailType describes an email
		req, err := http.NewRequest(
			"POST",
			conf.ConfigStrings[conf.MailgunAPIURL],
			strings.NewReader(formBody.Encode()),
		)

		req.Header.Set(
			"Content-Type",
			"application/x-www-form-urlencoded; charset=UTF-8",
		)

		req.SetBasicAuth("api", mailgunAPIKey)

		client := &http.Client{}

		resp, err := client.Do(req)
		if err != nil {
			glog.Errorf("Failed to send email: %s", err.Error())
			return http.StatusInternalServerError, err
		}
		defer resp.Body.Close()

		type MailgunResp struct {
			ID      string `json:"id"`
			Message string `json:"message"`
		}

		parsedResp := MailgunResp{}
		err = json.NewDecoder(resp.Body).Decode(&parsedResp)
		if err != nil {
			glog.Errorf("Failed to read mailgun response: %s", err.Error())
			return http.StatusInternalServerError, err
		}

		glog.Infof("Success: %v %s", parsedResp.ID, parsedResp.Message)
	} else {
		glog.Warningf("No email provider configured")
	}

	return http.StatusOK, nil
}

// AnchorRelativeUrls takes a HTML string that contains links like:
//   href="/profiles/"
// and adds the given site's absolute URL so that it becomes:
//   href="https://key.microco.sm/profiles/"
func AnchorRelativeUrls(siteID int64, bodyText string) string {
	site, _, err := GetSite(siteID)
	if err != nil {
		glog.Errorf("Failed to get site: %+v", err)
		return bodyText
	}

	siteURL := site.GetURL()

	bodyText = strings.Replace(bodyText, `img src="/`, `img src="`+siteURL+`/`, -1)
	bodyText = strings.Replace(bodyText, `a href="/`, `a href="`+siteURL+`/`, -1)

	return bodyText
}
