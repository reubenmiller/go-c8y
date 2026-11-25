package api

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"encoding/pem"
	"errors"
	"fmt"
	ctxhelpers "github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/contexthelpers"
	"net"
	"strings"
	"time"

	"github.com/reubenmiller/go-c8y/v2/internal/jwe"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/authentication"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/tenants/logintokens"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/users/currentuser/totp"
)

// certificateLogin exchanges the client certificate (Auth.Certificate and
// Auth.CertificateKey) for an access token, using the transport chosen by
// Auth.CertificateTransport. opts may be nil (token refresh): then a TFA
// challenge cannot be answered and is returned as an error.
func (c *Client) certificateLogin(ctx context.Context, opts *LoginOptions) (string, time.Time, error) {
	// The exchange must not go through the token source, which would start
	// another exchange (the token source is what calls this when it is empty).
	ctx = ctxhelpers.WithSkipTokenSource(ctx)
	cert, err := loadClientKeyPair(c.Auth.Certificate, c.Auth.CertificateKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("certificate login: load certificate: %w", err)
	}
	transport, err := authentication.ResolveCertificateTransport(c.Auth.CertificateTransport, cert.PrivateKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("certificate login: %w", err)
	}
	if transport == authentication.CertificateTransportMTLS {
		tok := c.Devices.CreateAccessToken(ctx)
		if tok.Err != nil {
			return "", time.Time{}, mtlsLoginError(tok.Err, c.common.MTLSPort)
		}
		raw := tok.Data.AccessToken()
		return raw, tokenExpiry(raw, 0), nil
	}
	return c.exchangeCertificate(ctx, c.Auth.Tenant, cert, opts)
}

// exchangeCertificate performs the HTTPS certificate exchange
// (POST /tenant/oauth/certificate) and decrypts the returned JWE.
func (c *Client) exchangeCertificate(ctx context.Context, tenant string, cert tls.Certificate, opts *LoginOptions) (string, time.Time, error) {
	key, ok := cert.PrivateKey.(*rsa.PrivateKey)
	if !ok {
		return "", time.Time{}, fmt.Errorf("certificate login over HTTPS needs an RSA key, got %T", cert.PrivateKey)
	}
	if tenant == "" {
		return "", time.Time{}, errors.New("certificate login over HTTPS needs the tenant ID (AuthOptions.Tenant)")
	}
	req := logintokens.CertificateTokenOptions{Tenant: tenant, Chain: certificateChainPEM(cert)}

	res := c.Tenants.LoginTokens.CreateWithCertificate(ctx, req)
	if res.Err != nil && opts != nil {
		code, handled, err := c.certificateTFACode(ctx, res.Err, *opts)
		if err != nil {
			return "", time.Time{}, err
		}
		if handled {
			req.TFACode = code
			res = c.Tenants.LoginTokens.CreateWithCertificate(ctx, req)
		}
	}
	if res.Err != nil {
		return "", time.Time{}, fmt.Errorf("certificate login: %w", res.Err)
	}

	plaintext, err := jwe.Decrypt(res.Data.AccessToken(), key)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("certificate login: decrypt token: %w", err)
	}
	token := string(plaintext)
	return token, tokenExpiry(token, res.Data.ExpiresIn()), nil
}

// certificateTFACode answers a TFA challenge from the certificate exchange
// with the login options' TOTP or SMS callbacks. handled is false when err is
// not a TFA challenge.
func (c *Client) certificateTFACode(ctx context.Context, err error, opts LoginOptions) (code string, handled bool, outErr error) {
	var apiErr *core.Error
	if !errors.As(err, &apiErr) || apiErr.StatusCode() != 401 {
		return "", false, nil
	}
	msg := strings.ToLower(apiErr.Error())
	switch {
	case strings.Contains(msg, "pin") && strings.Contains(msg, "generated"):
		if opts.SMSCode == nil {
			return "", false, nil
		}
		code, outErr = opts.SMSCode(ctx, SMSChallenge{Message: msg})
		if outErr != nil {
			return "", false, fmt.Errorf("sms: code input: %w", outErr)
		}
		return code, true, nil
	case strings.Contains(msg, "totp") || strings.Contains(msg, "tfa") || strings.Contains(msg, "two-factor"):
		if opts.TOTPSecret == "" && opts.TOTPCode == nil {
			return "", false, nil
		}
		code, outErr = c.totpCodeFrom(ctx, opts, totp.TOTPChallenge{Message: msg})
		if outErr != nil {
			return "", false, fmt.Errorf("totp: code input: %w", outErr)
		}
		return code, true, nil
	}
	return "", false, nil
}

// certificateChainPEM encodes the certificate chain of cert as PEM.
func certificateChainPEM(cert tls.Certificate) []byte {
	var out []byte
	for _, der := range cert.Certificate {
		out = append(out, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})...)
	}
	return out
}

// tokenExpiry returns when a token expires: from expiresIn (seconds) when
// given, else from the JWT exp claim, else in 55 minutes.
func tokenExpiry(token string, expiresIn int64) time.Time {
	if expiresIn > 0 {
		return time.Now().Add(time.Duration(expiresIn) * time.Second)
	}
	if claims, err := authentication.ParseToken(token); err == nil && claims.ExpiresAt != nil {
		return claims.ExpiresAt.Time
	}
	return time.Now().Add(55 * time.Minute)
}

// mtlsLoginError adds a hint about the mTLS port to network errors.
func mtlsLoginError(err error, port string) error {
	if port == "" {
		port = "8443"
	}
	var netErr net.Error
	var opErr *net.OpError
	if errors.As(err, &netErr) || errors.As(err, &opErr) || strings.Contains(strings.ToLower(err.Error()), "connection refused") {
		return fmt.Errorf("certificate login over mTLS failed: is port %s reachable? Set ClientOptions.MTLSPort if the platform uses another port: %w", port, err)
	}
	return fmt.Errorf("certificate login over mTLS: %w", err)
}
