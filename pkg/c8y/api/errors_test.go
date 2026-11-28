package api_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/internal/pkg/testclient"

	api "github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/authentication"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newErrServer(t *testing.T, status int, contentType, body string) *api.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if contentType != "" {
			w.Header().Set("Content-Type", contentType)
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return testclient.Must(api.ClientOptions{
		BaseURL: srv.URL,
		Auth:    authentication.AuthOptions{Tenant: "t1", Username: "u", Password: "p"},
	})
}

// Test_NonJSONErrorBodies_KeepStatus checks that the HTTP status survives error
// bodies which are not JSON (proxies, empty bodies).
func Test_NonJSONErrorBodies_KeepStatus(t *testing.T) {
	cases := []struct {
		name        string
		status      int
		contentType string
		body        string
		sentinel    error
		is          func(error) bool
	}{
		{"401 html", 401, "text/html", "<html><body>denied</body></html>", api.ErrUnauthorized, api.IsUnauthorized},
		{"401 empty", 401, "", "", api.ErrUnauthorized, api.IsUnauthorized},
		{"401 text", 401, "text/plain", "denied", api.ErrUnauthorized, api.IsUnauthorized},
		{"403 html", 403, "text/html; charset=utf-8", "<h1>Forbidden</h1>", api.ErrForbidden, api.IsForbidden},
		{"404 html", 404, "text/html", "<h1>Not found</h1>", api.ErrNotFound, api.IsNotFound},
		{"404 empty", 404, "", "", api.ErrNotFound, api.IsNotFound},
		{"409 html", 409, "text/html", "<p>conflict</p>", api.ErrConflict, api.IsConflict},
		{"409 empty", 409, "", "", api.ErrConflict, api.IsConflict},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := newErrServer(t, tc.status, tc.contentType, tc.body)
			result := client.Alarms.Get(context.Background(), "1")
			require.Error(t, result.Err)
			assert.True(t, errors.Is(result.Err, tc.sentinel), "errors.Is %v", result.Err)
			if tc.is != nil {
				assert.True(t, tc.is(result.Err))
			}
			code, ok := api.StatusCode(result.Err)
			assert.True(t, ok)
			assert.Equal(t, tc.status, code)
			assert.NotContains(t, result.Err.Error(), "<h1>")
		})
	}
}

func Test_NotFoundUnification(t *testing.T) {
	// real 404 from the server
	client := newErrServer(t, 404, "application/json", `{"error":"inventory/notFound","message":"nope"}`)
	err := client.Alarms.Get(context.Background(), "1").Err
	assert.True(t, errors.Is(err, api.ErrNotFound))
	assert.True(t, errors.Is(err, op.ErrNotFound))
	assert.True(t, api.IsNotFound(err))

	// op "no result"
	_, err = op.Result[[]int]{}.First()
	assert.True(t, errors.Is(err, op.ErrNotFound))
	assert.True(t, errors.Is(err, api.ErrNotFound))
	assert.True(t, api.IsNotFound(err))

	// core constructor, wrapped
	err = fmt.Errorf("lookup: %w", error(core.ErrNotFound("missing %s", "x")))
	assert.True(t, errors.Is(err, api.ErrNotFound))
	assert.True(t, errors.Is(err, op.ErrNotFound))

	// not mixed up with other statuses
	assert.False(t, errors.Is(op.ErrNotFound, api.ErrConflict))
	assert.False(t, errors.Is(api.ErrConflict, api.ErrNotFound))
	assert.False(t, errors.Is(errors.New("x"), api.ErrNotFound))
}

func Test_ErrorHelpers(t *testing.T) {
	assert.True(t, api.IsUnauthorized(fmt.Errorf("w: %w", error(&api.Error{Code: 401}))))
	assert.True(t, api.IsForbidden(&api.Error{Code: 403}))
	assert.True(t, api.IsConflict(api.ErrConflict))
	assert.False(t, api.IsConflict(nil))
	assert.False(t, api.IsNotFound(errors.New("x")))

	code, ok := api.StatusCode(fmt.Errorf("w: %w", error(&api.Error{Code: 418})))
	assert.True(t, ok)
	assert.Equal(t, 418, code)
	code, ok = api.StatusCode(api.ErrBadRequest)
	assert.True(t, ok)
	assert.Equal(t, 400, code)
	_, ok = api.StatusCode(errors.New("x"))
	assert.False(t, ok)
	_, ok = api.StatusCode(core.NewError(errors.New("transport")))
	assert.False(t, ok)
	_, ok = api.StatusCode(nil)
	assert.False(t, ok)
}

// Example_errorHandling shows how to classify errors with errors.Is and the helpers.
func Example_errorHandling() {
	err := fmt.Errorf("get device: %w", error(&api.Error{Code: 404, Message: "Finding device data from database failed"}))

	fmt.Println(errors.Is(err, api.ErrNotFound))
	fmt.Println(api.IsNotFound(err), api.IsUnauthorized(err), api.IsConflict(err))
	if code, ok := api.StatusCode(err); ok {
		fmt.Println("status:", code)
	}
	// Output:
	// true
	// true false false
	// status: 404
}
