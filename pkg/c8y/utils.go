package c8y

import "strings"

// IsID reports whether v looks like a Cumulocity numeric ID (digits only,
// ignoring surrounding whitespace).
func IsID(v string) bool {
	isNotDigit := func(c rune) bool { return c < '0' || c > '9' }
	value := strings.TrimSpace(v)
	return value != "" && strings.IndexFunc(value, isNotDigit) <= -1
}
