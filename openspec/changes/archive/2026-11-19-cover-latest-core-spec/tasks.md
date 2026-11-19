## 1. Spec and drift tooling

- [x] 1.1 Run `task fetch-spec` and `task generate`; commit the new spec and generated files
- [x] 1.2 Make `c8ygen lint` infer the HTTP method for each path literal from its request builder
- [x] 1.3 Support method-prefixed waivers; require a reason comment on every waiver
- [x] 1.4 Add a method-level waiver for `POST /tenant/trusted-certificates/settings/crl` (covered by `PUT` file upload)
- [x] 1.5 Unit tests for method inference and waiver matching in `tools/c8ygen`
- [x] 1.6 Keep the CI api-drift job strict: method-level drift is zero after this change, so no non-strict period is needed

## 2. New operations

- [x] 2.1 Add `pkg/c8y/api/tenants/restrictedroles` (`List`, `Replace`, `Add`, `Remove`) and wire it into `Tenants`
- [x] 2.2 Add `SmartRestTemplates.BindExternalID`
- [x] 2.3 Add `TrustedCertificates.CreateBulk` (`/bulk`); `TrustedCertificates.VerifyChain` was added by add-certificate-token-login
- [x] 2.4 Offline tests (scripted httptest servers in pkg/c8y/api/coverage_ops_test.go, linter unit tests in tools/c8ygen) for every scenario in `specs/api-coverage/spec.md`

## 3. Remaining waivers

- [x] 3.1 Check how the realtime client reaches `/notification/realtime`; replace the TODO waiver with a reasoned waiver or implement it
- [x] 3.2 Remove the "Known coverage gaps" TODO block from the overlay once 2.3 and 3.1 are done
- [x] 3.3 Note in the users package godoc that `withCertificates` / `provisionedCertificates` are not in the public spec

## 3a. Found by method-level drift

- [x] 3a.1 `UnassignMany` on the three child services (DELETE with a body needs `SetMethodDeleteAllowPayload`)
- [x] 3a.2 `TOTP.GetActivity` and `TOTPSecret.IsActive()`
- [x] 3a.3 `Applications.DownloadBinary`
- [x] 3a.4 Fix `ApiApplicationByOwner`, which pointed at `applicationsByTenant`

## 4. Documentation

- [x] 4.1 Update the coverage numbers and phase status in `docs/API_GEN.md`
