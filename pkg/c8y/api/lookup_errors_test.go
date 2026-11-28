package api_test

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	neturl "net/url"
	"strings"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/internal/pkg/testclient"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/remoteaccess/remoteaccess_configurations"
)

// lookupCalls resolve a reference by name with every service which does it. The
// message of each error is the text the SDK has always used (the CLI prints it).
func lookupCalls(c *api.Client) []struct {
	resource string
	message  string
	resolve  func(ctx context.Context) error
} {
	type call = struct {
		resource string
		message  string
		resolve  func(ctx context.Context) error
	}
	return []call{
		{"managed object", "managed object not found with name: nosuch", func(ctx context.Context) error {
			_, err := c.ManagedObjects.ResolveID(ctx, c.ManagedObjects.ByName("nosuch"), nil)
			return err
		}},
		{"application", "application not found with name: nosuch", func(ctx context.Context) error {
			_, err := c.Applications.ResolveID(ctx, c.Applications.ByName("nosuch", ""), nil)
			return err
		}},
		{"application", "application not found with name: nosuch, type: HOSTED", func(ctx context.Context) error {
			_, err := c.Applications.ResolveID(ctx, c.Applications.ByName("nosuch", "HOSTED"), nil)
			return err
		}},
		{"microservice", "microservice not found with name: nosuch", func(ctx context.Context) error {
			_, err := c.Microservices.ResolveID(ctx, "name:nosuch", nil)
			return err
		}},
		{"ui plugin", "ui plugin not found with name: nosuch", func(ctx context.Context) error {
			_, err := c.UIPlugins.ResolveID(ctx, "name:nosuch", nil)
			return err
		}},
		{"agent", "agent not found: nosuch", func(ctx context.Context) error {
			_, err := c.Agents.ResolveID(ctx, "name:nosuch", nil)
			return err
		}},
		{"device group", "device group not found: nosuch", func(ctx context.Context) error {
			_, err := c.DeviceGroups.ResolveID(ctx, "name:nosuch", nil)
			return err
		}},
		{"smart group", "smart group not found: nosuch", func(ctx context.Context) error {
			_, err := c.SmartGroups.ResolveID(ctx, "name:nosuch", nil)
			return err
		}},
		{"device profile", "device profile not found: nosuch", func(ctx context.Context) error {
			_, err := c.DeviceProfiles.ResolveID(ctx, "name:nosuch", nil)
			return err
		}},
		{"configuration", "configuration not found: nosuch", func(ctx context.Context) error {
			_, err := c.Configuration.ResolveID(ctx, "name:nosuch", nil)
			return err
		}},
		{"configuration", "configuration not found: name=nosuch", func(ctx context.Context) error {
			_, err := c.Repository.Configuration.ResolveID(ctx, "name:nosuch", nil)
			return err
		}},
		{"firmware", "firmware not found: name=nosuch", func(ctx context.Context) error {
			_, err := c.Repository.Firmware.ResolveID(ctx, "name:nosuch", nil)
			return err
		}},
		{"software", "software not found with name=nosuch", func(ctx context.Context) error {
			_, err := c.Repository.Software.ResolveID(ctx, "name:nosuch", nil)
			return err
		}},
	}
}

// TestLookupErrorsNameNotFound checks that a name which matches nothing is
// reported with an error a caller can detect without matching its text.
func TestLookupErrorsNameNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"managedObjects":[],"applications":[],"statistics":{"pageSize":5,"currentPage":1}}`))
	}))
	defer srv.Close()
	client := testclient.Must(api.ClientOptions{BaseURL: srv.URL})

	for _, tc := range lookupCalls(client) {
		t.Run(tc.message, func(t *testing.T) {
			err := tc.resolve(context.Background())
			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(err.Error(), tc.message) {
				t.Errorf("message changed: %q does not contain %q", err.Error(), tc.message)
			}
			if !errors.Is(err, core.ErrNameNotFound) {
				t.Errorf("errors.Is(err, core.ErrNameNotFound) is false for %v", err)
			}
			if !core.IsNotFound(err) {
				t.Errorf("core.IsNotFound is false for %v", err)
			}
			var nameErr *core.NameNotFoundError
			if !errors.As(err, &nameErr) || nameErr.Resource != tc.resource || nameErr.Name != "nosuch" {
				t.Errorf("errors.As NameNotFoundError = %+v, want resource %q name nosuch", nameErr, tc.resource)
			}
			if resource, name, ok := core.NameNotFound(err); !ok || resource != tc.resource || name != "nosuch" {
				t.Errorf("core.NameNotFound = (%q, %q, %v)", resource, name, ok)
			}
			var lookupErr *core.LookupError
			if !errors.As(err, &lookupErr) || lookupErr.Resource != tc.resource {
				t.Errorf("errors.As LookupError = %+v, want resource %q", lookupErr, tc.resource)
			}
		})
	}
}

// TestLookupErrorsTransport checks that a failure of the lookup request itself
// (no connection) is marked as an error of the lookup, and that the error of
// the request is still reachable.
func TestLookupErrorsTransport(t *testing.T) {
	// a port nothing listens on
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	_ = l.Close()
	client := testclient.Must(api.ClientOptions{BaseURL: "http://" + addr})

	for _, tc := range lookupCalls(client) {
		t.Run(tc.message, func(t *testing.T) {
			err := tc.resolve(context.Background())
			if err == nil {
				t.Fatal("expected an error")
			}
			var lookupErr *core.LookupError
			if !errors.As(err, &lookupErr) || lookupErr.Resource != tc.resource {
				t.Fatalf("errors.As LookupError = %+v, want resource %q (error: %v)", lookupErr, tc.resource, err)
			}
			var urlErr *neturl.Error
			if !errors.As(err, &urlErr) {
				t.Errorf("the transport error is not reachable: %v", err)
			}
			if errors.Is(err, core.ErrNameNotFound) {
				t.Errorf("a transport error is not a name which was not found: %v", err)
			}
		})
	}
}

// TestLookupErrorsListFailure checks that a failed list request of a lookup is
// returned, not turned into a name which was not found.
func TestLookupErrorsListFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"general/internalError","message":"boom"}`))
	}))
	defer srv.Close()
	client := testclient.Must(api.ClientOptions{BaseURL: srv.URL})

	calls := map[string]func() error{
		"applications": func() error {
			_, err := client.Applications.ResolveID(context.Background(), "name:x", nil)
			return err
		},
		"microservices": func() error {
			_, err := client.Microservices.ResolveID(context.Background(), "name:x", nil)
			return err
		},
		"ui plugins": func() error {
			_, err := client.UIPlugins.ResolveID(context.Background(), "name:x", nil)
			return err
		},
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			err := call()
			if err == nil {
				t.Fatal("expected an error")
			}
			if errors.Is(err, core.ErrNameNotFound) {
				t.Errorf("a failed request is not a name which was not found: %v", err)
			}
			if !core.ErrHasStatus(err, http.StatusInternalServerError) {
				t.Errorf("want the 500 of the request, got %v", err)
			}
			if !core.IsLookupError(err) {
				t.Errorf("want a lookup error, got %v", err)
			}
		})
	}
}

// TestLookupErrorsRemoteAccessConfiguration checks the error of a remote access
// configuration name which the device does not have.
func TestLookupErrorsRemoteAccessConfiguration(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":"1","name":"webssh","protocol":"SSH"}]`))
	}))
	defer srv.Close()
	client := testclient.Must(api.ClientOptions{BaseURL: srv.URL})

	_, err := client.RemoteAccess.Configurations.ResolveID(context.Background(), "d1", "name:missing")
	if err == nil || err.Error() != "remote access configuration not found: name=missing" {
		t.Fatalf("error = %v", err)
	}
	if !errors.Is(err, remoteaccess_configurations.ErrConfigurationNotFound) || !errors.Is(err, core.ErrNameNotFound) {
		t.Errorf("error does not match the sentinels: %v", err)
	}
	var notFound *remoteaccess_configurations.ConfigurationNotFoundError
	if !errors.As(err, &notFound) || notFound.Name != "missing" {
		t.Errorf("errors.As ConfigurationNotFoundError = %+v", notFound)
	}
	if resource, name, ok := core.NameNotFound(err); !ok || resource != "remote access configuration" || name != "missing" {
		t.Errorf("core.NameNotFound = (%q, %q, %v)", resource, name, ok)
	}
	if !core.IsLookupError(err) {
		t.Errorf("want a lookup error, got %v", err)
	}
}

// TestLookupErrorsDirectID checks that a plain id is no lookup.
func TestLookupErrorsDirectID(t *testing.T) {
	client := testclient.Must(api.ClientOptions{BaseURL: "http://127.0.0.1:1"})
	id, err := client.ManagedObjects.ResolveID(context.Background(), "123", nil)
	if err != nil || id != "123" {
		t.Fatalf("ResolveID = (%q, %v)", id, err)
	}
}
