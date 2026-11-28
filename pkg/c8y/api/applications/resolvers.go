package applications

import (
	"context"
	"fmt"
	"strings"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/source"
)

// nameResolver looks up an application by name with optional type filter
type nameResolver struct {
	Name   string
	Type   string
	Lookup func(ctx context.Context, name, appType string) (string, map[string]any, error)
}

func (n nameResolver) ResolveID(ctx context.Context) (source.ResolveResult, error) {
	if n.Lookup == nil {
		return source.ResolveResult{}, fmt.Errorf("no lookup function configured for application name")
	}
	id, meta, err := n.Lookup(ctx, n.Name, n.Type)
	if err != nil {
		return source.ResolveResult{}, err
	}
	if meta == nil {
		meta = make(map[string]any)
	}
	meta["namePattern"] = n.Name
	meta["source"] = "name"
	if n.Type != "" {
		meta["type"] = n.Type
	}
	return source.ResolveResult{ID: id, Meta: meta}, nil
}

func (n nameResolver) String() string {
	if n.Type != "" {
		return fmt.Sprintf("name:%s:%s", n.Name, n.Type)
	}
	return fmt.Sprintf("name:%s", n.Name)
}

// newResolvers builds the reference schemes specific to applications:
//   - "12345" -> direct ID
//   - "id:12345" -> direct ID
//   - "name:cockpit" -> name lookup
//   - "name:cockpit:HOSTED" -> name lookup with type filter
//   - "custom:..." -> custom resolver (if registered)
func (s *Service) newResolvers() *source.Registry {
	return source.NewRegistry("application", map[string]source.SchemeParser{
		"name": func(value string) (source.Resolver, error) {
			// Format: name:appName or name:appName:type
			appName, appType, _ := strings.Cut(value, ":")
			return nameResolver{
				Name: appName,
				Type: appType,
				// s.lookupByName is read at resolve time, so it can be replaced.
				Lookup: func(ctx context.Context, name, appType string) (string, map[string]any, error) {
					return s.lookupByName(ctx, name, appType)
				},
			}, nil
		},
	})
}
