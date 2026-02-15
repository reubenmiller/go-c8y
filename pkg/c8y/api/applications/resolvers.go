package applications

import (
	"context"
	"fmt"
	"strings"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/source"
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

// parseResolver parses a source reference string into a resolver
// Supports formats specific to applications:
//   - "12345" -> direct ID
//   - "id:12345" -> direct ID
//   - "name:cockpit" -> name lookup
//   - "name:cockpit:HOSTED" -> name lookup with type filter
//   - "custom:..." -> custom resolver (if registered)
func (s *Service) parseResolver(str string) (source.Resolver, error) {
	if str == "" {
		return nil, fmt.Errorf("empty application reference string")
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
		// Format: name:appName or name:appName:type
		appName := value
		appType := ""
		idx2 := strings.IndexByte(value, ':')
		if idx2 != -1 {
			appName = value[:idx2]
			appType = value[idx2+1:]
		}
		return nameResolver{
			Name:   appName,
			Type:   appType,
			Lookup: s.lookupByName,
		}, nil

	default:
		return nil, fmt.Errorf("unknown application resolver scheme: %s", prefix)
	}
}
