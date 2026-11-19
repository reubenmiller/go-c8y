## 1. Preparation

- [x] 1.1 Land PR #180 (resty v3.0.0-rc.4 and the DryRunTransport body copy) (merged)
- [x] 1.2 Add a CI check that fails if `go doc -all ./pkg/...` shows a `resty.` type other than `Client.Resty()`, initially in report-only mode
- [x] 1.3 List go-c8y-cli call sites that use resty types or any `Middleware*` helper through go-c8y, to size the migration (3 `HTTPClient.AddResponseMiddleware` calls in pkg/c8ystream → `Resty()`; its `PrepareRequest`/`OnResponse`/`SendRequest` uses are the v1 client)

## 2. Core layer

- [x] 2.1 Move `ExecuteResponseOnly` to internal `httpcore` (no public raw-response type needed)
- [x] 2.2 Move `NewMultiPartFile` / `NewMultiPartFileFields` to internal `httpcore` (no public multipart type needed)
- [x] 2.3 Change `BinaryResponse` to hold `*http.Response`; update `NewBinaryResponse`
- [x] 2.4 Make `core.Service.Client`, `TryRequest.Request`, `NewTryRequest`, `TryRequest.Funcs`, and `NewResult` unexported or move them behind internal helpers
- [x] 2.5 Unit tests for each scenario in `specs/http-client-contract/spec.md` that targets `core`

## 3. Client layer

- [x] 3.1 Add `api.Middleware` and `ClientOptions.Middleware []api.Middleware`, applied around the transport
- [x] 3.2 Unexport `MiddlewareAddUserAgent`, `MiddlewareAddHost`, `MiddlewareRemoveEmptyTenantID`, `TokenSourceMiddleware`, `MiddlewareContextAuthorization`, `TokenRenewalRetry`, `SetAuth`, and `NoAcceptHeader`; delete the unused `MiddlewareAddCookies`, `MiddlewareAuthorization`, and `TokenRenewalRetryMiddleware`
- [x] 3.2a Replace `MiddlewareCountRequests` / `MiddlewareCountByMethodAndPath` with `api.CountRequests` / `api.CountByMethodAndPath` (`api.Middleware`); update `examples/pipeline` and `test/c8y_api_test/managed_object_bulk_stream_test.go`
- [x] 3.3 Change `RequestOptions.PrepareRequest` and `OnResponse` to standard library types; make `OnResponse` read a buffered body
- [x] 3.4 Change `RequestResult.Response` to an SDK-owned type
- [x] 3.5 Make `Client.HTTPClient` unexported; add `(*Client).Resty()` with the compatibility warning in godoc
- [x] 3.6 Offline tests for middleware order, header injection, hook errors, and body re-reads

## 4. Service layer

- [x] 4.1 Replace `core.Service` embedding with an unexported field in all service packages (mechanical change; one commit per group of packages)
- [x] 4.2 Run `go test ./pkg/...` and the offline suite after each group

## 5. Documentation and rollout

- [x] 5.1 Add a migration table (old resty-based call → new call) to docs and the CHANGELOG
- [x] 5.2 Update `docs/API_DESIGN.md`, `docs/REQUEST_INSPECTION.md`, and examples
- [x] 5.3 Switch the CI check from task 1.2 to failing mode
- [ ] 5.4 Upgrade to resty v3.0.0 when released and confirm only internal files change (follow-up: resty v3.0.0 is not released yet)
