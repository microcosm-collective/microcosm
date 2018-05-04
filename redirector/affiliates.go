package redirector

import (
	"strings"

	"github.com/cloudflare/ahocorasick"

	"github.com/microcosm-cc/microcosm/models"
)

var affDomainParts = append(
	append(
		append(
			append(
				[]string{},
				lfgssDomainParts...,
			),
			affwinDomainParts...,
		),
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

	// Affiliate Window
	if !(len(ahocorasick.NewStringMatcher(affwinDomainParts).Match([]byte(strings.ToLower(link.Domain)))) == 0) {
		m := affWinLink{Link: link}
		if ok, u := m.getDestination(); ok {
			return u
		}
	}

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

	// Site specific, LFGSS
	if !(len(ahocorasick.NewStringMatcher(lfgssDomainParts).Match([]byte(strings.ToLower(link.Domain)))) == 0) {
		m := lfgssLink{Link: link}
		if ok, u := m.getDestination(); ok {
			return u
		}
	}

	return link.URL
}
