package models

import (
	"bytes"
	"fmt"
	"net/http"
	"time"
)

// LegalDoc describes a legal document
type LegalDoc struct {
	Link         interface{} `json:"link,omitempty"`
	HTML         string      `json:"html,omitempty"`
	LastModified time.Time   `json:"lastModified,omitempty"`
}

// LegalData is used by the HTML template and provides the values of the strings
// to be inserted into the HTML templates
type LegalData struct {
	CustomerName  string
	CustomerURL   string
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

// GetLegalDocument returns a legal document according to the type requested
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
		return doc, http.StatusBadRequest, fmt.Errorf("document does not exist")
	}

	if err != nil {
		return doc, http.StatusInternalServerError, err
	}

	doc.HTML = buff.String()

	return doc, http.StatusOK, nil
}

// GetLegalDataForSite returns the template vars used by the legal docs
func GetLegalDataForSite(site SiteType) (LegalData, int, error) {
	data := LegalData{}

	// Customer info
	data.CustomerName = site.Title
	data.CustomerURL = site.GetURL()

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
	data.LinkToCookiePolicy = data.CustomerURL + "/legal/cookies"
	data.LinkToFees = "http://microcosm.app/compare"
	data.LinkToOpenSourceLicenses =
		"https://github.com/microcosm-collective/microweb/blob/master/OPENSOURCE.md"
	data.LinkToPrivacyPolicy = data.CustomerURL + "/legal/privacy"

	// Links in forum content
	data.LinkToAccountDeactivation = "http://meta.microcosm.app"
	data.LinkToBestPractises = "http://meta.microcosm.app"
	data.LinkToVerificationPolicy = "http://meta.microcosm.app"

	return data, http.StatusOK, nil
}
