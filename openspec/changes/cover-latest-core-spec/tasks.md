## 1. Spec and drift tooling

- [ ] 1.1 Run `task fetch-spec` and `task generate`; commit the new spec and generated files
- [ ] 1.2 Make `c8ygen lint` infer the HTTP method for each path literal from its request builder
- [ ] 1.3 Support method-prefixed waivers; require a reason comment on every waiver
- [ ] 1.4 Add a method-level waiver for `POST /tenant/trusted-certificates/settings/crl` (covered by `PUT` file upload)
- [ ] 1.5 Unit tests for method inference and waiver matching in `tools/c8ygen`
- [ ] 1.6 Run the CI api-drift job non-strict for one release, then restore `--strict`

## 2. New operations

- [ ] 2.1 Add `pkg/c8y/api/tenants/restrictedroles` (`List`, `Replace`, `Add`, `Remove`) and wire it into `Tenants`
- [ ] 2.2 Add `SmartRestTemplates.BindExternalID`
- [ ] 2.3 Add `TrustedCertificates.CreateBulk` (`/bulk`) and `TrustedCertificates.VerifyChain`
- [ ] 2.4 Fake server handlers and offline tests for every scenario in `specs/api-coverage/spec.md`

## 3. Remaining waivers

- [ ] 3.1 Check how the realtime client reaches `/notification/realtime`; replace the TODO waiver with a reasoned waiver or implement it
- [ ] 3.2 Remove the "Known coverage gaps" TODO block from the overlay once 2.3 and 3.1 are done
- [ ] 3.3 Note in the users package godoc that `withCertificates` / `provisionedCertificates` are not in the public spec

## 4. Documentation

- [ ] 4.1 Update the coverage numbers and phase status in `docs/API_GEN.md`
