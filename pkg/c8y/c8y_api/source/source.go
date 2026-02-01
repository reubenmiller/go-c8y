package source

import (
	"context"
	"fmt"
)

// ResolveResult contains the resolved ID and optional metadata about what was resolved
type ResolveResult struct {
	// ID is the resolved managed object ID
	ID string

	// Meta contains optional metadata about the resolved object
	// Common keys: "name", "type", "externalType", "externalID"
	// Resolvers can include any metadata they find useful for display/logging
	Meta map[string]any
}

// GetString retrieves a string value from metadata
func (r ResolveResult) GetString(key string) string {
	if r.Meta == nil {
		return ""
	}
	if v, ok := r.Meta[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// GetName is a convenience method to get the "name" metadata
func (r ResolveResult) GetName() string {
	return r.GetString("name")
}

// GetType is a convenience method to get the "type" metadata
func (r ResolveResult) GetType() string {
	return r.GetString("type")
}

// Resolver is an interface for resolving a managed object ID from various sources
type Resolver interface {
	// ResolveID resolves the reference to a managed object ID and returns metadata
	ResolveID(ctx context.Context) (ResolveResult, error)

	// String returns a string representation (for logging/debugging)
	String() string
}

// ID represents a direct managed object ID (no lookup needed)
type ID string

func (id ID) ResolveID(ctx context.Context) (ResolveResult, error) {
	return ResolveResult{
		ID:   string(id),
		Meta: map[string]any{"source": "direct-id"},
	}, nil
}

func (id ID) String() string {
	return string(id)
}

// ExternalID represents a lookup by external identity
type ExternalID struct {
	Type       string
	ExternalID string
	Lookup     func(ctx context.Context, typ, extID string) (string, map[string]any, error)
}

func (e ExternalID) GetType() string {
	if e.Type == "" {
		return "c8y_Serial"
	}
	return e.Type
}

func (e ExternalID) ResolveID(ctx context.Context) (ResolveResult, error) {
	if e.Lookup == nil {
		return ResolveResult{}, fmt.Errorf("no lookup function configured for external ID")
	}
	id, meta, err := e.Lookup(ctx, e.GetType(), e.ExternalID)
	if err != nil {
		return ResolveResult{}, err
	}
	// Merge provided metadata with external ID info
	if meta == nil {
		meta = make(map[string]any)
	}
	meta["externalType"] = e.GetType()
	meta["externalID"] = e.ExternalID
	meta["source"] = "external-id"
	return ResolveResult{ID: id, Meta: meta}, nil
}

func (e ExternalID) String() string {
	return fmt.Sprintf("ext:%s:%s", e.GetType(), e.ExternalID)
}

// Name represents a lookup by device name using inventory query
type Name struct {
	Name   string
	Lookup func(ctx context.Context, name string) (string, map[string]any, error)
}

func (n Name) ResolveID(ctx context.Context) (ResolveResult, error) {
	if n.Lookup == nil {
		return ResolveResult{}, fmt.Errorf("no lookup function configured for name")
	}
	id, meta, err := n.Lookup(ctx, n.Name)
	if err != nil {
		return ResolveResult{}, err
	}
	// Merge provided metadata with name info
	if meta == nil {
		meta = make(map[string]any)
	}
	meta["namePattern"] = n.Name
	meta["source"] = "name"
	return ResolveResult{ID: id, Meta: meta}, nil
}

func (n Name) String() string {
	return fmt.Sprintf("name:%s", n.Name)
}

// Query represents a custom inventory query that should return exactly one result
type Query struct {
	Query  string
	Lookup func(ctx context.Context, query string) (string, map[string]any, error)
}

func (q Query) ResolveID(ctx context.Context) (ResolveResult, error) {
	if q.Lookup == nil {
		return ResolveResult{}, fmt.Errorf("no lookup function configured for query")
	}
	id, meta, err := q.Lookup(ctx, q.Query)
	if err != nil {
		return ResolveResult{}, err
	}
	// Merge provided metadata with query info
	if meta == nil {
		meta = make(map[string]any)
	}
	meta["query"] = q.Query
	meta["source"] = "query"
	return ResolveResult{ID: id, Meta: meta}, nil
}

func (q Query) String() string {
	return fmt.Sprintf("query:%s", q.Query)
}

// Custom allows users to define their own resolution logic
type Custom struct {
	Description string
	Resolve     func(ctx context.Context) (string, map[string]any, error)
}

func (c Custom) ResolveID(ctx context.Context) (ResolveResult, error) {
	if c.Resolve == nil {
		return ResolveResult{}, fmt.Errorf("no resolve function configured")
	}
	id, meta, err := c.Resolve(ctx)
	if err != nil {
		return ResolveResult{}, err
	}
	// Merge provided metadata with custom resolver info
	if meta == nil {
		meta = make(map[string]any)
	}
	meta["description"] = c.Description
	meta["source"] = "custom"
	return ResolveResult{ID: id, Meta: meta}, nil
}

func (c Custom) String() string {
	if c.Description != "" {
		return fmt.Sprintf("custom:%s", c.Description)
	}
	return "custom"
}

// Resolve is a helper that resolves any Resolver to a ResolveResult
// If the input is already a string, it returns it as a ResolveResult (for backward compatibility)
// If it's a Resolver, it calls ResolveID
// Otherwise returns an error
func Resolve(ctx context.Context, src any) (ResolveResult, error) {
	if src == nil {
		return ResolveResult{}, nil
	}

	// Direct string (backward compatibility)
	if str, ok := src.(string); ok {
		return ResolveResult{
			ID:   str,
			Meta: map[string]any{"source": "string"},
		}, nil
	}

	// Resolver interface
	if resolver, ok := src.(Resolver); ok {
		return resolver.ResolveID(ctx)
	}

	return ResolveResult{}, fmt.Errorf("unsupported source type: %T", src)
}

// MustResolve is like Resolve but panics on error (useful for tests)
func MustResolve(ctx context.Context, src any) ResolveResult {
	result, err := Resolve(ctx, src)
	if err != nil {
		panic(err)
	}
	return result
}
