package source_test

import (
	"context"
	"errors"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/source"
)

type fakeResolver struct{ id string }

func (f fakeResolver) ResolveID(context.Context) (source.ResolveResult, error) {
	return source.ResolveResult{ID: f.id, Meta: map[string]any{"source": "fake"}}, nil
}
func (f fakeResolver) String() string { return "fake:" + f.id }

func newTestRegistry() *source.Registry {
	return source.NewRegistry("widget", map[string]source.SchemeParser{
		"name": func(value string) (source.Resolver, error) { return fakeResolver{id: "id-of-" + value}, nil },
		"bad":  func(string) (source.Resolver, error) { return nil, errors.New("bad reference") },
	})
}

func TestRegistryResolve(t *testing.T) {
	r := newTestRegistry()
	ctx := context.Background()

	tests := []struct {
		ref        string
		want       string
		wantSource string
	}{
		{"123", "123", "direct-id"},
		{"id:123", "123", "direct-id"},
		{"name:abc", "id-of-abc", "fake"},
		{"name:a:b", "id-of-a:b", "fake"},
	}
	for _, tt := range tests {
		meta := map[string]any{}
		got, err := r.Resolve(ctx, tt.ref, meta)
		if err != nil || got != tt.want {
			t.Errorf("Resolve(%q) = %q, %v; want %q", tt.ref, got, err, tt.want)
		}
		if meta["source"] != tt.wantSource {
			t.Errorf("Resolve(%q) source = %v, want %q", tt.ref, meta["source"], tt.wantSource)
		}
	}

	// A nil meta is allowed.
	if got, err := r.Resolve(ctx, "name:x", nil); err != nil || got != "id-of-x" {
		t.Errorf("Resolve with nil meta = %q, %v", got, err)
	}
}

func TestRegistryErrors(t *testing.T) {
	r := newTestRegistry()
	ctx := context.Background()

	tests := []struct {
		ref  string
		want string
	}{
		{"", "empty widget reference"},
		{"nope:x", `unknown widget resolver scheme: "nope"`},
		{"bad:x", "bad reference"},
	}
	for _, tt := range tests {
		_, err := r.Resolve(ctx, tt.ref, nil)
		if err == nil || err.Error() != tt.want {
			t.Errorf("Resolve(%q) error = %v, want %q", tt.ref, err, tt.want)
		}
		var lookupErr *core.LookupError
		if !errors.As(err, &lookupErr) || lookupErr.Resource != "widget" || lookupErr.Reference != tt.ref {
			t.Errorf("Resolve(%q) error %v is not a LookupError of the widget", tt.ref, err)
		}
	}
}

func TestRegistryCustomResolverWins(t *testing.T) {
	r := newTestRegistry()
	r.Register("name", fakeResolver{id: "custom"})
	r.Register("extra", fakeResolver{id: "extra-id"})

	for ref, want := range map[string]string{"name:abc": "custom", "extra:1": "extra-id"} {
		if got, err := r.Resolve(context.Background(), ref, nil); err != nil || got != want {
			t.Errorf("Resolve(%q) = %q, %v; want %q", ref, got, err, want)
		}
	}
}
