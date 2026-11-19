## Context

- `README.md` is the v1 README. `CHANGELOG.md` has an "Unreleased" section with v2 entries.
- `pkg/c8y` (root package) holds 6 helper files kept from v1. `api/client_compat.go` provides `SendRequest` / `RequestResult` as a migration path for go-c8y-cli.
- CI (`.github/workflows/v2.yml`) runs lint, api-drift, offline tests on 3 operating systems, and live tests. Nothing checks API compatibility between releases.

## Goals / Non-Goals

**Goals:**
- A v2 README and docs that compile when copied.
- A clear list of supported exported packages.
- Automatic detection of breaking API changes after v2.0.0.

**Non-Goals:**
- Changes covered by the other stabilization changes (resty contracts, context helpers, batch outcomes, certificate login, generic methods).

## Decisions

**Compatibility check with `golang.org/x/exp/cmd/gorelease`.** It compares the module against the latest tag and reports incompatible changes, suggesting the right next version. Alternative: `apidiff` on each package. `gorelease` covers the whole module in one run. Run it as a CI job on PRs to `v2`, failing on incompatible changes once `v2.0.0` exists.

**Docs examples must compile.** Code blocks in `README.md` and `docs/*.md` that start with `package main` are extracted and built in CI. This catches stale import paths and renamed APIs.

**v1 helpers: keep only what go-c8y-cli needs.** For each exported identifier in `pkg/c8y/*.go` and `client_compat.go`, check go-c8y-cli usage. Used ones are documented as supported (with godoc stating the stability promise); unused ones move to `internal`.

## Risks / Trade-offs

- [Extracting code blocks is brittle] → Only blocks starting with `package main` are built; others are illustrative.
- [go-c8y-cli still needs compat APIs that block cleanup] → Keep them as supported, documented API; revisit in v3.

## Open Questions

- Should v2.0.0 wait for go-c8y-cli to migrate to v2, as the main real-world validation of the API?
