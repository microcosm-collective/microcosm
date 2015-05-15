package models

import (
	"bytes"
	crand "crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math"
	mrand "math/rand"
	"net/url"
	"regexp"
	"strings"

	"github.com/golang/glog"
	"golang.org/x/net/html"

	conf "github.com/microcosm-cc/microcosm/config"

	h "github.com/microcosm-cc/microcosm/helpers"
)

var regUrlHead = regexp.MustCompile(`^(https?:\/\/)(www\.)?`)

func ProcessLinks(
	revisionId int64,
	src []byte,
	siteId int64,
) (
	[]byte,
	error,
) {

	site, _, _ := GetSite(siteId)

	if !bytes.Contains(src, []byte(`<a `)) {
		return src, nil
	}

	// Read and parse HTML
	doc, err := html.Parse(bytes.NewReader(src))
	if err != nil {
		return []byte{}, err
	}

	// Start the tree walk
	links := []Link{}
	err = ParseLinks(revisionId, site, doc, &links)
	if err != nil {
		return []byte{}, err
	}

	// Render the modified HTML tree
	b := new(bytes.Buffer)
	err = html.Render(b, doc)
	if err != nil {
		return []byte{}, err
	}

	// Pipe out, and because go.net/html gives us a full doc, convert
	// back to a fragment
	return b.Bytes(), nil
}

func ParseLinks(
	revisionId int64,
	site SiteType,
	element *html.Node,
	links *[]Link,
) error {

	// Strip markdown introduced element ID attributes
	if element.Type == html.ElementNode {
		// Convert links to shortUrls
		if element.Data == "a" {
			var titleAttr string
			attributes := element.Attr

			for ii, attribute := range attributes {

				if attribute.Key == "href" &&
					!strings.Contains(attribute.Val, h.JumpUrl) &&
					!strings.HasPrefix(attribute.Val, "mailto:") {

					u, err := url.Parse(attribute.Val)
					if err != nil {
						// It's not a valid URL, so let's not link it
						break
					}
					fullUrl := u.String()
					host := u.Host
					if host == "" {
						break
					}

					if element.FirstChild == nil {
						// If there's nothing in this anchor then this link does
						// nothing
						break
					}

					shortUrl, text, title, err := ShortenLink(
						revisionId,
						fullUrl,
						host,
						site,
						element.FirstChild.Data,
						links,
					)
					if err != nil {
						return err
					}

					// Write a title so people know where they're going
					titleAttr = title

					// Write our new link and text to the anchor
					attribute.Val = shortUrl
					attributes[ii] = attribute

					element.FirstChild.Data = text
					element.Attr = attributes
					break
				}
			}

			// Add the title if we have one to add (i.e. a shortened link)
			if titleAttr != "" {
				var found bool

				for ii, attribute := range attributes {
					// Update existing title attr
					if attribute.Key == "title" {
						attribute.Val = titleAttr
						attributes[ii] = attribute
						found = true
						break
					}
				}
				if !found {
					// Add new title attr
					attributes = append(
						attributes,
						html.Attribute{Key: "title", Val: titleAttr},
					)
				}

				element.Attr = attributes
			}
		}
	}

	// Walk the tree
	for child := element.FirstChild; child != nil; child = child.NextSibling {
		err := ParseLinks(revisionId, site, child, links)
		if err != nil {
			return err
		}
	}

	return nil
}

func ShortenLink(
	revisionId int64,
	fullUrl string,
	host string,
	site SiteType,
	text string,
	links *[]Link,
) (
	string,
	string,
	string,
	error,
) {

	// For the text...
	// strip URLs of protocol (to encourage clicking on the link and to make
	// it prettier)

	var addTitle bool
	if text == "" {
		text = fullUrl
	} else {
		if text != fullUrl {
			addTitle = true
		}
	}

	// Don't process intra-site links
	//
	// We basically convert fully qualified URLs into absolute URLs by stripping
	// the prefix
	if site.Domain == "" {
		// If site.Domain were not blank this would cause issues as it would
		// break /api/v1/files/* links.
		prefix := "https://" + site.SubdomainKey + conf.CONFIG_STRING[conf.KEY_MICROCOSM_DOMAIN]
		if strings.HasPrefix(fullUrl, prefix) {
			if len(fullUrl) > len(prefix) {
				fullUrl = fullUrl[len(prefix):]
				if fullUrl == "." {
					fullUrl = "/"
				}
				return fullUrl, text, "", nil
			} else {
				return "/", text, "", nil
			}
		}
	} else {
		// We should not shortern this... it's a link to a file we know about,
		// an attachment or something.
		prefix := "https://" + site.SubdomainKey + conf.CONFIG_STRING[conf.KEY_MICROCOSM_DOMAIN]
		if strings.HasPrefix(fullUrl, prefix) {
			return fullUrl, text, "", nil
		}

		// We handle both the http and https as we cannot know what how it will
		// be displayed in future.
		prefix = "http://" + site.Domain
		if strings.HasPrefix(fullUrl, prefix) {
			if len(fullUrl) > len(prefix) {
				fullUrl = fullUrl[len(prefix):]
				if fullUrl == "." {
					fullUrl = "/"
				}
				return fullUrl, text, "", nil
			} else {
				return "/", text, "", nil
			}
		}
		prefix = "https://" + site.Domain
		if strings.HasPrefix(fullUrl, prefix) {
			if len(fullUrl) > len(prefix) {
				fullUrl = fullUrl[len(prefix):]
				if fullUrl == "." {
					fullUrl = "/"
				}
				return fullUrl, text, "", nil
			} else {
				return "/", text, "", nil
			}
		}
	}

	// If host is empty then this is a local (absolute or relative) link
	if host == "" {
		return fullUrl, text, "", nil
	}

	var b bytes.Buffer
	b.Write(regUrlHead.ReplaceAll([]byte(text), []byte("")))
	text = string(b.Bytes())

	// Provide a meaningful title only if the contents of the anchor is not
	// the fullUrl
	var title string
	if addTitle {
		upperBound := int(math.Ceil((float64(len(fullUrl)) / 100) * 80))
		title = fullUrl[0:upperBound]
		if len(fullUrl) > upperBound {
			title += "..."
		}
		var b2 bytes.Buffer
		b2.Write(regUrlHead.ReplaceAll([]byte(title), []byte("")))
		title = string(b2.Bytes())
	}

	var (
		link  Link
		found bool
	)
	// Get the shortened version
	for _, l := range *links {
		if l.URL == fullUrl {
			link = l
			found = true
		}
	}

	if !found {
		l, err := GetOrCreateLink(revisionId, fullUrl, host, text)
		if err != nil {
			glog.Warningf("Failed to shorten %s: %+v", fullUrl, err)
			// We don't care so much about failures to create short URLs, just
			// return a working URL and don't fail
			return fullUrl, text, "", nil
		}
		link = l
		*links = append(*links, l)
	}

	return fmt.Sprintf("%s%s", h.JumpUrl, link.ShortURL), text, title, nil
}

func GetOrCreateLink(
	revisionId int64,
	fullUrl string,
	host string,
	text string,
) (
	Link,
	error,
) {

	db, err := h.GetConnection()
	if err != nil {
		return Link{}, err
	}

	rows, err := db.Query(`
SELECT link_id
      ,short_url
      ,domain
      ,url
      ,inner_text
      ,created
      ,resolved_url
      ,resolved
      ,hits
  FROM links
 WHERE url = $1`,
		fullUrl,
	)
	if err != nil {
		return Link{}, err
	}
	defer rows.Close()

	var links []Link
	for rows.Next() {
		tmpLink := Link{}
		err = rows.Scan(
			&tmpLink.ID,
			&tmpLink.ShortURL,
			&tmpLink.Domain,
			&tmpLink.URL,
			&tmpLink.Text,
			&tmpLink.Created,
			&tmpLink.ResolvedURL,
			&tmpLink.Resolved,
			&tmpLink.Hits,
		)
		links = append(links, tmpLink)
	}
	err = rows.Err()
	if err != nil {
		return Link{}, err
	}
	rows.Close()

	if len(links) > 0 {
		err = CreateRevisionLink(db, revisionId, links[0].ID)
		if err != nil {
			return Link{}, err
		}

		return links[0], nil
	}

	link := Link{}
	link.ID, err = getNextLinkId(db)
	if err != nil {
		return Link{}, err
	}

	link.ShortURL = createShortUrl(
		link.ID,
		toSafeBase(int64(getRandomByte(1))%BASE_LEN),
	)
	link.Domain = host
	link.URL = fullUrl
	link.Text = text

	_, err = db.Exec(`
INSERT INTO links(
       link_id,
       short_url,
       domain,
       url,
       inner_text
) VALUES (
       $1,
       $2,
       $3,
       $4,
       $5
)`,
		link.ID,
		link.ShortURL,
		link.Domain,
		link.URL,
		link.Text,
	)
	if err != nil {
		return Link{}, errors.New(
			fmt.Sprintf("Could not create link (%s): %+v", link.URL, err),
		)
	}

	err = CreateRevisionLink(db, revisionId, link.ID)
	if err != nil {
		return Link{}, err
	}

	return link, nil
}

func CreateRevisionLink(db *sql.DB, revisionId int64, linkId int64) error {

	_, err := db.Exec(`
INSERT INTO revision_links(
       revision_id,
       link_id
) VALUES (
       $1,
       $2
)`,
		revisionId,
		linkId,
	)
	if err != nil {
		return errors.New(
			fmt.Sprintf(
				"Could not create revision_link (%d, %d): %v",
				revisionId,
				linkId,
				err.Error(),
			),
		)
	}

	return nil
}

func getNextLinkId(db *sql.DB) (int64, error) {
	var insertId int64
	err := db.QueryRow(
		`SELECT NEXTVAL('links_link_id_seq') AS link_id`,
	).Scan(
		&insertId,
	)
	if err != nil {
		return 0, errors.New(fmt.Sprintf("Error fetching nextval: %+v", err))
	}

	return insertId, nil
}

func createShortUrl(id int64, r string) string {
	return r + toSafeBase(id)
}

var baseChars = []byte{
	'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j',
	'k', 'm', 'n', 'p', 'q', 'r', 's', 't', 'u', 'v',
	'w', 'x', 'y', 'z', '2', '3', '4', '5', '6', '7',
	'8', '9', 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H',
	'J', 'K', 'L', 'M', 'N', 'P', 'Q', 'R', 'S', 'T',
	'U', 'V', 'W', 'X', 'Y', 'Z',
}

const BASE_LEN int64 = int64(56)

// toSafeBase converts a decimal number (base 10) to the safe base
// representation
func toSafeBase(n int64) string {
	if n < BASE_LEN {
		return string(baseChars[n])
	}

	var buff bytes.Buffer

	for n != 0 {
		buff.WriteByte(baseChars[n%BASE_LEN])
		n /= BASE_LEN
	}

	return buff.String()
}

func getRandomByte(n int) byte {
	b := make([]byte, n)
	x, err := crand.Read(b)
	if err != nil || x != n {
		for i := range b {
			b[i] = byte(mrand.Int31())
		}
	}

	return b[0]
}
