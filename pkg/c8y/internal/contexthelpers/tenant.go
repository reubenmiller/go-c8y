package contexthelpers

import "context"

type tenantKey struct{}

// WithTenant returns a context that overrides the {tenantId} path parameter for
// tenant-scoped endpoints (e.g. /application/applicationsByTenant/{tenantId}).
// Unlike authentication.WithAuth it does not change which credentials are used,
// only which tenant the URL refers to.
func WithTenant(ctx context.Context, tenant string) context.Context {
	return context.WithValue(ctx, tenantKey{}, tenant)
}

// TenantFromContext returns the tenant ID stored in the context via WithTenant.
// The second return value is false when no tenant override is present.
func TenantFromContext(ctx context.Context) (string, bool) {
	tenant, ok := ctx.Value(tenantKey{}).(string)
	return tenant, ok
}
