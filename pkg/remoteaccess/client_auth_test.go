package remoteaccess

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/authentication"
)

// The websocket handshake carries the same credentials as a request of the
// client: basic auth with the tenant, or the bearer token.
func TestRemoteAccessConnectionAuthorization(t *testing.T) {
	tests := []struct {
		name string
		auth authentication.AuthOptions
		want string
	}{
		{
			name: "basic auth with the tenant",
			auth: authentication.AuthOptions{Tenant: "t123", Username: "user", Password: "secret"},
			want: api.NewBasicAuthString("t123", "user", "secret"),
		},
		{
			name: "token",
			auth: authentication.AuthOptions{Token: "abc"},
			want: "Bearer abc",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got, gotPath string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got, gotPath = r.Header.Get("Authorization"), r.URL.Path
				w.WriteHeader(http.StatusUnauthorized)
			}))
			defer server.Close()

			client := api.NewClient(api.ClientOptions{BaseURL: server.URL, Auth: tc.auth, SkipTenantLookup: true})
			ra := NewRemoteAccessClient(client, RemoteAccessOptions{ManagedObjectID: "12345", RemoteAccessID: "1"})
			if _, _, err := ra.createRemoteAccessConnection(context.Background()); err == nil {
				t.Fatal("expected the handshake to be rejected")
			}
			if gotPath != "/service/remoteaccess/client/12345/configurations/1" {
				t.Errorf("path = %q", gotPath)
			}
			if got != tc.want {
				t.Errorf("Authorization = %q, want %q", got, tc.want)
			}
		})
	}
}
