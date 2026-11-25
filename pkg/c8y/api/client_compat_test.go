package api

import (
	"context"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type formPart struct {
	Name        string
	FileName    string
	ContentType string
	Value       string
}

// readFormParts parses a multipart request body into its parts, in order.
func readFormParts(t *testing.T, r *http.Request, body string) []formPart {
	t.Helper()
	_, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	require.NoError(t, err)
	reader := multipart.NewReader(strings.NewReader(body), params["boundary"])
	var parts []formPart
	for {
		p, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		value, err := io.ReadAll(p)
		require.NoError(t, err)
		parts = append(parts, formPart{
			Name:        p.FormName(),
			FileName:    p.FileName(),
			ContentType: p.Header.Get("Content-Type"),
			Value:       string(value),
		})
	}
	return parts
}

func Test_SendRequest_FormParts(t *testing.T) {
	ms := newMWServer(t)
	client := newMWClient(ms)

	file := filepath.Join(t.TempDir(), "notes.txt")
	require.NoError(t, os.WriteFile(file, []byte("from disk"), 0o600))
	f, err := os.Open(file)
	require.NoError(t, err)

	result := client.SendRequest(context.Background(), RequestOptions{
		Method: http.MethodPost,
		Path:   "/example",
		FormData: map[string]io.Reader{
			"upload":  &FormFile{Reader: strings.NewReader("custom"), FileName: "custom.bin", ContentType: "application/x-custom"},
			"disk":    f,
			"comment": &FormField{Reader: strings.NewReader("hello")},
			"meta":    FormField{Reader: strings.NewReader(`{"a":1}`), ContentType: "application/json"},
			"object":  strings.NewReader(`{"name":"x"}`),
			"file":    strings.NewReader("raw"),
		},
	})
	require.NoError(t, result.Error)

	parts := readFormParts(t, ms.last(t), ms.bodies[len(ms.bodies)-1])
	assert.Equal(t, []formPart{
		{Name: "comment", Value: "hello"},
		{Name: "disk", FileName: "notes.txt", ContentType: "text/plain; charset=utf-8", Value: "from disk"},
		{Name: "file", FileName: "file", ContentType: "application/octet-stream", Value: "raw"},
		{Name: "meta", ContentType: "application/json", Value: `{"a":1}`},
		{Name: "object", ContentType: "application/json", Value: `{"name":"x"}`},
		{Name: "upload", FileName: "custom.bin", ContentType: "application/x-custom", Value: "custom"},
	}, parts)
}

func Test_SendRequest_HostKeepsQueryOnce(t *testing.T) {
	ms := newMWServer(t)
	client := newMWClient(ms)

	result := client.SendRequest(context.Background(), RequestOptions{
		Method: http.MethodGet,
		Host:   ms.server.URL + "/service/example",
		Path:   "/items?id=1",
	})
	require.NoError(t, result.Error)
	req := ms.last(t)
	assert.Equal(t, "/service/example/items", req.URL.Path)
	assert.Equal(t, "id=1", req.URL.RawQuery)
}

func Test_SendRequest_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/html" {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte("<html>denied</html>"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"inventory/notFound","message":"Finding device data from database failed"}`))
	}))
	t.Cleanup(server.Close)
	client := newMWClient(&mwServer{server: server})
	client.Resty().SetRetryCount(0)

	result := client.SendRequest(context.Background(), RequestOptions{Method: http.MethodGet, Path: "/inventory/managedObjects/1"})
	var apiErr *core.Error
	require.ErrorAs(t, result.Error, &apiErr)
	assert.Equal(t, http.StatusNotFound, apiErr.StatusCode())
	assert.Equal(t, "inventory/notFound", apiErr.Type)
	assert.Equal(t, "Finding device data from database failed", apiErr.Message)
	assert.NotEmpty(t, result.Body(), "the body stays readable")

	result = client.SendRequest(context.Background(), RequestOptions{Method: http.MethodGet, Path: "/html"})
	require.ErrorAs(t, result.Error, &apiErr)
	assert.Equal(t, http.StatusForbidden, apiErr.StatusCode())
	assert.Equal(t, "<html>denied</html>", apiErr.Message)
}

func Test_SendRequest_ContextOptionsAndRepeatedQueryKeys(t *testing.T) {
	ms := newMWServer(t)
	client := newMWClient(ms)

	ctx := WithExtraHeaders(context.Background(), map[string]string{"X-Custom": "1"})
	ctx = WithExtraQueryParams(ctx, url.Values{"extra": {"a", "b"}})
	ctx = WithProcessingMode(ctx, "TRANSIENT")

	result := client.SendRequest(ctx, RequestOptions{
		Method: http.MethodGet,
		Path:   "/items?type=a&type=b",
		Query:  "other=1",
	})
	require.NoError(t, result.Error)
	req := ms.last(t)
	assert.Equal(t, url.Values{"type": {"a", "b"}, "other": {"1"}, "extra": {"a", "b"}}, req.URL.Query())
	assert.Equal(t, "1", req.Header.Get("X-Custom"))
	assert.Equal(t, "TRANSIENT", req.Header.Get("X-Cumulocity-Processing-Mode"))
	assert.Equal(t, "application/json", req.Header.Get("Accept"))
}
