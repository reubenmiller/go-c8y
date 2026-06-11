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
		v := root.Get(path)
		if !v.Exists() {
			continue
		}
		out, err = sjson.SetRawBytesOptions(out, path, []byte(v.Raw), setOptions)
		if err != nil {
			return jsondoc.Empty(), err
		}
	}

	if len(s.wildcards) > 0 {
		walk(root, "", func(path string, v gjson.Result) bool {
			for _, re := range s.wildcards {
				if re.MatchString(path) {
					out, err = sjson.SetRawBytesOptions(out, path, []byte(v.Raw), setOptions)
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
