package redirector

import (
	"net/url"

	"github.com/microcosm-cc/microcosm/models"
)

var lfgssDomainParts = []string{
	"bikmo.com",
}

type lfgssLink struct {
	Link models.Link
}

func (m *lfgssLink) getDestination() (bool, string) {
	// Construct a link based on domain
	switch m.Link.Domain {
	case "bikmo.com":
		u, _ := url.Parse(m.Link.URL)
		q := u.Query()
		q.Set("ref", "lfgss")
		u.RawQuery = q.Encode()
		return true, u.String()
	default:
		return false, m.Link.URL
	}
}
