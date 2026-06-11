package shape

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/tidwall/gjson"
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

	// pruneSegments, when non-nil, holds the lowercased top-level segments
	// the positive patterns can match. Documents are reduced to these
	// top-level fields before flattening — a large win when a few
	// properties are selected from large documents. It is nil (no pruning)
	// when any positive pattern starts with a wildcard or escape sequence.
	pruneSegments map[string]struct{}
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
	s.pruneSegments = resolvePruneSegments(s.patterns)
	return s
}

// resolvePruneSegments determines the set of top-level document fields the
// positive patterns can possibly match. It returns nil (pruning disabled)
// when a positive pattern's first segment is not a plain literal.
func resolvePruneSegments(patterns []selectPattern) map[string]struct{} {
	segments := make(map[string]struct{}, len(patterns))
	for _, p := range patterns {
		if p.glob.IsNegative() {
			// negative patterns only exclude already-selected keys
			continue
		}
		pattern := strings.TrimLeft(p.glob.String(), "!")
		seg, _, _ := strings.Cut(pattern, ".")
		if seg == "" || strings.ContainsAny(seg, `*?\[{`) {
			return nil
		}
		segments[seg] = struct{}{}
	}
	if len(segments) == 0 {
		return nil
	}
	return segments
}

// pruneDocument reduces a JSON object to the top-level fields which the
// selector's patterns can match, so only the relevant subtrees are decoded
// and flattened.
func pruneDocument(doc []byte, segments map[string]struct{}) []byte {
	root := gjson.ParseBytes(doc)
	if !root.IsObject() {
		return doc
	}
	var buf bytes.Buffer
	buf.Grow(len(doc) / 4)
	buf.WriteByte('{')
	first := true
	root.ForEach(func(k, v gjson.Result) bool {
		if _, ok := segments[strings.ToLower(k.String())]; !ok {
			return true
		}
		if !first {
			buf.WriteByte(',')
		}
		first = false
		if k.Raw != "" {
			buf.WriteString(k.Raw)
		} else {
			b, _ := json.Marshal(k.String())
			buf.Write(b)
		}
		buf.WriteByte(':')
		buf.WriteString(v.Raw)
		return true
	})
	buf.WriteByte('}')
	return buf.Bytes()
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
	src, err := s.flattenDoc(doc)
	if err != nil {
		return nil, err
	}
	sourceKeys := sortedNaturalKeys(src)

	sel := &Selection{values: make(map[string]any)}
	for _, p := range s.patterns {
		matchPattern(p, sourceKeys, src, sel)
	}
	s.removeNegatedKeys(sel)
	return sel, nil
}

func (s *Selector) flattenDoc(doc []byte) (map[string]any, error) {
	if s.pruneSegments != nil {
		doc = pruneDocument(doc, s.pruneSegments)
	}
	nested := make(map[string]any)
	decoder := json.NewDecoder(bytes.NewReader(doc))
	decoder.UseNumber()
	if err := decoder.Decode(&nested); err != nil {
		return nil, err
	}
	return flattenValue(nested), nil
}

// sortedNaturalKeys returns the map keys natural-sorted so matching is stable
// when a pattern can match multiple values, and array elements order as
// 1, 2, 10.
func sortedNaturalKeys(src map[string]any) []string {
	keys := make([]string, 0, len(src))
	for key := range src {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return naturalLess(keys[i], keys[j]) })
	return keys
}

// matchPattern selects the source keys matching a single pattern into sel.
func matchPattern(p selectPattern, sourceKeys []string, src map[string]any, sel *Selection) {
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

// removeNegatedKeys filters out keys matched by negated patterns.
func (s *Selector) removeNegatedKeys(sel *Selection) {
	matchingKeys := make([]string, 0, len(sel.keys))
	for _, key := range sel.keys {
		if s.isNegatedKey(key) {
			delete(sel.values, key)
		} else {
			matchingKeys = append(matchingKeys, key)
		}
	}
	sel.keys = matchingKeys
	for i := range sel.groups {
		groupKeys := make([]string, 0, len(sel.groups[i].Keys))
		for _, key := range sel.groups[i].Keys {
			if !s.isNegatedKey(key) {
				groupKeys = append(groupKeys, key)
			}
		}
		sel.groups[i].Keys = groupKeys
	}
}

func (s *Selector) isNegatedKey(key string) bool {
	keyl := strings.ToLower(key)
	for _, p := range s.patterns {
		if p.glob.IsNegative() && p.glob.MatchString(keyl) {
			return true
		}
	}
	return false
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
		if aliased, ok := aliasWildcardSuffix(p.alias, pattern, key, keyl); ok {
			return aliased
		}
	}
	return p.alias
}

// aliasWildcardSuffix keeps the unmatched remainder of the key below the
// alias for trailing-wildcard patterns (e.g. "c8y_Hardware.*").
func aliasWildcardSuffix(alias, pattern, key, keyl string) (string, bool) {
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
		return alias + key[len(commonprefix):], true
	}
	return "", false
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
