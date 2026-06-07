package encoding_test

import (
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/encoding"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeDecodeUTF16RoundTrip(t *testing.T) {
	// Note: DecodeUTF16 cannot decode an empty buffer (it requires >= 2 bytes
	// for the BOM check), so the empty string is intentionally excluded.
	inputs := []string{"Hello", "a", "héllo wörld", "12345"}
	for _, in := range inputs {
		t.Run(in, func(t *testing.T) {
			encoded := encoding.EncodeUTF16(in, false)
			// big-endian encoding produces 2 bytes per UTF-16 code unit
			assert.Equal(t, 0, len(encoded)%2)

			decoded, err := encoding.DecodeUTF16(encoded)
			require.NoError(t, err)
			assert.Equal(t, in, decoded)
		})
	}
}

func TestEncodeUTF16WithBOM(t *testing.T) {
	encoded := encoding.EncodeUTF16("Hi", true)
	require.GreaterOrEqual(t, len(encoded), 2)
	// Big-endian BOM is prepended
	assert.Equal(t, byte(0xFE), encoded[0])
	assert.Equal(t, byte(0xFF), encoded[1])
	assert.Equal(t, int8(1), encoding.UTF16Bom(encoded))
}

func TestDecodeUTF16Errors(t *testing.T) {
	t.Run("odd length", func(t *testing.T) {
		_, err := encoding.DecodeUTF16([]byte{0x00})
		assert.Error(t, err)
	})

	t.Run("too small for BOM", func(t *testing.T) {
		// even-length but empty: passes the even check, then BOM check fails
		_, err := encoding.DecodeUTF16([]byte{})
		assert.Error(t, err)
	})
}

func TestUTF16Bom(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  int8
	}{
		{"too small", []byte{0x00}, -1},
		{"big endian", []byte{0xFE, 0xFF}, 1},
		{"little endian", []byte{0xFF, 0xFE}, 2},
		{"no bom", []byte{0x00, 0x41}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, encoding.UTF16Bom(tt.input))
		})
	}
}

func TestIsUTF16(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"big endian bom", string([]byte{0xFE, 0xFF, 0x00, 0x41}), true},
		{"little endian bom", string([]byte{0xFF, 0xFE, 0x41, 0x00}), true},
		{"no bom", "hello", false},
		{"too short", string([]byte{0xFE}), false},
		{"bom but odd length", string([]byte{0xFE, 0xFF, 0x00}), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, encoding.IsUTF16(tt.input))
		})
	}
}
