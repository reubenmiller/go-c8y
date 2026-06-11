package filter_test

import (
	"encoding/json"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/output/filter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var doc = jsondoc.New([]byte(`{
	"id": "12345",
	"name": "Linux-Device-0001",
	"type": "c8y_Linux",
	"count": 42,
	"active": true,
	"creationTime": "2026-01-15T10:00:00Z",
	"c8y_Hardware": {"model": "RPi4", "serialNumber": "SN-001"},
	"c8y_SupportedOperations": ["c8y_Restart", "c8y_Configuration"],
	"description": "Primary gateway device"
}`))

func match(t *testing.T, expr string) bool {
	t.Helper()
	p, err := filter.Parse(expr)
	require.NoError(t, err, "expression %q must parse", expr)
	return p(doc)
}

func TestParseOperators(t *testing.T) {
	cases := []struct {
		expr string
		want bool
	}{
		// like / notlike: case-insensitive, anchored wildcards
		{"name like linux*", true},
		{"name like LINUX*", true},
		{"name like linux", false},
		{"name notlike linux*", false},
		{"name -like linux*", true},

		// match / notmatch: case-insensitive, unanchored regex
		{"name match device-\\d+", true},
		{"name match ^device", false},
		{"name notmatch ^device", true},

		// equality with type coercion
		{"count eq 42", true},
		{"count = 42", true},
		{"count neq 42", false},
		{"active eq true", true},
		{"type eq c8y_Linux", true},
		{"id eq '12345'", true}, // quoted: string comparison
		{"id eq \"12345\"", true},

		// numeric comparison
		{"count gt 41", true},
		{"count gt 42", false},
		{"count gte 42", true},
		{"count lt 100", true},
		{"count lte 41", false},

		// contains (default operator) — case-insensitive
		{"description contains gateway", true},
		{"description contains GATEWAY", true},
		{"description strictcontains GATEWAY", false},
		{"Primary", true},      // bare value → contains on whole document
		{"NoSuchValue", false}, // bare value → contains on whole document

		// prefix/suffix
		{"name startswith Linux", true},
		{"name startswith linux", false},
		{"name endswith 0001", true},

		// key existence
		{"has c8y_Hardware", true},
		{"has c8y_Hardware.model", true},
		{"has c8y_Mobile", false},
		{"hasnot c8y_Mobile", true},
		{"missing c8y_Mobile", true},
		{"c8y_Hardware has model", true},

		// dates
		{"creationTime datelt 2026-06-01", true},
		{"creationTime dategt 2026-06-01", false},
		{"creationTime newerthan 2025-01-01", true},

		// dates with relative references (resolved against now)
		{"creationTime datelt -1h", true},
		{"creationTime dategt -1h", false},
		{"creationTime datelt now", true},
		{"creationTime dategt '2026-01-15T10:00:00Z' - 1h", true},
		{"creationTime datelt '2026-01-15T10:00:00Z' + 1h", true},
		{"creationTime dategt '2026-01-15T10:00:00Z' + 1h", false},

		// length
		{"c8y_SupportedOperations leneq 2", true},
		{"c8y_SupportedOperations lengt 1", true},
		{"c8y_SupportedOperations lenlt 2", false},

		// array membership
		{"c8y_SupportedOperations includes c8y_Restart", true},
		{"c8y_SupportedOperations includes c8y_Reboot", false},
		{"c8y_SupportedOperations notincludes c8y_Reboot", true},

		// nested paths
		{"c8y_Hardware.model eq RPi4", true},
		{"c8y_Hardware.serialNumber like sn-*", true},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, match(t, tc.expr), "expression: %s", tc.expr)
	}
}

func TestParseMultipleExpressionsAreANDed(t *testing.T) {
	p, err := filter.Parse("name like linux*", "count gt 40")
	require.NoError(t, err)
	assert.True(t, p(doc))

	p, err = filter.Parse("name like linux*", "count gt 100")
	require.NoError(t, err)
	assert.False(t, p(doc))
}

func TestParseQuotedValuesWithSpaces(t *testing.T) {
	p, err := filter.Parse(`description eq 'Primary gateway device'`)
	require.NoError(t, err)
	assert.True(t, p(doc))
}

func TestParseErrors(t *testing.T) {
	for _, expr := range []string{
		"",
		"name unknownop value",
		"count gt notanumber",
		"name match [invalid",
		"creationTime datelt notadate",
	} {
		_, err := filter.Parse(expr)
		assert.Error(t, err, "expression %q must fail to parse", expr)
	}
}

func TestCondition(t *testing.T) {
	cases := []struct {
		property string
		operator string
		value    any
		want     bool
	}{
		{"count", "eq", 42, true},
		{"count", "eq", float64(42), true},
		{"count", "gt", 41, true},
		{"count", "eq", "42", true}, // string compare against number still matches via String()
		{"active", "eq", true, true},
		{"name", "like", "linux*", true},
		{"name", "keyIn", "doesNotExist", false},
		{".", "keyIn", "c8y_Hardware", true},
	}
	for _, tc := range cases {
		p, err := filter.Condition(tc.property, tc.operator, tc.value)
		require.NoError(t, err, "%s %s %v", tc.property, tc.operator, tc.value)
		assert.Equal(t, tc.want, p(doc), "%s %s %v", tc.property, tc.operator, tc.value)
	}

	_, err := filter.Condition("a", "eq", []string{"x"})
	assert.Error(t, err, "unsupported value types must error so callers can fall back")
}

// TestVersionOperator mirrors go-c8y-cli's legacy gojsonq "version" macro:
// semver constraint checks where invalid versions (including "") are treated
// as 0.0.0, build metadata is ignored when comparing, and non-string values
// never match.
func TestVersionOperator(t *testing.T) {
	versionDoc := func(v string) jsondoc.JSONDoc {
		b, _ := json.Marshal(map[string]string{"value": v})
		return jsondoc.New(b)
	}
	cases := []struct {
		constraint string
		value      string
		want       bool
	}{
		{">=1.10.3+deb10", "1.10.3+deb10", true},
		{">=1.10.3+deb10", "2.0.0", true},
		{"<1.99", "1.10.3+deb10", true},
		{"<1.99", "2.0.0", false},
		// build metadata is not significant: 1.10.3+deb10 satisfies >=1.10.3+deb11
		{">=1.10.3+deb11, < 2.0.0", "1.10.3+deb10", true},
		{">=1.10.3+deb11, < 2.0.0", "2.0.0", false},
		// invalid/empty versions are treated as 0.0.0
		{">0", "", false},
		{">=0", "", true},
		{"<1.0", "not-a-version", true},
	}
	for _, tc := range cases {
		p, err := filter.Condition("value", "version", tc.constraint)
		require.NoError(t, err, "constraint %q", tc.constraint)
		assert.Equal(t, tc.want, p(versionDoc(tc.value)), "value=%q version %s", tc.value, tc.constraint)
	}

	// version expressions also work via Parse
	p, err := filter.Parse("value version >=1.0.0")
	require.NoError(t, err)
	assert.True(t, p(versionDoc("1.2.3")))
	assert.False(t, p(versionDoc("0.9.0")))

	// non-string values never match
	p, err = filter.Condition("count", "version", ">=0")
	require.NoError(t, err)
	assert.False(t, p(doc))

	// missing properties never match
	assert.False(t, p(jsondoc.New([]byte(`{}`))))

	// invalid constraints fail at compile time
	_, err = filter.Condition("value", "version", "not >= a constraint")
	assert.Error(t, err)
}

func TestParseMissingPropertyNeverMatches(t *testing.T) {
	assert.False(t, match(t, "doesNotExist like *"))
	assert.False(t, match(t, "doesNotExist gt 1"))
	assert.False(t, match(t, "doesNotExist datelt 2030-01-01"))
}
