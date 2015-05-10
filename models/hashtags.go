package models

import (
	"bytes"
	"regexp"
)

var regHashtags = regexp.MustCompile(`(?i)(\s|\A)#([a-z0-9_]*[a-z_][a-z0-9_]*)`)

func ProcessHashtags(siteId int64, src []byte) []byte {

	if !bytes.Contains(src, []byte(`#`)) {
		return src
	}

	s, _, _ := GetSite(siteId)
	return regHashtags.ReplaceAll(
		src,
		[]byte(`$1[#$2](`+s.GetURL()+`/search/?q=%23$2)`),
	)
}
