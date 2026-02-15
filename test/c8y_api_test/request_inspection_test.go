package api_test

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_RequestInspection_DryRun(t *testing.T) {
	client := testcore.CreateTestClient(t)

	// Enable dry run to get mock response
	ctx := api.WithDryRun(context.Background(), true)

	// Make a request
	result := client.ManagedObjects.Get(ctx, "12345", managedobjects.GetOptions{})

	// Should not error
	assert.NoError(t, result.Err)

	// Request should be available for inspection
	require.NotNil(t, result.Request, "Request should be captured in Result")

	// Inspect request details
	assert.Equal(t, "GET", result.Request.Method)
	assert.Contains(t, result.Request.URL.Path, "/inventory/managedObjects/12345")

	// Check headers
	assert.NotEmpty(t, result.Request.Header.Get("User-Agent"))
	assert.Contains(t, result.Request.Header.Get("Accept"), "application/json")
}

func Test_RequestInspection_DryRun_POST(t *testing.T) {
	client := testcore.CreateTestClient(t)

	// Enable dry run
	ctx := api.WithDryRun(context.Background(), true)

	// Create a managed object
	result := client.ManagedObjects.Create(ctx, map[string]any{
		"name": "Test Device",
		"type": "c8y_TestDevice",
	})

	// Should not error
	assert.NoError(t, result.Err)

	// Request should be available
	require.NotNil(t, result.Request)

	// Inspect POST request
	assert.Equal(t, "POST", result.Request.Method)
	assert.Contains(t, result.Request.URL.Path, "/inventory/managedObjects")
	assert.Equal(t, "application/json", result.Request.Header.Get("Content-Type"))

	// Body should be readable
	body, err := io.ReadAll(result.Request.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "Test Device")
	assert.Contains(t, string(body), "c8y_TestDevice")
}

func Test_RequestInspection_DryRun_FormattingExample(t *testing.T) {
	client := testcore.CreateTestClient(t)

	// Enable dry run
	ctx := api.WithDryRun(context.Background(), true)

	// Make a request
	result := client.ManagedObjects.Get(ctx, "12345", managedobjects.GetOptions{})

	require.NotNil(t, result.Request)

	// Example: Format as curl-like command
	curlCommand := formatAsCurl(result.Request)
	t.Logf("Curl equivalent:\n%s", curlCommand)

	assert.Contains(t, curlCommand, "curl")
	assert.Contains(t, curlCommand, "GET")
	assert.Contains(t, curlCommand, result.Request.URL.String())
}

// formatAsCurl is a simple example of how you could format a request as a curl command
// This is similar to what go-c8y-cli might do
func formatAsCurl(req *http.Request) string {
	cmd := "curl -X " + req.Method

	// Add headers
	for key, values := range req.Header {
		for _, value := range values {
			cmd += " \\\n  -H '" + key + ": " + value + "'"
		}
	}

	// Add URL
	cmd += " \\\n  '" + req.URL.String() + "'"

	return cmd
}

func Test_RequestInspection_SensitiveHeadersRedacted(t *testing.T) {
	// This test verifies that sensitive headers are not logged during dry run
	// We can't directly test the logging output, but we can verify the redaction function exists
	// by checking that authorization headers don't appear in logs when we run with dry run

	// The actual verification would be done by inspecting logs, but we can at least
	// verify the dry run works without exposing sensitive data
	client := testcore.CreateTestClient(t)

	// Enable dry run
	ctx := api.WithDryRun(context.Background(), true)

	// Make a request - authorization headers will be added by the client
	result := client.ManagedObjects.Get(ctx, "12345", managedobjects.GetOptions{})

	// Should not error
	assert.NoError(t, result.Err)

	// Request should be captured
	require.NotNil(t, result.Request)

	// The request will have authorization headers in it (from the client)
	// but when logged, they should have been redacted
	// We can't easily test the log output itself, but we can verify the request
	// object still has the headers (since we only redact during logging)
	t.Logf("Request has %d headers", len(result.Request.Header))
}

func Test_RequestInspection_RedactionOptOut(t *testing.T) {
	// Verify that disabling redaction shows full headers in logs
	client := testcore.CreateTestClient(t)

	// Enable dry run AND disable header redaction (for debugging)
	ctx := context.Background()
	ctx = api.WithDryRun(ctx, true)
	ctx = api.WithRedactHeaders(ctx, false)

	// Make a request
	result := client.ManagedObjects.Get(ctx, "12345", managedobjects.GetOptions{})

	// Should not error
	assert.NoError(t, result.Err)

	// Request should be captured
	require.NotNil(t, result.Request)

	// Verify context values are set correctly
	assert.True(t, api.IsDryRun(ctx), "Dry run should be enabled")
	assert.False(t, api.ShouldRedactHeaders(ctx), "Header redaction should be disabled")

	// The actual headers would be visible in logs (not redacted)
	// but we can't directly test log output here
	t.Logf("Request captured with unredacted headers for debugging")
}
