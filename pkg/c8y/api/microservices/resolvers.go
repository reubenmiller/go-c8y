package microservices

import (
	"context"
	"fmt"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/source"
)

// nameResolver looks up a microservice by name
type nameResolver struct {
	Name   string
	Lookup func(ctx context.Context, name string) (string, map[string]any, error)
}

func (n nameResolver) ResolveID(ctx context.Context) (source.ResolveResult, error) {
	if n.Lookup == nil {
		return source.ResolveResult{}, fmt.Errorf("no lookup function configured for microservice name")
	}
	id, meta, err := n.Lookup(ctx, n.Name)
	if err != nil {
		return source.ResolveResult{}, err
	}
	if meta == nil {
		meta = make(map[string]any)
	}
	meta["namePattern"] = n.Name
	meta["source"] = "name"
	return source.ResolveResult{ID: id, Meta: meta}, nil
}

func (n nameResolver) String() string {
	return fmt.Sprintf("name:%s", n.Name)
}

// contextPathResolver looks up a microservice by context path
type contextPathResolver struct {
	ContextPath string
	Lookup      func(ctx context.Context, contextPath string) (string, map[string]any, error)
}

func (c contextPathResolver) ResolveID(ctx context.Context) (source.ResolveResult, error) {
	if c.Lookup == nil {
		return source.ResolveResult{}, fmt.Errorf("no lookup function configured for microservice contextPath")
	}
	id, meta, err := c.Lookup(ctx, c.ContextPath)
	if err != nil {
		return source.ResolveResult{}, err
	}
	if meta == nil {
		meta = make(map[string]any)
	}
	meta["contextPathPattern"] = c.ContextPath
	meta["source"] = "contextPath"
	return source.ResolveResult{ID: id, Meta: meta}, nil
}

func (c contextPathResolver) String() string {
	return fmt.Sprintf("contextPath:%s", c.ContextPath)
}

// newResolvers builds the reference schemes specific to microservices:
//   - "12345" -> direct ID
//   - "id:12345" -> direct ID
//   - "name:my-microservice" -> name lookup
//   - "contextPath:/my-microservice" -> contextPath lookup
//   - "custom:..." -> custom resolver (if registered)
func (s *Service) newResolvers() *source.Registry {
	return source.NewRegistry("microservice", map[string]source.SchemeParser{
		"name": func(value string) (source.Resolver, error) {
			return nameResolver{Name: value, Lookup: func(ctx context.Context, name string) (string, map[string]any, error) {
				return s.lookupByName(ctx, name)
			}}, nil
		},
		"contextPath": func(value string) (source.Resolver, error) {
			return contextPathResolver{ContextPath: value, Lookup: func(ctx context.Context, contextPath string) (string, map[string]any, error) {
				return s.lookupByContextPath(ctx, contextPath)
			}}, nil
		},
	})
}
