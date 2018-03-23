package redirector

import (
	"net/url"
	"strings"

	"github.com/microcosm-cc/microcosm/models"
)

var lfgssDomainParts = []string{
	"bikmo.com",
	"wahoofitness.com",
}

type lfgssLink struct {
	Link models.Link
}

func (m *lfgssLink) getDestination() (bool, string) {
	// Construct a link based on domain
	switch {
	case m.Link.Domain == "bikmo.com":
		u, _ := url.Parse(m.Link.URL)
		q := u.Query()
		q.Set("ref", "lfgss")
		u.RawQuery = q.Encode()
		return true, u.String()
	case strings.Contains(m.Link.Domain, "wahoofitness.com"):
		u, _ := url.Parse(m.Link.URL)
		q := u.Query()
		q.Set("___store", "uk_english")
		q.Set("acc", "bf62768ca46b6c3b5bea9515d1a1fc45")
		u.RawQuery = q.Encode()
		return true, u.String()
	default:
		return false, m.Link.URL
	}
}
