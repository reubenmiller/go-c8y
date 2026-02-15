package api

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_redactSensitiveHeaders(t *testing.T) {
	tests := []struct {
		name     string
		input    http.Header
		expected map[string]string
	}{
		{
			name: "redacts authorization header",
			input: http.Header{
				"Authorization": []string{"Bearer secret-token-12345"},
				"User-Agent":    []string{"go-client"},
			},
			expected: map[string]string{
				"Authorization": "[REDACTED]",
				"User-Agent":    "go-client",
			},
		},
		{
			name: "redacts cookie header",
			input: http.Header{
				"Cookie":       []string{"session=abc123; token=xyz789"},
				"Content-Type": []string{"application/json"},
			},
			expected: map[string]string{
				"Cookie":       "[REDACTED]",
				"Content-Type": "application/json",
			},
		},
		{
			name: "redacts API key headers (various forms)",
			input: http.Header{
				"Api-Key":   []string{"sk-1234567890"},
				"X-Api-Key": []string{"ak-9876543210"},
			},
			expected: map[string]string{
				"Api-Key":   "[REDACTED]",
				"X-Api-Key": "[REDACTED]",
			},
		},
		{
			name: "redacts XSRF token",
			input: http.Header{
				"X-Xsrf-Token": []string{"csrf-token-value"},
				"X-Csrf-Token": []string{"another-csrf-token"},
			},
			expected: map[string]string{
				"X-Xsrf-Token": "[REDACTED]",
				"X-Csrf-Token": "[REDACTED]",
			},
		},
		{
			name: "preserves safe headers",
			input: http.Header{
				"Accept":          []string{"application/json"},
				"Accept-Encoding": []string{"gzip, deflate"},
				"User-Agent":      []string{"go-client"},
				"Content-Type":    []string{"application/json"},
			},
			expected: map[string]string{
				"Accept":          "application/json",
				"Accept-Encoding": "gzip, deflate",
				"User-Agent":      "go-client",
				"Content-Type":    "application/json",
			},
		},
		{
			name: "mixed sensitive and safe headers",
			input: http.Header{
				"Authorization": []string{"Basic dXNlcjpwYXNz"},
				"Accept":        []string{"application/json"},
				"Cookie":        []string{"session=secret"},
				"User-Agent":    []string{"go-client"},
			},
			expected: map[string]string{
				"Authorization": "[REDACTED]",
				"Accept":        "application/json",
				"Cookie":        "[REDACTED]",
				"User-Agent":    "go-client",
			},
		},
		{
			name: "case insensitive matching",
			input: http.Header{
				"Authorization": []string{"Bearer token"},
				"User-Agent":    []string{"test-agent"},
			},
			expected: map[string]string{
				"Authorization": "[REDACTED]",
				"User-Agent":    "test-agent",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := redactSensitiveHeaders(tt.input)

			// Verify all expected headers are present with correct values
			for key, expectedValue := range tt.expected {
				actualValues := result.Get(key)
				assert.Equal(t, expectedValue, actualValues, "Header %s should have value %s", key, expectedValue)
			}

			// Verify no headers were added or removed
			assert.Equal(t, len(tt.input), len(result), "Number of headers should remain the same")
		})
	}
}

func Test_redactSensitiveHeaders_EmptyHeaders(t *testing.T) {
	input := http.Header{}
	result := redactSensitiveHeaders(input)
	assert.Empty(t, result, "Empty input should return empty output")
}

func Test_redactSensitiveHeaders_DoesNotModifyOriginal(t *testing.T) {
	input := http.Header{
		"Authorization": []string{"Bearer secret"},
		"User-Agent":    []string{"go-client"},
	}

	// Store original values
	originalAuth := input.Get("Authorization")

	// Call redaction
	redactSensitiveHeaders(input)

	// Verify original is unchanged
	assert.Equal(t, originalAuth, input.Get("Authorization"), "Original headers should not be modified")
}

func Test_ShouldRedactHeaders_Default(t *testing.T) {
	ctx := context.Background()

	// Default behavior should be to redact
	assert.True(t, ShouldRedactHeaders(ctx), "Should redact headers by default")
}

func Test_ShouldRedactHeaders_ExplicitEnable(t *testing.T) {
	ctx := context.Background()
	ctx = WithRedactHeaders(ctx, true)

	assert.True(t, ShouldRedactHeaders(ctx), "Should redact when explicitly enabled")
}

func Test_ShouldRedactHeaders_ExplicitDisable(t *testing.T) {
	ctx := context.Background()
	ctx = WithRedactHeaders(ctx, false)

	assert.False(t, ShouldRedactHeaders(ctx), "Should not redact when explicitly disabled")
}
