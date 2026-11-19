## 1. Documentation

- [ ] 1.1 Rewrite `README.md` for v2 (install, quick start, minimum Go version, stability promise, links to design docs)
- [ ] 1.2 Fix `/v2` import paths in `docs/V2.md` and `docs/DEFERRED_EXECUTION.md`
- [ ] 1.3 Document the difference between `applications.List` filters and the `ListBy*` methods
- [ ] 1.4 Document that `Tenant` on tenant-scoped options defaults to the client tenant and can be overridden with `api.WithTenant`

## 2. Exported surface

- [ ] 2.1 List exported identifiers in `pkg/c8y/*.go` and `api/client_compat.go`; check go-c8y-cli usage for each
- [ ] 2.2 Document used identifiers as supported; move unused ones to `internal`
- [ ] 2.3 Review `go doc -all ./pkg/...` once more for anything exported by accident

## 3. CI

- [ ] 3.1 Add a CI job that fails on v1 import paths in `README.md` and `docs/`
- [ ] 3.2 Add a CI job that extracts and builds `package main` code blocks from docs
- [ ] 3.3 Add a `gorelease` job comparing PRs to the latest v2 tag (non-blocking until `v2.0.0` exists, then blocking)

## 4. Release

- [ ] 4.1 Confirm the must-land changes are merged and the adopt-generic-methods decision is recorded
- [ ] 4.2 Vendor the core spec again; `c8ygen lint --strict` passes
- [ ] 4.3 Group the CHANGELOG by area with migration tables for every breaking change
- [ ] 4.4 Tag `v2.0.0`
