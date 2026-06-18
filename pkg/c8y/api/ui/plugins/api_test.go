package plugins

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"resty.dev/v3"
)

func newTestService() *Service {
	return NewService(&core.Service{
		Client: resty.New().SetBaseURL("https://example.cumulocity.com"),
	})
}

// testServerService spins up an httptest server and returns a Service pointed at it.
func testServerService(handler http.HandlerFunc) (*Service, func()) {
	ts := httptest.NewServer(handler)
	return NewService(&core.Service{Client: resty.New().SetBaseURL(ts.URL)}), ts.Close
}

func TestListB(t *testing.T) {
	s := newTestService()
	req := s.listB(ListOptions{Name: "my-plugin"})

	assert.Equal(t, resty.MethodGet, req.Request.Method)
	assert.Equal(t, types.MimeTypeApplicationJSON, req.Request.Header.Get("Accept"))
	assert.Equal(t, ApiPlugins, req.URL().Path)
}

func TestGetB(t *testing.T) {
	s := newTestService()
	req := s.getB("12345")

	assert.Equal(t, resty.MethodGet, req.Request.Method)
	assert.Equal(t, ApiPlugin, req.URL().Path)
}

func TestCreateB(t *testing.T) {
	s := newTestService()
	plugin := NewPlugin("test-plugin")
	req := s.createB(plugin)

	assert.Equal(t, resty.MethodPost, req.Request.Method)
	assert.Equal(t, ApiPlugins, req.URL().Path)
}

func TestUpdateB(t *testing.T) {
	s := newTestService()
	plugin := NewPlugin("test")
	req := s.updateB("12345", plugin)

	assert.Equal(t, resty.MethodPut, req.Request.Method)
	assert.Equal(t, ApiPlugin, req.URL().Path)
}

func TestDeleteB(t *testing.T) {
	s := newTestService()
	req := s.deleteB("12345")

	assert.Equal(t, resty.MethodDelete, req.Request.Method)
	assert.Equal(t, ApiPlugin, req.URL().Path)
}

func TestNewPlugin(t *testing.T) {
	plugin := NewPlugin("my-plugin")

	assert.Equal(t, "my-plugin", plugin.Name)
	assert.Equal(t, "my-plugin-key", plugin.Key)
	assert.Equal(t, ApplicationTypeHosted, plugin.Type)
	assert.NotNil(t, plugin.Manifest)
}

// TestListPluckAndQuery verifies the collection is plucked into items and that
// the providedFor filter and the forced hasVersions=true flag are emitted.
func TestListPluckAndQuery(t *testing.T) {
	var query string
	svc, closeFn := testServerService(func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"applications":[{"id":"1","name":"a"},{"id":"2","name":"b"}]}`))
	})
	defer closeFn()

	res := svc.List(context.Background(), ListOptions{ProvidedFor: "t123"})
	require.NoError(t, res.Err)

	var ids []string
	for item := range res.Items() {
		ids = append(ids, item.ID())
	}
	assert.Equal(t, []string{"1", "2"}, ids)
	assert.Contains(t, query, "providedFor=t123", "providedFor must use the v1 query param name")
	assert.Contains(t, query, "hasVersions=true", "List forces hasVersions=true")
}

// TestResolveIDByName verifies a name ref is resolved by listing HOSTED plugins
// (with versions) and matching on name or contextPath.
func TestResolveIDByName(t *testing.T) {
	var listQuery string
	svc, closeFn := testServerService(func(w http.ResponseWriter, r *http.Request) {
		listQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"applications":[{"id":"42","name":"my-plugin","contextPath":"my-context"}]}`))
	})
	defer closeFn()

	// match by name
	id, err := svc.ResolveID(context.Background(), "name:my-plugin", nil)
	require.NoError(t, err)
	assert.Equal(t, "42", id)
	assert.Contains(t, listQuery, "type=HOSTED")
	assert.Contains(t, listQuery, "hasVersions=true")

	// match by contextPath
	id, err = svc.ResolveID(context.Background(), "name:my-context", nil)
	require.NoError(t, err)
	assert.Equal(t, "42", id)
}

// TestResolveIDPlainPassthrough verifies a plain id resolves without any server
// round-trip (so resolution is safe under --dry).
func TestResolveIDPlainPassthrough(t *testing.T) {
	called := false
	svc, closeFn := testServerService(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	defer closeFn()

	id, err := svc.ResolveID(context.Background(), "12345", nil)
	require.NoError(t, err)
	assert.Equal(t, "12345", id)
	assert.False(t, called, "a plain id must not trigger a lookup request")
}

// TestUpdateRawBody verifies Update accepts a raw body (map/json.RawMessage) and
// passes it through unchanged on a PUT to the resolved plugin id.
func TestUpdateRawBody(t *testing.T) {
	var method, path string
	var body map[string]any
	svc, closeFn := testServerService(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		path = r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(raw)
	})
	defer closeFn()

	res := svc.Update(context.Background(), "55", json.RawMessage(`{"availability":"SHARED","contextPath":"cp"}`))
	require.NoError(t, res.Err)
	assert.Equal(t, http.MethodPut, method)
	assert.Equal(t, "/application/applications/55", path)
	assert.Equal(t, "SHARED", body["availability"])
	assert.Equal(t, "cp", body["contextPath"])
}
