# certificate-authentication Specification

## Purpose
Defines how the SDK obtains and refreshes access tokens with X.509 certificates (HTTPS exchange for RSA keys, mTLS for other keys), and how accounts opt in to certificate-only authentication.
## Requirements
### Requirement: Transport chosen by key type
`AuthOptions.CertificateTransport` SHALL accept `CertificateTransportAuto` (default, empty string), `CertificateTransportHTTPS`, and `CertificateTransportMTLS`. With `Auto`, `LoginMethodCertificate` SHALL use the HTTPS exchange (`POST /tenant/oauth/certificate`) for RSA keys and mTLS (`POST /devicecontrol/deviceAccessToken` on `MTLSPort`) for all other key types. `Auto` SHALL NOT fall back from one transport to the other.

#### Scenario: RSA key uses HTTPS
- **WHEN** a client logs in with `LoginMethodCertificate` and an RSA certificate and key
- **THEN** the SDK sends `POST /tenant/oauth/certificate` on the standard HTTPS port and no mTLS connection is opened

#### Scenario: EC key uses mTLS
- **WHEN** a client logs in with `LoginMethodCertificate` and an ECDSA certificate and key
- **THEN** the SDK sends `POST /devicecontrol/deviceAccessToken` over mTLS on `MTLSPort` and does not call `/tenant/oauth/certificate`

#### Scenario: No fallback on failure
- **WHEN** the mTLS request for an EC key fails because port 8443 is unreachable
- **THEN** the login fails with an error that mentions the port and `MTLSPort`, and no HTTPS exchange is attempted

#### Scenario: Forced HTTPS with an EC key
- **WHEN** `CertificateTransport` is `HTTPS` and the key is ECDSA
- **THEN** the login fails before any request with an error saying the HTTPS exchange needs an RSA key

### Requirement: HTTPS exchange request
The HTTPS exchange SHALL send the certificate chain in PEM format with `Content-Type: application/x-pem-file`, SHALL identify the tenant with the `tenant_id` query parameter, and SHALL NOT send an `Authorization` header.

#### Scenario: Request format
- **WHEN** the HTTPS exchange runs for tenant `t123` with a leaf and one intermediate certificate
- **THEN** the request is `POST /tenant/oauth/certificate?tenant_id=t123` with both certificates in the PEM body and no `Authorization` header

### Requirement: JWE token decryption
The SDK SHALL decrypt the returned `access_token` as a compact JWE with `alg` `RSA-OAEP-256` and `enc` `A256GCM`, using the certificate's private key, and SHALL use the decrypted JWT as the bearer token. Any other `alg` or `enc` SHALL produce an error that names the received values.

#### Scenario: Successful decryption
- **WHEN** the fake server returns a JWE encrypted to the client certificate's public key
- **THEN** later API requests carry `Authorization: Bearer <decrypted JWT>`

#### Scenario: Unsupported algorithm
- **WHEN** the JWE header has `alg` `RSA1_5`
- **THEN** the login fails with an error naming `RSA1_5`

#### Scenario: Tampered token
- **WHEN** one byte of the JWE ciphertext is changed
- **THEN** the login fails with a decryption error and no token is stored

### Requirement: Second factor for the HTTPS exchange
When the platform requires TFA for the HTTPS exchange, the SDK SHALL obtain the code from `LoginOptions.TOTPSecret`, `TOTPCode`, or `SMSCode` and send it in the `X-Cumulocity-TFA-Code` header.

#### Scenario: TOTP required
- **WHEN** the first exchange returns `401` asking for TFA and `LoginOptions.TOTPCode` is set
- **THEN** the SDK repeats the exchange once with `X-Cumulocity-TFA-Code` set to the returned code

### Requirement: Token refresh uses the same transport
The cached token source SHALL refresh certificate tokens with the same transport rule as the first login, and SHALL compute expiry from `expires_in`, falling back to the JWT `exp` claim.

#### Scenario: Refresh before expiry
- **WHEN** a certificate token with `expires_in: 60` is about to expire and the client sends another request
- **THEN** the SDK performs a new exchange with the same transport before sending the request

### Requirement: Certificate opt-in helpers
The SDK SHALL provide `Users.CurrentUser.SetRequiredAuthType(ctx, authType)`, `Applications.Current.SetAuthType(ctx, authType)`, `Applications.ResetAuthType(ctx, id)`, and `jsonmodels.User.RequiredAuthType()`, where `authType` is `NONE` or `CERTIFICATES`.

#### Scenario: Regular user opts in
- **WHEN** `Users.CurrentUser.SetRequiredAuthType(ctx, "CERTIFICATES")` is called
- **THEN** the SDK sends `PUT /user/currentUser` with `{"requiredAuthType":"CERTIFICATES"}`

#### Scenario: Service user opts in
- **WHEN** `Applications.Current.SetAuthType(ctx, "CERTIFICATES")` is called
- **THEN** the SDK sends `PUT /application/currentApplication/authType` with `{"requiredAuthType":"CERTIFICATES"}`

#### Scenario: Admin resets a service user
- **WHEN** `Applications.ResetAuthType(ctx, "42")` is called
- **THEN** the SDK sends `PUT /application/applications/42/authType` with `{"requiredAuthType":"NONE"}`

### Requirement: Lockout-safe opt-in
`Client.EnableCertificateAuth(ctx, opts)` SHALL, before setting `requiredAuthType` to `CERTIFICATES`, check locally that the key is RSA, matches the certificate, and that the certificate is currently valid, and SHALL verify the chain with `POST /tenant/trusted-certificates/verify-cert-chain` (unless `SkipChainVerification` is set). It SHALL NOT change `requiredAuthType` when a check fails. After opting in it SHALL confirm with a certificate exchange; if that fails, the error SHALL name the recovery path. (The platform only issues certificate tokens to accounts that already opted in, so an exchange cannot be the first check.)

#### Scenario: Untrusted chain
- **WHEN** `EnableCertificateAuth` is called and the tenant does not trust the certificate chain
- **THEN** it returns an error matching `ErrCertificateNotTrusted` and no `PUT` request is sent

#### Scenario: Success
- **WHEN** the chain is trusted and the exchange succeeds after the opt-in
- **THEN** the requests are, in order, the chain verification, the `PUT` of `requiredAuthType`, and the certificate exchange, and the client uses the certificate token

### Requirement: Certificate tokens for service users
`ClientOptions` SHALL accept an optional `ContextCertificateSource func(tenant string) (tls.Certificate, error)`, used when `ContextTokenExchange` is enabled. When it returns a certificate for a tenant, per-tenant tokens SHALL be obtained with the certificate exchange instead of the service user's password.

#### Scenario: Service user with a certificate
- **WHEN** a request runs with service-user credentials for tenant `t123` and `ContextCertificateSource` returns an RSA certificate for `t123`
- **THEN** the SDK obtains the token from `POST /tenant/oauth/certificate?tenant_id=t123` and does not call `POST /tenant/oauth/token`

