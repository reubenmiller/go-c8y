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

// newResolvers builds the reference schemes specific to managed objects:
//   - "12345" -> direct ID
//   - "id:12345" -> direct ID
//   - "ext:c8y_Serial:ABC123" -> external ID lookup
//   - "name:MyDevice" -> name lookup
//   - "query:name eq 'MyDevice'" -> query lookup
//   - "custom:..." -> custom resolver (if registered)
func (s *Service) newResolvers() *source.Registry {
	return source.NewRegistry("managed object", map[string]source.SchemeParser{
		"ext": func(value string) (source.Resolver, error) {
			// Format: ext:type:externalId
			typ, extID, ok := strings.Cut(value, ":")
			if !ok {
				return nil, fmt.Errorf("invalid external ID format, expected ext:type:externalId")
			}
			return externalIDResolver{
				Type:       typ,
				ExternalID: extID,
				Lookup: func(ctx context.Context, typ, extID string) (string, map[string]any, error) {
					return s.lookupByExternalID(ctx, typ, extID)
				},
			}, nil
		},
		"name": func(value string) (source.Resolver, error) {
			return nameResolver{Name: value, Lookup: func(ctx context.Context, name string) (string, map[string]any, error) {
				return s.lookupByName(ctx, name)
			}}, nil
		},
		"query": func(value string) (source.Resolver, error) {
			return queryResolver{Query: value, Lookup: func(ctx context.Context, query string) (string, map[string]any, error) {
				return s.lookupByQuery(ctx, query)
			}}, nil
		},
	})
}
