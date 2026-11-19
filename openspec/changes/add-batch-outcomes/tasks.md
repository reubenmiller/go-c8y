## 1. Fake server support

- [ ] 1.1 Add bulk error responses to `internal/pkg/fakeserver`: validation (`422` with `failedIndex`), conflict before write (`409`), partial write (`409`/`500` with `fragmentType` and marker fragments on created objects)
- [ ] 1.2 Support `fragmentType` queries and bulk update setting a fragment to `null`

## 2. Core error and status

- [ ] 2.1 Add `bulkRequest` parsing to `core.APIError`
- [ ] 2.2 Add `core.BatchError`, `core.BatchApplied` and its constants; return `*BatchError` from error coupling when `bulkRequest` is present
- [ ] 2.3 Add `op.StatusPartial`; set it, with `Retryable = false`, when `Applied == BatchAppliedPartial`
- [ ] 2.4 Confirm bulk POSTs are never retried by the client's retry policy; add a test
- [ ] 2.5 Unit and offline tests for every scenario in `specs/batch-outcomes/spec.md`, including `Measurements.CreateList`

## 3. Recovery

- [ ] 3.1 Add `pipeline.BatchRecoverer[T]`
- [ ] 3.2 Implement `ManagedObjects.RecoverCreateBulk` (lookup with retry, resend missing positions, remove marker, return in request order)
- [ ] 3.3 Offline tests for the recovery scenarios

## 4. Managed object bulk API

- [ ] 4.1 Validate 1 to 50 items in `CreateBulk` and `UpdateBulk`
- [ ] 4.2 Add `managedobjects.ExternalID` and `WithExternalIDs`; reject duplicate external IDs within one request before sending
- [ ] 4.3 Set `BatchError.Offset` in `createBulkSeq` and `updateBulkSeq`
- [ ] 4.4 Add `BulkStreamOptions.RecoverPartial` and wire it to `RecoverCreateBulk`
- [ ] 4.5 Rewrite `GetOrCreateByExternalID` to use `externalIds` on create, with the fallback path
- [ ] 4.6 Offline tests for every scenario in `specs/managed-object-bulk/spec.md`

## 5. Documentation

- [ ] 5.1 Godoc for `BatchError`, `StatusPartial`, `RecoverCreateBulk`, and the stream options
- [ ] 5.2 Note in the `UpdateBulk` godoc that `externalIds` cannot be changed by bulk update (use the Identity API)
- [ ] 5.3 Add a bulk create and recovery example to `docs/IMPLEMENTATION_PATTERNS.md`
- [ ] 5.4 Point the `UpdateBulkStream` godoc to `pipeline.Batch` instead of `rill.Batch`
