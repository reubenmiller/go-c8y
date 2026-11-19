## ADDED Requirements

### Requirement: Result transformation methods
`op.Result[T]` SHALL provide `Map[R any](fn func(T) R) Result[R]` and `FlatMap[R any](fn func(T) Result[R]) Result[R]`. Both SHALL keep the original error, status, and HTTP metadata when the result is an error, without calling `fn`.

#### Scenario: Map on success
- **WHEN** `r` is a successful `Result[jsonmodels.ManagedObject]` and `r.Map(func(mo jsonmodels.ManagedObject) string { return mo.Name() })` is called
- **THEN** the returned `Result[string]` holds the name and the original HTTP status

#### Scenario: Map on error
- **WHEN** `r` is a failed result
- **THEN** `r.Map(fn)` does not call `fn` and returns a result with the same error

### Requirement: Typed decoding
`op.Result[T]` SHALL provide `Decode[U any]() (U, error)` and `IterAs[U any]() iter.Seq2[U, error]`. `jsondoc.JSONDoc` SHALL provide `Decode[U any]() (U, error)`, `GetAs[U any](key string) (U, error)`, and `Fragment[U interface{ FragmentKey() string }]() (U, error)`, available on every `jsonmodels` type through embedding. The constraint is written structurally so `jsondoc` does not import `model`; every `model.Fragment` satisfies it.

#### Scenario: Decode a managed object into a user struct
- **WHEN** `client.ManagedObjects.Get(ctx, id, opts).Decode[MyDevice]()` is called
- **THEN** it returns a `MyDevice` filled from the response JSON

#### Scenario: Iterate a collection as user structs
- **WHEN** `r.IterAs[MyDevice]()` is ranged over for a list result with 3 items
- **THEN** it yields 3 `MyDevice` values and nil errors

#### Scenario: Field access on a model
- **WHEN** `mo.GetAs[int]("c8y_Battery.level")` is called on a managed object whose `c8y_Battery.level` is `87`
- **THEN** it returns `87` and a nil error

#### Scenario: Result without raw JSON
- **WHEN** `Decode[MyDevice]()` is called on a `Result[int64]`
- **THEN** it returns an error saying the result data has no JSON document

### Requirement: Iterator transformation methods
`pagination.Iterator[T]` SHALL provide `Map[R any](fn func(T) R) iter.Seq2[R, error]` and `As[U any]() iter.Seq2[U, error]`, which preserve pagination and stop at the first page error.

#### Scenario: Map across pages
- **WHEN** `client.ManagedObjects.ListAll(ctx, opts).Map(nameOf)` is ranged over 2 pages of 5 items
- **THEN** it yields 10 names

### Requirement: No duplicate free functions
The free functions replaced by methods (`op.MapResult`, `op.FlatMap`, `op.IterAs`, `op.IterAsErr`, `op.Iter`, `op.Iter2`, `op.ToSliceR`, `op.CollectR`, `jsondoc.Decode`, `jsonmodels.GetAs`, `jsonmodels.GetFragment`) SHALL be removed before v2.0.0.

#### Scenario: Removed function
- **WHEN** code calls `op.MapResult(r, fn)` against the v2.0.0 release
- **THEN** it does not compile, and the CHANGELOG names `r.Map(fn)` as the replacement

### Requirement: Service methods stay non-generic
Exported methods on service types under `pkg/c8y/api/*` SHALL NOT have type parameters.

#### Scenario: Services can be mocked
- **WHEN** a user defines `type DeviceGetter interface { Get(ctx context.Context, id string, opts managedobjects.GetOptions) op.Result[jsonmodels.ManagedObject] }`
- **THEN** `client.ManagedObjects` satisfies it

#### Scenario: Static check
- **WHEN** the CI API check scans exported methods of service types
- **THEN** it finds no method with type parameters

### Requirement: Minimum Go version
`go.mod` SHALL declare `go 1.27` or later, and the README SHALL state the minimum Go version.

#### Scenario: Older toolchain
- **WHEN** a user builds the module with Go 1.26 and `GOTOOLCHAIN=local`
- **THEN** the build fails with Go's standard "go.mod requires go >= 1.27" error
