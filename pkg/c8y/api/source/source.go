package source

import (
	"context"
	"fmt"
)

// ResolveResult contains the resolved ID and optional metadata about what was resolved
type ResolveResult struct {
	// ID is the resolved managed object ID (or other entity ID)
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

// Resolver is an interface for resolving an entity ID from various sources.
// Each service (managedobjects, applications, users, etc.) implements resolvers
// specific to their entity type.
type Resolver interface {
	// ResolveID resolves the reference to an entity ID and returns metadata
	ResolveID(ctx context.Context) (ResolveResult, error)

	// String returns a string representation (for logging/debugging)
	String() string
}

// ID represents a direct entity ID (no lookup needed).
// This is a basic resolver that all services can use.
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

// Custom represents a resolver with custom logic.
// This allows users to implement their own resolution strategies.
type Custom struct {
	Description string
	Resolve     func(ctx context.Context) (string, map[string]any, error)
}

func (c Custom) ResolveID(ctx context.Context) (ResolveResult, error) {
	if c.Resolve == nil {
		return ResolveResult{}, fmt.Errorf("no resolve function provided for custom resolver")
	}
	id, meta, err := c.Resolve(ctx)
	if err != nil {
		return ResolveResult{}, err
	}
	if meta == nil {
		meta = make(map[string]any)
	}
	meta["source"] = "custom"
	meta["description"] = c.Description
	return ResolveResult{ID: id, Meta: meta}, nil
}

func (c Custom) String() string {
	return fmt.Sprintf("custom:%s", c.Description)
}
