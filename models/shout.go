package models

import (
	"strings"
)

// ShoutToWhisper takes a string, and will lowercase it if it is in all caps
func ShoutToWhisper(s string) string {
	if strings.ToUpper(s) == s {
		s = strings.ToLower(s)
	}

	return s
}
