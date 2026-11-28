package source

import (
	"context"
	"fmt"
	"strings"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
)

// SchemeParser builds the [Resolver] of a reference of one scheme, given the
// text after the scheme prefix (for "name:cockpit:HOSTED" it receives
// "cockpit:HOSTED").
type SchemeParser func(value string) (Resolver, error)

// Registry parses and resolves the references of one kind of resource, such as
// "12345", "id:12345" or "name:cockpit". A reference without a scheme is a
// plain id. It holds the schemes a service understands plus the custom
// resolvers registered by the caller, which take precedence over the
// built-in schemes.
//
// A Registry is the shared implementation of the ResolveID and RegisterResolver
// methods of the services; it is not safe to call Register concurrently with
// Parse or Resolve.
type Registry struct {
	resource string
	schemes  map[string]SchemeParser
	custom   map[string]Resolver
}

// NewRegistry creates a registry for resource (the singular noun used in error
// messages, e.g. "application") with the given built-in schemes. The "id"
// scheme is always available and does not need to be listed.
func NewRegistry(resource string, schemes map[string]SchemeParser) *Registry {
	return &Registry{
		resource: resource,
		schemes:  schemes,
		custom:   make(map[string]Resolver),
	}
}

// Register adds a custom resolver for scheme. It replaces a built-in scheme of
// the same name and a custom resolver registered before.
func (r *Registry) Register(scheme string, resolver Resolver) {
	r.custom[scheme] = resolver
}

// Parse turns a reference into the resolver which can resolve it. It fails for
// an empty reference and for an unknown scheme.
func (r *Registry) Parse(ref string) (Resolver, error) {
	if ref == "" {
		return nil, fmt.Errorf("empty %s reference", r.resource)
	}
	scheme, value, hasScheme := strings.Cut(ref, ":")
	if !hasScheme {
		return ID(ref), nil
	}
	if resolver, ok := r.custom[scheme]; ok {
		return resolver, nil
	}
	if scheme == "id" {
		return ID(value), nil
	}
	if parse, ok := r.schemes[scheme]; ok {
		return parse(value)
	}
	return nil, fmt.Errorf("unknown %s resolver scheme: %q", r.resource, scheme)
}

// Resolve resolves a reference to an id. When meta is not nil, the metadata of
// the resolution is merged into it. Every error is marked as a
// [core.LookupError] of the resource.
func (r *Registry) Resolve(ctx context.Context, ref string, meta map[string]any) (_ string, resolveErr error) {
	defer func() { resolveErr = core.WrapLookup(r.resource, ref, resolveErr) }()
	resolver, err := r.Parse(ref)
	if err != nil {
		return "", err
	}
	result, err := resolver.ResolveID(ctx)
	if err != nil {
		return "", err
	}
	for k, v := range result.Meta {
		if meta != nil {
			meta[k] = v
		}
	}
	return result.ID, nil
}
