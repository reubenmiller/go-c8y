## 1. Preparation

- [ ] 1.1 Land PR #180 (resty v3.0.0-rc.4 and the DryRunTransport body copy)
- [ ] 1.2 Add a CI check that fails if `go doc -all ./pkg/...` shows a `resty.` type other than `Client.Resty()`, initially in report-only mode
- [ ] 1.3 List go-c8y-cli call sites that use resty types through go-c8y, to size the migration

## 2. Core layer

- [ ] 2.1 Add `core.RawResponse` (status, headers, body bytes, duration, copied `*http.Request`) and change `ExecuteResponseOnly` to return it
- [ ] 2.2 Add `core.MultipartField`; change `NewMultiPartFile` / `NewMultiPartFileFields` to return it
- [ ] 2.3 Change `BinaryResponse` to hold `*http.Response`; update `NewBinaryResponse`
- [ ] 2.4 Make `core.Service.Client`, `TryRequest.Request`, `NewTryRequest`, `TryRequest.Funcs`, and `NewResult` unexported or move them behind internal helpers
- [ ] 2.5 Unit tests for each scenario in `specs/http-client-contract/spec.md` that targets `core`

## 3. Client layer

- [ ] 3.1 Add `api.Middleware` and `ClientOptions.Middleware []api.Middleware`, applied around the transport
- [ ] 3.2 Convert or unexport `Middleware*`, `TokenSourceMiddleware`, `TokenRenewalRetry*`, `SetAuth`, and `NoAcceptHeader`
- [ ] 3.3 Change `RequestOptions.PrepareRequest` and `OnResponse` to standard library types; make `OnResponse` read a buffered body
- [ ] 3.4 Change `RequestResult.Response` to an SDK-owned type
- [ ] 3.5 Make `Client.HTTPClient` unexported; add `(*Client).Resty()` with the compatibility warning in godoc
- [ ] 3.6 Offline tests for middleware order, header injection, hook errors, and body re-reads

## 4. Service layer

- [ ] 4.1 Replace `core.Service` embedding with an unexported field in all service packages (mechanical change; one commit per group of packages)
- [ ] 4.2 Run `go test ./pkg/...` and the offline suite after each group

## 5. Documentation and rollout

- [ ] 5.1 Add a migration table (old resty-based call → new call) to docs and the CHANGELOG
- [ ] 5.2 Update `docs/API_DESIGN.md`, `docs/REQUEST_INSPECTION.md`, and examples
- [ ] 5.3 Switch the CI check from task 1.2 to failing mode
- [ ] 5.4 Upgrade to resty v3.0.0 when released and confirm only internal files change
