package models

import (
	"github.com/microcosm-cc/bluemonday"
)

var textPolicy = bluemonday.StripTagsPolicy()
var htmlPolicy = bluemonday.UGCPolicy()
var initHTMLPolicy bool

// SanitiseHTML strips any HTML not on the cleanse whitelist, leaving a safe
// set of HTML intact that is not going to pose an XSS risk
func SanitiseHTML(src []byte) []byte {
	if !initHTMLPolicy {
		htmlPolicy.RequireNoFollowOnLinks(false)
		htmlPolicy.RequireNoFollowOnFullyQualifiedLinks(true)
		htmlPolicy.AddTargetBlankToFullyQualifiedLinks(true)
		initHTMLPolicy = true
	}

	return htmlPolicy.SanitizeBytes(src)
}

// SanitiseText strips all HTML tags from text
func SanitiseText(s string) string {
	return textPolicy.Sanitize(s)
}
