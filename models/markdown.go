package models

import (
	"bytes"
	"database/sql"
	"regexp"

	"github.com/russross/blackfriday"
	"golang.org/x/net/html"

	h "github.com/microcosm-cc/microcosm/helpers"
)

const htmlCruft = `<html><head></head><body>`

var (
	longWords      = regexp.MustCompile(`([^\s]{40})`)
	breakLongWords = "${1}\u00AD"
	unlinkedURLs   = regexp.MustCompile(`(?i)(^|[^\/>\]\w])(www\.[^\s<\[]+)`)
	linkURLs       = []byte(`${1}http://${2}`)
)

// ProcessCommentMarkdown will turn the latest revision markdown into HTML
func ProcessCommentMarkdown(
	tx *sql.Tx,
	revisionID int64,
	markdown string,
	siteID int64,
	itemTypeID int64,
	itemID int64,
	sendUpdates bool,
) (
	string,
	error,
) {
	markdown = stripChars(markdown, true, true, true, true)

	src := []byte(markdown)

	// Autolinkify
	src = unlinkedURLs.ReplaceAll(src, linkURLs)

	// 2014-09-15 (DK): Commented out as it affects code blocks as it is not
	// context aware.
	//src = PreProcessMentions(src)

	// Convert any BBCode to Markdown
	src = ProcessBBCode(src)

	// Find and link hashtags
	src = ProcessHashtags(siteID, src)

	// Use blackfriday to convert MarkDown to HTML
	src = MarkdownToHTML(src)

	// Convert all links to shortened URLs and record which links are in
	// which revisions (of a comment)
	src, err := ProcessLinks(revisionID, src, siteID)
	if err != nil {
		return "", err
	}

	// Parse mentions in the form of +Velocio or @Velocio
	if itemTypeID == h.ItemTypes[h.ItemTypeHuddle] {
		sendUpdates = false
	}
	src, err = ProcessMentions(
		tx,
		revisionID,
		src,
		siteID,
		itemTypeID,
		itemID,
		sendUpdates,
	)
	if err != nil {
		return "", err
	}

	// The treewalking leaves behind a stub root node
	if bytes.HasPrefix(src, []byte(htmlCruft)) {
		src = src[len([]byte(htmlCruft)):]
	}

	// Scrub the generated HTML of anything nasty
	// NOTE: This *MUST* always be the last thing to avoid introducing a
	// security vulnerability, any step beyond this must be known and trusted
	src = SanitiseHTML(src)

	// Now we have trusted input, add embeds that we trust (they're hard-coded
	// into this codebase)
	src = Embedly(src)

	return string(src), nil
}

// MarkdownToHTML wraps Black Friday and provides default settings for Black Friday
func MarkdownToHTML(src []byte) []byte {

	extensions := 0

	// detect embedded URLs that are not explicitly marked
	extensions |= blackfriday.EXTENSION_AUTOLINK

	// render fenced code blocks
	extensions |= blackfriday.EXTENSION_FENCED_CODE

	// translate newlines into line breaks
	extensions |= blackfriday.EXTENSION_HARD_LINE_BREAK

	// ignore emphasis markers inside words
	extensions |= blackfriday.EXTENSION_NO_INTRA_EMPHASIS

	// be strict about prefix header rules
	extensions |= blackfriday.EXTENSION_SPACE_HEADERS

	// strikethrough text using ~~test~~
	extensions |= blackfriday.EXTENSION_STRIKETHROUGH

	// render HTML tables
	extensions |= blackfriday.EXTENSION_TABLES

	// No need to insert an empty line to start a (code, quote, order list,
	// unorder list) block
	extensions |= blackfriday.EXTENSION_NO_EMPTY_LINE_BEFORE_BLOCK

	//// The following are not enabled by design decision

	// loosen up HTML block parsing rules
	//extensions |= blackfriday.EXTENSION_LAX_HTML_BLOCKS

	// Pandoc-style footnotes
	//extensions |= blackfriday.EXTENSION_FOOTNOTES

	htmlFlags := 0

	// generate XHTML output instead of HTML
	htmlFlags |= blackfriday.HTML_USE_XHTML

	//// Not enabled by design decision

	// enable smart punctuation substitutions
	//htmlFlags |= blackfriday.HTML_USE_SMARTYPANTS

	// enable smart fractions (with HTML_USE_SMARTYPANTS)
	//htmlFlags |= blackfriday.HTML_SMARTYPANTS_FRACTIONS

	// enable LaTeX-style dashes (with HTML_USE_SMARTYPANTS)
	//htmlFlags |= blackfriday.HTML_SMARTYPANTS_LATEX_DASHES

	// skip preformatted HTML blocks
	//htmlFlags |= blackfriday.HTML_SKIP_HTML

	// skip embedded images
	//htmlFlags |= blackfriday.HTML_SKIP_IMAGES

	// skip all links
	//htmlFlags |= blackfriday.HTML_SKIP_LINKS

	// skip embedded <script> elements
	//htmlFlags |= blackfriday.HTML_SKIP_SCRIPT

	// skip embedded <style> elements
	//htmlFlags |= blackfriday.HTML_SKIP_STYLE

	// only link to trusted protocols
	//htmlFlags |= blackfriday.HTML_SAFELINK

	// generate a table of contents
	//htmlFlags |= blackfriday.HTML_TOC

	// skip the main contents (for a standalone table of contents)
	//htmlFlags |= blackfriday.HTML_OMIT_CONTENTS

	// generate a complete HTML page
	//htmlFlags |= blackfriday.HTML_COMPLETE_PAGE

	renderer := blackfriday.HtmlRenderer(htmlFlags, "", "")

	htmlBytes := blackfriday.Markdown(src, renderer, extensions)

	// Final task is to insert \u00AD (HTML &shy;) every 40 chars within any
	// long words.
	//
	// 40 chars was chosen as ~42 chars is our smallest supported screen
	// (iPhone 3) and most browsers will show 90 chars, meaning that 80 chars
	// would be shown before wrapping there.
	htmlRoot, err := html.Parse(bytes.NewReader(htmlBytes))
	if err != nil {
		return []byte{}
	}

	var replaceLongStrings func(*html.Node)
	replaceLongStrings = func(n *html.Node) {

		if n.Type == html.TextNode {
			n.Data = longWords.ReplaceAllString(n.Data, breakLongWords)
		}

		// Walk the tree
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			replaceLongStrings(c)
		}
	}
	// Start the tree walk
	replaceLongStrings(htmlRoot)

	// Render the modified HTML tree
	b := new(bytes.Buffer)
	if html.Render(b, htmlRoot) != nil {
		return []byte{}
	}
	return b.Bytes()
}
