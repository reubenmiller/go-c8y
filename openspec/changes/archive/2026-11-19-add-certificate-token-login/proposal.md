## Why

Cumulocity added `POST /tenant/oauth/certificate`, which exchanges an X.509 certificate for an access token over standard HTTPS. It works for regular users and microservice service users, not only devices. The SDK only supports the device flow (mTLS on port 8443, `POST /devicecontrol/deviceAccessToken`) and has no helpers to opt an account in to certificate authentication. The HTTPS exchange only works for RSA keys, because the platform returns the token encrypted to the certificate's public key (JWE, `RSA-OAEP-256` / `A256GCM`). EC certificates must keep using mTLS on port 8443; a platform bug that currently lets EC keys use HTTPS is being fixed. Review: docs/V2_STABILIZATION_REVIEW.md §3 and §4.7.

## What Changes

- **BREAKING** (behavior of an existing login method) `LoginMethodCertificate` chooses its transport from the private key type, through new `AuthOptions.CertificateTransport` (`Auto` default, `HTTPS`, `MTLS`): RSA keys use the HTTPS exchange, all other keys use mTLS on `MTLSPort`.
- New HTTPS token exchange with JWE decryption implemented with the standard library (no new dependency), including the tenant and the TFA code (`X-Cumulocity-TFA-Code`) from the existing `LoginOptions` TOTP/SMS callbacks.
- The cached token source refreshes certificate tokens through the same transport rule.
- Opt-in helpers: `CurrentUser.SetRequiredAuthType`, `CurrentApplication.SetAuthType`, `Applications.ResetAuthType`, and `jsonmodels.User.RequiredAuthType()`.
- Per-tenant certificate source for `ContextTokenExchange`, so microservice service users can use certificate tokens.

The login method's transport rule must be fixed before v2.0.0. The helpers are additive.

## Capabilities

### New Capabilities
- `certificate-authentication`: obtaining and refreshing access tokens with X.509 certificates (HTTPS exchange and mTLS), and opting accounts in to certificate-only authentication.

### Modified Capabilities
<!-- none: no existing specs -->

## Impact

- `pkg/c8y/api/authentication` (`AuthOptions`, `CertificateTransport`), `pkg/c8y/api/client.go` (login flow, `fetchToken`).
- `pkg/c8y/api/tenants/logintokens` (new exchange call).
- `pkg/c8y/api/users/currentuser`, `pkg/c8y/api/applications`, `pkg/c8y/api/applications/currentapplication`, `pkg/c8y/jsonmodels`.
- New OAS operations: `postCertificateAccessToken`, `putCurrentApplicationResourceAuthType`, `putApplicationResourceAuthType` (also listed in cover-latest-core-spec).
- Fake server: certificate exchange endpoint returning a JWE.
