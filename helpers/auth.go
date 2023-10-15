package helpers

import (
	"crypto/rand"
	"fmt"
)

// NoAuthMessage is the unauthorised error message
const NoAuthMessage = "You do not have permission to do that"

// AllowedChars is the range of chars that can appear in a random string
var AllowedChars = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

// RandString returns a random string for a required length
func RandString(length int) (string, error) {
	bytes := make([]byte, length)
	randomByte := make([]byte, 1)

	for i := 0; i < length; i++ {
		// attempt to generate a random number
		// that is no greater than 'max'
	attempt:
		numBytes, err := rand.Read(randomByte)
		if numBytes != 1 {
			return string(bytes), fmt.Errorf("failed to read a random byte")
		}

		if err != nil {
			return string(bytes), err
		}

		// if the number is too big, start again at 'attempt'
		if int(randomByte[0]) >= len(AllowedChars) {
			goto attempt
		}

		// use the char corresponding to this number
		bytes[i] = AllowedChars[randomByte[0]]

	}

	return string(bytes), nil
}
