## ADDED Requirements

### Requirement: Bulk size limit
`ManagedObjects.CreateBulk` and `ManagedObjects.UpdateBulk` SHALL reject a slice with more than 50 items before sending any request, with an error that states the limit.

#### Scenario: Too many items
- **WHEN** `CreateBulk` is called with 51 items
- **THEN** no request is sent and the result error says the limit is 50 items

#### Scenario: Empty slice
- **WHEN** `UpdateBulk` is called with 0 items
- **THEN** no request is sent and the result error says at least 1 item is required

### Requirement: External IDs on create
The SDK SHALL provide `managedobjects.ExternalID{Type, ExternalID string}` and `managedobjects.WithExternalIDs(body any, ids ...ExternalID)`, which returns a create body containing the `externalIds` field, for use with `Create` and `CreateBulk`. Each entry SHALL contain only `type` and `externalId`. (`model.Identity` is not used because it also serializes `self` and `managedObject`.)

#### Scenario: Create with an external ID
- **WHEN** `Create` is called with `WithExternalIDs(map[string]any{"name": "dev1"}, managedobjects.ExternalID{Type: "c8y_Serial", ExternalID: "SN1"})`
- **THEN** the request body contains `"externalIds":[{"type":"c8y_Serial","externalId":"SN1"}]` and no other fields in that entry

#### Scenario: Duplicate external ID in one request
- **WHEN** `CreateBulk` is called with two items that use the same external ID
- **THEN** no request is sent and the error names both positions

### Requirement: Recover a partial bulk create
`ManagedObjects.RecoverCreateBulk(ctx, items, err)` SHALL, for an error with `Applied == BatchAppliedPartial`: find created objects with `fragmentType=<Marker>`, create only the items whose positions were not found, remove the marker fragment from the objects that carry it, and return all objects in the original request order.

#### Scenario: Two of five items were created
- **WHEN** a 5-item bulk create fails with a marker and objects for positions 0 and 1 exist with that marker
- **THEN** `RecoverCreateBulk` creates only positions 2 to 4, removes the marker from the 2 objects that carry it, and returns 5 objects in request order

#### Scenario: Lookup fails
- **WHEN** the marker lookup returns `503` twice and then succeeds
- **THEN** `RecoverCreateBulk` retries the lookup and never resends the original bulk request

#### Scenario: Not a partial error
- **WHEN** `RecoverCreateBulk` is called with an error whose `Applied` is `BatchAppliedNone`
- **THEN** it returns an error without sending any request

### Requirement: Stream positions and recovery
`CreateBulkStream` and `UpdateBulkStream` SHALL set `BatchError.Offset` to the stream position of the failing request's first item. When `BulkStreamOptions.RecoverPartial` is true, `CreateBulkStream` SHALL call `RecoverCreateBulk` on a partial error and continue with the next batch.

#### Scenario: Failure in the third batch
- **WHEN** a stream of 120 items fails in the third request with `failedIndex: 4`
- **THEN** the yielded error has `Offset == 100` and `Index == 4`

#### Scenario: Recovery in a stream
- **WHEN** `RecoverPartial` is true and the second batch fails with a marker
- **THEN** the stream yields every object of the second batch once and continues with the third batch

### Requirement: Atomic get-or-create by external ID
`GetOrCreateByExternalID` SHALL create a missing object and its external ID in one request using `externalIds`. On `409`, it SHALL return the existing object bound to that external ID with `Status == op.StatusDuplicate` (the existing contract for a conflict).

#### Scenario: Object does not exist
- **WHEN** the external ID lookup returns `404`
- **THEN** the SDK sends one `POST /inventory/managedObjects` containing `externalIds` and no separate identity request

#### Scenario: Concurrent creation
- **WHEN** the create returns `409` because another client bound the external ID first
- **THEN** the SDK fetches and returns the existing object and deletes nothing

#### Scenario: Older platform without externalIds support
- **WHEN** the platform rejects the `externalIds` field
- **THEN** the SDK falls back to create, then assign identity, then delete on conflict
