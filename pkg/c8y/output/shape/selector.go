package shape

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/tidwall/sjson"
)

// KeyGroup records which concrete json keys a single select pattern resolved
// to within one row. Keys is empty when the pattern did not match anything
// in the row.
type KeyGroup struct {
	// Pattern original (non-lowercased) select pattern
	Pattern string

	// Keys resolved json keys (empty if the pattern did not match)
	Keys []string
}

// Selector is a compiled set of property-selection patterns (go-c8y-cli's
// --select). Pattern parsing, alias extraction and glob compilation happen
// once at construction; Apply performs a single flatten+match pass per
// document.
//
// A pattern is a case-insensitive path glob (see pathGlob) optionally
// prefixed with an alias ("alias:path"), which renames the matched keys.
// Selecting a path also selects its nested subtree. Patterns prefixed with
// '!' exclude keys matched by other patterns.
type Selector struct {
	patterns []selectPattern
	globOnly bool
}

type selectPattern struct {
	glob     *pathGlob
	alias    string
	original string // alias-stripped, original case
}

// NewSelector compiles selection patterns. Each property may contain
// multiple comma separated patterns. Invalid patterns are skipped (matching
// go-c8y-cli's behavior). When no patterns are given, everything is selected
// ("**").
func NewSelector(properties ...string) *Selector {
	patterns := make([]string, 0, len(properties))
	for _, p := range properties {
		patterns = append(patterns, strings.Split(p, ",")...)
	}
	if len(patterns) == 0 {
		patterns = []string{"**"}
	}

	s := &Selector{
		globOnly: len(patterns) == 1 && patterns[0] == "**",
	}
	for _, p := range patterns {
		alias := ""
		if idx := strings.Index(p, ":"); idx > -1 {
			alias = p[0:idx]
			p = p[idx+1:]
		}
		if p == "" {
			continue
		}
		g, err := compilePathGlob(strings.ToLower(p))
		if err != nil {
			continue
		}
		s.patterns = append(s.patterns, selectPattern{glob: g, alias: alias, original: p})
	}
	return s
}

// SelectsEverything reports whether the selector consists of the single
// globstar pattern, i.e. documents pass through unshaped.
func (s *Selector) SelectsEverything() bool {
	return s.globOnly
}

// Selection is the result of applying a Selector to a single document: the
// resolved keys (in pattern order, then natural key order), their values,
// and the keys grouped by the pattern which resolved them.
type Selection struct {
	keys   []string
	values map[string]any
	groups []KeyGroup
}

// Keys returns the resolved flat keys in selection order. Patterns which did
// not match are included as-is (so e.g. csv output keeps a stable column for
// them).
func (s *Selection) Keys() []string { return s.keys }

// Groups returns the resolved keys grouped by select pattern.
func (s *Selection) Groups() []KeyGroup { return s.groups }

// Size returns the number of distinct selected keys.
func (s *Selection) Size() int { return len(s.values) }

// Value returns the selected value for a resolved key.
func (s *Selection) Value(key string) (any, bool) {
	v, ok := s.values[key]
	return v, ok
}

// Apply selects properties from a single JSON object. The per-document work
// is one decode+flatten pass and one match per (pattern, key) pair; all
// pattern compilation already happened in NewSelector.
func (s *Selector) Apply(doc []byte) (*Selection, error) {
	nested := make(map[string]any)
	decoder := json.NewDecoder(bytes.NewReader(doc))
	decoder.UseNumber()
	if err := decoder.Decode(&nested); err != nil {
		return nil, err
	}

	src := flattenValue(nested)

	// Natural sort the source keys so matching is stable when a pattern can
	// match multiple values, and array elements order as 1, 2, 10.
	sourceKeys := make([]string, 0, len(src))
	for key := range src {
		sourceKeys = append(sourceKeys, key)
	}
	sort.Slice(sourceKeys, func(i, j int) bool { return naturalLess(sourceKeys[i], sourceKeys[j]) })

	sel := &Selection{values: make(map[string]any)}

	for _, p := range s.patterns {
		found := false
		group := KeyGroup{Pattern: p.original}
		for _, key := range sourceKeys {
			value := src[key]

			// normalize key, and strip the integer key marker
			keyl := strings.ReplaceAll(strings.ToLower(key), keyPrefix, "")

			if strings.HasPrefix(keyl, p.glob.String()+".") || (p.glob.MatchString(keyl) && !p.glob.IsNegative()) {
				key = applyAlias(p, key, keyl)
				sel.values[key] = value
				sel.keys = append(sel.keys, key)
				group.Keys = append(group.Keys, key)
				found = true
			}
		}
		if !found && !p.glob.IsNegative() {
			// store non-matching patterns so e.g. csv output keeps a stable
			// column for them
			sel.keys = append(sel.keys, group.Pattern)
		}
		if !p.glob.IsNegative() {
			sel.groups = append(sel.groups, group)
		}
	}

	// filter out keys matched by negated patterns
	isNegatedKey := func(key string) bool {
		keyl := strings.ToLower(key)
		for _, p := range s.patterns {
			if p.glob.IsNegative() && p.glob.MatchString(keyl) {
				return true
			}
		}
		return false
	}

	matchingKeys := make([]string, 0, len(sel.keys))
	for _, key := range sel.keys {
		if isNegatedKey(key) {
			delete(sel.values, key)
		} else {
			matchingKeys = append(matchingKeys, key)
		}
	}
	sel.keys = matchingKeys
	for i := range sel.groups {
		groupKeys := make([]string, 0, len(sel.groups[i].Keys))
		for _, key := range sel.groups[i].Keys {
			if !isNegatedKey(key) {
				groupKeys = append(groupKeys, key)
			}
		}
		sel.groups[i].Keys = groupKeys
	}

	return sel, nil
}

// applyAlias renames a matched key using the pattern's alias. Wildcard
// patterns keep the unmatched remainder of the key below the alias.
func applyAlias(p selectPattern, key, keyl string) string {
	if p.alias == "" {
		return key
	}
	pattern := p.glob.String()
	if !strings.Contains(pattern, "*") {
		return p.alias
	}

	if strings.HasPrefix(pattern, "*") {
		return p.alias + "." + key
	}
	if strings.HasSuffix(pattern, "*") {
		paths := strings.Split(pattern, ".")
		keyPaths := strings.Split(key, ".")
		commonpath := bytes.Buffer{}
		for idxPart, part := range paths {
			if strings.Contains(part, "**") || part == "*" {
				break
			}
			// get the real key path rather than the wildcard
			if strings.Contains(part, "*") && idxPart < len(keyPaths) {
				part = keyPaths[idxPart]
				commonpath.WriteString("." + part)
				break
			}
			commonpath.WriteString("." + part)
		}
		commonprefix := strings.TrimLeft(commonpath.String(), ".")
		if strings.HasPrefix(keyl, strings.ToLower(commonprefix)) {
			return p.alias + key[len(commonprefix):]
		}
	}
	return p.alias
}

// JSON rebuilds the selection as a nested JSON document, with key order
// following the selection order.
func (s *Selection) JSON() ([]byte, error) {
	output := []byte("{}")
	var err error
	for _, k := range s.keys {
		if v, ok := s.values[k]; ok {
			k = strings.ReplaceAll(k, keyPrefix, ":")
			output, err = sjson.SetBytes(output, escapeGJSONPath(k), v)
			if err != nil {
				return nil, err
			}
		}
	}
	return output, nil
}

// FlatJSON renders the selection as a flat JSON object of dot-separated
// keys (alphabetically ordered, like encoding/json map marshalling).
func (s *Selection) FlatJSON() ([]byte, error) {
	return json.Marshal(s.values)
}

// CSV renders the selection's values joined by the separator, in selection
// order. Unresolved keys produce empty fields. String values are unquoted
// unless they contain the separator-sensitive comma.
func (s *Selection) CSV(separator string) string {
	buf := bytes.Buffer{}
	if separator == "" {
		separator = ","
	}
	for i, key := range s.keys {
		if i != 0 {
			// handle for empty non-existent values by leaving it blank
			buf.WriteString(separator)
		}
		if value, ok := s.values[key]; ok {
			marshalledValue, err := json.Marshal(value)
			if err != nil {
				continue
			}
			if !bytes.Contains(marshalledValue, []byte(",")) {
				buf.Write(bytes.Trim(marshalledValue, "\""))
			} else {
				buf.Write(marshalledValue)
			}
		}
	}
	return buf.String()
}

// String implements fmt.Stringer for debugging.
func (s *Selection) String() string {
	return fmt.Sprintf("Selection{keys: %v}", s.keys)
}
