## Why

The v2 public API exposes resty types (`*resty.Client`, `*resty.Request`, `*resty.Response`, `resty.RequestMiddleware`). resty v3 is still a release candidate. The beta to rc.4 upgrade (PR #180) needed six renames and fixed a silent change in how request buffers are pooled. Any such change in resty after v2.0.0 would break go-c8y users, even though go-c8y itself did not change its API. Review: docs/V2_STABILIZATION_REVIEW.md §5.1.

## What Changes

- **BREAKING** Replace resty types in exported signatures with SDK-owned types built on the standard library:
  - Middleware becomes `api.Middleware func(next http.RoundTripper) http.RoundTripper`.
  - Hooks use `*http.Request` and `*http.Response` (`RequestOptions.PrepareRequest`, `RequestOptions.OnResponse`, `RequestResult.Response`).
  - `core.ExecuteResponseOnly` returns an SDK response type, not `*resty.Response`.
  - Multipart and binary helpers use `core.MultipartField` and `*http.Response`.
- **BREAKING** Make `Client.HTTPClient` and `core.Service.Client` unexported. Add one escape hatch, `(*Client).Resty() *resty.Client`, documented as outside the v2 compatibility guarantee.
- **BREAKING** Services stop embedding `core.Service` in their exported types, so its fields are not promoted into every service's public API.
- Built-in `Middleware*` helpers are either converted to `api.Middleware` or made unexported.
- Track resty v3.0.0 when it is released. Only internal code changes after this work.

Must land before v2.0.0.

## Capabilities

### New Capabilities
- `http-client-contract`: the SDK's public HTTP extension points (middleware, request and response hooks, raw response access) and the guarantee that they use only standard library and SDK-owned types.

### Modified Capabilities
<!-- none: no existing specs -->

## Impact

- `pkg/c8y/api` (23 exported references to resty), `pkg/c8y/api/core` (15), `pkg/c8y` (1).
- All ~66 service packages under `pkg/c8y/api/*` (embedding of `core.Service`).
- `examples/` and docs that use `PrepareRequest`, `OnResponse`, or `Middleware*`.
- go-c8y-cli and other consumers that register resty middleware need migration notes.
