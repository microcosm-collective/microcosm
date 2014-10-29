package helpers

import (
	"crypto/rand"
	"errors"
)

const (
	NoAuthMessage = "You do not have permission to do that"
)

var AllowedChars = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func RandString(length int) (string, error) {

	bytes := make([]byte, length)

	random_byte := make([]byte, 1)

	for i := 0; i < length; i++ {

		// attempt to generate a random number
		// that is no greater than 'max'
	attempt:
		num_bytes, err := rand.Read(random_byte)
		if num_bytes != 1 {
			return string(bytes), errors.New("Failed to read a random byte")
		}

		if err != nil {
			return string(bytes), err
		}

		// if the number is too big, start again at 'attempt'
		if int(random_byte[0]) >= len(AllowedChars) {
			goto attempt
		}

		// use the char corresponding to this number
		bytes[i] = AllowedChars[random_byte[0]]

	}

	return string(bytes), nil

}
