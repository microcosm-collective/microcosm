package redirector

import (
	"fmt"
	"net/url"
	"regexp"

	"github.com/golang/glog"

	"git.dee.kitchen/buro9/microcosm/models"
)

// https://developer.ebay.com/devzone/shopping/docs/callref/getsingleitem.html
// Max length: 19 (Note: The eBay database specifies 38. Currently, Item IDs are usually 9 to 12 digits).
var ebayItemIDRegexp = regexp.MustCompile("[0-9]{9,19}")

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
		q.Del("campid")
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
	itemID := getEbayItemIDFromURL(m.Link.URL)
	if isEbayLink && itemID != "" {
		u, _ := url.Parse(fmt.Sprintf(`https://www.ebay.co.uk/itm/%s`, itemID))
		q := u.Query()
		q.Del("mkevt")
		q.Del("mkcid")
		q.Del("mkrid")
		q.Del("campid")
		q.Del("toolid")
		u.RawQuery = q.Encode()

		return true, u.String()
	}

	u, err := url.Parse(m.Link.URL)
	if err != nil {
		glog.Errorf("url.Parse(`%s`) %+v", m.Link.URL, err)
		return false, m.Link.URL
	}

	q := u.Query()
	q.Del("mkevt")
	q.Del("mkcid")
	q.Del("mkrid")
	q.Del("campid")
	q.Del("toolid")
	u.RawQuery = q.Encode()

	return true, u.String()
}

func getEbayItemIDFromURL(str string) string {
	u, _ := url.Parse(str)
	q := u.Query()
	q.Del("mkevt")
	q.Del("mkcid")
	q.Del("mkrid")
	q.Del("campid")
	q.Del("toolid")
	u.RawQuery = q.Encode()
	return ebayItemIDRegexp.FindString(u.String())
}
