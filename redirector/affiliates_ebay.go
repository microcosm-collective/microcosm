package redirector

import (
	"fmt"
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
		u, _ := url.Parse(fmt.Sprintf(`https://www.ebay.co.uk/itm/%s`, itemID))
		q := u.Query()

		q.Add("mkrid", "710-53481-19255-0")
		q.Add("siteid", "3")
		q.Add("mkcid", "1")
		q.Add("campid", ebayCampaignID)
		q.Add("toolid", "1001")
		q.Add("mkevt", "1")

		u.RawQuery = q.Encode()

		return true, u.String()
	}

	u, err := url.Parse(m.Link.URL)
	if err != nil {
		glog.Errorf("url.Parse(`%s`) %+v", m.Link.URL, err)
		return false, m.Link.URL
	}

	q := u.Query()
	q.Del("mkrid")
	q.Add("mkrid", "710-53481-19255-0")
	q.Del("campid")
	q.Add("campid", ebayCampaignID)
	q.Del("siteid")
	q.Add("siteid", "3")
	q.Del("mkcid")
	q.Add("mkcid", "1")
	q.Del("toolid")
	q.Add("toolid", "1001")
	q.Del("mkevt")
	q.Add("mkevt", "1")
	u.RawQuery = q.Encode()

	return true, u.String()
}
