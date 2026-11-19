## 1. Decision

- [ ] 1.1 Decide: adopt before v2.0.0 (Go 1.27 minimum, remove duplicates) or in v2.x (keep functions); record the outcome in design.md
- [ ] 1.2 If adopting before v2.0.0: raise `go.mod` and all example modules to `go 1.27`; update the CI Go version matrix

## 2. Methods

- [ ] 2.1 Add `Map`, `FlatMap`, `Decode`, `IterAs`, `Collect` to `op.Result[T]`
- [ ] 2.2 Add `Map` and `As` to `pagination.Iterator[T]`
- [ ] 2.3 Add `Decode`, `GetAs`, `Fragment` to `jsondoc.JSONDoc`
- [ ] 2.4 Unit tests for each scenario in `specs/generic-helpers/spec.md`

## 3. Duplicates and rule enforcement

- [ ] 3.1 Replace internal and example uses of the old free functions with the methods
- [ ] 3.2 Remove the old free functions (only if adopting before v2.0.0)
- [ ] 3.3 Add a CI check that exported service methods have no type parameters

## 4. Documentation

- [ ] 4.1 Update `docs/API_DESIGN.md` and `docs/V2.md` examples to the method style
- [ ] 4.2 Add an old function → new method table to the CHANGELOG
- [ ] 4.3 State the minimum Go version in the README
