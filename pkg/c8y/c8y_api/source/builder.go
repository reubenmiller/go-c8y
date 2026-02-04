package source

import (
	"context"
	"fmt"
	"strings"
)

// Builder provides a fluent API for creating source resolvers
// This is injected with client-specific lookup functions
type Builder struct {
	lookupByExternalID func(ctx context.Context, typ, extID string) (string, map[string]any, error)
	lookupByName       func(ctx context.Context, name string) (string, map[string]any, error)
	lookupByQuery      func(ctx context.Context, query string) (string, map[string]any, error)
	customResolvers    map[string]Resolver
}

// NewBuilder creates a new source builder with lookup functions
func NewBuilder(
	lookupByExternalID func(ctx context.Context, typ, extID string) (string, map[string]any, error),
	lookupByName func(ctx context.Context, name string) (string, map[string]any, error),
	lookupByQuery func(ctx context.Context, query string) (string, map[string]any, error),
) *Builder {
	return &Builder{
		lookupByExternalID: lookupByExternalID,
		lookupByName:       lookupByName,
		lookupByQuery:      lookupByQuery,
		customResolvers:    make(map[string]Resolver),
	}
}

// ByID creates a direct ID reference (no lookup)
func (b *Builder) ByID(id string) Resolver {
	return ID(id)
}

// ByExternalID creates a resolver that looks up by external identity
func (b *Builder) ByExternalID(typ, externalID string) Resolver {
	return ExternalID{
		Type:       typ,
		ExternalID: externalID,
		Lookup:     b.lookupByExternalID,
	}
}

// ByName creates a resolver that looks up by device name
func (b *Builder) ByName(name string) Resolver {
	return Name{
		Name:   name,
		Lookup: b.lookupByName,
	}
}

// ByQuery creates a resolver that looks up using a custom inventory query
func (b *Builder) ByQuery(query string) Resolver {
	return Query{
		Query:  query,
		Lookup: b.lookupByQuery,
	}
}

// Custom creates a resolver with custom logic
func (b *Builder) Custom(description string, resolver func(ctx context.Context) (string, map[string]any, error)) Resolver {
	return Custom{
		Description: description,
		Resolve:     resolver,
	}
}

// Parse parses a source reference from a string with prefixes
// Supports formats like:
//   - "12345" -> direct ID
//   - "id:12345" -> direct ID
//   - "ext:c8y_Serial:ABC123" -> external ID lookup
//   - "name:MyDevice" -> name lookup
//   - "query:name eq 'MyDevice'" -> query lookup
//   - "custom:..." -> custom resolver (if registered)
func (b *Builder) Parse(s string) (Resolver, error) {
	if s == "" {
		return nil, fmt.Errorf("empty source string")
	}

	// Try to parse with prefix
	idx := strings.IndexByte(s, ':')
	if idx == -1 {
		// No prefix, treat as direct ID
		return b.ByID(s), nil
	}

	prefix := s[:idx]
	value := s[idx+1:]

	// Check custom resolvers first
	if resolver, ok := b.customResolvers[prefix]; ok {
		return resolver, nil
	}

	switch prefix {
	case "id":
		return b.ByID(value), nil

	case "ext":
		// Format: ext:type:externalId
		idx2 := strings.IndexByte(value, ':')
		if idx2 == -1 {
			return nil, fmt.Errorf("invalid external ID format, expected ext:type:externalId")
		}
		typ := value[:idx2]
		extID := value[idx2+1:]
		return b.ByExternalID(typ, extID), nil

	case "name":
		return b.ByName(value), nil

	case "query":
		return b.ByQuery(value), nil

	default:
		return nil, fmt.Errorf("unknown resolver scheme: %s", prefix)
	}
}

// RegisterResolver registers a custom resolver for a given scheme
func (b *Builder) RegisterResolver(scheme string, resolver Resolver) {
	b.customResolvers[scheme] = resolver
}
