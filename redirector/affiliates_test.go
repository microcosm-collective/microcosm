package redirector

import (
	"testing"

	"github.com/microcosm-cc/microcosm/models"
)

func TestAffiliatesMatching(t *testing.T) {

	m := models.Link{
		Domain: "www.chainreactioncycles.com",
		URL:    "http://www.chainreactioncycles.com/michelin-pro4-service-course-road-bike-tyre/rp-prod73626",
	}

	if !affiliateMayExist(m.Domain) {
		t.Error(`affiliateMayExist("www.chainreactioncycles.com") should be true`)
	}

	s := getAffiliateLink(m)
	if s != `http://www.awin1.com/cread.php?awinaffid=101164&awinmid=2698&clickref=&p=http%3A%2F%2Fwww.chainreactioncycles.com%2Fmichelin-pro4-service-course-road-bike-tyre%2Frp-prod73626` {
		t.Error("Chain Reaction URL (Affiliate Window) did not match expected value")
	}
}
