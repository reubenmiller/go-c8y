package api

import (
	"context"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/authentication"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/events"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// headerServer records the path, query and headers of every request.
type headerServer struct {
	mu       sync.Mutex
	requests []*http.Request
	server   *httptest.Server
}

func newHeaderServer(t *testing.T) *headerServer {
	t.Helper()
	hs := &headerServer{}
	hs.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hs.mu.Lock()
		hs.requests = append(hs.requests, r.Clone(context.Background()))
		hs.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/event/events" && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"1","type":"test"}`))
		default:
			_, _ = w.Write([]byte(`{"users":[],"statistics":{"currentPage":1,"pageSize":5}}`))
		}
	}))
	t.Cleanup(hs.server.Close)
	return hs
}

func (hs *headerServer) count() int {
	hs.mu.Lock()
	defer hs.mu.Unlock()
	return len(hs.requests)
}

func (hs *headerServer) last(t *testing.T) *http.Request {
	t.Helper()
	hs.mu.Lock()
	defer hs.mu.Unlock()
	require.NotEmpty(t, hs.requests)
	return hs.requests[len(hs.requests)-1]
}

func newOptionsTestClient(hs *headerServer) *Client {
	return mustNewClient(ClientOptions{
		BaseURL: hs.server.URL,
		Auth: authentication.AuthOptions{
			Tenant:   "t100",
			Username: "user",
			Password: "pass",
			AuthType: []authentication.AuthType{authentication.AuthTypeBasic},
		},
	})
}

func Test_ContextOptions_ProcessingModeFromAPI(t *testing.T) {
	hs := newHeaderServer(t)
	client := newOptionsTestClient(hs)

	ctx := WithProcessingModeTransient(context.Background())
	result := client.Events.Create(ctx, events.CreateOptions{Type: "test", Text: "test", Source: "12345"})
	require.NoError(t, result.Err)
	assert.Equal(t, "TRANSIENT", hs.last(t).Header.Get("X-Cumulocity-Processing-Mode"))
	assert.Equal(t, "TRANSIENT", string(ProcessingMode(ctx)))
}

func Test_ContextOptions_ExtraHeadersAndQueryParams(t *testing.T) {
	hs := newHeaderServer(t)
	client := newOptionsTestClient(hs)

	ctx := WithExtraHeaders(context.Background(), map[string]string{"X-Test": "1"})
	ctx = WithExtraQueryParams(ctx, url.Values{"withExtra": []string{"true"}})
	result := client.Users.List(ctx, users.ListOptions{})
	require.NoError(t, result.Err)

	req := hs.last(t)
	assert.Equal(t, "1", req.Header.Get("X-Test"))
	assert.Equal(t, "true", req.URL.Query().Get("withExtra"))
	assert.Equal(t, map[string]string{"X-Test": "1"}, ExtraHeaders(ctx))
	assert.Equal(t, "true", ExtraQueryParams(ctx).Get("withExtra"))
}

func Test_ContextOptions_DryRunHandler(t *testing.T) {
	hs := newHeaderServer(t)
	client := newOptionsTestClient(hs)

	var handled *http.Request
	ctx := WithDryRunHandler(WithDryRun(context.Background(), true), func(r *http.Request) {
		handled = r
	})
	_ = client.Users.List(ctx, users.ListOptions{})

	require.NotNil(t, handled, "dry-run handler should receive the request")
	assert.Equal(t, "/user/t100/users", handled.URL.Path)
	assert.Equal(t, 0, hs.count(), "nothing should be sent to the server")
}

func Test_ContextOptions_StringTenantKeyIsIgnored(t *testing.T) {
	hs := newHeaderServer(t)
	client := newOptionsTestClient(hs)

	//nolint:staticcheck // deliberately uses an untyped key to prove it is ignored
	ctx := context.WithValue(context.Background(), "tenant", "t999") //lint:ignore SA1029 test of ignored key
	result := client.Users.List(ctx, users.ListOptions{})
	require.NoError(t, result.Err)
	assert.Equal(t, "/user/t100/users", hs.last(t).URL.Path)
}

func Test_ContextOptions_WithTenantOverride(t *testing.T) {
	hs := newHeaderServer(t)
	client := newOptionsTestClient(hs)

	ctx := WithTenant(context.Background(), "t999")
	result := client.Users.List(ctx, users.ListOptions{})
	require.NoError(t, result.Err)
	assert.Equal(t, "/user/t999/users", hs.last(t).URL.Path)

	tenant, ok := TenantFromContext(ctx)
	assert.True(t, ok)
	assert.Equal(t, "t999", tenant)
}

// No public package may be named "context": it shadows the standard library.
func Test_NoPublicPackageNamedContext(t *testing.T) {
	root := filepath.Join("..", "..") // pkg/c8y
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == "context" {
			if _, statErr := os.Stat(filepath.Join(path)); statErr == nil {
				t.Errorf("public package directory named context found: %s", path)
			}
		}
		return nil
	})
	require.NoError(t, err)
}
