package shape

import (
	"strconv"
	"strings"
)

// keyPrefix marks map keys that look like integers, so they can be
// distinguished from array indices when the selection is rebuilt into a
// nested document. Matches go-c8y-cli's pkg/flatten.
const keyPrefix = "::k::"

// flattenValue flattens a decoded JSON document into dot-separated leaf
// paths. Empty objects and arrays are leaves; dots inside keys are escaped;
// integer-looking map keys are marked with keyPrefix.
// Credit to https://github.com/jeremywohl/flatten (same semantics as
// go-c8y-cli's pkg/flatten with the dot style).
func flattenValue(nested map[string]any) map[string]any {
	flatMap := make(map[string]any)
	flatten(true, flatMap, nested, "")
	return flatMap
}

func flatten(top bool, flatMap map[string]any, nested any, prefix string) {
	assign := func(newKey string, v any) {
		switch typedV := v.(type) {
		case map[string]any:
			if len(typedV) == 0 {
				flatMap[newKey] = typedV
				return
			}
			flatten(false, flatMap, v, newKey)
		case []any:
			if len(typedV) == 0 {
				flatMap[newKey] = typedV
				return
			}
			flatten(false, flatMap, v, newKey)
		default:
			flatMap[newKey] = v
		}
	}

	switch nestedValue := nested.(type) {
	case map[string]any:
		if len(nestedValue) == 0 {
			assign(prefix, nestedValue)
			return
		}
		for k, v := range nestedValue {
			if isInteger(k) {
				k = keyPrefix + k
			}
			assign(enkey(top, prefix, k), v)
		}
	case []any:
		if len(nestedValue) == 0 {
			assign(prefix, nestedValue)
			return
		}
		for i, v := range nestedValue {
			assign(enkey(top, prefix, strconv.Itoa(i)), v)
		}
	}
}

func enkey(top bool, prefix, subkey string) string {
	if strings.Contains(subkey, ".") {
		subkey = strings.ReplaceAll(subkey, ".", "\\.")
	}
	if top {
		return prefix + subkey
	}
	return prefix + "." + subkey
}

func isInteger(v string) bool {
	value := strings.TrimSpace(v)
	if value == "" {
		return true
	}
	for _, c := range value {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// escapeGJSONPath escapes the special characters of a gjson/sjson path
// segment. Matches go-c8y-cli's pkg/gjsonpath.EscapePath.
func escapeGJSONPath(s string) string {
	s = strings.ReplaceAll(s, "|", "\\|")
	s = strings.ReplaceAll(s, "#", "\\#")
	s = strings.ReplaceAll(s, "@", "\\@")
	s = strings.ReplaceAll(s, "*", "\\*")
	return strings.ReplaceAll(s, "?", "\\?")
}
