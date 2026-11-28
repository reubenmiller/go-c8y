// Package lookup holds the reference resolution shared by the services whose
// resources are stored as managed objects: agents, device groups, smart groups
// and device profiles. Each of them resolves "name:<name>" and "query:<q>"
// references with an inventory query scoped to its own kind of managed object.
package lookup

import (
	"context"
	"fmt"
	"strings"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/contexthelpers"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
)

// Item is a resource with an id and a name.
type Item interface {
	ID() string
	Name() string
}

// Scoped describes how the references of one managed-object backed resource
// are resolved.
type Scoped[T Item] struct {
	// Resource is the singular noun used in error messages, e.g. "device group".
	Resource string

	// NameQuery builds the inventory query which finds the resource with the
	// given name, scoped to the kind of resource.
	NameQuery func(name string) string

	// ScopeQuery builds the inventory query which combines the scope of the
	// resource with the raw filter of a "query:" reference.
	ScopeQuery func(filter string) string

	// First lists the first resource matching an inventory query.
	First func(ctx context.Context, query string) op.Result[T]
}

// ResolveID resolves a reference to an id. A plain string is used as-is;
// "id:<id>", "name:<name>" and "query:<q>" are supported. The metadata of the
// lookup (id, name and source) is added to meta when it is not nil.
//
// A reference which finds nothing fails with an error which wraps a 404
// [core.Error]; for a name it is a [core.NameNotFoundError]. Every error is a
// [core.LookupError].
func (s Scoped[T]) ResolveID(ctx context.Context, ref string, meta map[string]any) (_ string, resolveErr error) {
	defer func() { resolveErr = core.WrapLookup(s.Resource, ref, resolveErr) }()
	if ref == "" {
		return "", fmt.Errorf("empty %s reference", s.Resource)
	}
	scheme, value, hasScheme := strings.Cut(ref, ":")
	if !hasScheme {
		return ref, nil
	}
	// Resolve for real even under dry run / deferred execution.
	ctx = contexthelpers.ResolutionContext(ctx)
	switch scheme {
	case "id":
		return value, nil
	case "name":
		return s.first(ctx, s.NameQuery(value), value, "name", meta)
	case "query":
		return s.first(ctx, s.ScopeQuery(value), value, "query", meta)
	default:
		return "", fmt.Errorf("unknown %s resolver scheme: %q", s.Resource, scheme)
	}
}

// first returns the id of the first resource matching query. label is the
// original reference, for error messages and metadata.
func (s Scoped[T]) first(ctx context.Context, query, label, source string, meta map[string]any) (string, error) {
	result := s.First(ctx, query)
	if result.Err != nil {
		return "", result.Err
	}
	for item := range result.Seq() {
		if meta != nil {
			meta["id"] = item.ID()
			meta["name"] = item.Name()
			meta["source"] = source
		}
		return item.ID(), nil
	}
	if source == "name" {
		return "", core.NewNameNotFoundError(s.Resource, label, "%s not found: %s", s.Resource, label)
	}
	return "", queryNotFoundError{
		msg: fmt.Sprintf("%s not found: %s", s.Resource, label),
		err: core.ErrNotFound("%s not found: %s", s.Resource, label),
	}
}

// queryNotFoundError is the error of a "query:" reference which matches nothing.
// It keeps the plain message such a reference has always reported, while
// [core.IsNotFound] recognises it as a 404 like the not found error of a name.
type queryNotFoundError struct {
	msg string
	err *core.Error
}

func (e queryNotFoundError) Error() string { return e.msg }

// Unwrap gives access to the 404 error, so errors.As and core.IsNotFound work.
func (e queryNotFoundError) Unwrap() error { return e.err }
