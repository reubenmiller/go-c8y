## 1. Fake server support

- [x] 1.1 Fake server: bulk validation errors (`422`/`404` with `failedIndex`, `409 identity/Duplicate`) and `externalIds` binding on create. Partial writes are tested with scripted `httptest` servers in `pkg/c8y/api/batch_outcomes_test.go`
- [x] 1.2 Marker lookup (`fragmentType`) and marker removal (`null` via bulk update) are covered by the scripted `httptest` servers

## 2. Core error and status

- [x] 2.1 Add `bulkRequest` parsing to `core.APIError`
- [x] 2.2 Add `core.BatchError`, `core.BatchApplied` and its constants; return `*BatchError` from error coupling when `bulkRequest` is present
- [x] 2.3 Add `op.StatusPartial`; set it, with `Retryable = false`, when `Applied == BatchAppliedPartial`
- [x] 2.4 Confirm bulk POSTs are never retried by the client's retry policy; add a test
- [x] 2.5 Unit and offline tests for every scenario in `specs/batch-outcomes/spec.md`, including `Measurements.CreateList`

## 3. Recovery

- [x] 3.1 Add `pipeline.BatchRecoverer[T]`
- [x] 3.2 Implement `ManagedObjects.RecoverCreateBulk` (lookup with retry, resend missing positions, remove marker, return in request order)
- [x] 3.3 Offline tests for the recovery scenarios

## 4. Managed object bulk API

- [x] 4.1 Validate 1 to 50 items in `CreateBulk` and `UpdateBulk`
- [x] 4.2 Add `managedobjects.ExternalID` and `WithExternalIDs`; reject duplicate external IDs within one request before sending
- [x] 4.3 Set `BatchError.Offset` in `createBulkSeq` and `updateBulkSeq`
- [x] 4.4 Add `BulkStreamOptions.RecoverPartial` and wire it to `RecoverCreateBulk`
- [x] 4.5 Rewrite `GetOrCreateByExternalID` to use `externalIds` on create, with the fallback path
- [x] 4.6 Offline tests for every scenario in `specs/managed-object-bulk/spec.md`

## 5. Documentation

- [x] 5.1 Godoc for `BatchError`, `StatusPartial`, `RecoverCreateBulk`, and the stream options
- [x] 5.2 Note in the `UpdateBulk` godoc that `externalIds` cannot be changed by bulk update (use the Identity API)
- [x] 5.3 Add a bulk create and recovery example to `docs/IMPLEMENTATION_PATTERNS.md`
- [x] 5.4 Point the `UpdateBulkStream` godoc to `pipeline.Batch` instead of `rill.Batch`
