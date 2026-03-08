package fakeserver

import (
	"encoding/json"
	"net/http"
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
		if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
			filters = append(filters, dateAfterOrEqual("time", t))
		}
	}
	if v := q.Get("dateTo"); v != "" {
		if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
			filters = append(filters, dateBeforeOrEqual("time", t))
		}
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

func dateAfterOrEqual(path string, threshold time.Time) func(json.RawMessage) bool {
	// Truncate to seconds to match Cumulocity query parameter precision (RFC3339, no sub-seconds).
	threshold = threshold.Truncate(time.Second)
	return func(doc json.RawMessage) bool {
		val := getJSONPath(doc, path)
		t, err := time.Parse(time.RFC3339Nano, val)
		if err != nil {
			t, err = time.Parse(time.RFC3339, val)
			if err != nil {
				return true // if we can't parse, include it
			}
		}
		return !t.Truncate(time.Second).Before(threshold)
	}
}

func dateBeforeOrEqual(path string, threshold time.Time) func(json.RawMessage) bool {
	// Truncate to seconds to match Cumulocity query parameter precision (RFC3339, no sub-seconds).
	threshold = threshold.Truncate(time.Second)
	return func(doc json.RawMessage) bool {
		val := getJSONPath(doc, path)
		t, err := time.Parse(time.RFC3339Nano, val)
		if err != nil {
			t, err = time.Parse(time.RFC3339, val)
			if err != nil {
				return true
			}
		}
		return !t.Truncate(time.Second).After(threshold)
	}
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
