package api

// Request-scoped options.
//
// Every option that changes how a single request behaves is attached to a
// context.Context with the functions in this file, and applies to all API
// calls made with that context:
//
//	ctx := api.WithProcessingModeTransient(context.Background())
//	ctx = api.WithTenant(ctx, "t12345")
//	result := client.Events.Create(ctx, body)
//
// This package is the only public place for these options.

import (
	"context"
	"net/http"
	"net/url"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/authentication"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	ctxhelpers "github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/contexthelpers"
)

// WithDryRun returns a context with dry run enabled
// Dry run mode logs requests for inspection/validation without sending them
func WithDryRun(ctx context.Context, enabled bool) context.Context {
	return ctxhelpers.WithDryRun(ctx, enabled)
}

// IsDryRun checks if dry run is enabled in the context
func IsDryRun(ctx context.Context) bool {
	return ctxhelpers.IsDryRun(ctx)
}

// WithMockResponses returns a context with mock responses enabled
// When enabled, HTTP requests will return mock data from embedded JSON files
// instead of making real API calls. Useful for unit testing without network dependencies.
// Can be combined with WithDryRun for logging + mock data, or used independently.
func WithMockResponses(ctx context.Context, enabled bool) context.Context {
	return ctxhelpers.WithMockResponses(ctx, enabled)
}

// IsMockResponses checks if mock responses are enabled in the context
func IsMockResponses(ctx context.Context) bool {
	return ctxhelpers.IsMockResponses(ctx)
}

// WithRedactHeaders returns a context with header redaction control
// By default, headers are redacted for security. Set to false to disable redaction for debugging.
// Example: ctx = api.WithRedactHeaders(ctx, false) // Disable redaction for debugging
func WithRedactHeaders(ctx context.Context, redact bool) context.Context {
	return ctxhelpers.WithRedactHeaders(ctx, redact)
}

// ShouldRedactHeaders checks if header redaction is enabled in the context
// Returns true by default (secure by default)
func ShouldRedactHeaders(ctx context.Context) bool {
	return ctxhelpers.ShouldRedactHeaders(ctx)
}

// WithDeferredExecution returns a context with deferred execution enabled
// When enabled, operations will prepare the request (including parameter resolution)
// but won't execute the HTTP call until Result.Execute() is called.
// This is useful for confirmation prompts before destructive operations.
func WithDeferredExecution(ctx context.Context, enabled bool) context.Context {
	return ctxhelpers.WithDeferredExecution(ctx, enabled)
}

// IsDeferredExecution checks if deferred execution is enabled in the context
func IsDeferredExecution(ctx context.Context) bool {
	return ctxhelpers.IsDeferredExecution(ctx)
}

// WithAuth returns a context that carries per-request credentials. API calls made
// with the returned context use these credentials instead of the client-level
// credentials, so one client can act on behalf of multiple tenants.
func WithAuth(ctx context.Context, auth authentication.AuthOptions) context.Context {
	return authentication.WithAuth(ctx, auth)
}

// WithServiceUser returns a context that carries the given service user's
// credentials. API calls made with the returned context execute as that
// tenant's service user (the typical pattern in MULTI_TENANT microservices).
func WithServiceUser(ctx context.Context, user model.ServiceUser) context.Context {
	return authentication.WithAuth(ctx, authentication.AuthOptions{
		Tenant:   user.Tenant,
		Username: user.Username,
		Password: user.Password,
	})
}

// WithTenant returns a context that overrides the {tenantId} path parameter for
// tenant-scoped endpoints without changing which credentials are used.
func WithTenant(ctx context.Context, tenant string) context.Context {
	return ctxhelpers.WithTenant(ctx, tenant)
}

// TenantFromContext returns the tenant set with WithTenant. The second return
// value is false when no tenant override is present.
func TenantFromContext(ctx context.Context) (string, bool) {
	return ctxhelpers.TenantFromContext(ctx)
}

// WithDryRunHandler returns a context that hands each dry-run request to fn
// instead of logging it. It only has an effect together with WithDryRun.
func WithDryRunHandler(ctx context.Context, fn func(*http.Request)) context.Context {
	return ctxhelpers.WithDryRunHandler(ctx, fn)
}

// WithProcessingMode returns a context that sets the X-Cumulocity-Processing-Mode
// header on every request made with it.
func WithProcessingMode(ctx context.Context, mode types.ProcessingMode) context.Context {
	return ctxhelpers.WithProcessingMode(ctx, mode)
}

// WithProcessingModePersistent sets the processing mode to PERSISTENT (the
// platform default): data is stored and sent to the streaming analytics engine.
func WithProcessingModePersistent(ctx context.Context) context.Context {
	return ctxhelpers.WithProcessingModePersistent(ctx)
}

// WithProcessingModeTransient sets the processing mode to TRANSIENT: data is
// sent to the streaming analytics engine but not stored.
func WithProcessingModeTransient(ctx context.Context) context.Context {
	return ctxhelpers.WithProcessingModeTransient(ctx)
}

// WithProcessingModeQuiescent sets the processing mode to QUIESCENT: like
// PERSISTENT, but no real-time notifications are sent.
func WithProcessingModeQuiescent(ctx context.Context) context.Context {
	return ctxhelpers.WithProcessingModeQuiescent(ctx)
}

// WithProcessingModeCEP sets the processing mode to CEP: like TRANSIENT, but no
// real-time notifications are sent.
func WithProcessingModeCEP(ctx context.Context) context.Context {
	return ctxhelpers.WithProcessingModeCEP(ctx)
}

// ProcessingMode returns the processing mode set on the context, or an empty
// value when none is set.
func ProcessingMode(ctx context.Context) types.ProcessingMode {
	return ctxhelpers.GetProcessingMode(ctx)
}

// WithExtraHeaders returns a context that adds the given headers to every
// request made with it.
func WithExtraHeaders(ctx context.Context, headers map[string]string) context.Context {
	return ctxhelpers.WithExtraHeaders(ctx, headers)
}

// ExtraHeaders returns the headers set with WithExtraHeaders.
func ExtraHeaders(ctx context.Context) map[string]string {
	return ctxhelpers.GetExtraHeaders(ctx)
}

// WithExtraQueryParams returns a context that adds the given query parameters
// to every request made with it.
func WithExtraQueryParams(ctx context.Context, params url.Values) context.Context {
	return ctxhelpers.WithExtraQueryParams(ctx, params)
}

// ExtraQueryParams returns the query parameters set with WithExtraQueryParams.
func ExtraQueryParams(ctx context.Context) url.Values {
	return ctxhelpers.GetExtraQueryParams(ctx)
}

// WithNoAccept returns a context that removes the Accept header, so a POST or
// PUT returns the resource as stored instead of a forced representation.
func WithNoAccept(ctx context.Context, enabled bool) context.Context {
	return ctxhelpers.WithNoAccept(ctx, enabled)
}

// WithNoAuth returns a context that sends requests without an Authorization
// header, for endpoints that must be called unauthenticated.
func WithNoAuth(ctx context.Context) context.Context {
	return ctxhelpers.WithNoAuth(ctx)
}
