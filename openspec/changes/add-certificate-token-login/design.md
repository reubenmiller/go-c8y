## Context

- Today `LoginMethodCertificate` calls `Devices.CreateAccessToken` (mTLS on `MTLSPort`, default 8443). `fetchToken` uses the same path for refresh.
- `POST /tenant/oauth/certificate`: body is the PEM leaf or chain (`application/x-pem-file`); tenant from `?tenant_id=` or `X-Cumulocity-TenantId`; TFA code in `X-Cumulocity-TFA-Code`; response `{access_token, token_type, expires_in}` where `access_token` is a compact JWE (`alg: RSA-OAEP-256`, `enc: A256GCM`) encrypted to the certificate's public key.
- Opt-in: a regular user sets `requiredAuthType: CERTIFICATES` on `PUT /user/currentUser` (self-service only). A service user calls `PUT /application/currentApplication/authType`. An admin can only reset a service user to `NONE` with `PUT /application/applications/{id}/authType`. After switching, password-derived tokens are rejected immediately.
- EC keys cannot receive an encrypted token securely; they must use mTLS on 8443.

## Goals / Non-Goals

**Goals:**
- One login method; the SDK picks the transport the platform supports for the key.
- No new dependency for JWE.
- Safe opt-in: hard to lock an account out by accident.

**Non-Goals:**
- Generating or enrolling certificates (already covered by device enrollment and `certutil`).
- General JWE support beyond `RSA-OAEP-256` + `A256GCM`.

## Decisions

**Transport chosen by key type (`CertificateTransportAuto`).** RSA → HTTPS exchange; ECDSA / Ed25519 → mTLS. Alternative: try HTTPS first, fall back to mTLS. Rejected: for EC keys HTTPS only works through the platform bug, so a fallback would hide a future breakage, and for RSA keys a silent fallback to 8443 would hide misconfiguration. `Auto` never falls back between transports.

**Explicit overrides fail fast.** `CertificateTransportHTTPS` with a non-RSA key returns an error before any request. `CertificateTransportMTLS` with an RSA key is allowed (devices that already use 8443).

**JWE decryption with the standard library.** Split the compact JWE into 5 parts; check the header is exactly `RSA-OAEP-256` / `A256GCM`; unwrap the content key with `rsa.DecryptOAEP(sha256.New(), …)`; decrypt with `cipher.NewGCM` using the protected header (ASCII) as additional data. About 60 lines. Alternative: `github.com/go-jose/go-jose/v4`. Rejected to avoid a dependency for one fixed algorithm pair; revisit if the platform adds algorithms.

**Refresh through the same rule.** `fetchToken` calls a shared `certificateLogin(ctx)` that applies the transport rule, so cached token refresh behaves the same as the first login. Token expiry comes from `expires_in`, then the JWT `exp` claim.

**TFA through existing callbacks.** The HTTPS exchange reuses `LoginOptions.TOTPSecret`, `TOTPCode`, and `SMSCode`. Device users are never asked for a second factor.

**Lockout-safe opt-in helper.** `EnableCertificateAuth(ctx, cert, key)` first performs a certificate exchange; it sets `requiredAuthType` only if the exchange succeeds. The low-level setters stay available with godoc warnings.

**Service users.** `ClientOptions` gets an optional `ContextCertificateSource func(tenant string) (tls.Certificate, error)`, used by `ContextTokenExchange`. When it returns a certificate, the exchange uses it instead of the service user's password. Service-user certificates should be RSA, because microservice containers often cannot reach port 8443.

## Risks / Trade-offs

- [Platform adds new JWE algorithms] → Unknown `alg`/`enc` gives a clear error naming the values; adding support is a local change.
- [Users switch to CERTIFICATES and lose their certificate] → Lockout-safe helper; godoc points to the admin reset (`Applications.ResetAuthType`) and the "forgot password" reset for regular users.
- [Port 8443 blocked] → mTLS errors mention the port and the `MTLSPort` option.

## Open Questions

- Does the platform return a specific error when an EC key is used on the HTTPS endpoint after the bug fix? The SDK should map it to a clear message.
- What does the exchange return when TFA is required but no code was sent? The spec scenario assumes a `401` that can be recognized as a TFA challenge, like the OAI-Secure login. Confirm against a live tenant; if it cannot be recognized, send the code up front whenever `TOTPSecret` is set.
- Should `Auto` read the key type from `CertificateKey` only, or also accept a `crypto.Signer` (for hardware keys) in a later change?
