package op

import (
	"testing"
)

func TestDesiredStateMatches(t *testing.T) {
	existing := []byte(`{
		"id": "12345",
		"name": "tedge",
		"type": "c8y_Software",
		"softwareType": "apt",
		"description": "Software package: tedge",
		"c8y_Filter": {"type": "arm64"},
		"c8y_Global": {},
		"lastUpdated": "2026-06-10T10:30:00Z",
		"additionParents": {"references": []},
		"count": 3,
		"tags": ["a", "b"]
	}`)

	testCases := []struct {
		name    string
		desired map[string]any
		matches bool
	}{
		{
			name:    "empty desired always matches",
			desired: map[string]any{},
			matches: true,
		},
		{
			name: "subset of fields matches",
			desired: map[string]any{
				"name":         "tedge",
				"softwareType": "apt",
			},
			matches: true,
		},
		{
			name: "nested object subset matches",
			desired: map[string]any{
				"c8y_Filter": map[string]any{"type": "arm64"},
			},
			matches: true,
		},
		{
			name: "nested object via typed map matches",
			desired: map[string]any{
				"c8y_Filter": map[string]string{"type": "arm64"},
			},
			matches: true,
		},
		{
			name: "different scalar value does not match",
			desired: map[string]any{
				"description": "changed",
			},
			matches: false,
		},
		{
			name: "missing field does not match",
			desired: map[string]any{
				"newFragment": map[string]any{"a": 1},
			},
			matches: false,
		},
		{
			name: "different nested value does not match",
			desired: map[string]any{
				"c8y_Filter": map[string]any{"type": "amd64"},
			},
			matches: false,
		},
		{
			name: "numbers compare after JSON normalisation",
			desired: map[string]any{
				"count": 3,
			},
			matches: true,
		},
		{
			name: "arrays compare wholesale - equal",
			desired: map[string]any{
				"tags": []string{"a", "b"},
			},
			matches: true,
		},
		{
			name: "arrays compare wholesale - reordered is a change",
			desired: map[string]any{
				"tags": []string{"b", "a"},
			},
			matches: false,
		},
		{
			name: "object vs scalar does not match",
			desired: map[string]any{
				"name": map[string]any{"x": 1},
			},
			matches: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := DesiredStateMatches(tc.desired, existing); got != tc.matches {
				t.Errorf("DesiredStateMatches() = %v, want %v", got, tc.matches)
			}
		})
	}
}

func TestDesiredStateMatchesInvalidInputs(t *testing.T) {
	if DesiredStateMatches(map[string]any{"a": 1}, []byte("not-json")) {
		t.Error("invalid existing JSON should not match")
	}
	if DesiredStateMatches([]string{"not", "an", "object"}, []byte(`{}`)) {
		t.Error("non-object desired should not match")
	}
	if DesiredStateMatches(map[string]any{"a": func() {}}, []byte(`{}`)) {
		t.Error("unmarshallable desired should not match")
	}
}
