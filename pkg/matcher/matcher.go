package matcher

import (
	"fmt"
	"regexp"
	"strings"
)

// Glob represents a compiled glob pattern for efficient matching.
// It converts glob wildcards (*, ?) to regular expressions and provides
// case-sensitive matching suitable for channel subscriptions.
type Glob struct {
	pattern string
	regex   *regexp.Regexp
}

// Compile compiles a glob pattern into a reusable Glob matcher.
// Supported glob syntax:
//   - * matches zero or more characters
//   - ? matches exactly one character
//   - All other characters are matched literally (case-sensitive)
//
// Example patterns:
//   - "/alarms/*" matches "/alarms/12345" and "/alarms/anything"
//   - "/events/?" matches "/events/1" but not "/events/12"
//   - "/measurements/123*" matches "/measurements/12345"
func Compile(pattern string) (*Glob, error) {
	r, err := compileGlobPattern(pattern, false)
	if err != nil {
		return nil, err
	}

	return &Glob{
		pattern: pattern,
		regex:   r,
	}, nil
}

// String returns the original glob pattern string.
func (g *Glob) String() string {
	return g.pattern
}

// MatchString reports whether the string s matches the glob pattern.
// Matching is case-sensitive.
func (g *Glob) MatchString(s string) bool {
	return g.regex.MatchString(s)
}

func compileGlobPattern(pattern string, caseInsensitive bool) (*regexp.Regexp, error) {
	// Escape backslashes first
	pattern = strings.ReplaceAll(pattern, "\\", "\\\\")

	// Escape special regex characters (except *, ? which are our glob wildcards)
	specialChars := []string{".", "+", "(", ")", "|", "[", "]", "{", "}", "^", "$"}
	for _, char := range specialChars {
		pattern = strings.ReplaceAll(pattern, char, "\\"+char)
	}

	// Convert glob wildcards to regex
	pattern = strings.ReplaceAll(pattern, "*", ".*")
	pattern = strings.ReplaceAll(pattern, "?", ".")

	// Add anchors for full string matching
	prefix := "^"
	if caseInsensitive {
		prefix = "(?i)^"
	}
	pattern = prefix + pattern + "$"

	r, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid glob pattern: %w", err)
	}

	return r, nil
}

func MatchWithWildcards(s, pattern string) (bool, error) {
	r, err := compileGlobPattern(pattern, true)
	if err != nil {
		return false, err
	}
	return r.MatchString(s), nil
}

func MatchWithRegex(s, pattern string) (bool, error) {
	// case-insensitive matching
	pattern = "(?i)" + pattern

	r, err := regexp.Compile(pattern)

	if err != nil {
		return false, fmt.Errorf("invalid regex pattern")
	}

	return r.MatchString(s), nil
}
