package models

import (
	"bytes"
	"errors"
	"net/http"
	"time"
)

type LegalDoc struct {
	Link         interface{} `json:"link,omitempty"`
	Html         string      `json:"html,omitempty"`
	LastModified time.Time   `json:"lastModified,omitempty"`
}

type LegalData struct {
	CustomerName  string
	CustomerUrl   string
	CustomerEmail string

	LinkToAccountDeactivation string
	LinkToBestPractises       string
	LinkToCookiePolicy        string
	LinkToFees                string
	LinkToOpenSourceLicenses  string
	LinkToPrivacyPolicy       string
	LinkToVerificationPolicy  string

	CookiePolicyLastModified     string
	PrivacyPolicyLastModified    string
	ServiceAgreementLastModified string
	TermsOfUseLastModified       string

	LegalEntity      string
	MicrocosmEmail   string
	MicrocosmAddress string
}

func GetLegalDocument(
	site SiteType,
	documentRequested string,
) (
	LegalDoc,
	int,
	error,
) {

	doc := LegalDoc{}

	data, status, err := GetLegalDataForSite(site)
	if err != nil {
		return doc, status, err
	}

	var buff bytes.Buffer

	switch documentRequested {
	case "cookies":
		err = legalCookiePolicy.Execute(&buff, data)
		doc.LastModified = legalCookiePolicyLastModified
	case "privacy":
		err = legalPrivacyPolicy.Execute(&buff, data)
		doc.LastModified = legalPrivacyPolicyLastModified
	case "service":
		err = legalServiceAgreement.Execute(&buff, data)
		doc.LastModified = legalServiceAgreementLastModified
	case "terms":
		err = legalTermsOfUse.Execute(&buff, data)
		doc.LastModified = legalTermsOfUseLastModified
	default:
		return doc, http.StatusBadRequest, errors.New("Document does not exist")
	}

	if err != nil {
		return doc, http.StatusInternalServerError, err
	}

	doc.Html = buff.String()

	return doc, http.StatusOK, nil
}

func GetLegalDataForSite(site SiteType) (LegalData, int, error) {
	data := LegalData{}

	// Customer info
	data.CustomerName = site.Title
	data.CustomerUrl = site.GetURL()

	profile, status, err := GetProfile(site.ID, site.OwnedByID)
	if err != nil {
		return data, status, err
	}

	user, status, err := GetUser(profile.UserID)
	if err != nil {
		return data, status, err
	}

	data.CustomerEmail = user.Email

	// Modified dates
	data.CookiePolicyLastModified = legalCookiePolicyLastModified.String()
	data.PrivacyPolicyLastModified = legalPrivacyPolicyLastModified.String()
	data.ServiceAgreementLastModified =
		legalServiceAgreementLastModified.String()
	data.TermsOfUseLastModified = legalTermsOfUseLastModified.String()

	// Microcosm
	data.LegalEntity = "Microcosm"
	data.MicrocosmEmail = "support@microcosm.cc"
	data.MicrocosmAddress = "74 Fraser House, Green Dragon Lane, London TW8 0DQ"

	// Links
	data.LinkToCookiePolicy = data.CustomerUrl + "/legal/cookies"
	data.LinkToFees = "http://microco.sm/compare"
	data.LinkToOpenSourceLicenses =
		"https://github.com/microcosm-cc/microweb/blob/master/OPENSOURCE.md"
	data.LinkToPrivacyPolicy = data.CustomerUrl + "/legal/privacy"

	// Links in forum content
	data.LinkToAccountDeactivation = "http://meta.microco.sm"
	data.LinkToBestPractises = "http://meta.microco.sm"
	data.LinkToVerificationPolicy = "http://meta.microco.sm"

	return data, http.StatusOK, nil
}
