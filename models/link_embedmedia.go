package models

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/cloudflare/ahocorasick"
	"github.com/golang/glog"

	h "github.com/microcosm-cc/microcosm/helpers"
)

// RewriteRule describes a matching HTML string and a regex replacement
type RewriteRule struct {
	ID           int64
	Title        string
	RegexMatch   string
	RegexReplace string
	Enabled      bool
	FetchTitle   bool
	Sequence     int
	Valid        bool
}

// UpdateEmbedsForDomain is used when we have new media definitions
// (i.e. for YouTube)
func UpdateEmbedsForDomain(domain string) (int, error) {
	// 1. Find all revisions that have links to the given domain
	// 2. Iterate
	// 3. Clear HTML cache, and re-run markdown processing
	return http.StatusOK, nil
}

// EmbedAllMedia is called after a revision has been created to perform media
// embedding and gets all links in the revision and processes them all
func EmbedAllMedia(revisionID int64) (int, error) {

	db, err := h.GetConnection()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	// Are there any external links in this revision?
	rows, err := db.Query(`--EmbedAllMedia
SELECT l.link_id
      ,l.short_url
      ,l.domain
      ,l.url
      ,l.inner_text
      ,l.created
      ,l.resolved_url
      ,l.resolved
      ,l.hits
  FROM revision_links r
  JOIN links l ON l.link_id = r.link_id
 WHERE r.revision_id = $1
 GROUP BY l.link_id
         ,l.short_url
         ,l.domain
         ,l.url
         ,l.inner_text
         ,l.created
         ,l.resolved_url
         ,l.resolved
         ,l.hits
`,
		revisionID,
	)
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("get links failed: %v", err.Error())
	}
	defer rows.Close()

	links := []Link{}
	for rows.Next() {
		link := Link{}
		err = rows.Scan(
			&link.ID,
			&link.ShortURL,
			&link.Domain,
			&link.URL,
			&link.Text,
			&link.Created,
			&link.ResolvedURL,
			&link.Resolved,
			&link.Hits,
		)
		if err != nil {
			return http.StatusInternalServerError,
				fmt.Errorf("row parsing error: %v", err.Error())
		}

		links = append(links, link)
	}
	err = rows.Err()
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("error fetching rows: %v", err.Error())
	}
	rows.Close()

	// Now process each one
	for _, link := range links {
		embedMediaForLink(link, revisionID)
	}

	return http.StatusOK, nil
}

// embedMediaForLink takes a single link in a revision, fetch it and sees
// whether we can embed it
func embedMediaForLink(m Link, revisionID int64) (int, error) {
	rule, status, err := m.fetchRewriteRule()
	if err != nil {
		glog.Errorf("%s %+v", "m.FetchRewriteRule()", err)
		return status, err
	}

	if rule.Valid {
		glog.Infof("%s %t", "rule.Valid", rule.Valid)
		return m.embedMediaUsingRule(revisionID, rule)
	}

	// TODO fetch destination and poke around the HTML

	// m.EmbedMediaUsingOpenGraph(revisionID)

	// m.EmbedMediaUsingTwitterCard(revisionID)

	return http.StatusOK, nil
}

func (m *Link) fetchRewriteRule() (RewriteRule, int, error) {
	rewriteRule := RewriteRule{}

	if !m.rewriteRuleMayExist() {
		return rewriteRule, http.StatusOK, nil
	}

	db, err := h.GetConnection()
	if err != nil {
		return rewriteRule, http.StatusInternalServerError, err
	}

	rows, err := db.Query(`
SELECT r.rule_id
      ,r.name As title
      ,r.match_regex
      ,r.replace_regex
      ,r.is_enabled
      ,r.sequence
  FROM rewrite_domains d
       JOIN rewrite_domain_rules dr ON dr.domain_id = d.domain_id
       JOIN rewrite_rules r ON r.rule_id = dr.rule_id
 WHERE r.is_enabled IS NOT FALSE
   AND $1 ~ d.domain_regex
 ORDER BY r.sequence`,
		m.Domain,
	)
	if err != nil {
		return rewriteRule, http.StatusInternalServerError,
			fmt.Errorf("get links failed: %+v", err)
	}
	defer rows.Close()

	rules := []RewriteRule{}
	for rows.Next() {
		rule := RewriteRule{}
		err = rows.Scan(
			&rule.ID,
			&rule.Title,
			&rule.RegexMatch,
			&rule.RegexReplace,
			&rule.Enabled,
			&rule.Sequence,
		)
		if err != nil {
			return rewriteRule, http.StatusInternalServerError,
				fmt.Errorf("row parsing error: %+v", err)
		}

		rules = append(rules, rule)
	}
	err = rows.Err()
	if err != nil {
		return rewriteRule, http.StatusInternalServerError,
			fmt.Errorf("error fetching rows: %+v", err)
	}
	rows.Close()

	for _, rule := range rules {
		matched, err := regexp.Match(`(?i)`+rule.RegexMatch, []byte(m.URL))
		if err != nil {
			glog.Errorf("%s %+v", "regexp.Compile(rule.RegexMatch)", err)
			continue
		}

		if matched {
			rule.Valid = true

			return rule, http.StatusOK, nil
		}
	}

	return rewriteRule, http.StatusOK, nil
}

func (m *Link) rewriteRuleMayExist() bool {
	// A super-quick pre-check for determining whether we are likely to have a
	// rewrite rule in the database. This is hard-coded for speed, when you add
	// a new unique domain rule, add the domain keyword here. This is string
	// matching and does not use regular expressions.
	domains := ahocorasick.NewStringMatcher([]string{
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
	})
	hits := domains.Match([]byte(strings.ToLower(m.Domain)))

	return !(len(hits) == 0)
}

func (m *Link) embedMediaUsingRule(
	revisionID int64,
	rule RewriteRule,
) (
	int,
	error,
) {
	// Build the embed HTML
	matchURL, err := regexp.Compile(`(?i)` + rule.RegexMatch)
	if err != nil {
		glog.Errorf("%s %+v", "regexp.Compile(rule.RegexMatch)", err)
		return http.StatusInternalServerError,
			fmt.Errorf("could not compile match URL: %+v", err)
	}

	embedHTML := matchURL.ReplaceAllString(m.URL, rule.RegexReplace)

	// Use string manipulation to insert the embed
	shortURL := h.JumpURL + m.ShortURL

	return m.embedMedia(revisionID, shortURL, embedHTML)
}

func (m *Link) embedMedia(
	revisionID int64,
	shortURL string,
	embedHTML string,
) (
	int,
	error,
) {
	tx, err := h.GetTransaction()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	var (
		commentID int64
		html      string
	)
	err = tx.QueryRow(`
SELECT comment_id
      ,html
  FROM revisions
 WHERE revision_id = $1`,
		revisionID,
	).Scan(
		&commentID,
		&html,
	)
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("error fetching HTML for revision: %+v", err)
	}

	// Use string manipulation to insert the embed
	//
	// Essence of this is:
	// 1) Split on the shortURL
	// 2) Ignoring index 0, on each subsequent segment find the first instance
	//    of </a>
	// 3) Test whether this embed has been done by looking for embed="true"
	// 4) Replace the first close anchor with the new one and insert the embed
	// 5) Copy these new segments into a new array
	// 6) Join the array back together by the shortURL to recreate the comment
	const closeAnchor string = "</a>"
	const done string = `embed="true"`
	htmlOut := []string{}

	replacementsMade := false
	htmlIn := strings.Split(html, shortURL)
	for i, part := range htmlIn {
		if i == 0 {
			htmlOut = append(htmlOut, part)
			continue
		}

		if part[2:len(done)+2] == done {
			// Already done
			htmlOut = append(htmlOut, part)
			continue
		}

		new := closeAnchor + "<br />\n" + embedHTML + "<br />\n"

		part = strings.Replace(`" `+done+part[1:], closeAnchor, new, 1)

		htmlOut = append(htmlOut, part)
		replacementsMade = true
	}
	html = strings.Join(htmlOut, shortURL)

	if !replacementsMade {
		// No embeds were made, so this must have been processed already
		tx.Rollback()
		return http.StatusOK, nil
	}

	// Update the html
	_, err = tx.Exec(`
UPDATE revisions
   SET html = $2
 WHERE revision_id = $1`,
		revisionID,
		html,
	)
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("could not save HTML: %v", err.Error())
	}

	err = tx.Commit()
	if err != nil {
		return http.StatusInternalServerError,
			fmt.Errorf("transaction failed: %v", err.Error())
	}

	PurgeCache(h.ItemTypes[h.ItemTypeComment], commentID)

	return http.StatusOK, nil
}
