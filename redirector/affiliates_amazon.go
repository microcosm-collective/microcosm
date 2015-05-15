package redirector

import (
	"net/url"

	"github.com/golang/glog"

	"github.com/microcosm-cc/microcosm/models"
)

const (
	amazonCampaignID string = "1634"
	amazonTagID      string = "buro9"
	amazonCreativeID string = "6738"
)

var amazonDomainParts = []string{
	".amazon.",
}

type amazonLink struct {
	Link models.Link
}

func (m *amazonLink) getDestination() (bool, string) {

	var (
		isAmazonLink bool
	)

	switch m.Link.Domain {
	case "www.amazon.co.uk":
		isAmazonLink = true
	}

	if !isAmazonLink {
		return false, m.Link.URL
	}

	u, err := url.Parse(m.Link.URL)
	if err != nil {
		glog.Errorf("url.Parse(`%s`) %+v", m.Link.URL, err)
		return false, m.Link.URL
	}

	// Create our affiliate link
	q := u.Query()
	q.Del("camp")
	q.Add("camp", amazonCampaignID)
	q.Del("tag")
	q.Add("tag", amazonTagID)
	q.Del("creative")
	q.Add("creative", amazonCreativeID)
	q.Del("linkCode")
	q.Del("linkId")
	u.RawQuery = q.Encode()

	return true, u.String()
}
