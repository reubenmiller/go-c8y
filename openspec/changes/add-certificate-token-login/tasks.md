## 1. Fake server and test fixtures

- [ ] 1.1 Add `POST /tenant/oauth/certificate` to the fake server: checks the PEM body and `tenant_id`, returns a JWE encrypted to the presented certificate's key
- [ ] 1.2 Add RSA and ECDSA test certificates and keys (generated in tests with `crypto/x509`, not committed secrets)
- [ ] 1.3 Add TFA challenge support to the fake exchange endpoint

## 2. JWE and exchange

- [ ] 2.1 Implement compact JWE decryption for `RSA-OAEP-256` / `A256GCM` with the standard library, with unit tests including tampered and unsupported tokens
- [ ] 2.2 Add the exchange call in `tenants/logintokens` (PEM body, `tenant_id`, no `Authorization`, optional TFA header)
- [ ] 2.3 Add `certutil` helper to build the PEM chain from a `tls.Certificate`

## 3. Login flow

- [ ] 3.1 Add `AuthOptions.CertificateTransport` and its constants
- [ ] 3.2 Implement the `Auto` rule (key type detection for RSA, ECDSA, Ed25519) and fail-fast checks for explicit overrides
- [ ] 3.3 Route `LoginMethodCertificate` and `fetchToken` through one `certificateLogin` function
- [ ] 3.4 Wire TFA (`TOTPSecret`, `TOTPCode`, `SMSCode`) into the HTTPS exchange
- [ ] 3.5 Improve the mTLS error message to mention the port and `MTLSPort`
- [ ] 3.6 Offline tests for every transport, TFA, and refresh scenario

## 4. Opt-in helpers

- [ ] 4.1 Add `jsonmodels.User.RequiredAuthType()`
- [ ] 4.2 Add `CurrentUser.SetRequiredAuthType`, `CurrentApplication.SetAuthType`, `Applications.ResetAuthType`
- [ ] 4.3 Add `EnableCertificateAuth` (exchange first, then opt in)
- [ ] 4.4 Godoc warnings about lockout and the recovery paths
- [ ] 4.5 Offline tests for the helper scenarios

## 5. Service users

- [ ] 5.1 Add `ClientOptions.ContextCertificateSource` (used when `ContextTokenExchange` is on)
- [ ] 5.2 Use it per tenant in `MiddlewareContextAuthorization` token exchange; fall back to password only when no certificate is returned
- [ ] 5.3 Offline test for the service-user scenario

## 6. Documentation

- [ ] 6.1 Add a "Certificate login" section to `docs/API_DESIGN.md` covering transports, RSA vs EC, port 8443, TFA, and opt-in
- [ ] 6.2 Add an example under `examples/` for RSA HTTPS login
- [ ] 6.3 Live test against a tenant with certificate auth enabled (manual, before release)
