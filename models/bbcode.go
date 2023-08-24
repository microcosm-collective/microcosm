package models

import (
	"bytes"
	"regexp"
)

type bbCode struct {
	test    []byte
	match   *regexp.Regexp
	replace []byte
}

var (
	bbcodeQuote     = regexp.MustCompile(`(?imsU)\[quote(?:=[^\]]+)?\](.+)\[/quote\]`)
	bbcodeQuoteLine = regexp.MustCompile(`(?imU)^(.*)$`)

	bbCodes = []bbCode{
		{
			test:    []byte(`[b]`),
			match:   regexp.MustCompile(`(?i)\[b\]((?:.|\n)+?)\[\/b\]`),
			replace: []byte(`**$1**`),
		},
		{
			test:    []byte(`[u]`),
			match:   regexp.MustCompile(`(?i)\[u\]((?:.|\n)+?)\[\/u\]`),
			replace: []byte(`*$1*`),
		},
		{
			test:    []byte(`[i]`),
			match:   regexp.MustCompile(`(?i)\[i\]((?:.|\n)+?)\[\/i\]`),
			replace: []byte(`*$1*`),
		},
		{
			test:    []byte(`[s]`),
			match:   regexp.MustCompile(`(?i)\[s\]((?:.|\n)+?)\[\/s\]`),
			replace: []byte(`~~$1~~`),
		},
		{
			test:    []byte(`[color`),
			match:   regexp.MustCompile(`(?i)\[color\=.+?\]((?:.|\n)+?)\[\/color\]`),
			replace: []byte(`$1`),
		},
		{
			test:    []byte(`[list`),
			match:   regexp.MustCompile(`(?i)\[\/?list(=1)?\]`),
			replace: []byte(``),
		},
		{
			test:    []byte(`[*]`),
			match:   regexp.MustCompile(`(\n)\[\*\]`),
			replace: []byte(`$1* `),
		},
		{
			test:    []byte(`[img]`),
			match:   regexp.MustCompile(`(?i)\[img\]((?:.|\n)+?)\[\/img\]`),
			replace: []byte(`![]($1)`),
		},
		{
			test:    []byte(`[attach]`),
			match:   regexp.MustCompile(`(?i)\[attach\]((?:.|\n)+?)\[\/attach\]`),
			replace: []byte(`[/attachments/$1](/attachments/$1)`),
		},
		{
			test:    []byte(`[url]`),
			match:   regexp.MustCompile(`(?i)\[url\]((?:.|\n)+?)\[\/url\]`),
			replace: []byte(`$1`),
		},
		{
			test:    []byte(`[url=`),
			match:   regexp.MustCompile(`(?i)\[url="?(.+?)"?\]((?:.|\n)+?)\[\/url\]`),
			replace: []byte(`[$2]($1)`),
		},
		{
			test:    []byte(`[email]`),
			match:   regexp.MustCompile(`(?i)\[email\]((?:.|\n)+?)\[\/email\]`),
			replace: []byte(`<$1>`),
		},
		{
			test:    []byte(`[email=`),
			match:   regexp.MustCompile(`(?i)\[email="?(.+?)"?\]((?:.|\n)+?)\[\/email\]`),
			replace: []byte(`<$1>`),
		},
		{
			test:    []byte(`[cite`),
			match:   regexp.MustCompile(`(?i)\[cite\]\s+?((?:.|\n)+?):?\[\/cite\]`),
			replace: []byte(`*$1* `),
		},
	}
)

func bbcodeQuoteToMarkdown(input []byte) []byte {
	// Find quotes
	for _, quote := range bbcodeQuote.FindAll(input, -1) {

		var b bytes.Buffer

		// Each line in a quote must be prefixed with `> `
		// and newlines added around the quote
		b.Write([]byte("\n"))
		b.Write(bbcodeQuoteLine.ReplaceAll(
			bbcodeQuote.ReplaceAll(quote, []byte("$1")), []byte("> $1"),
		))
		b.Write([]byte("\n"))

		// Replace the quote with our parsed version
		input = bytes.Replace(input, quote, b.Bytes(), 1)
	}

	return input
}

// ProcessBBCode replaces BBCode with Markdown
func ProcessBBCode(src []byte) []byte {

	testSrc := bytes.ToLower(src)

	if bytes.Contains(testSrc, []byte(`[quote`)) {
		src = bbcodeQuoteToMarkdown(src)
	}

	for _, bb := range bbCodes {
		if bytes.Contains(testSrc, bb.test) {
			src = bb.match.ReplaceAll(src, bb.replace)
		}
	}

	return src
}
