package jsonUtilities_test

import (
	"encoding/json"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/jsonUtilities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergePatch(t *testing.T) {
	tests := []struct {
		name  string
		base  string
		patch string
		want  map[string]any
	}{
		{
			name:  "add new key",
			base:  `{"a":1}`,
			patch: `{"b":2}`,
			want:  map[string]any{"a": float64(1), "b": float64(2)},
		},
		{
			name:  "replace value",
			base:  `{"a":1}`,
			patch: `{"a":2}`,
			want:  map[string]any{"a": float64(2)},
		},
		{
			name:  "remove key with null",
			base:  `{"a":1,"b":2}`,
			patch: `{"b":null}`,
			want:  map[string]any{"a": float64(1)},
		},
		{
			name:  "recursive merge of nested objects",
			base:  `{"obj":{"a":1,"b":2}}`,
			patch: `{"obj":{"b":3,"c":4}}`,
			want:  map[string]any{"obj": map[string]any{"a": float64(1), "b": float64(3), "c": float64(4)}},
		},
		{
			name:  "patch object replaces non-object base value",
			base:  `{"a":1}`,
			patch: `{"a":{"nested":true}}`,
			want:  map[string]any{"a": map[string]any{"nested": true}},
		},
		{
			name:  "empty patch returns base",
			base:  `{"a":1}`,
			patch: ``,
			want:  map[string]any{"a": float64(1)},
		},
		{
			name:  "empty base treated as empty object",
			base:  ``,
			patch: `{"a":1}`,
			want:  map[string]any{"a": float64(1)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := jsonUtilities.MergePatch([]byte(tt.base), []byte(tt.patch))
			require.NoError(t, err)

			var gotMap map[string]any
			require.NoError(t, json.Unmarshal(got, &gotMap))
			assert.Equal(t, tt.want, gotMap)
		})
	}
}

func TestMergePatchInvalidInput(t *testing.T) {
	t.Run("invalid base", func(t *testing.T) {
		_, err := jsonUtilities.MergePatch([]byte(`{not json}`), []byte(`{"a":1}`))
		assert.Error(t, err)
	})

	t.Run("invalid patch", func(t *testing.T) {
		_, err := jsonUtilities.MergePatch([]byte(`{"a":1}`), []byte(`{not json}`))
		assert.Error(t, err)
	})
}
