package source

import "strings"

// DefaultSchemes are the reference schemes understood by the SDK's resolvers.
// ParseRef keeps a value carrying one of them unchanged.
var DefaultSchemes = []string{"id", "name", "ext", "query"}

// ParseRef turns free-form user input (for example a CLI flag value) into a
// resolver reference, using the convention every resolver in the SDK accepts:
//
//   - ""                 -> ""
//   - "12345"            -> "12345"           (all digits: an id)
//   - "name:dev01", "ext:c8y_Serial:ABC", "query:...", "id:1" -> unchanged
//   - "dev01"            -> "name:dev01"      (anything else is a name)
//   - "83001,1.0.0"      -> "83001"           (comma list: the first id wins,
//     otherwise the first non-empty part is a name)
//
// extraSchemes adds service-specific schemes (e.g. "contextPath") that are
// kept unchanged as well.
func ParseRef(value string, extraSchemes ...string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if hasScheme(value, extraSchemes) {
		return value
	}
	if strings.Contains(value, ",") {
		name := ""
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			if IsID(part) {
				return part
			}
			if name == "" {
				name = part
			}
		}
		if name == "" {
			return ""
		}
		value = name
		if hasScheme(value, extraSchemes) {
			return value
		}
	}
	if IsID(value) {
		return value
	}
	return "name:" + value
}

// IsID reports whether s looks like a Cumulocity id (non-empty, all digits).
func IsID(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func hasScheme(value string, extra []string) bool {
	idx := strings.IndexByte(value, ':')
	if idx <= 0 {
		return false
	}
	prefix := value[:idx]
	for _, s := range DefaultSchemes {
		if prefix == s {
			return true
		}
	}
	for _, s := range extra {
		if prefix == s {
			return true
		}
	}
	return false
}
