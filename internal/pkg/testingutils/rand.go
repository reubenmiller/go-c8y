package testingutils

import (
	"math/rand"
	"time"

	"github.com/sethvargo/go-password/password"
)

const charset = "abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))

func stringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

// RandomString returns a random string of the given length
func RandomString(length int) string {
	return stringWithCharset(length, charset)
}

// RandomPassword generate a random password that meets the default
// Cumulocity IoT password policy
func RandomPassword(length int) string {
	value, err := password.Generate(length, 10, 10, false, false)
	if err != nil {
		// Panic as this should not happen
		panic("could not generate password")
	}
	return value
}
