## Why

Go 1.27 allows methods with their own type parameters. Many v2 helpers are free functions only because methods could not take type parameters before: `op.MapResult(r, fn)`, `op.IterAs[U](r)`, `jsondoc.Decode[T](d)`, `jsonmodels.GetAs[T](mo, key)`. As methods they are discoverable with autocomplete and can be chained (`client.ManagedObjects.Get(ctx, id, opts).Decode[MyDevice]()`). Adopting them before v2.0.0 lets the SDK ship one way of doing each thing; after v2.0.0, both the functions and the methods would have to stay for the whole major version. Review: docs/V2_STABILIZATION_REVIEW.md §6.

## What Changes

- Add generic convenience methods on value types:
  - `op.Result[T]`: `Map`, `FlatMap`, `IterAs`, `Decode`, `Collect`.
  - `pagination.Iterator[T]`: `Map`, `As`.
  - `jsondoc.JSONDoc` (and so every `jsonmodels` type through embedding): `Decode`, `GetAs`, `Fragment`.
- **BREAKING** Remove the free functions these replace: `op.MapResult`, `op.FlatMap`, `op.IterAs`, `op.IterAsErr`, `op.Iter`, `op.Iter2`, `op.ToSliceR`, `op.CollectR`, `jsondoc.Decode`, `jsonmodels.GetAs`, `jsonmodels.GetFragment`.
- **BREAKING** (for users) Raise the minimum Go version in `go.mod` to 1.27.
- Rule: service methods (`Get`, `List`, `Create`, …) stay non-generic, so services can still be wrapped in user-defined interfaces for mocking.

**Decision (2026-11-19):** adopt before v2.0.0. v2.0.0 requires Go 1.27 and ships only the method forms; the free functions they replace are removed. Must land before v2.0.0.

## Capabilities

### New Capabilities
- `generic-helpers`: the generic convenience API on results, iterators, and JSON documents, and the rule that service methods stay non-generic.

### Modified Capabilities
<!-- none: no existing specs -->

## Impact

- `go.mod` (`go 1.27`; CI picks it up through `go-version-file`), example modules.
- `pkg/c8y/op`, `pkg/c8y/api/pagination`, `pkg/c8y/jsondoc`, `pkg/c8y/jsonmodels`.
- Callers of the removed free functions inside the SDK, examples, docs, and go-c8y-cli.
