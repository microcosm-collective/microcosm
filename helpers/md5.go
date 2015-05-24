package helpers

import (
	"crypto/md5"
	"fmt"
	"io"
)

// MD5Sum calculates the MD5 of a string
func MD5Sum(s string) string {
	m := md5.New()
	io.WriteString(m, s)
	return fmt.Sprintf("%x", m.Sum(nil))
}
