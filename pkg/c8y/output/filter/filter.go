// Package filter provides compiled predicates over JSON documents for use
// with output.Filter. All pattern compilation happens at construction time,
// so the per-item cost is a single gjson path lookup plus a comparison —
// in contrast to query engines that re-parse the document per filter.
package filter

import (
	"regexp"
	"strings"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
)

// Predicate reports whether a document matches.
type Predicate func(jsondoc.JSONDoc) bool

// And matches when all predicates match.
func And(preds ...Predicate) Predicate {
	return func(d jsondoc.JSONDoc) bool {
		for _, p := range preds {
			if !p(d) {
				return false
			}
		}
		return true
	}
}

// Or matches when at least one predicate matches.
func Or(preds ...Predicate) Predicate {
	return func(d jsondoc.JSONDoc) bool {
		for _, p := range preds {
			if p(d) {
				return true
			}
		}
		return false
	}
}

// Not inverts a predicate.
func Not(p Predicate) Predicate {
	return func(d jsondoc.JSONDoc) bool { return !p(d) }
}

// Exists matches documents where the path exists.
func Exists(path string) Predicate {
	return func(d jsondoc.JSONDoc) bool { return d.Get(path).Exists() }
}

// Eq matches documents where the value at path equals want.
// Numeric values are compared numerically, everything else as strings.
func Eq(path string, want any) Predicate {
	switch w := want.(type) {
	case string:
		return func(d jsondoc.JSONDoc) bool {
			v := d.Get(path)
			return v.Exists() && v.String() == w
		}
	case bool:
		return func(d jsondoc.JSONDoc) bool {
			v := d.Get(path)
			return v.Exists() && v.Bool() == w
		}
	default:
		f := toFloat(want)
		return func(d jsondoc.JSONDoc) bool {
			v := d.Get(path)
			return v.Exists() && v.Float() == f
		}
	}
}

// Gt, Gte, Lt, Lte match documents where the numeric value at path compares
// against want.
func Gt(path string, want float64) Predicate {
	return func(d jsondoc.JSONDoc) bool { v := d.Get(path); return v.Exists() && v.Float() > want }
}

func Gte(path string, want float64) Predicate {
	return func(d jsondoc.JSONDoc) bool { v := d.Get(path); return v.Exists() && v.Float() >= want }
}

func Lt(path string, want float64) Predicate {
	return func(d jsondoc.JSONDoc) bool { v := d.Get(path); return v.Exists() && v.Float() < want }
}

func Lte(path string, want float64) Predicate {
	return func(d jsondoc.JSONDoc) bool { v := d.Get(path); return v.Exists() && v.Float() <= want }
}

// Contains matches documents where the string value at path contains substr.
func Contains(path, substr string) Predicate {
	return func(d jsondoc.JSONDoc) bool {
		return strings.Contains(d.Get(path).String(), substr)
	}
}

// Like matches the string value at path against a case-insensitive glob
// pattern where '*' matches any sequence and '?' matches a single character
// (the same semantics as go-c8y-cli's "like" filter). The pattern is compiled
// once at construction.
func Like(path, pattern string) Predicate {
	re := globToRegexp(pattern)
	return func(d jsondoc.JSONDoc) bool {
		return re.MatchString(d.Get(path).String())
	}
}

// Match matches the string value at path against a regular expression,
// compiled once at construction. It panics if the expression is invalid
// (use regexp.Compile and a custom predicate to handle errors).
func Match(path, expr string) Predicate {
	re := regexp.MustCompile(expr)
	return func(d jsondoc.JSONDoc) bool {
		return re.MatchString(d.Get(path).String())
	}
}

func globToRegexp(pattern string) *regexp.Regexp {
	var sb strings.Builder
	sb.WriteString("(?i)^")
	for _, r := range pattern {
		switch r {
		case '*':
			sb.WriteString(".*")
		case '?':
			sb.WriteString(".")
		default:
			sb.WriteString(regexp.QuoteMeta(string(r)))
		}
	}
	sb.WriteString("$")
	return regexp.MustCompile(sb.String())
}

func toFloat(v any) float64 {
	switch n := v.(type) {
	case int:
		return float64(n)
	case int32:
		return float64(n)
	case int64:
		return float64(n)
	case float32:
		return float64(n)
	case float64:
		return n
	}
	return 0
}
