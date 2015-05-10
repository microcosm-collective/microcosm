package models

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/golang/glog"
	"golang.org/x/net/html"

	h "github.com/microcosm-cc/microcosm/helpers"
)

const (
	UrlProfile         string = "/profiles/"
	mentionPunctuation string = `,.:?!-`
)

var (
	regPreMentions    = regexp.MustCompile(`(?:^|\W)([+@](\S+))`)
	regMarkdownChars  = regexp.MustCompile("([\\\\*_{}[\\]()#-.!])")
	replMarkdownChars = []byte(`\$1`)
	regMentions       = regexp.MustCompile(`(^|\W)[+@](\S+)`)
)

// PreProcessMentions will escape any characters in a username that markdown
// treats as special. This results in the username emerging from the markdown
// parser as it appears before this function is applied.
//
// In other words: preserves usernames during the markdown process.
//
// NOTE (buro9 2014-09-15): This unfortunately applies to code that is processed
// too, and results in text in code blocks being escaped, but not being
// unescaped by the Markdown processor... it is *not* aware of code blocks and
// is therefore buggy.
func PreProcessMentions(md []byte) []byte {

	if !(bytes.Contains(md, []byte(`+`)) || bytes.Contains(md, []byte(`@`))) {
		return md
	}

	for _, s := range regPreMentions.FindAll(md, -1) {
		md = bytes.Replace(md, s, regMarkdownChars.ReplaceAll(s, replMarkdownChars), 1)
	}

	return md
}

func ProcessMentions(
	tx *sql.Tx,
	revisionId int64,
	src []byte,
	siteId int64,
	itemTypeId int64,
	itemId int64,
	sendUpdates bool,
) (
	[]byte,
	error,
) {

	if !(bytes.Contains(src, []byte(`+`)) || bytes.Contains(src, []byte(`@`))) {
		return src, nil
	}

	// Read and parse HTML
	doc, err := html.Parse(bytes.NewReader(src))
	if err != nil {
		return []byte{}, err
	}

	// Track mentions as we walk the tree
	var mentions map[string]string
	mentions = make(map[string]string)
	var profileNames map[string]int64
	profileNames = make(map[string]int64)

	// Function used as we need recursion to treewalk the Html
	var links func(*html.Node)
	links = func(n *html.Node) {

		if n.Type == html.TextNode {

			// Convert links to shortUrls
			if strings.Contains(n.Data, "+") || strings.Contains(n.Data, "@") {
				matches := regMentions.FindAllString(n.Data, -1)
				for _, v := range matches {
					// We track both the mentions and the profiles
					var mention = strings.TrimRight(
						strings.TrimLeft(v, " \n\t"),
						mentionPunctuation,
					)
					var profileName = strings.TrimLeft(mention, "+@")

					mentions[mention] = profileName
					profileNames[profileName] = int64(0)
				}
			}
		}

		// Walk the tree
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			links(c)
		}
	}
	// Start the tree walk
	links(doc)

	// Render the modified HTML tree
	b := new(bytes.Buffer)
	err = html.Render(b, doc)
	if err != nil {
		return []byte{}, err
	}
	src = b.Bytes()

	if len(profileNames) > 0 {
		// Fetch knowledge of existing mentions in older comment revisions
		type Mention struct {
			CommentId   int64
			MentionedBy int64
			ProfileId   sql.NullInt64
		}

		rows, err := tx.Query(`
SELECT c.comment_id
      ,c.profile_id as mentioned_by
      ,for_profile_id
  FROM comments c
       JOIN revisions r ON r.comment_id = c.comment_id
       LEFT OUTER JOIN updates u ON
            u.item_type_id = 4 -- comment
        AND u.item_id = c.comment_id
        AND u.created_by = c.profile_id
        AND u.update_type_id = 3 -- mention
 WHERE r.revision_id = $1`,
			revisionId,
		)
		if err != nil {
			glog.Errorf("%s %+v", "tx.Query()", err)
			return []byte{}, err
		}
		defer rows.Close()

		var existingMentions []Mention
		for rows.Next() {
			mention := Mention{}
			err = rows.Scan(
				&mention.CommentId,
				&mention.MentionedBy,
				&mention.ProfileId,
			)
			existingMentions = append(existingMentions, mention)
		}
		err = rows.Err()
		if err != nil {
			glog.Errorf("%s %+v", "rows.Next()", err)
			return []byte{}, err
		}
		rows.Close()

		if len(existingMentions) == 0 {
			return []byte{}, errors.New("Data integrity failure, " +
				"comment must exist for revision to be processed")
		}

		// Save all new mentions
		for profileName, _ := range profileNames {
			profileNames[profileName] = FetchProfileId(tx, profileName, revisionId)

			if profileNames[profileName] > 0 {
				var found bool
				for _, m := range existingMentions {
					if m.ProfileId.Valid &&
						m.ProfileId.Int64 == profileNames[profileName] {

						found = true
					}
				}
				if !found {
					err = ProcessMention(
						tx,
						existingMentions[0].CommentId,
						revisionId,
						existingMentions[0].MentionedBy,
						profileNames[profileName],
						siteId,
						itemTypeId,
						itemId,
						sendUpdates,
					)
					if err != nil {
						return []byte{}, err
					}
				}

				for mkey, mval := range mentions {
					if mval == profileName {
						src = bytes.Replace(
							src,
							[]byte(mkey),
							[]byte(
								fmt.Sprintf(
									`<a href="%s%d">%s</a>`,
									UrlProfile,
									profileNames[profileName],
									mkey,
								),
							),
							-1,
						)
					}
				}
			}
		}
	}

	return src, nil
}

// Returns 0 if profile does not exist
func FetchProfileId(tx *sql.Tx, profileName string, revisionId int64) int64 {
	var profileId int64
	rows, err := tx.Query(`
SELECT profile_id
  FROM profiles
 WHERE LOWER(profile_name) = $1
   AND site_id = (
           SELECT site_id
             FROM revisions r
                  LEFT JOIN profiles p ON r.profile_id = p.profile_id
            WHERE revision_id = $2
       )
 ORDER BY profile_id ASC
 LIMIT 1
OFFSET 0`,
		strings.ToLower(profileName),
		revisionId,
	)
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&profileId)
		if err != nil {
			return 0
		}
	}
	err = rows.Err()
	if err != nil {
		return 0
	}
	rows.Close()

	return profileId
}

// ProcessMention processes username mentions using the `+username` syntax and generates
// alerts for the mentioned user if they have are enabled in their preferences.
func ProcessMention(
	tx *sql.Tx,
	commentId int64,
	revisionId int64,
	createdBy int64,
	profileId int64,
	siteId int64,
	itemTypeId int64,
	itemId int64,
	sendUpdates bool,
) error {

	if !sendUpdates {
		// When this is false, it indicates that mentions should not be
		// processed and updates not sent.
		//
		// This is the case when the html has been purged and the comment is
		// old and we are just re-generated the html and the mentions were
		// processed a long long time ago.
		return nil
	}

	// Send the update
	var update = UpdateType{}
	update.SiteID = siteId
	update.UpdateTypeID = h.UpdateTypes[h.UpdateTypeMentioned]
	update.ForProfileID = profileId
	update.ItemTypeID = h.ItemTypes[h.ItemTypeComment]
	update.ItemID = commentId
	update.Meta.CreatedById = createdBy
	_, err := update.insert(tx)
	if err != nil {
		glog.Errorf("%s %+v", "update.insert(tx)", err)
		return err
	}

	go SendUpdatesForNewMentionInComment(
		siteId,
		update.ForProfileID,
		update.ItemID,
	)

	return nil
}
