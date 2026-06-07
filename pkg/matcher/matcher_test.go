package matcher_test

import (
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/matcher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGlobMatchString(t *testing.T) {
	tests := []struct {
		pattern string
		input   string
		want    bool
	}{
		{"/alarms/*", "/alarms/12345", true},
		{"/alarms/*", "/alarms/", true},
		{"/alarms/*", "/events/12345", false},
		{"/events/?", "/events/1", true},
		{"/events/?", "/events/12", false},
		{"/measurements/123*", "/measurements/12345", true},
		{"/measurements/123*", "/measurements/999", false},
		// literal matching of regex special characters
		{"a.b", "a.b", true},
		{"a.b", "axb", false},
		{"(group)", "(group)", true},
		// case sensitive
		{"Alarm", "alarm", false},
		{"Alarm", "Alarm", true},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.input, func(t *testing.T) {
			g, err := matcher.Compile(tt.pattern)
			require.NoError(t, err)
			assert.Equal(t, tt.want, g.MatchString(tt.input))
		})
	}
}

func TestGlobString(t *testing.T) {
	g, err := matcher.Compile("/alarms/*")
	require.NoError(t, err)
	assert.Equal(t, "/alarms/*", g.String())
}

func TestMatchWithWildcards(t *testing.T) {
	tests := []struct {
		pattern string
		input   string
		want    bool
	}{
		// MatchWithWildcards is case-insensitive
		{"Alarm*", "alarm123", true},
		{"alarm*", "ALARM123", true},
		{"/events/?", "/events/1", true},
		{"/events/?", "/events/12", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.input, func(t *testing.T) {
			got, err := matcher.MatchWithWildcards(tt.input, tt.pattern)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMatchWithRegex(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		input   string
		want    bool
	}{
		{"prefix", "^/alarms/", "/alarms/123", true},
		{"case-insensitive", "alarm", "ALARM", true},
		{"no match", "^/events/", "/alarms/123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := matcher.MatchWithRegex(tt.input, tt.pattern)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMatchWithRegexInvalid(t *testing.T) {
	_, err := matcher.MatchWithRegex("anything", "[invalid(")
	assert.Error(t, err)
}
