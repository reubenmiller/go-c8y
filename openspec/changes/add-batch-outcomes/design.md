## Context

The platform's bulk create of managed objects (max 50 items):
- validates all items and external IDs first; a failed check creates nothing (`422`, `409` with only `failedIndex`);
- if a write fails part-way (`409` for a concurrently bound external ID, `500`), some objects exist and the error includes `fragmentType`; every created object carries `"<fragmentType>": {"index": n}`;
- to complete without duplicates: query `GET /inventory/managedObjects?fragmentType=<fragmentType>&pageSize=50` (retry the lookup, never the bulk request), resend only the missing positions, then remove the marker fragment (bulk update setting it to `null`).

Bulk update validates first and reports `failedIndex`; a datastore failure may leave a partial update with no marker.

Today `core.APIError` has only `error`, `message`, `info`, and `CreateBulkStream` stops at the first error with a batch-relative index.

## Goals / Non-Goals

**Goals:**
- One error type and one status for every batch endpoint, parsed in `core`.
- Safe recovery of partial bulk creates without duplicates.
- No change to existing method signatures.

**Non-Goals:**
- Automatic recovery inside `CreateBulk` (single call). Recovery is explicit, or opt-in on streams.
- Recovery for measurements (the platform gives no marker for them).
- Typed slices for `CreateBulk` / `UpdateBulk` (review §5.7). A generic service method would break the rule that service methods stay non-generic (see adopt-generic-methods). `[]any` stays; callers convert typed slices with a loop or `pipeline.FromSlice`.

## Decisions

**Parse `bulkRequest` generically in `core`.** `APIError` gets a `BulkRequest *bulkRequestInfo` field. When present, `core` returns `*BatchError` (which wraps `*Error`, so existing `errors.As(err, &*core.Error)` checks keep working). Alternative: a managed-object-specific error. Rejected because measurements already use the schema.

**`Applied` is derived, not guessed by callers.**
- `fragmentType` present → `BatchAppliedPartial`.
- `404`, `409`, or `422` without `fragmentType` → `BatchAppliedNone` (validated before writing).
- anything else (for example `500` without `fragmentType`, or bulk update `500`) → `BatchAppliedUnknown`.

**`Index` stays request-relative; streams add `Offset`.** This keeps `Index` identical to the platform's `failedIndex`. Streams set `Offset` to the position of the request's first item, so `Offset + Index` is the stream position.

**New status `op.StatusPartial`.** Fits the existing outcome model (`StatusDuplicate`, `StatusSkipped`, …), so `switch result.Status` handles it. `IsError()` stays true and `Retryable` is false: the platform says never to retry a partial bulk request.

**Recovery as a function shape, not an interface.** `type BatchRecoverer[T any] func(ctx context.Context, items []any, err *core.BatchError) ([]T, error)`. Services expose concrete methods (`RecoverCreateBulk`); streams accept the function. Alternative: an interface with a generic method. Not possible: interfaces cannot have type-parameterized methods.

**Atomic get-or-create.** Use `externalIds` on create: look up → create with the external ID → on `409`, fetch the existing object. Keep the current create-then-assign path as a fallback when the platform rejects the `externalIds` field (older platforms).

## Risks / Trade-offs

- [Platform changes the error format] → Parsing is tolerant: unknown or missing fields give `BatchAppliedUnknown`, never a parse failure.
- [Recovery lookup is eventually consistent] → Follow the platform's guidance: retry the lookup with backoff; only resend positions that are still missing after the lookup succeeds.
- [Removing the marker fails] → Report it in the result's metadata and leave the marker; it is harmless and documented as ignorable by the platform.

## Open Questions

- How can the SDK detect "platform does not support `externalIds` on create" reliably (specific error type or status)? Needs a check against an older tenant.
