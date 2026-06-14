package fakeserver

import (
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

// FilterItems applies standard Cumulocity query parameters to a slice of documents.
// Supported filters: source, type, status, severity, dateFrom, dateTo, fragmentType, text,
// deviceId, agentId.
func FilterItems(r *http.Request, items []json.RawMessage) []json.RawMessage {
	q := r.URL.Query()
	var filters []func(json.RawMessage) bool

	if v := q.Get("source"); v != "" {
		filters = append(filters, fieldEquals("source.id", v))
	}
	if v := q.Get("type"); v != "" {
		// type can be comma-separated for alarms
		types := strings.Split(v, ",")
		filters = append(filters, fieldIn("type", types))
	}
	if v := q.Get("status"); v != "" {
		statuses := strings.Split(v, ",")
		filters = append(filters, fieldIn("status", statuses))
	}
	if v := q.Get("severity"); v != "" {
		severities := strings.Split(v, ",")
		filters = append(filters, fieldIn("severity", severities))
	}
	if v := q.Get("dateFrom"); v != "" {
		filters = append(filters, dateAfterOrEqual(v))
	}
	if v := q.Get("dateTo"); v != "" {
		filters = append(filters, dateBeforeOrEqual(v))
	}
	if v := q.Get("fragmentType"); v != "" {
		filters = append(filters, hasField(v))
	}
	if v := q.Get("text"); v != "" {
		filters = append(filters, fieldContains("text", v))
	}
	if v := q.Get("deviceId"); v != "" {
		filters = append(filters, fieldEquals("deviceId", v))
	}
	if v := q.Get("agentId"); v != "" {
		filters = append(filters, fieldEquals("agentId", v))
	}
	if v := q.Get("name"); v != "" {
		filters = append(filters, fieldEquals("name", v))
	}

	if len(filters) == 0 {
		return items
	}

	var result []json.RawMessage
	for _, item := range items {
		match := true
		for _, f := range filters {
			if !f(item) {
				match = false
				break
			}
		}
		if match {
			result = append(result, item)
		}
	}
	return result
}

// ReverseItems returns items in reverse order (newest first, matching Cumulocity default).
func ReverseItems(items []json.RawMessage) []json.RawMessage {
	n := len(items)
	out := make([]json.RawMessage, n)
	for i, item := range items {
		out[n-1-i] = item
	}
	return out
}

// --- filter predicates ---

func fieldEquals(path, value string) func(json.RawMessage) bool {
	return func(doc json.RawMessage) bool {
		return getJSONPath(doc, path) == value
	}
}

func fieldIn(path string, values []string) func(json.RawMessage) bool {
	set := make(map[string]struct{}, len(values))
	for _, v := range values {
		set[strings.TrimSpace(v)] = struct{}{}
	}
	return func(doc json.RawMessage) bool {
		val := getJSONPath(doc, path)
		_, ok := set[val]
		return ok
	}
}

func hasField(field string) func(json.RawMessage) bool {
	return func(doc json.RawMessage) bool {
		var m map[string]json.RawMessage
		if err := json.Unmarshal(doc, &m); err != nil {
			return false
		}
		_, ok := m[field]
		return ok
	}
}

func fieldContains(path, substr string) func(json.RawMessage) bool {
	lower := strings.ToLower(substr)
	return func(doc json.RawMessage) bool {
		val := getJSONPath(doc, path)
		return strings.Contains(strings.ToLower(val), lower)
	}
}

// dateAfterOrEqual builds a "doc.time >= dateFrom" predicate. The comparison
// precision follows the supplied query value: a value without a fractional
// second is compared at second granularity (matching a second-precision client),
// a value with sub-second digits is compared at full precision (so a
// millisecond keyset cursor positions exactly between same-second items).
func dateAfterOrEqual(raw string) func(json.RawMessage) bool {
	threshold, prec, ok := parseQueryTime(raw)
	if !ok {
		return func(json.RawMessage) bool { return true }
	}
	threshold = threshold.Truncate(prec)
	return func(doc json.RawMessage) bool {
		t, ok := docTime(doc)
		if !ok {
			return true // if we can't parse, include it
		}
		return !t.Truncate(prec).Before(threshold)
	}
}

// dateBeforeOrEqual builds a "doc.time <= dateTo" predicate with the same
// precision-adaptive comparison as dateAfterOrEqual.
func dateBeforeOrEqual(raw string) func(json.RawMessage) bool {
	threshold, prec, ok := parseQueryTime(raw)
	if !ok {
		return func(json.RawMessage) bool { return true }
	}
	threshold = threshold.Truncate(prec)
	return func(doc json.RawMessage) bool {
		t, ok := docTime(doc)
		if !ok {
			return true
		}
		return !t.Truncate(prec).After(threshold)
	}
}

// parseQueryTime parses an RFC3339(/Nano) timestamp from a query parameter and
// reports the precision implied by its fractional-second digits: no fraction →
// seconds, 3 digits → milliseconds (Cumulocity's resolution and what the SDK
// sends), 9 → nanoseconds, etc. Stored item times are compared truncated to this
// precision, so a millisecond cursor matches millisecond-stored data exactly
// instead of being defeated by sub-millisecond noise.
func parseQueryTime(raw string) (time.Time, time.Duration, bool) {
	t, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		t, err = time.Parse(time.RFC3339, raw)
		if err != nil {
			return time.Time{}, 0, false
		}
	}
	return t, fractionalPrecision(raw), true
}

// fractionalPrecision returns the time precision implied by the count of
// fractional-second digits in an RFC3339 value (0 → second, 3 → millisecond,
// 9 → nanosecond), clamped at nanosecond.
func fractionalPrecision(raw string) time.Duration {
	dot := strings.LastIndex(raw, ".")
	if dot < 0 {
		return time.Second
	}
	n := 0
	for i := dot + 1; i < len(raw) && raw[i] >= '0' && raw[i] <= '9'; i++ {
		n++
	}
	p := time.Second
	for i := 0; i < n && p > time.Nanosecond; i++ {
		p /= 10
	}
	return p
}

// docTime returns the document's logical timestamp, preferring the "time" field
// (events/alarms/measurements) and falling back to "creationTime" (operations,
// audit records). The boolean reports whether a timestamp was found.
func docTime(doc json.RawMessage) (time.Time, bool) {
	for _, field := range []string{"time", "creationTime"} {
		val := getJSONPath(doc, field)
		if val == "" {
			continue
		}
		if t, err := time.Parse(time.RFC3339Nano, val); err == nil {
			return t, true
		}
		if t, err := time.Parse(time.RFC3339, val); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// OrderTimeItems sorts time-series items the way Cumulocity does: newest first by
// default, oldest first when revert=true. Items sharing a timestamp are ordered
// by numeric id in the same direction, giving a stable, deterministic order at
// page boundaries (real Cumulocity tie-breaks on id; the old reverse-insertion
// behaviour was non-deterministic for duplicate timestamps). Sorts in place; the
// caller passes a freshly built slice.
func OrderTimeItems(r *http.Request, items []json.RawMessage) []json.RawMessage {
	ascending := queryBool(r, "revert")
	sort.SliceStable(items, func(i, j int) bool {
		ti, _ := docTime(items[i])
		tj, _ := docTime(items[j])
		if !ti.Equal(tj) {
			if ascending {
				return ti.Before(tj)
			}
			return ti.After(tj)
		}
		ii := itemIDNum(items[i])
		ij := itemIDNum(items[j])
		if ascending {
			return ii < ij
		}
		return ii > ij
	})
	return items
}

// queryBool reads a boolean query parameter (true only for the literal "true").
func queryBool(r *http.Request, key string) bool {
	return r.URL.Query().Get(key) == "true"
}

// itemIDNum returns the document's numeric id, or 0 when absent/non-numeric.
func itemIDNum(doc json.RawMessage) int64 {
	n, _ := strconv.ParseInt(getJSONPath(doc, "id"), 10, 64)
	return n
}

// compareValues compares two scalar query values, numerically when both parse as
// numbers and lexically otherwise (RFC3339 timestamps order correctly either
// way). Returns -1, 0 or 1.
func compareValues(a, b string) int {
	if af, aerr := strconv.ParseFloat(a, 64); aerr == nil {
		if bf, berr := strconv.ParseFloat(b, 64); berr == nil {
			switch {
			case af < bf:
				return -1
			case af > bf:
				return 1
			default:
				return 0
			}
		}
	}
	return strings.Compare(a, b)
}

// getJSONPath extracts a dotted path value from raw JSON.
// Supports simple paths like "source.id" (max depth 3).
func getJSONPath(doc json.RawMessage, path string) string {
	parts := strings.Split(path, ".")
	var current any
	if err := json.Unmarshal(doc, &current); err != nil {
		return ""
	}
	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current = m[part]
	}
	switch v := current.(type) {
	case string:
		return v
	case float64:
		if v == float64(int64(v)) {
			return strings.TrimRight(strings.TrimRight(json.Number(strings.Replace(
				strings.Replace(json.Number("").String(), "", "", 1), "", "", 1)).String(), "0"), ".")
		}
		b, _ := json.Marshal(v)
		return string(b)
	case json.Number:
		return v.String()
	default:
		if current == nil {
			return ""
		}
		b, _ := json.Marshal(current)
		return strings.Trim(string(b), `"`)
	}
}
