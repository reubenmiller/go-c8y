package authentication

import (
	"errors"
	"strings"
)

const (
	// LoginTypeOAuth2Internal OAuth2 internal mode
	LoginTypeOAuth2Internal = "OAUTH2_INTERNAL"

	// LoginTypeOAuth2 OAuth2 external provider
	LoginTypeOAuth2 = "OAUTH2"

	// LoginTypeBasic Basic authentication
	LoginTypeBasic = "BASIC"

	// LoginTypeNone no authentication
	LoginTypeNone = "NONE"
)

var (
	ErrInvalidLoginType = errors.New("invalid login type")
)

// Parse the login type and select as default if no value options are found
// It returns the selected method, and if the input was valid or not
func ParseLoginType(v string) (string, error) {
	v = strings.ToUpper(v)
	switch v {
	case LoginTypeBasic:
		return LoginTypeBasic, nil
	case LoginTypeNone:
		return LoginTypeNone, nil
	case LoginTypeOAuth2Internal:
		return LoginTypeOAuth2Internal, nil
	case LoginTypeOAuth2:
		return LoginTypeOAuth2, nil
	default:
		return "", ErrInvalidLoginType
	}
}

// LoginMethod identifies the specific login flow to use with LoginWithOptions.
//
// Use Method to enforce exactly one flow, or Preference to specify a priority
// list where the first locally-satisfied and available method wins.
type LoginMethod string

const (
	// LoginMethodBasic uses HTTP Basic authentication on every request.
	// No token exchange is performed; Username and Password must be set.
	LoginMethodBasic LoginMethod = "BASIC"

	// LoginMethodOAuth2Internal exchanges Username and Password for an
	// OAI-Secure short-lived token. Supports TFA (TOTP, SMS) and
	// forced-password-change challenges via the LoginOptions callbacks.
	LoginMethodOAuth2Internal LoginMethod = "OAUTH2_INTERNAL"

	// LoginMethodCertificate authenticates using a client/device certificate.
	// Certificate and CertificateKey must be set on AuthOptions.
	LoginMethodCertificate LoginMethod = "CERTIFICATE"

	// LoginMethodOAuth2DeviceFlow obtains a token via the RFC 8628 device
	// authorization flow. Requires an external OAUTH2 provider on the tenant.
	LoginMethodOAuth2DeviceFlow LoginMethod = "OAUTH2_DEVICE_FLOW"

	// LoginMethodOAuth2BrowserFlow obtains a token via the Authorization Code
	// flow (RFC 6749), opening the system browser. Requires an external
	// OAUTH2 provider on the tenant.
	LoginMethodOAuth2BrowserFlow LoginMethod = "OAUTH2_BROWSER_FLOW"
)

var ErrInvalidLoginMethod = errors.New("invalid login method")

// ParseLoginMethod parses a string (case-insensitive) into a LoginMethod.
func ParseLoginMethod(v string) (LoginMethod, error) {
	switch strings.ToUpper(v) {
	case string(LoginMethodBasic):
		return LoginMethodBasic, nil
	case string(LoginMethodOAuth2Internal):
		return LoginMethodOAuth2Internal, nil
	case string(LoginMethodCertificate):
		return LoginMethodCertificate, nil
	case string(LoginMethodOAuth2DeviceFlow):
		return LoginMethodOAuth2DeviceFlow, nil
	case string(LoginMethodOAuth2BrowserFlow):
		return LoginMethodOAuth2BrowserFlow, nil
	default:
		return "", ErrInvalidLoginMethod
	}
}
