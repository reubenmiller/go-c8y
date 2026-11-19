# batch-outcomes Specification

## Purpose
Defines how the SDK reports the outcome of any request that carries several items (core.BatchError, op.StatusPartial), and the shared shape for recovering a partially written batch.
## Requirements
### Requirement: Batch errors are typed
When an error response body contains a `bulkRequest` object, the SDK SHALL return a `*core.BatchError` that wraps the existing `*core.Error`. `BatchError.Index` SHALL equal `bulkRequest.failedIndex`, or -1 when `failedIndex` is absent. `BatchError.Marker` SHALL equal `bulkRequest.fragmentType`, or be empty when it is absent.

#### Scenario: Validation failure on bulk create
- **WHEN** a bulk create returns `422` with `{"error":"undefined/validationError","bulkRequest":{"failedIndex":2}}`
- **THEN** `errors.As(result.Err, &be)` succeeds for `be *core.BatchError`, with `be.Index == 2`, `be.Marker == ""`, and `be.Applied == core.BatchAppliedNone`

#### Scenario: Existing error checks still work
- **WHEN** a bulk request fails with a `bulkRequest` body
- **THEN** `errors.As(result.Err, &e)` for `e *core.Error` also succeeds and reports the HTTP status

#### Scenario: Error for the whole request
- **WHEN** a bulk request with 51 items is rejected with a `422` body that has no `bulkRequest`
- **THEN** the error is a plain `*core.Error`, not a `*core.BatchError`

#### Scenario: Measurements use the same type
- **WHEN** `Measurements.CreateList` receives `422` with `bulkRequest.failedIndex: 0`
- **THEN** the returned error is a `*core.BatchError` with `Index == 0`

### Requirement: Applied state is derived from the response
The SDK SHALL set `BatchError.Applied` to `BatchAppliedPartial` when `fragmentType` is present; to `BatchAppliedNone` for `404`, `409`, or `422` without `fragmentType`; and to `BatchAppliedUnknown` otherwise.

#### Scenario: Partial create after a concurrent conflict
- **WHEN** a bulk create returns `409` with `bulkRequest: {failedIndex: 3, fragmentType: "c8y_BatchId_abc"}`
- **THEN** `Applied == core.BatchAppliedPartial` and `Marker == "c8y_BatchId_abc"`

#### Scenario: Datastore failure without marker
- **WHEN** a bulk update returns `500` with `bulkRequest: {failedIndex: 17}`
- **THEN** `Applied == core.BatchAppliedUnknown`

### Requirement: Partial outcome status
When `Applied` is `BatchAppliedPartial`, the result SHALL have `Status == op.StatusPartial`, `IsError() == true`, and `Retryable == false`.

#### Scenario: Partial result is not retryable
- **WHEN** a bulk create returns a partial-write error
- **THEN** `result.Status == op.StatusPartial` and `result.Retryable == false`

### Requirement: Batch requests are never retried automatically
The SDK SHALL NOT automatically resend a bulk create request after a response with a `bulkRequest` error.

#### Scenario: No automatic resend
- **WHEN** a bulk create returns `500` with a `fragmentType`
- **THEN** the fake server receives exactly one `POST /inventory/managedObjects` request

### Requirement: Shared recovery function shape
Package `pipeline` SHALL define `BatchRecoverer[T any]` as `func(ctx context.Context, items []any, err *core.BatchError) ([]T, error)`. Every service method that recovers a partial batch SHALL match this shape.

#### Scenario: Recoverer can be passed to a stream
- **WHEN** `ManagedObjects.RecoverCreateBulk` is assigned to a variable of type `pipeline.BatchRecoverer[jsonmodels.ManagedObject]`
- **THEN** the code compiles

