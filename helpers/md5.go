package helpers

import (
	"crypto/md5"
	"fmt"
	"io"
)

func Md5sum(s string) string {
	m := md5.New()
	io.WriteString(m, s)
	return fmt.Sprintf("%x", m.Sum(nil))
}
