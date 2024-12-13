package models

import (
	"bytes"
	"regexp"
	"strings"

	"golang.org/x/net/html"

	"github.com/cloudflare/ahocorasick"
)

// EmbedRule describes a link within user generated content
type EmbedRule struct {
	Name    string
	Match   *regexp.Regexp
	Replace string
	Enabled bool
}

// FindReplace is a pair of strings that we will search for within a comment
// and replace with the found string.
//
// The "Find" will be a whole anchor, i.e. from an opening `<a ` through to
// the first closing `</a>`
//
// The "Replace" will be the string to Replace the found string, which is
// typically the link followed by the HTML for the embed.
type FindReplace struct {
	Find    string
	Replace string
}

// embedOnDomains is a list of domains that will be matched against within the
// src HTML, and if a match is found then we will try and match the EmbedRules
var embedOnDomains = []string{
	"bikely",
	"bikemap.net",
	"everytrail.com",
	"garmin",
	"google.com",
	"gpsies.com",
	"plotaroute.com",
	"ridewithgps.com",
	"strava",
	"vimeo",
	"youtube",
	"youtu.be",
}

// EmbedRules is an ordered list of the embeds to match on and then process,
// the whole idea is first match wins, and sometimes specificity is hard so we
// should put some things before others
//
// If you add a new thing here, be sure to add a domain match to embedOnDomains too
var EmbedRules = []EmbedRule{
	{
		// Must come before other Strava ones
		Name:    `Strava Recent Rides`,
		Match:   regexp.MustCompile(`https?://((app|www).strava.com/(clubs/[a-z\-0-9]+/latest-rides/[a-f0-9]+(\?show_rides=true)|athletes/[0-9]+/latest-rides/[a-f0-9]+))`),
		Replace: `<iframe height="454" width="300" frameborder="0" allowtransparency="true" scrolling="no" src="https://$1"></iframe>`,
		Enabled: true,
	},
	{
		// Must come before other YouTube ones
		Name:    `YouTube Playlists`,
		Match:   regexp.MustCompile(`(?:http|https):\/\/(?:www.|)(?:youtu\.be|youtube\.com)\/.*[&?]list=([a-zA-Z0-9_]+)`),
		Replace: `<iframe width="560" height="315" src="https://www.youtube-nocookie.com/embed/videoseries?list=${1}" frameborder="0" allowfullscreen></iframe>`,
		Enabled: true,
	},
	{
		Name:    `Bikely`,
		Match:   regexp.MustCompile(`http://www.bikely.com/maps/bike-path/([\w\d_-]+)`),
		Replace: `<div id="routemapiframe" style="width: 100%; border: 1px solid #d0d0d0; background: #755; overflow: hidden; white-space: nowrap;"><iframe id="rmiframe" style="height:360px;  background: #eee;" width="100%" frameborder="0" scrolling="no" src="http://www.bikely.com/maps/bike-path/$1/embed/1"></iframe></div>`,
		Enabled: true,
	},
	{
		Name:    `Bikemap`,
		Match:   regexp.MustCompile(`https?://www.bikemap.net/(?:en/)route/(\d+-?[^\/]*)/?`),
		Replace: `<iframe src="https://www.bikemap.net/en/route/$1/widget/?width=640&amp;extended=1&amp;distance_markers=1&amp;height=480&amp;unit=metric" width="640" height="628" border="0" frameborder="0" marginheight="0" marginwidth="0" scrolling="no"></iframe>`,
		Enabled: true,
	},
	{
		Name:    `Every Trail`,
		Match:   regexp.MustCompile(`https?://www.everytrail.com/view_trip.php\?trip_id=(\d+)`),
		Replace: `<iframe src="https://www.everytrail.com/iframe2.php?trip_id=$1&width=600&height=400" marginheight="0" marginwidth="0" frameborder="0" scrolling="no" width="600" height="400"></iframe>`,
		Enabled: true,
	},
	{
		Name:    `Garmin Activity`,
		Match:   regexp.MustCompile(`https?:\/\/connect\.garmin\.com\/(?:modern\/)activity\/([0-9]+)`),
		Replace: `<iframe width="465" height="548" frameborder="0" src="https://connect.garmin.com/activity/embed/$1"></iframe>`,
		Enabled: true,
	},
	{
		Name:    `Garmin Course`,
		Match:   regexp.MustCompile(`https?:\/\/connect\.garmin\.com\/course\/([0-9]+)`),
		Replace: `<iframe width="560" height="600" frameborder="0" src="https://connect.garmin.com/course/embed/$1"></iframe>`,
		Enabled: true,
	},
	{
		Name:    `Google Calendar`,
		Match:   regexp.MustCompile(`https?://(www.google.com/calendar/embed.*)`),
		Replace: `<iframe src="https://$1" style="border: 0" width="100%" height="600" frameborder="0" scrolling="no"></iframe>`,
		Enabled: true,
	},
	{
		Name:    `Google Map Embed`,
		Match:   regexp.MustCompile(`https?://www.google.com/maps/d/viewer\?mid=([^&]+).*`),
		Replace: `<iframe src="https://www.google.com/maps/d/embed?mid=$1" width="560" height="480"></iframe>`,
		Enabled: true,
	},
	{
		Name:    `Google Map Embed`,
		Match:   regexp.MustCompile(`https?://www.google.com/maps/d/embed\?mid=([^&]+).*`),
		Replace: `<iframe src="https://www.google.com/maps/d/embed?mid=$1" width="560" height="480"></iframe>`,
		Enabled: true,
	},
	{
		Name:    `GPSies`,
		Match:   regexp.MustCompile(`http://www.gpsies.com/map.do\?fileId=([\w\d_-]+)`),
		Replace: `<iframe src="http://www.gpsies.com/mapOnly.do?fileId=$1" width="600" height="400" frameborder="0" scrolling="no" marginheight="0" marginwidth="0"></iframe>`,
		Enabled: true,
	},
	{
		Name:    `Plot a Route`,
		Match:   regexp.MustCompile(`https?://www.plotaroute.com/route/([0-9]+)`),
		Replace: `<iframe name="plotaroute_map_$1" src="https://www.plotaroute.com/embedmap/$1?units=km" frameborder="0" scrolling="no" width="100%" height="650px" allowfullscreen webkitallowfullscreen mozallowfullscreen oallowfullscreen msallowfullscreen></iframe>`,
		Enabled: true,
	},
	{
		Name:    `Plot a Route Map`,
		Match:   regexp.MustCompile(`https?://www.plotaroute.com/map/([0-9]+)`),
		Replace: `<iframe name="plotaroute_map_$1" src="https://www.plotaroute.com/embedmap/$1?units=km" frameborder="0" scrolling="no" width="100%" height="650px" allowfullscreen webkitallowfullscreen mozallowfullscreen oallowfullscreen msallowfullscreen></iframe>`,
		Enabled: true,
	},
	{
		Name:    `Ride With GPS`,
		Match:   regexp.MustCompile(`https?://ridewithgps.com/routes/([0-9]+)`),
		Replace: `<iframe src="https://ridewithgps.com/routes/$1/embed" height="650px" width="100%" frameborder="0"></iframe>`,
		Enabled: true,
	},
	{
		Name:    `Ride With GPS`,
		Match:   regexp.MustCompile(`https?://ridewithgps.com/trips/([0-9]+)`),
		Replace: `<iframe src="https://ridewithgps.com/trips/$1/embed" height="500px" width="100%" frameborder="0"></iframe>`,
		Enabled: true,
	},
	{
		Name:    `Strava Activity Summary`,
		Match:   regexp.MustCompile(`https?://((?:app|www).strava.com/athletes/[0-9]+/activity-summary/[a-f0-9-]+)`),
		Replace: `<iframe height="360" width="360" frameborder="0" allowtransparency="true" scrolling="no" src="https://$1"></iframe>`,
		Enabled: true,
	},
	{
		Name:    `Strave Club Summary`,
		Match:   regexp.MustCompile(`https?://((app|www).strava.com/clubs/[a-z\-0-9]+/latest-rides/[a-f0-9]+(\?show_rides=false)?)`),
		Replace: `<iframe height="360" width="360" frameborder="0" allowtransparency="true" scrolling="no" src="https://$1"></iframe>`,
		Enabled: true,
	},
	{
		Name:    `Strave Course Segment`,
		Match:   regexp.MustCompile(`https?://(?:app|www).strava.com/segments/[a-z0-9-]+-([0-9]+)`),
		Replace: `<iframe height="360" width="460" frameborder="0" allowtransparency="true" scrolling="no" src="https://www.strava.com/segments/$1/embed"></iframe>`,
		Enabled: true,
	},
	{
		Name:    `Strava Course Segment`,
		Match:   regexp.MustCompile(`https?://((?:app|www).strava.com/activities/[0-9]+/embed/[a-f0-9-]+)`),
		Replace: `<iframe height="404" width="590" frameborder="0" allowtransparency="true" scrolling="no" src="https://$1"></iframe>`,
		Enabled: true,
	},
	{
		Name:    `Vimeo`,
		Match:   regexp.MustCompile(`https?://(?:[\w]+.)?vimeo.com\/(?:(?:groups\/[^\/]+\/videos\/)|(?:video\/))?([0-9]+).*`),
		Replace: `<iframe src="https://player.vimeo.com/video/$1" width="500" height="281" frameborder="0" webkitallowfullscreen mozallowfullscreen allowfullscreen></iframe>`,
		Enabled: true,
	},
	{
		Name:    `YouTube`,
		Match:   regexp.MustCompile(`^(?:https?:|)(?:\/\/)(?:www.|)(?:youtu\.be\/|youtube\.com(?:\/embed\/|\/v\/|\/watch\?(?:.*)v=|\/ytscreeningroom\?(?:.*)v=|\/feeds\/api\/videos\/|\/user\S*[^\w\-\s]))([\w\-]{11})(?:\W.*)?$`),
		Replace: `<iframe width="560" height="315" src="https://www.youtube-nocookie.com/embed/$1" frameborder="0" allowfullscreen></iframe>`,
		Enabled: true,
	},
}

func embedMayExist(src []byte) bool {
	// A super-quick pre-check for determining whether we are likely to have a
	// rewrite rule in the database. This is hard-coded for speed, when you add
	// a new unique domain rule, add the domain keyword here. This is string
	// matching and does not use regular expressions.
	domains := ahocorasick.NewStringMatcher(embedOnDomains)
	hits := domains.Match(src)

	return !(len(hits) == 0)
}

func Embedly(src []byte) []byte {
	// If there are no links, do nothing
	if !bytes.Contains(src, []byte(`<a `)) {
		return src
	}

	// Links are present in the src HTML, so are any likely to be embeds
	if !embedMayExist(src) {
		return src
	}

	// We are not going to work on the whole of the HTML, we chop this into
	// smaller parts, what we're going to operate on is the text up to a
	// closing anchor.
	//
	// We create a slice of chunks of HTML, by simply splitting on `</a>`,
	// note that we will return to actually find and replace on the original
	// full HTML, we just need to break it down into "sections that each have
	// a single link" so that we can iterate over the sections and process
	// one link at a time.
	var findReplaces = []FindReplace{}

	html := string(src)
	htmlChunks := strings.Split(html, `</a>`)

	for _, htmlChunk := range htmlChunks {
		// Links should be present within each htmlChunk, and we already determined
		// that one of the hard-coded list of domains is likely somewhere within the
		// HTML, so let's try and do some processing.
		//
		// We're not actually doing any DOM manipulation here, this is both far
		// simpler and far more messy, what we're doing is string replacements.
		// We start by finding if links exist that match our rules, and if so we
		// add those (in their entirety) to an array of things to find, whilst also
		// adding the full replacement.
		//
		// Only when we have composed a list of all the changes that we want to make
		// do we actually do so.
		//
		// This approach is a bit messy on the string manipulation required, but
		// makes it easy to reason about the fact that we only go through the HTML
		// once, and match the first regex each time.
		//
		// The process is:
		// 1. For each embed regexp
		// 2. Find the point at which one of our embed regexp's matches
		// 3. Find the first </a> after that match, we now have the substring we
		//    want to replace
		// 4. If we have not processed this, then match the original string and
		//    perform the regexp replace, take the output of that and add it after
		//    the closing `</a>`
		// 5. Copy the before and after into an array of strings to find and replace
		// 6. Apply the array of changes to the original source

		// We do not want to match on the whole HTML, but only the first link, so we
		// can split again on `<a ` and work on the last chunk
		htmlParts := strings.Split(htmlChunk, `<a `)
		if len(htmlParts) == 1 {
			// There was no opening anchor!?
			continue
		}

		// And now we have the full link that we want to actually do something with,
		// this will also form the "find" part of our findReplace
		find := `<a ` + htmlParts[len(htmlParts)-1] + `</a>`

		// We do actually now have to work on the HTML as a fragment of DOM as we
		// need to safely get the `href` value without consuming too much of the HTML
		href := extractHrefUrlFromFirstAnchor(find)
		if href == `` {
			continue
		}

		for _, embedRule := range EmbedRules {
			// https://pkg.go.dev/regexp#Regexp.FindStringIndex returns the start and
			// end of any match
			if !embedRule.Match.MatchString(href) {
				continue
			}

			// We now have a matching pattern, let's do it
			replace := embedRule.Match.ReplaceAllString(href, embedRule.Replace)
			if href == replace {
				// Surgery was not successful as nothing changed, do nothing
				continue
			}

			findReplaces = append(findReplaces, FindReplace{
				Find:    find,
				Replace: find + "<br />\n" + replace,
			})

			break
		}
	}

	// Finally do the actual replacements and put the embeds in place
	for _, findReplace := range findReplaces {
		html = strings.Replace(html, findReplace.Find, findReplace.Replace, 1)
	}

	return []byte(html)
}

func extractHrefUrlFromFirstAnchor(src string) string {
	doc, err := html.Parse(strings.NewReader(src))
	if err != nil {
		return ``
	}

	var href string
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, a := range n.Attr {
				if a.Key == "href" {
					href = a.Val
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	return href
}
