## Why

Cumulocity's bulk endpoints report which item failed (`bulkRequest.failedIndex`) and, when part of a bulk create was written, how to find it (`bulkRequest.fragmentType`, a `c8y_BatchId_*` fragment). The SDK drops both fields, so a caller cannot tell "nothing was created" from "some objects were created", and cannot recover without creating duplicates. The same `bulkRequestError` schema is used by `POST /inventory/managedObjects`, `PUT /inventory/managedObjects`, and `POST /measurement/measurements`, and other collection endpoints are likely to adopt it. Review: docs/V2_STABILIZATION_REVIEW.md §2.

## What Changes

- **BREAKING** (error contract) New `core.BatchError`, produced for any error response that contains `bulkRequest`, regardless of endpoint. It wraps `*core.Error` and adds `Index`, `Offset`, `Applied` (`None` / `Partial` / `Unknown`), and `Marker`.
- **BREAKING** (outcome contract) New `op.StatusPartial`, set when part of a batch request was written.
- New shared recovery shape `pipeline.BatchRecoverer[T]`, with a first implementation `ManagedObjects.RecoverCreateBulk` that follows the platform's documented steps.
- `CreateBulkStream` / `UpdateBulkStream` report the position in the whole input stream (`Offset + Index`) and can recover partial creates with `BulkStreamOptions.RecoverPartial`.
- `CreateBulk` / `UpdateBulk` reject more than 50 items before sending.
- Typed `externalIds` on create (`managedobjects.WithExternalIDs`), and an atomic `GetOrCreateByExternalID` that creates the object and its external ID in one request.

The error type and status must land before v2.0.0. The recovery helpers and typed options are additive.

## Capabilities

### New Capabilities
- `batch-outcomes`: how the SDK reports and recovers the outcome of any request that carries several items.
- `managed-object-bulk`: bulk create and update of managed objects, including external IDs, size limits, streaming, and get-or-create by external ID.

### Modified Capabilities
<!-- none: no existing specs -->

## Impact

- `pkg/c8y/api/core` (error parsing), `pkg/c8y/op` (status), `pkg/c8y/api/pipeline` (recoverer type).
- `pkg/c8y/api/inventory/managedobjects` (bulk methods, streams, get-or-create).
- `pkg/c8y/api/measurements` (`CreateList` gets `BatchError` automatically).
- Fake server in `internal/pkg/fakeserver` needs bulk error responses for offline tests.
