package api

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"errors"
	"fmt"
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/trustedcertificates"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/users/currentuser"
)

// CertificateAuthTarget selects which account EnableCertificateAuth changes.
type CertificateAuthTarget int

const (
	// CertificateAuthCurrentUser opts in the user the client is logged in as.
	CertificateAuthCurrentUser CertificateAuthTarget = iota

	// CertificateAuthServiceUser opts in a microservice's own service user
	// (the client must use service user credentials).
	CertificateAuthServiceUser
)

// EnableCertificateAuthOptions configures EnableCertificateAuth.
type EnableCertificateAuthOptions struct {
	// Target is the account to opt in. Defaults to the current user.
	Target CertificateAuthTarget

	// SkipChainVerification skips checking the certificate chain against the
	// tenant's trusted certificates. That check needs tenant management read
	// rights or a service user; skip it only if you have verified the chain
	// another way.
	SkipChainVerification bool
}

// ErrCertificateNotTrusted is returned when the tenant does not trust the
// client certificate chain.
var ErrCertificateNotTrusted = errors.New("certificate chain is not trusted by the tenant")

// EnableCertificateAuth switches the account to certificate-only
// authentication (requiredAuthType CERTIFICATES), using the client's
// certificate (AuthOptions.Certificate and CertificateKey).
//
// The platform only issues certificate tokens to accounts that have already
// opted in, so the certificate cannot be fully tested beforehand. To reduce
// the risk of a lockout, this method:
//  1. checks locally that the key is RSA, matches the certificate, and the
//     certificate is currently valid;
//  2. verifies the chain against the tenant's trusted certificates (unless
//     SkipChainVerification is set);
//  3. opts the account in;
//  4. confirms with a certificate exchange and switches the client to the
//     certificate token.
//
// If step 4 fails, the account is opted in but cannot log in with the
// certificate. The returned error names the recovery path: the "forgot
// password" email flow for users, or applications.Service.ResetAuthType by a
// tenant administrator for service users.
func (c *Client) EnableCertificateAuth(ctx context.Context, opts EnableCertificateAuthOptions) error {
	cert, err := loadClientKeyPair(c.Auth.Certificate, c.Auth.CertificateKey)
	if err != nil {
		return fmt.Errorf("enable certificate auth: load certificate: %w", err)
	}
	if _, ok := cert.PrivateKey.(*rsa.PrivateKey); !ok {
		return fmt.Errorf("enable certificate auth: the certificate exchange needs an RSA key, got %T", cert.PrivateKey)
	}
	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return fmt.Errorf("enable certificate auth: parse certificate: %w", err)
	}
	if now := time.Now(); now.Before(leaf.NotBefore) || now.After(leaf.NotAfter) {
		return fmt.Errorf("enable certificate auth: certificate is not valid now (valid %s to %s)", leaf.NotBefore, leaf.NotAfter)
	}

	if !opts.SkipChainVerification {
		res := c.TrustedCertificates.VerifyChain(ctx, trustedcertificates.VerifyChainOptions{
			Chain:  certificateChainPEM(cert),
			Tenant: c.Auth.Tenant,
		})
		if res.Err != nil {
			if core.ErrHasStatus(res.Err, 403) {
				return fmt.Errorf("enable certificate auth: verifying the chain needs tenant management read rights or a service user (set SkipChainVerification to skip): %w", res.Err)
			}
			return fmt.Errorf("enable certificate auth: %w: %w", ErrCertificateNotTrusted, res.Err)
		}
		if !res.Data.SuccessfullyValidated() {
			return fmt.Errorf("enable certificate auth: %w", ErrCertificateNotTrusted)
		}
	}

	recovery := "use the \"forgot password\" email flow to reset it to NONE"
	switch opts.Target {
	case CertificateAuthServiceUser:
		recovery = "a tenant administrator can reset it with Applications.ResetAuthType"
		if res := c.Applications.Current.SetAuthType(ctx, currentuser.AuthTypeCertificates); res.Err != nil {
			return fmt.Errorf("enable certificate auth: %w", res.Err)
		}
	default:
		if res := c.Users.CurrentUser.SetRequiredAuthType(ctx, currentuser.AuthTypeCertificates); res.Err != nil {
			return fmt.Errorf("enable certificate auth: %w", res.Err)
		}
	}

	token, _, err := c.exchangeCertificate(ctx, c.Auth.Tenant, cert, nil)
	if err != nil {
		return fmt.Errorf("enable certificate auth: the account now requires certificates, but the certificate exchange failed (%s): %w", recovery, err)
	}
	c.seedTokenSource(token)
	return nil
}
