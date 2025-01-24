package models

import (
	"bytes"
	"net/url"
	"strings"

	"golang.org/x/net/html"

	conf "github.com/microcosm-collective/microcosm/config"
	h "github.com/microcosm-collective/microcosm/helpers"
)

// ProcessLinks will fetch the HTML for a revision and extract and shorten all
// hyperlinks
func ProcessLinks(
	revisionID int64,
	src []byte,
	siteID int64,
) (
	[]byte,
	error,
) {

	site, _, _ := GetSite(siteID)

	// If there are no links, do nothing
	if !bytes.Contains(src, []byte(`<a `)) {
		return src, nil
	}

	// Read and parse HTML
	doc, err := html.Parse(bytes.NewReader(src))
	if err != nil {
		return []byte{}, err
	}

	// Start the tree walk of the HTML
	err = WalkHtmlAndModifyLinks(revisionID, site, doc)
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

// WalkHtmlAndModifyLinks will recursively walk the DOM and find the links
func WalkHtmlAndModifyLinks(
	revisionID int64,
	site SiteType,
	element *html.Node,
) error {

	// Strip markdown introduced element ID attributes
	if element.Type == html.ElementNode {
		// Convert links to shortUrls
		if element.Data == "a" {
			var titleAttr string
			attributes := element.Attr

			for ii, attribute := range attributes {

				if attribute.Key == "href" &&
					!strings.Contains(attribute.Val, h.JumpURL) &&
					!strings.HasPrefix(attribute.Val, "mailto:") {

					u, err := url.Parse(attribute.Val)
					if err != nil {
						// It's not a valid URL, so let's not link it
						break
					}

					host := u.Host
					if host == "" {
						break
					}

					if element.FirstChild == nil {
						// If there's nothing in this anchor then this link does
						// nothing
						break
					}

					newURL, text, err := ProcessLink(
						u,
						site,
						element.FirstChild.Data,
					)
					if err != nil {
						return err
					}

					// Write our new link and text to the anchor
					attribute.Val = newURL
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
		err := WalkHtmlAndModifyLinks(revisionID, site, child)
		if err != nil {
			return err
		}
	}

	return nil
}

// ProcessLink sanitizes out the tracking, if the tracking is predictable
func ProcessLink(
	u *url.URL,
	site SiteType,
	text string,
) (
	string,
	string,
	error,
) {
	fullURL := u.String()
	// Don't process intra-site links
	//
	// We basically convert fully qualified URLs into absolute URLs by stripping
	// the prefix
	if site.Domain == "" {
		// If site.Domain were not blank this would cause issues as it would
		// break /api/v1/files/* links.
		prefix := "https://" + site.SubdomainKey + conf.ConfigStrings[conf.MicrocosmDomain]
		if strings.HasPrefix(fullURL, prefix) {
			if len(fullURL) > len(prefix) {
				fullURL = fullURL[len(prefix):]
				if fullURL == "." {
					fullURL = "/"
				}
				return fullURL, text, nil
			}

			return "/", text, nil
		}
	} else {
		// We should not process this... it's a link to a file we know about,
		// an attachment or something.
		prefix := "https://" + site.SubdomainKey + conf.ConfigStrings[conf.MicrocosmDomain]
		if strings.HasPrefix(fullURL, prefix) {
			return fullURL, text, nil
		}

		// We preserve both the http and https as we cannot know what how it will
		// be displayed in future:
		prefix = "http://" + site.Domain
		if strings.HasPrefix(fullURL, prefix) {
			if len(fullURL) > len(prefix) {
				fullURL = fullURL[len(prefix):]
				if fullURL == "." {
					fullURL = "/"
				}
				return fullURL, text, nil
			}

			return "/", text, nil
		}
		prefix = "https://" + site.Domain
		if strings.HasPrefix(fullURL, prefix) {
			if len(fullURL) > len(prefix) {
				fullURL = fullURL[len(prefix):]
				if fullURL == "." {
					fullURL = "/"
				}
				return fullURL, text, nil
			}

			return "/", text, nil
		}
	}

	// If host is empty then this is a local (absolute or relative) link
	if u.Host == "" {
		return fullURL, text, nil
	}

	// Now we can actually process the URL, as we believe this is now an
	// external URL
	//
	// We will strip out any tracking parameters from the URL using these rules:
	// https://github.com/newhouse/url-tracking-stripper/blob/dea6c144/README.md#documentation
	q := u.Query()

	// Google's Urchin Tracking Module & Google Adwords
	q.Del("utm_source")
	q.Del("utm_medium")
	q.Del("utm_term")
	q.Del("utm_campaign")
	q.Del("utm_content")
	q.Del("utm_name")
	q.Del("utm_cid")
	q.Del("utm_reader")
	q.Del("utm_viz_id")
	q.Del("utm_referrer")
	q.Del("utm_pubreferrer")
	q.Del("utm_swu")
	q.Del("stm_source")
	q.Del("stm_medium")
	q.Del("stm_term")
	q.Del("stm_campaign")
	q.Del("stm_content")
	q.Del("stm_name")
	q.Del("stm_cid")
	q.Del("stm_reader")
	q.Del("stm_viz_id")
	q.Del("stm_referrer")
	q.Del("stm_pubreferrer")
	q.Del("stm_swu")
	q.Del("gclid")

	// Amazon
	if strings.Contains(u.Host, "amazon.") {
		q.Del("dib")
		q.Del("dib_tag")
		q.Del("psc")
		q.Del("smid")
		q.Del("tag")
	}

	// Adobe Omniture SiteCatalyst
	q.Del("ICID")
	q.Del("icid")

	// Facebook
	q.Del("fbclid")

	// Hubspot
	q.Del("_hsenc")
	q.Del("_hsmi")

	// Instagram
	q.Del("igshid")

	// Klaviyo
	q.Del("igshid")

	// Marketo
	q.Del("mkt_tok")
	q.Del("mkt_uniq")

	// MailChimp
	q.Del("mc_cid")
	q.Del("mc_eid")

	// Microsoft
	q.Del("cvid")
	q.Del("oicd")

	// Olytics
	q.Del("oly_anon_id")
	q.Del("oly_enc_id")
	q.Del("otc")

	// Simple Reach
	q.Del("sr_share")

	// Vero
	q.Del("vero_conv")
	q.Del("vero_id")

	// Wicked
	q.Del("wickedid")

	// Yahoo
	q.Del("soc_src")
	q.Del("soc_trk")

	// Yankdex
	q.Del("_openstat")
	q.Del("yclid")

	// YouTube
	if strings.Contains(u.Host, "youtu.be") || strings.Contains(u.Host, "youtube.com") {
		q.Del("si")
	}

	// Unknown
	q.Del("nr_email_referer")
	q.Del("ncid")
	q.Del("ref")

	u.RawQuery = q.Encode()

	return u.String(), text, nil
}
