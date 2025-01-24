package redirector

import (
	"strings"

	"github.com/cloudflare/ahocorasick"

	"github.com/microcosm-collective/microcosm/models"
)

var affDomainParts = append(
	append(
		[]string{},
		ebayDomainParts...,
	),
	amazonDomainParts...,
)

func affiliateMayExist(domain string) bool {
	domains := ahocorasick.NewStringMatcher(affDomainParts)
	hits := domains.Match([]byte(strings.ToLower(domain)))

	return !(len(hits) == 0)
}

func getAffiliateLink(link models.Link) string {

	// We look for affiliate links to strip the tracking

	// Ebay Partner Network
	if !(len(ahocorasick.NewStringMatcher(ebayDomainParts).Match([]byte(strings.ToLower(link.Domain)))) == 0) {
		m := ebayLink{Link: link}
		if ok, u := m.getDestination(); ok {
			return u
		}
	}

	// Amazon
	if !(len(ahocorasick.NewStringMatcher(amazonDomainParts).Match([]byte(strings.ToLower(link.Domain)))) == 0) {
		m := amazonLink{Link: link}
		if ok, u := m.getDestination(); ok {
			return u
		}
	}

	return link.URL
}
