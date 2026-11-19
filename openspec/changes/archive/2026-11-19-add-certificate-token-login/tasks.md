## 1. Fake server and test fixtures

- [x] 1.1 Add `POST /tenant/oauth/certificate` to the fake server: checks the PEM body and `tenant_id`, returns a JWE encrypted to the presented certificate's key
- [x] 1.2 Add RSA and ECDSA test certificates and keys (generated in tests with `crypto/x509`, not committed secrets)
- [x] 1.3 Add TFA challenge support to the fake exchange endpoint

## 2. JWE and exchange

- [x] 2.1 Implement compact JWE decryption for `RSA-OAEP-256` / `A256GCM` with the standard library, with unit tests including tampered and unsupported tokens
- [x] 2.2 Add the exchange call in `tenants/logintokens` (PEM body, `tenant_id`, no `Authorization`, optional TFA header)
- [x] 2.3 Add `certutil` helper to build the PEM chain from a `tls.Certificate`

## 3. Login flow

- [x] 3.1 Add `AuthOptions.CertificateTransport` and its constants
- [x] 3.2 Implement the `Auto` rule (key type detection for RSA, ECDSA, Ed25519) and fail-fast checks for explicit overrides
- [x] 3.3 Route `LoginMethodCertificate` and `fetchToken` through one `certificateLogin` function
- [x] 3.4 Wire TFA (`TOTPSecret`, `TOTPCode`, `SMSCode`) into the HTTPS exchange
- [x] 3.5 Improve the mTLS error message to mention the port and `MTLSPort`
- [x] 3.6 Offline tests for every transport, TFA, and refresh scenario

## 4. Opt-in helpers

- [x] 4.1 Add `jsonmodels.User.RequiredAuthType()`
- [x] 4.2 Add `CurrentUser.SetRequiredAuthType`, `CurrentApplication.SetAuthType`, `Applications.ResetAuthType`
- [x] 4.3 Add `EnableCertificateAuth` (local checks and chain verification, then opt in, then confirm with an exchange)
- [x] 4.4 Godoc warnings about lockout and the recovery paths
- [x] 4.5 Offline tests for the helper scenarios

## 5. Service users

- [x] 5.1 Add `ClientOptions.ContextCertificateSource` (used when `ContextTokenExchange` is on)
- [x] 5.2 Use it per tenant in `MiddlewareContextAuthorization` token exchange; fall back to password only when no certificate is returned
- [x] 5.3 Offline test for the service-user scenario

## 6. Documentation

- [x] 6.1 Add a "Certificate login" section to `docs/API_DESIGN.md` covering transports, RSA vs EC, port 8443, TFA, and opt-in
- [x] 6.2 Add an example under `examples/` for RSA HTTPS login
- [ ] 6.3 Live test against a tenant with certificate auth enabled (manual, before release; part of the final post-check)
