## 1. Decision

- [x] 1.1 Decide: adopt before v2.0.0 (Go 1.27 minimum, remove duplicates); recorded in design.md (2026-11-19)
- [x] 1.2 Raise `go.mod` and all example modules to `go 1.27`; CI reads the Go version from `go.mod`, but golangci-lint had to move from v2.11 to v2.15 (Go 1.27 support arrived in v2.13.0)

## 2. Methods

- [x] 2.1 Add `Map`, `FlatMap`, `Decode`, `IterAs`, `Collect` to `op.Result[T]`
- [x] 2.2 Add `Map` and `As` to `pagination.Iterator[T]`
- [x] 2.3 Add `Decode`, `GetAs`, `Fragment` to `jsondoc.JSONDoc`
- [x] 2.4 Unit tests for each scenario in `specs/generic-helpers/spec.md`

## 3. Duplicates and rule enforcement

- [x] 3.1 Replace internal and example uses of the old free functions with the methods
- [x] 3.2 Remove the old free functions
- [x] 3.3 Add a CI check that exported service methods have no type parameters

## 4. Documentation

- [x] 4.1 Update `docs/API_DESIGN.md` and `docs/V2.md` examples to the method style
- [x] 4.2 Add an old function → new method table to the CHANGELOG
- [ ] 4.3 State the minimum Go version in the README (done as part of the prepare-v2-release README rewrite)
