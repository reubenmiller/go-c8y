package authentication

import "context"

type authContextKey struct{}

// WithAuth returns a context that carries per-request authentication credentials.
// Any API call made with the returned context will use these credentials instead
// of the credentials configured on the client, making it possible to share a
// single client across multiple tenants (e.g. inside a MULTI_TENANT microservice).
//
// Credential selection within the AuthOptions follows the same precedence as the
// client itself: a Token takes priority, otherwise Username/Password (Basic) is used.
// The Tenant field additionally overrides the {tenantId} path parameter used by
// tenant-scoped endpoints.
//
// Example:
//
//	ctx := authentication.WithAuth(context.Background(), authentication.AuthOptions{
//		Tenant:   "t12345",
//		Username: "service_myapp",
//		Password: "...",
//	})
//	result := client.Devices.List(ctx, devices.ListOptions{})
func WithAuth(ctx context.Context, auth AuthOptions) context.Context {
	return context.WithValue(ctx, authContextKey{}, auth)
}

// AuthFromContext returns the per-request credentials stored in the context via
// WithAuth. The second return value is false when no credentials are present.
func AuthFromContext(ctx context.Context) (AuthOptions, bool) {
	auth, ok := ctx.Value(authContextKey{}).(AuthOptions)
	return auth, ok
}
