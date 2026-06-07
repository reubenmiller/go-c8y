package jsonUtilities_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/jsonUtilities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsValidJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"object", `{"name":"test"}`, true},
		{"array", `[1,2,3]`, true},
		{"object with whitespace", "  {\"a\":1}\n", true},
		{"bare string", `"hello"`, false},
		{"bare number", `123`, false},
		{"invalid", `{not json}`, false},
		{"empty", ``, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, jsonUtilities.IsValidJSON([]byte(tt.input)))
		})
	}
}

func TestIsJSONArrayAndObject(t *testing.T) {
	assert.True(t, jsonUtilities.IsJSONArray([]byte(`[1,2]`)))
	assert.False(t, jsonUtilities.IsJSONArray([]byte(`{"a":1}`)))
	assert.True(t, jsonUtilities.IsJSONObject([]byte(`{"a":1}`)))
	assert.False(t, jsonUtilities.IsJSONObject([]byte(`[1,2]`)))
}

func TestUnescapeJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"unicode escape", `é`, "é"},
		{"plain text", `hello`, "hello"},
		{"newline escape", `line1\nline2`, "line1\nline2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, jsonUtilities.UnescapeJSON([]byte(tt.input)))
		})
	}
}

func TestDecodeJSONFile(t *testing.T) {
	dir := t.TempDir()

	t.Run("valid", func(t *testing.T) {
		path := filepath.Join(dir, "valid.json")
		require.NoError(t, os.WriteFile(path, []byte(`{"name":"test","count":3}`), 0600))

		got, err := jsonUtilities.DecodeJSONFile(path)
		require.NoError(t, err)
		assert.Equal(t, "test", got["name"])
		assert.EqualValues(t, 3, got["count"])
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := jsonUtilities.DecodeJSONFile(filepath.Join(dir, "does-not-exist.json"))
		assert.ErrorIs(t, err, jsonUtilities.ErrOpenFile)
	})

	t.Run("invalid json", func(t *testing.T) {
		path := filepath.Join(dir, "invalid.json")
		require.NoError(t, os.WriteFile(path, []byte(`{not json}`), 0600))

		_, err := jsonUtilities.DecodeJSONFile(path)
		assert.ErrorIs(t, err, jsonUtilities.ErrJSONDecode)
	})
}
