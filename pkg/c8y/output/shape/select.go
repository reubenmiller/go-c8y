// Package shape provides response-shaping stages: projecting documents down
// to a selected set of properties (the equivalent of go-c8y-cli's --select).
//
// Patterns are compiled once when the stage is constructed; per item the cost
// is a handful of gjson path lookups written into a small output document —
// no flatten/sort/unflatten cycle.
package shape

import (
	"regexp"
	"strings"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/output"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// Select returns a stage that reduces each document to the given properties.
// A pattern is either a concrete gjson path (e.g. "id", "c8y_Hardware.model")
// or a case-insensitive glob over dotted key paths where '*' matches any
// sequence (e.g. "c8y_Hardware.*", "*.serialNumber"). Matched values keep
// their position in the document hierarchy.
//
// Arrays are treated as leaf values: they are copied whole when selected,
// not traversed by glob patterns.
func Select(patterns ...string) output.Stage {
	sel := compileSelector(patterns)
	return output.Map(sel.apply)
}

type selector struct {
	exact     []string
	wildcards []*regexp.Regexp
}

func compileSelector(patterns []string) *selector {
	sel := &selector{}
	for _, p := range patterns {
		if strings.ContainsAny(p, "*?") {
			sel.wildcards = append(sel.wildcards, globPathToRegexp(p))
		} else {
			sel.exact = append(sel.exact, p)
		}
	}
	return sel
}

var setOptions = &sjson.Options{Optimistic: true}

func (s *selector) apply(doc jsondoc.JSONDoc) (jsondoc.JSONDoc, error) {
	root := doc.Get()
	out := []byte("{}")
	var err error

	for _, path := range s.exact {
		v := root.Get(escapeGJSONPath(path))
		if !v.Exists() {
			continue
		}
		out, err = sjson.SetRawBytesOptions(out, safeSetPath(root, path), []byte(v.Raw), setOptions)
		if err != nil {
			return jsondoc.Empty(), err
		}
	}

	if len(s.wildcards) > 0 {
		walk(root, "", func(path string, v gjson.Result) bool {
			for _, re := range s.wildcards {
				if re.MatchString(path) {
					out, err = sjson.SetRawBytesOptions(out, safeSetPath(root, path), []byte(v.Raw), setOptions)
					return err == nil
				}
			}
			return true
		})
		if err != nil {
			return jsondoc.Empty(), err
		}
	}

	return jsondoc.New(out), nil
}

// safeSetPath rewrites a gjson path into an sjson set path that addresses the
// intended literal keys. It handles two hazards:
//
//   - Numeric object keys: sjson treats an all-digit path segment as an array
//     index and pre-allocates an array of that size, so a large numeric object
//     key (e.g. a Cumulocity c8y_Dashboard widget id like "15426326034650895")
//     would make it try to allocate ~10^16 elements and exhaust memory. Each
//     all-digit segment whose parent in the source is an object is prefixed
//     with ':' to force a literal key; genuine array indices keep their form.
//   - Special characters: a key containing gjson/sjson syntax (|, #, @, *, ?)
//     is escaped so the segment is taken literally rather than as a query.
func safeSetPath(root gjson.Result, path string) string {
	segs := splitPath(path)
	node := root
	for i, seg := range segs {
		esc := escapeGJSONPath(seg)
		if isAllDigits(seg) && !node.IsArray() {
			esc = ":" + esc
		}
		node = node.Get(escapeGJSONPath(seg))
		segs[i] = esc
	}
	return strings.Join(segs, ".")
}

// splitPath splits a gjson path on unescaped '.' separators, keeping any
// escape sequences (e.g. "\.") within their segment.
func splitPath(path string) []string {
	var segs []string
	var cur strings.Builder
	for i := 0; i < len(path); i++ {
		if path[i] == '\\' && i+1 < len(path) {
			cur.WriteByte(path[i])
			cur.WriteByte(path[i+1])
			i++
			continue
		}
		if path[i] == '.' {
			segs = append(segs, cur.String())
			cur.Reset()
			continue
		}
		cur.WriteByte(path[i])
	}
	segs = append(segs, cur.String())
	return segs
}

// isAllDigits reports whether s is non-empty and consists only of ASCII
// digits — the form sjson interprets as an array index.
func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// walk visits every leaf of the document in document order with its dotted
// key path. Objects are traversed; arrays and scalars are leaves.
func walk(res gjson.Result, prefix string, visit func(string, gjson.Result) bool) bool {
	cont := true
	res.ForEach(func(k, v gjson.Result) bool {
		path := k.String()
		if prefix != "" {
			path = prefix + "." + path
		}
		if v.IsObject() {
			cont = walk(v, path, visit)
		} else {
			cont = visit(path, v)
		}
		return cont
	})
	return cont
}

func globPathToRegexp(pattern string) *regexp.Regexp {
	var sb strings.Builder
	sb.WriteString("(?i)^")
	rs := []rune(pattern)
	for i := 0; i < len(rs); i++ {
		// A backslash escapes the next character, so an escaped '*'/'?' (e.g.
		// a key that literally contains those characters) matches literally
		// rather than acting as a wildcard.
		if rs[i] == '\\' && i+1 < len(rs) {
			sb.WriteString(regexp.QuoteMeta(string(rs[i+1])))
			i++
			continue
		}
		switch rs[i] {
		case '*':
			sb.WriteString(".*")
		case '?':
			sb.WriteString(".")
		default:
			sb.WriteString(regexp.QuoteMeta(string(rs[i])))
		}
	}
	sb.WriteString("$")
	return regexp.MustCompile(sb.String())
}
