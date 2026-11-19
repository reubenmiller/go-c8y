## Context

resty is used for request building, retries, circuit breaking, and middleware. It also appears in exported API:

- `pkg/c8y/api`: `Client.HTTPClient *resty.Client`, `RequestOptions.PrepareRequest func(*resty.Request)`, `RequestOptions.OnResponse func(*resty.Response) error`, `RequestResult.Response *resty.Response`, `SetAuth(*resty.Client, …)`, `NoAcceptHeader(*resty.Request)`, `TokenRenewalRetry`, `TokenRenewalRetryMiddleware`, `TokenSourceMiddleware`, and every `Middleware*` constructor.
- `pkg/c8y/api/core`: `Service.Client`, `TryRequest.Request`, `NewTryRequest`, `TryRequest.Funcs`, `ExecuteResponseOnly`, `BinaryResponse.Response`, `NewBinaryResponse`, `NewMultiPartFile*`, `NewResult`.

rc.4 also showed that resty's internal buffer handling can change without any signature change: `RawRequest.Body` now points into a pooled buffer that is cleared after the request.

## Goals / Non-Goals

**Goals:**
- No resty type in any exported signature, field, or embedded type, except the documented `Resty()` escape hatch.
- Users can extend the client with standard `http.RoundTripper` wrappers.
- resty upgrades only touch internal code.

**Non-Goals:**
- Replacing resty internally. Retries, circuit breaking, and request building stay on resty.
- Changing service method signatures (`List`, `Get`, …).

## Decisions

**Middleware as `func(next http.RoundTripper) http.RoundTripper`.** This is the standard way to extend Go HTTP clients and works with any library. Alternative considered: SDK-defined `RequestMiddleware func(*Request) error` with an SDK `Request` type. Rejected because it recreates resty's shape and needs its own compatibility promise. Trade-off: RoundTripper middleware runs per attempt (inside retries), not once per logical request. Document this.

**Hooks on `*http.Request` / `*http.Response`.** `PrepareRequest` runs on the built `*http.Request` before sending (headers, query, body can still be changed). `OnResponse` receives `*http.Response` with the body already buffered so it can be read more than once.

**Escape hatch `(*Client).Resty()`.** Some users need resty-level control (custom retry conditions, debug). One accessor keeps that possible while making the risk explicit. Alternative: no escape hatch. Rejected because go-c8y-cli uses resty features directly today.

**Stop embedding `core.Service`.** Exported service types hold `svc *core.Service` in an unexported field. Alternative: move `core` to `internal/`. Rejected because promoted fields from an embedded internal type are still accessible to users, so moving alone does not hide them.

**SDK-owned raw response type.** `core.ExecuteResponseOnly` returns `*core.RawResponse` (status, headers, body bytes, duration, and the `*http.Request` copy). The body and request are copied into SDK-owned memory, which also removes the pooled-buffer exposure found in PR #180.

## Risks / Trade-offs

- [Large mechanical diff across ~66 packages] → Do it in one PR per layer (core, api, services) with the offline suite as the regression net.
- [go-c8y-cli breaks] → Provide a migration table (old resty-based call → new call) in docs and the CHANGELOG.
- [RoundTripper middleware sees retries as separate requests] → Document it; offer a request-level hook (`PrepareRequest`) for once-per-request logic.

## Migration Plan

1. Add new types and functions next to the old ones.
2. Move internal callers and examples to the new API.
3. Remove the resty-typed exports in the same pre-v2.0.0 release window (no deprecation period is needed before the first stable tag).

## Open Questions

- Should the built-in `Middleware*` helpers stay public as `api.Middleware`, or become unexported options on `ClientOptions`?
- Does go-c8y-cli need any resty feature that `Resty()` cannot cover?
