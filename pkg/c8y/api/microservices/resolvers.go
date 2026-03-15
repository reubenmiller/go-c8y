package microservices

import (
	"context"
	"fmt"
	"strings"

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

// parseResolver parses a source reference string into a resolver
// Supports formats specific to microservices:
//   - "12345" -> direct ID
//   - "id:12345" -> direct ID
//   - "name:my-microservice" -> name lookup
//   - "contextPath:/my-microservice" -> contextPath lookup
//   - "custom:..." -> custom resolver (if registered)
func (s *Service) parseResolver(str string) (source.Resolver, error) {
	if str == "" {
		return nil, fmt.Errorf("empty microservice reference string")
	}

	// Try to parse with prefix
	idx := strings.IndexByte(str, ':')
	if idx == -1 {
		// No prefix, treat as direct ID
		return source.ID(str), nil
	}

	prefix := str[:idx]
	value := str[idx+1:]

	// Check custom resolvers first
	if s.customResolvers != nil {
		if resolver, ok := s.customResolvers[prefix]; ok {
			return resolver, nil
		}
	}

	switch prefix {
	case "id":
		return source.ID(value), nil

	case "name":
		return nameResolver{
			Name:   value,
			Lookup: s.lookupByName,
		}, nil

	case "contextPath":
		return contextPathResolver{
			ContextPath: value,
			Lookup:      s.lookupByContextPath,
		}, nil

	default:
		return nil, fmt.Errorf("unknown microservice resolver scheme: %s", prefix)
	}
}
