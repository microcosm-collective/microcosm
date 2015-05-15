package redirector

import (
	"net/url"
	"regexp"

	"github.com/golang/glog"

	"github.com/microcosm-cc/microcosm/models"
)

const (
	ebayPublisherID string = "5574889051"
	ebayCampaignID  string = "5336525415"
)

var ebayItemIDRegexp = regexp.MustCompile("[0-9]{10,}")

var ebayDomainParts = []string{
	".ebay.",
	"half.com",
}

type ebayLink struct {
	Link models.Link
}

func (m *ebayLink) getDestination() (bool, string) {

	// Hijack an existing affiliate link
	if m.Link.Domain == "rover.ebay.com" {
		u, err := url.Parse(m.Link.URL)
		if err != nil {
			glog.Errorf("url.Parse(`%s`) %+v", m.Link.URL, err)
			return false, m.Link.URL
		}

		q := u.Query()
		q.Del("pub")
		q.Add("pub", ebayPublisherID)
		q.Del("campid")
		q.Add("campid", ebayCampaignID)
		u.RawQuery = q.Encode()

		return true, u.String()
	}

	var (
		isEbayLink bool
		isHalfLink bool
	)

	switch m.Link.Domain {
	case "www.ebay.com":
		isEbayLink = true
	case "www.ebay.ie":
		isEbayLink = true
	case "www.ebay.at":
		isEbayLink = true
	case "www.ebay.au":
		isEbayLink = true
	case "www.ebay.be":
		isEbayLink = true
	case "www.ebay.ca":
		isEbayLink = true
	case "www.ebay.fr":
		isEbayLink = true
	case "www.ebay.com.de":
		isEbayLink = true
	case "www.ebay.it":
		isEbayLink = true
	case "www.ebay.es":
		isEbayLink = true
	case "www.ebay.ch":
		isEbayLink = true
	case "www.ebay.co.uk":
		isEbayLink = true
	case "www.ebay.nl":
		isEbayLink = true
	case "www.half.com":
		isHalfLink = true
	}

	if !isEbayLink && !isHalfLink {
		return false, m.Link.URL
	}

	// Determine if we have an itemID, which is a 64-bit integer currently at
	// least 10 digits long, in the URL.
	// If so, we will want to link directly to the item rather than use the
	// custom url link.
	itemID := ebayItemIDRegexp.FindString(m.Link.URL)
	if isEbayLink && itemID != "" {
		u, _ := url.Parse("http://rover.ebay.com/rover/1/710-53481-19255-0/1")
		q := u.Query()

		// These do not vary
		q.Add("toolid", "1001")
		q.Add("icep_ff3", "2")
		q.Add("ipn", "psmain")
		q.Add("icep_vectorid", "229508")
		q.Add("kwid", "902099")
		q.Add("mtid", "824")
		q.Add("kw", "lg")

		// These vary
		q.Add("campid", ebayCampaignID)
		q.Add("pub", ebayPublisherID)
		q.Add("icep_item", itemID)

		u.RawQuery = q.Encode()

		return true, u.String()
	}

	// Create our affiliate link
	u, _ := url.Parse("http://rover.ebay.com/rover/1/710-53481-19255-0/1")
	q := u.Query()

	// These do not vary
	q.Add("toolid", "1001")
	q.Add("ff3", "4")

	// These vary
	q.Add("campid", ebayCampaignID)
	q.Add("pub", ebayPublisherID)
	q.Add("mpre", m.Link.URL)

	u.RawQuery = q.Encode()

	return true, u.String()
}
