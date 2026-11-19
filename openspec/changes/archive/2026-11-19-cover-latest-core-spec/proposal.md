## Why

The vendored spec (`docs/c8y-oas.yml`, June 2026) is behind the published core OpenAPI spec. The latest spec has 9 new operations, 8 of them missing from the SDK, plus 3 older gaps still waived in the overlay. The drift check (`c8ygen lint`) compares paths only, so a new method on an existing path (for example `PUT /inventory/managedObjects`) is invisible to it. Review: docs/V2_STABILIZATION_REVIEW.md §1.

## What Changes

- Vendor the latest spec (`task fetch-spec`) and regenerate `pkg/c8y/api/spec`.
- Add the missing operations:
  - Restricted roles: `Tenants.RestrictedRoles` with `List`, `Replace`, `Add`, `Remove` (`/tenant/tenants/{tenantId}/applications/restricted-roles[/{roleId}]`).
  - SmartREST template external ID: `SmartRestTemplates.BindExternalID` (`POST /inventory/templates/globalId/{globalId}/externalId`).
  - Trusted certificates bulk upload: `TrustedCertificates.CreateBulk` (`POST …/trusted-certificates/bulk`).
  - Certificate chain check: `TrustedCertificates.VerifyChain` (`POST /tenant/trusted-certificates/verify-cert-chain`).
  - The three certificate-auth operations are specified in add-certificate-token-login, not here.
- Resolve the `/notification/realtime` waiver: confirm the realtime client covers it, and record the reason in the overlay, or implement it.
- Make `c8ygen lint` compare method + path, with method-level waivers.

Additive. No exported API changes.

## Capabilities

### New Capabilities
- `api-coverage`: which core OpenAPI operations the SDK exposes, and how drift between the spec and the SDK is detected and waived.

### Modified Capabilities
<!-- none: no existing specs -->

## Impact

- `docs/c8y-oas.yml`, `docs/c8y-oas.overlay.yml`, `pkg/c8y/api/spec` (generated).
- `tools/c8ygen` (lint), CI api-drift job.
- `pkg/c8y/api/tenants` (new `restrictedroles` sub-package), `pkg/c8y/api/smartresttemplates`, `pkg/c8y/api/trustedcertificates`.
