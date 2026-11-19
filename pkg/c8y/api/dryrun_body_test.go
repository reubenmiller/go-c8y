package api

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	ctxhelpers "github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/contexthelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The HTTP client builds the request body from a pooled buffer and clears it
// for reuse once the request finishes. A dry-run request must stay readable
// after that, and must not show another request's data.
func Test_DryRunTransport_RequestBodySurvivesBufferReuse(t *testing.T) {
	buf := bytes.NewBufferString(`{"name":"Test Device"}`)
	ctx := ctxhelpers.WithDryRun(context.Background(), true)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://example.com/inventory/managedObjects", buf)
	require.NoError(t, err)

	var handlerBody string
	ctx = ctxhelpers.WithDryRunHandler(ctx, func(r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		handlerBody = string(b)
	})
	req = req.WithContext(ctx)

	transport := &DryRunTransport{Transport: http.DefaultTransport}
	_, err = transport.RoundTrip(req)
	require.NoError(t, err)

	// Simulate the client returning the buffer to its pool and reusing it.
	buf.Reset()
	buf.WriteString(`{"name":"Other Request"}`)

	assert.Equal(t, `{"name":"Test Device"}`, handlerBody)

	body, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	assert.Equal(t, `{"name":"Test Device"}`, string(body))

	rc, err := req.GetBody()
	require.NoError(t, err)
	body, err = io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, `{"name":"Test Device"}`, string(body))
}

func Test_DryRunTransport_NoBody(t *testing.T) {
	ctx := ctxhelpers.WithDryRun(context.Background(), true)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://example.com/inventory/managedObjects", nil)
	require.NoError(t, err)

	transport := &DryRunTransport{Transport: http.DefaultTransport}
	_, err = transport.RoundTrip(req)
	require.NoError(t, err)
	assert.Nil(t, req.Body)
}
