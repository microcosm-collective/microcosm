package models

import (
	"strings"
)

func ShoutToWhisper(s string) string {
	if strings.ToUpper(s) == s {
		s = strings.ToLower(s)
	}

	return s
}
