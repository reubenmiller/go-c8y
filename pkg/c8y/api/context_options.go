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
//
// To observe or wrap the HTTP bodies, use WithRequestHook (final request),
// WithResponseBodyWrapper (response body), or the progress shortcuts
// WithUploadProgress and WithDownloadProgress. Typed uploads also take
// core.UploadFileOptions.Progress, which reports the file bytes and the file
// size. For endpoints without a service method use Client.Request, which
// returns an op.Result and applies the same options.

import (
	"context"
	"io"
	"net/http"
	"net/url"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/authentication"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
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
//
// auth.AuthType selects what is sent when auth has more than one kind of
// credentials: AuthTypeBasic sends Basic (no token, no token exchange),
// AuthTypeNone sends no Authorization header.
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

// ErrRequestNotApproved is the error of a request declined by the approver set
// with WithRequestApprover. Use errors.Is to detect it.
var ErrRequestNotApproved = ctxhelpers.ErrRequestNotApproved

// RequestApprover decides whether a request that changes data may be sent. It
// receives the final request (resolved ids, headers and query parameters
// applied) before it is sent, so it can show exactly what will happen.
// Returning false declines the request with ErrRequestNotApproved; returning
// an error declines it with that error.
type RequestApprover = ctxhelpers.RequestApprover

// WithRequestApprover returns a context whose requests that change data (any
// method other than GET, HEAD and OPTIONS) are passed to fn before they are
// sent. Use it for confirmation prompts: unlike deferred execution, the
// request is built and sent once, and streaming uploads work unchanged.
//
// The approver is not called for dry runs, mock responses, deferred
// preparation or the lookups that resolve references (names, external ids).
// Combine it with WithApprovalScope so an operation that sends several
// requests is approved once. A nil fn removes the approver.
func WithRequestApprover(ctx context.Context, fn RequestApprover) context.Context {
	return ctxhelpers.WithRequestApprover(ctx, fn)
}

// WithApprovalScope returns a context that starts a new approval scope: the
// approver is asked for the first request that changes data, and every later
// request made under the same scope reuses that decision. Start one scope per
// logical operation, for example per input item of a CLI command.
func WithApprovalScope(ctx context.Context) context.Context {
	return ctxhelpers.WithApprovalScope(ctx)
}

// ResolutionContext returns the context the SDK uses for reference lookups
// (names, external ids, queries): dry run, deferred execution, implicit mock
// responses and the request approver are removed, everything else (auth,
// tenant, extra headers and query parameters, processing mode, cancellation)
// is kept. Use it when resolving a reference yourself, so the lookup behaves
// exactly like the SDK's own.
func ResolutionContext(ctx context.Context) context.Context {
	return ctxhelpers.ResolutionContext(ctx)
}

// WithRequestHook returns a context whose requests are passed to fn on the
// final *http.Request just before they are sent, once per attempt (again on
// retries) and also for dry runs. fn may change headers or the URL, or replace
// req.Body (for example to wrap it and count bytes). Returning an error aborts
// the request. It applies to every request made with the context, typed or
// raw; for a single [Client.SendRequest] use RequestOptions.PrepareRequest.
// Hooks compose: a hook already on ctx (including the ones set by
// WithUploadProgress and PrepareRequest) runs first, then fn.
func WithRequestHook(ctx context.Context, fn func(req *http.Request) error) context.Context {
	return ctxhelpers.WithRequestHook(ctx, fn)
}

// WithResponseBodyWrapper returns a context whose responses have their body
// replaced by fn(resp) as soon as the headers arrive, for example to count the
// bytes of a download. fn returns the new body; returning nil keeps the
// original. The wrapper is not called for dry runs or mock responses.
// Wrappers compose: a wrapper already on ctx (for example the one set by
// WithDownloadProgress) wraps the body first, then fn wraps the result.
func WithResponseBodyWrapper(ctx context.Context, fn func(resp *http.Response) io.ReadCloser) context.Context {
	return ctxhelpers.WithResponseBodyWrapper(ctx, fn)
}

// WithUploadProgress returns a context that reports the request body bytes
// sent. total is the request's Content-Length, or -1 when it is not known
// (typically a multipart upload, which is streamed). For the file size of a
// typed upload use core.UploadFileOptions.Progress instead.
func WithUploadProgress(ctx context.Context, fn core.ProgressFunc) context.Context {
	return WithRequestHook(ctx, func(req *http.Request) error {
		if req.Body != nil && req.Body != http.NoBody {
			total := req.ContentLength
			if total <= 0 {
				total = -1
			}
			req.Body = core.NewProgressReader(req.Body, total, fn)
		}
		return nil
	})
}

// WithDownloadProgress returns a context that reports the response body bytes
// read, for example while saving a binary. total is the response's
// Content-Length, or -1 when it is not known.
func WithDownloadProgress(ctx context.Context, fn core.ProgressFunc) context.Context {
	return WithResponseBodyWrapper(ctx, func(resp *http.Response) io.ReadCloser {
		return core.NewProgressReader(resp.Body, resp.ContentLength, fn)
	})
}
