package api_test

import (
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Client_SetBaseURL(t *testing.T) {
	client := testcore.CreateTestClient(t)
	require.NoError(t, client.SetBaseURL("https://new.example.com"))
	assert.Contains(t, client.BaseURL.String(), "new.example.com")
}

func Test_Client_SetDebugWithAuth(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.SetDebugWithAuth(true)
	client.SetDebugWithAuth(false)
}

func Test_Client_FormatBaseURL(t *testing.T) {
	cases := map[string]string{
		"example.com":          "https://example.com/",
		"http://example.com":   "http://example.com/",
		"https://example.com":  "https://example.com/",
		"https://example.com/": "https://example.com/",
	}
	for in, expected := range cases {
		assert.Equal(t, expected, api.FormatBaseURL(in))
	}
}
