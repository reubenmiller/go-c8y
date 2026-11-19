## ADDED Requirements

### Requirement: Every spec operation is implemented or waived
For the vendored core spec, every operation (method + path) SHALL either be implemented by an SDK service method or be listed in the overlay's drift waivers with a reason comment.

#### Scenario: Strict drift check passes
- **WHEN** `go run ./tools/c8ygen lint --strict` runs against the vendored spec
- **THEN** it exits with status 0

#### Scenario: Waiver without reason
- **WHEN** a waiver entry in `docs/c8y-oas.overlay.yml` has no `#` reason comment
- **THEN** `c8ygen lint` reports it as an error

### Requirement: Method-aware drift detection
`c8ygen lint` SHALL compare operations as method + normalized path. A waiver MAY name a method; a waiver without a method SHALL match all methods on that path.

#### Scenario: New method on an existing path
- **WHEN** the spec adds `PUT /inventory/managedObjects` and the SDK only implements `GET` and `POST` on that path
- **THEN** `c8ygen lint` reports `PUT /inventory/managedObjects` as missing

#### Scenario: Method-specific waiver
- **WHEN** the overlay waives `POST /tenant/trusted-certificates/settings/crl`
- **THEN** `c8ygen lint` does not report that operation, but still reports a missing `GET` on the same path

### Requirement: Restricted roles
The SDK SHALL provide `Tenants.RestrictedRoles` with `List(ctx, opts)`, `Replace(ctx, opts, roles []string)`, `Add(ctx, opts, role string)`, and `Remove(ctx, opts, role string)`, where the tenant in `opts` defaults to the client tenant.

#### Scenario: List
- **WHEN** `Tenants.RestrictedRoles.List(ctx, restrictedroles.ListOptions{})` is called on a client for tenant `t1`
- **THEN** the SDK sends `GET /tenant/tenants/t1/applications/restricted-roles` and returns the role names

#### Scenario: Replace
- **WHEN** `Replace` is called with `["ROLE_TENANT_ADMIN"]`
- **THEN** the SDK sends `PUT` with the JSON body `["ROLE_TENANT_ADMIN"]`

#### Scenario: Add and remove
- **WHEN** `Add` and then `Remove` are called with `ROLE_TENANT_ADMIN`
- **THEN** the SDK sends `POST` and then `DELETE` to `…/restricted-roles/ROLE_TENANT_ADMIN`

### Requirement: SmartREST template external ID
The SDK SHALL provide `SmartRestTemplates.BindExternalID(ctx, globalID, xID string)`, which sends `POST /inventory/templates/globalId/{globalId}/externalId` with `type` `c8y_SmartRestDeviceIdentifier`.

#### Scenario: Bind an X-Id
- **WHEN** `BindExternalID(ctx, "123", "myTemplateXid")` is called
- **THEN** the request body is `{"externalId":"myTemplateXid","type":"c8y_SmartRestDeviceIdentifier"}`

### Requirement: Trusted certificate bulk upload and chain check
The SDK SHALL provide `TrustedCertificates.CreateBulk` for `POST /tenant/tenants/{tenantId}/trusted-certificates/bulk` and `TrustedCertificates.VerifyChain` for `POST /tenant/trusted-certificates/verify-cert-chain`.

#### Scenario: Bulk upload path
- **WHEN** `CreateBulk` is called for the client tenant `t1`
- **THEN** the SDK sends `POST /tenant/tenants/t1/trusted-certificates/bulk`

#### Scenario: Chain check
- **WHEN** `VerifyChain` is called with a certificate chain
- **THEN** the SDK sends `POST /tenant/trusted-certificates/verify-cert-chain` and returns the platform's validation result
