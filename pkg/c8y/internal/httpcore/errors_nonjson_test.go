package httpcore

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"resty.dev/v3"
)

// Test_coupleAPIErrors_UnparsedNonJSONBody covers the streaming path (body not
// read by resty): the status code must survive a body which is not JSON.
func Test_coupleAPIErrors_UnparsedNonJSONBody(t *testing.T) {
	for _, tc := range []struct {
		name, contentType, body string
		status                  int
		wantMsg                 string
	}{
		{"401 html", "text/html", "<html>nope</html>", 401, http.StatusText(http.StatusUnauthorized)},
		{"401 empty", "", "", 401, http.StatusText(http.StatusUnauthorized)},
		{"404 text", "text/plain", "gone fishing", 404, "gone fishing"},
		{"409 empty", "", "", 409, "Conflict"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tc.contentType != "" {
					w.Header().Set("Content-Type", tc.contentType)
				}
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()
			rc := resty.New()
			defer rc.Close()
			resp, err := rc.R().SetResponseDoNotParse(true).Get(srv.URL)
			_, err = coupleAPIErrors(resp, err)
			require.Error(t, err)

			var e *core.Error
			require.True(t, errors.As(err, &e))
			assert.Equal(t, tc.status, e.Code)
			assert.Equal(t, tc.wantMsg, e.Message)
			assert.Equal(t, tc.body, e.MessageRaw)
			assert.True(t, errors.Is(err, core.Error{Code: tc.status}))
		})
	}
}
