## Context

Verified by compiling test programs (Go 1.26.3 locally, Go 1.27.0 through toolchain download):

- Go 1.27 compiles methods with type parameters on generic and non-generic types; Go 1.26 rejects them ("method must have no type parameters").
- Generic methods are promoted through embedding, so a method on `jsondoc.JSONDoc` is callable on every `jsonmodels` type. Method values work.
- Interfaces cannot declare generic methods, and a generic method does not satisfy a non-generic interface method of the same name.
- A method cannot add a constraint to the receiver's type parameter. `Result[T any]` cannot require `T` to implement `jsondoc.Unwrapper` in a method; the method can constrain only its own new type parameters.

## Goals / Non-Goals

**Goals:**
- One discoverable, chainable way to transform and decode results.
- Keep services mockable through user-defined interfaces.

**Non-Goals:**
- Generic service methods such as `Service.GetAs[T]()`.
- Changing `pipeline` step combinators (`Pipe`, `Retry`, …); they compose functions and stay functions.
- Changing constructors (`op.OK[T]`, `pagination.Paginate[T, D]`) or multi-value helpers (`op.Combine`).

## Decisions

**Generic methods only on value types.** `op.Result`, `pagination.Iterator`, and `jsondoc.JSONDoc`. Service methods stay non-generic because interfaces cannot include generic methods, so a generic service method could never be mocked through an interface.

**Runtime check where the receiver's type cannot be constrained.** `Result[T].IterAs[U]()` and `Result[T].Decode[U]()` need `T` to expose raw JSON. They check `any(r.Data).(jsondoc.Unwrapper)` at run time and return an error result if it fails. Alternative: keep these as constrained free functions. Rejected for consistency; the failure is easy to detect in tests and documented in godoc.

**Structural constraint for fragments.** `JSONDoc.Fragment[U interface{ FragmentKey() string }]` instead of `U model.Fragment`, so `jsondoc` (which today depends only on `types`) does not start importing `model`, `authentication`, and `certutil`.

**Remove the duplicates.** Adopted before v2.0.0 (decision recorded below), so v2.0.0 ships only the method forms and the free functions are removed.

**Go version policy.** Go supports the two most recent major releases. Go 1.26 stays supported until Go 1.28 is released, so requiring 1.27 excludes users still on 1.26 until then.

## Risks / Trade-offs

- [Users on Go 1.26 cannot use go-c8y v2] → Accepted. State Go 1.27 as the minimum in the README and the v2.0.0 release notes. Go 1.26 leaves support when Go 1.28 is released.
- [Runtime decode errors instead of compile errors] → Typical `Result[T]` values in the SDK all hold `jsonmodels` types, which implement `Unwrapper`; test each service's result type once.
- [go-c8y-cli migration] → Mechanical replacement; provide a table of old function → new method.

## Adoption decision

**2026-11-19: adopt before v2.0.0.** The minimum Go version becomes 1.27, and the duplicate free functions are removed, so v2.0.0 has one way of doing each thing. Rejected alternative: tag v2.0.0 on Go 1.26 and add the methods in v2.x, which would keep both the functions and the methods for the whole major version.

## Open Questions

- None. The adoption decision is recorded above.
