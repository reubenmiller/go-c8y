package managedobjects

import (
	"context"
	"fmt"
	"strings"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/source"
)

// externalIDResolver looks up a managed object by external identity
type externalIDResolver struct {
	Type       string
	ExternalID string
	Lookup     func(ctx context.Context, typ, extID string) (string, map[string]any, error)
}

func (e externalIDResolver) GetType() string {
	if e.Type == "" {
		return "c8y_Serial"
	}
	return e.Type
}

func (e externalIDResolver) ResolveID(ctx context.Context) (source.ResolveResult, error) {
	if e.Lookup == nil {
		return source.ResolveResult{}, fmt.Errorf("no lookup function configured for external ID")
	}
	id, meta, err := e.Lookup(ctx, e.GetType(), e.ExternalID)
	if err != nil {
		return source.ResolveResult{}, err
	}
	if meta == nil {
		meta = make(map[string]any)
	}
	meta["externalType"] = e.GetType()
	meta["externalID"] = e.ExternalID
	meta["source"] = "external-id"
	return source.ResolveResult{ID: id, Meta: meta}, nil
}

func (e externalIDResolver) String() string {
	return fmt.Sprintf("ext:%s:%s", e.GetType(), e.ExternalID)
}

// nameResolver looks up a managed object by name
type nameResolver struct {
	Name   string
	Lookup func(ctx context.Context, name string) (string, map[string]any, error)
}

func (n nameResolver) ResolveID(ctx context.Context) (source.ResolveResult, error) {
	if n.Lookup == nil {
		return source.ResolveResult{}, fmt.Errorf("no lookup function configured for name")
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

// queryResolver looks up a managed object using a custom inventory query
type queryResolver struct {
	Query  string
	Lookup func(ctx context.Context, query string) (string, map[string]any, error)
}

func (q queryResolver) ResolveID(ctx context.Context) (source.ResolveResult, error) {
	if q.Lookup == nil {
		return source.ResolveResult{}, fmt.Errorf("no lookup function configured for query")
	}
	id, meta, err := q.Lookup(ctx, q.Query)
	if err != nil {
		return source.ResolveResult{}, err
	}
	if meta == nil {
		meta = make(map[string]any)
	}
	meta["query"] = q.Query
	meta["source"] = "query"
	return source.ResolveResult{ID: id, Meta: meta}, nil
}

func (q queryResolver) String() string {
	return fmt.Sprintf("query:%s", q.Query)
}

// parseResolver parses a source reference string into a resolver
// Supports formats specific to managed objects:
//   - "12345" -> direct ID
//   - "id:12345" -> direct ID
//   - "ext:c8y_Serial:ABC123" -> external ID lookup
//   - "name:MyDevice" -> name lookup
//   - "query:name eq 'MyDevice'" -> query lookup
//   - "custom:..." -> custom resolver (if registered)
func (s *Service) parseResolver(str string) (source.Resolver, error) {
	if str == "" {
		return nil, fmt.Errorf("empty source string")
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

	case "ext":
		// Format: ext:type:externalId
		idx2 := strings.IndexByte(value, ':')
		if idx2 == -1 {
			return nil, fmt.Errorf("invalid external ID format, expected ext:type:externalId")
		}
		typ := value[:idx2]
		extID := value[idx2+1:]
		return externalIDResolver{
			Type:       typ,
			ExternalID: extID,
			Lookup:     s.lookupByExternalID,
		}, nil

	case "name":
		return nameResolver{
			Name:   value,
			Lookup: s.lookupByName,
		}, nil

	case "query":
		return queryResolver{
			Query:  value,
			Lookup: s.lookupByQuery,
		}, nil

	default:
		return nil, fmt.Errorf("unknown resolver scheme: %s", prefix)
	}
}
