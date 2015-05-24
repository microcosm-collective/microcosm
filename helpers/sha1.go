package helpers

import (
	"crypto/sha1"
	"fmt"
)

// SHA1 calculates the SHA1 for a []byte
func SHA1(bytes []byte) (string, error) {
	s := sha1.New()
	_, err := s.Write(bytes)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", s.Sum(nil)), nil
}
