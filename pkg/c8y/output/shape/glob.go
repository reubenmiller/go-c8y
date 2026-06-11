package shape

import (
	"regexp"
	"strings"
)

// pathGlob is a compiled path glob with the same semantics as the globs used
// by go-c8y-cli's property selection (github.com/obeattie/ohmyglob with a
// '.' separator, anchored at both ends):
//
//   - '*'  matches any sequence of characters except the separator
//   - '**' matches across separators (and absorbs adjacent separators)
//   - '?'  matches a single character except the separator
//   - '\'  escapes the next character
//   - a leading '!' (repeatable) negates the pattern; negated patterns are
//     never selected, they exclude keys selected by other patterns
type pathGlob struct {
	pattern  string // the pattern as given (including any '!' prefix)
	re       *regexp.Regexp
	negative bool
}

func (g *pathGlob) String() string {
	return g.pattern
}

func (g *pathGlob) IsNegative() bool {
	return g.negative
}

func (g *pathGlob) MatchString(s string) bool {
	return g.re.MatchString(s)
}

const (
	tokLiteral = iota
	tokSeparator
	tokStar
	tokGlobStar
	tokAny
)

type globToken struct {
	typ int
	lit string
}

func compilePathGlob(pattern string) (*pathGlob, error) {
	g := &pathGlob{pattern: pattern}
	rest := pattern
	for strings.HasPrefix(rest, "!") {
		g.negative = !g.negative
		rest = rest[1:]
	}

	// Tokenize
	var tokens []globToken
	for i := 0; i < len(rest); {
		switch rest[i] {
		case '\\':
			if i+1 < len(rest) {
				tokens = append(tokens, globToken{tokLiteral, string(rest[i+1])})
				i += 2
			} else {
				tokens = append(tokens, globToken{tokLiteral, `\`})
				i++
			}
		case '.':
			tokens = append(tokens, globToken{typ: tokSeparator})
			i++
		case '?':
			tokens = append(tokens, globToken{typ: tokAny})
			i++
		case '*':
			if i+1 < len(rest) && rest[i+1] == '*' {
				tokens = append(tokens, globToken{typ: tokGlobStar})
				i += 2
			} else {
				tokens = append(tokens, globToken{typ: tokStar})
				i++
			}
		default:
			j := i
			for j < len(rest) && rest[j] != '\\' && rest[j] != '.' && rest[j] != '?' && rest[j] != '*' {
				j++
			}
			tokens = append(tokens, globToken{tokLiteral, rest[i:j]})
			i = j
		}
	}

	// Globstars take care of surrounding separators: a separator directly
	// after a globstar is consumed, consecutive globstars collapse, and a
	// trailing globstar absorbs the separator before it.
	var processed []globToken
	for i := 0; i < len(tokens); i++ {
		t := tokens[i]
		if t.typ == tokGlobStar {
			if i+1 < len(tokens) && tokens[i+1].typ == tokSeparator {
				i++
			}
			if n := len(processed); n > 0 && processed[n-1].typ == tokGlobStar {
				processed = processed[:n-1]
			}
			if n := len(processed); i+1 >= len(tokens) && n > 0 && processed[n-1].typ == tokSeparator {
				processed = processed[:n-1]
			}
		}
		processed = append(processed, t)
	}

	var sb strings.Builder
	sb.WriteString("^")
	for i, t := range processed {
		switch t.typ {
		case tokLiteral:
			sb.WriteString(regexp.QuoteMeta(t.lit))
		case tokSeparator:
			sb.WriteString(`\.`)
		case tokStar:
			sb.WriteString(`[^\.]*`)
		case tokAny:
			sb.WriteString(`[^\.]`)
		case tokGlobStar:
			isLast := i == len(processed)-1
			sb.WriteString("(?:")
			if isLast && i > 0 {
				sb.WriteString(`\.`)
			}
			sb.WriteString(".+")
			if !isLast {
				sb.WriteString(`\.`)
			}
			sb.WriteString(")?")
		}
	}
	sb.WriteString("$")

	re, err := regexp.Compile(sb.String())
	if err != nil {
		return nil, err
	}
	g.re = re
	return g, nil
}

// naturalLess compares two strings in natural order, e.g. "abc2" < "abc12".
// Non-digit sequences are compared bytewise, digit sequences numerically
// (leading zeros as tie-breaker, so "2" < "02"). Only ASCII digits are
// considered. Credit to https://github.com/fvbommel/sortorder
func naturalLess(str1, str2 string) bool {
	isdigit := func(b byte) bool { return '0' <= b && b <= '9' }
	idx1, idx2 := 0, 0
	for idx1 < len(str1) && idx2 < len(str2) {
		c1, c2 := str1[idx1], str2[idx2]
		dig1, dig2 := isdigit(c1), isdigit(c2)
		switch {
		case dig1 != dig2: // Digits before other characters.
			return dig1
		case !dig1:
			if c1 != c2 {
				return c1 < c2
			}
			idx1++
			idx2++
		default: // Digits
			// Eat zeros.
			for ; idx1 < len(str1) && str1[idx1] == '0'; idx1++ {
			}
			for ; idx2 < len(str2) && str2[idx2] == '0'; idx2++ {
			}
			// Eat all digits.
			nonZero1, nonZero2 := idx1, idx2
			for ; idx1 < len(str1) && isdigit(str1[idx1]); idx1++ {
			}
			for ; idx2 < len(str2) && isdigit(str2[idx2]); idx2++ {
			}
			// If lengths of numbers with non-zero prefix differ, the shorter
			// one is less.
			if len1, len2 := idx1-nonZero1, idx2-nonZero2; len1 != len2 {
				return len1 < len2
			}
			// If they're equal, string comparison is correct.
			if nr1, nr2 := str1[nonZero1:idx1], str2[nonZero2:idx2]; nr1 != nr2 {
				return nr1 < nr2
			}
			// Otherwise, the one with less zeros is less.
			if nonZero1 != nonZero2 {
				return nonZero1 < nonZero2
			}
		}
	}
	return len(str1) < len(str2)
}
