## ADDED Requirements

### Requirement: Documentation uses the v2 module path
The README and every file under `docs/` SHALL import packages with the module path `github.com/reubenmiller/go-c8y/v2`, and SHALL NOT reference the v1 import path `github.com/reubenmiller/go-c8y/pkg`.

#### Scenario: No v1 import paths
- **WHEN** CI searches `README.md` and `docs/` for `"github.com/reubenmiller/go-c8y/pkg`
- **THEN** it finds no matches

### Requirement: Documentation examples compile
Every Go code block in `README.md` and `docs/*.md` that starts with `package main` SHALL compile against the current module.

#### Scenario: README quick start
- **WHEN** CI extracts the README quick-start block and runs `go build` on it
- **THEN** the build succeeds

### Requirement: README describes v2
The README SHALL show the v2 install command, a quick start using `api.NewClient` or `api.NewClientFromEnvironment`, the minimum Go version (1.27), and the v2 API stability promise.

#### Scenario: Stability statement
- **WHEN** a user reads the README
- **THEN** it states that v2 follows semantic versioning and does not say there are no API stability guarantees

### Requirement: Supported exported packages are explicit
Every exported identifier in `pkg/c8y/*.go` and `pkg/c8y/api/client_compat.go` SHALL either be documented as supported v2 API or be moved to an internal package before v2.0.0.

#### Scenario: Compat API decision recorded
- **WHEN** a user reads the godoc for `api.Client.SendRequest`
- **THEN** it states whether the function is supported for v2 or what to use instead

### Requirement: Breaking changes are detected after v2.0.0
CI SHALL run an API compatibility check on every PR to `v2` against the latest `v2.x.y` tag and fail when the PR introduces an incompatible change.

#### Scenario: Removed exported function
- **WHEN** a PR removes an exported function that exists in the latest v2 tag
- **THEN** the compatibility job fails and names the removed function

#### Scenario: Additive change
- **WHEN** a PR only adds new exported functions
- **THEN** the compatibility job passes

### Requirement: Release checklist
`v2.0.0` SHALL be tagged only after: the changes decouple-http-client-contracts, consolidate-context-helpers, add-batch-outcomes, add-certificate-token-login, validate-microservice-callers, and adopt-generic-methods are merged; the core spec has been vendored within 30 days of tagging with `c8ygen lint --strict` passing; and the CHANGELOG lists every breaking change with its migration.

#### Scenario: Checklist incomplete
- **WHEN** add-batch-outcomes is not merged
- **THEN** the release is not tagged
