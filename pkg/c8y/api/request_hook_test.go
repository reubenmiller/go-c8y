package api

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/authentication"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/binaries"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type progressLog struct {
	mu    sync.Mutex
	reads []int64
	total int64
}

func (p *progressLog) fn(read, total int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.reads = append(p.reads, read)
	p.total = total
}

func (p *progressLog) assertComplete(t *testing.T, size int64) {
	t.Helper()
	p.mu.Lock()
	defer p.mu.Unlock()
	require.NotEmpty(t, p.reads)
	for i := 1; i < len(p.reads); i++ {
		assert.Greater(t, p.reads[i], p.reads[i-1], "progress must increase")
	}
	assert.Equal(t, size, p.reads[len(p.reads)-1], "last count is the total")
	assert.Equal(t, size, p.total)
}

func newProgressClient(t *testing.T, size int) (*Client, *httptest.Server) {
	t.Helper()
	payload := bytes.Repeat([]byte("0123456789abcdef"), size/16)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/inventory/binaries":
			_, _ = io.Copy(io.Discard, r.Body)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"1","name":"f.bin"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/inventory/binaries/1":
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Content-Length", "65536")
			_, _ = w.Write(payload)
		case r.URL.Path == "/missing":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"general/internalError","message":"nope"}`))
		case r.URL.Path == "/echo":
			b, _ := io.ReadAll(r.Body)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"got":"` + string(b) + `","hdr":"` + r.Header.Get("X-Hook") + `"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	client := mustNewClient(ClientOptions{
		BaseURL:          srv.URL,
		SkipTenantLookup: true,
		Auth: authentication.AuthOptions{
			Tenant: "t1", Username: "u", Password: "p",
			AuthType: []authentication.AuthType{authentication.AuthTypeBasic},
		},
	})
	return client, srv
}

func Test_WithRequestHook_WrapsBody(t *testing.T) {
	client, _ := newProgressClient(t, 65536)
	var seen string
	ctx := WithRequestHook(context.Background(), func(req *http.Request) error {
		req.Header.Set("X-Hook", "yes")
		b, _ := io.ReadAll(req.Body)
		seen = string(b)
		req.Body = io.NopCloser(strings.NewReader(strings.ToUpper(seen)))
		req.ContentLength = int64(len(seen))
		return nil
	})
	res := client.Request(ctx, RequestOptions{Method: http.MethodPost, Path: "/echo", Body: "hello", ContentType: "text/plain"})
	require.NoError(t, res.Err)
	assert.Equal(t, "hello", seen)
	assert.Equal(t, "HELLO", res.Data.Get("got").String())
	assert.Equal(t, "yes", res.Data.Get("hdr").String())
}

func Test_WithRequestHook_ErrorAborts(t *testing.T) {
	client, _ := newProgressClient(t, 65536)
	boom := errors.New("boom")
	ctx := WithRequestHook(context.Background(), func(*http.Request) error { return boom })
	res := client.Request(ctx, RequestOptions{Method: http.MethodGet, Path: "/echo"})
	require.Error(t, res.Err)
	assert.ErrorIs(t, res.Err, boom)
}

func Test_WithRequestHook_DryRun(t *testing.T) {
	client, srv := newProgressClient(t, 65536)
	calls := 0
	ctx := WithDryRun(WithRequestHook(context.Background(), func(req *http.Request) error {
		calls++
		req.Header.Set("X-Hook", "dry")
		return nil
	}), true)
	var dryReq *http.Request
	ctx = WithDryRunHandler(ctx, func(r *http.Request) { dryReq = r })
	res := client.Request(ctx, RequestOptions{Method: http.MethodPost, Path: "/echo", Body: "x"})
	require.NoError(t, res.Err)
	assert.Equal(t, 1, calls)
	require.NotNil(t, dryReq)
	assert.Equal(t, "dry", dryReq.Header.Get("X-Hook"))
	assert.Contains(t, dryReq.URL.String(), srv.URL)
}

func Test_UploadProgress_UploadFileOptions(t *testing.T) {
	const size = 65536
	client, _ := newProgressClient(t, size)
	path := filepath.Join(t.TempDir(), "f.bin")
	require.NoError(t, os.WriteFile(path, bytes.Repeat([]byte("z"), size), 0o600))

	t.Run("file path", func(t *testing.T) {
		var p progressLog
		res := client.Binaries.Create(context.Background(), binaries.UploadFileOptions{FilePath: path, Progress: p.fn})
		require.NoError(t, res.Err)
		p.assertComplete(t, size)
	})

	t.Run("reader", func(t *testing.T) {
		var p progressLog
		res := client.Binaries.Create(context.Background(), binaries.UploadFileOptions{
			Reader: bytes.NewReader(bytes.Repeat([]byte("z"), size)), Name: "f.bin", Progress: p.fn,
		})
		require.NoError(t, res.Err)
		p.assertComplete(t, size)
	})

	t.Run("update (raw body)", func(t *testing.T) {
		var p progressLog
		res := client.Binaries.Update(context.Background(), "1", binaries.UploadFileOptions{
			Reader: bytes.NewReader(bytes.Repeat([]byte("z"), size)), Progress: p.fn,
		})
		// the test server has no PUT route: only the body accounting matters
		_ = res
		p.assertComplete(t, size)
	})
}

func Test_UploadProgress_Context(t *testing.T) {
	client, _ := newProgressClient(t, 65536)
	var p progressLog
	ctx := WithUploadProgress(context.Background(), p.fn)
	res := client.Request(ctx, RequestOptions{Method: http.MethodPost, Path: "/echo", Body: "twelve bytes"})
	require.NoError(t, res.Err)
	p.assertComplete(t, 12)
}

func Test_DownloadProgress(t *testing.T) {
	const size = 65536
	client, _ := newProgressClient(t, size)
	var p progressLog
	ctx := WithDownloadProgress(context.Background(), p.fn)
	res := client.Binaries.Get(ctx, "1")
	require.NoError(t, res.Err)
	defer res.Data.Close()
	data, err := io.ReadAll(res.Data.Reader())
	require.NoError(t, err)
	assert.Len(t, data, size)
	p.assertComplete(t, size)
}

func Test_ResponseBodyWrapper_NotCalledOnDryRun(t *testing.T) {
	client, _ := newProgressClient(t, 65536)
	called := false
	ctx := WithResponseBodyWrapper(WithDryRun(context.Background(), true), func(*http.Response) io.ReadCloser {
		called = true
		return nil
	})
	res := client.Request(ctx, RequestOptions{Method: http.MethodGet, Path: "/echo"})
	require.NoError(t, res.Err)
	assert.False(t, called)
}

func Test_Request_TypedResult(t *testing.T) {
	client, _ := newProgressClient(t, 65536)

	res := client.Request(context.Background(), RequestOptions{Method: http.MethodPost, Path: "/echo", Body: "abc"})
	require.NoError(t, res.Err)
	assert.Equal(t, http.StatusCreated, res.HTTPStatus)
	assert.Equal(t, "abc", res.Data.Get("got").String())
	assert.NotNil(t, res.Request)
	assert.Contains(t, string(res.Response), `"got"`)
}

func Test_Request_ErrorResult(t *testing.T) {
	client, _ := newProgressClient(t, 65536)
	res := client.Request(context.Background(), RequestOptions{Method: http.MethodGet, Path: "/missing"})
	require.Error(t, res.Err)
	assert.True(t, res.IsError())
	assert.True(t, res.IsNotFound())
	var apiErr *core.Error
	require.ErrorAs(t, res.Err, &apiErr)
	assert.Equal(t, http.StatusNotFound, apiErr.Code)
}

func Test_Request_DryRunSendsNothing(t *testing.T) {
	client, srv := newProgressClient(t, 65536)
	srv.Close()
	res := client.Request(WithDryRun(context.Background(), true), RequestOptions{Method: http.MethodPost, Path: "/echo", Body: "x"})
	require.NoError(t, res.Err)
	assert.NotNil(t, res.Request)
}
