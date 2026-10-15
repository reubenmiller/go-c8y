package authentication

import (
	"log/slog"
	"os"
	"strings"
)

var (
	EnvironmentHost               = []string{"C8Y_BASEURL", "C8Y_URL", "C8Y_HOST"}
	EnvironmentTenant             = []string{"C8Y_TENANT"}
	EnvironmentToken              = []string{"C8Y_TOKEN"}
	EnvironmentPassword           = []string{"C8Y_PASSWORD"}
	EnvironmentUsername           = []string{"C8Y_USERNAME", "C8Y_USER"}
	EnvironmentCertificateKeyFile = []string{"C8Y_CERTIFICATE_KEY_FILE"}
	EnvironmentCertificateFile    = []string{"C8Y_CERTIFICATE_FILE"}
)

// LookupFunc resolves a setting by key, e.g. os.LookupEnv or a lookup over a
// KEY=VALUE credentials file. The second return value reports whether the key
// is set; empty and whitespace-only values are treated as unset by
// FromLookup and HostFromLookup.
type LookupFunc func(key string) (string, bool)

// LookupFromMap returns a LookupFunc backed by the given map.
func LookupFromMap(values map[string]string) LookupFunc {
	return func(key string) (string, bool) {
		v, ok := values[key]
		return v, ok
	}
}

// HostFromEnvironment returns the Cumulocity host from the C8Y_* environment
// variables. See HostFromLookup for the rules.
func HostFromEnvironment() string {
	return HostFromLookup(os.LookupEnv)
}

// HostFromLookup returns the Cumulocity host using the same rules as
// HostFromEnvironment, but reading the Environment* settings from lookup.
// The first non-empty value of EnvironmentHost wins; otherwise the issuer of
// the token is used when the token carries both an XSRF token and an issuer.
func HostFromLookup(lookup LookupFunc) string {
	// Prefer host instead of the token's Issuer, so it works well in situations
	// where the issuer might be from a URL which is not reachable for the device
	host := lookupValue(lookup, EnvironmentHost...)
	if host != "" {
		return host
	}
	if token := lookupValue(lookup, EnvironmentToken...); token != "" {
		if tok, err := ParseToken(token); err == nil && tok.XSRFToken != "" && tok.Issuer != "" {
			slog.Info("Setting host from issuer", "value", tok.Issuer)
			return tok.Issuer
		}
	}
	return ""
}

// FromEnvironment returns the authentication options from the C8Y_*
// environment variables. See FromLookup for the rules.
func FromEnvironment() AuthOptions {
	return FromLookup(os.LookupEnv)
}

// FromLookup returns the authentication options using the same rules as
// FromEnvironment, but reading the Environment* settings from lookup (for
// example a parsed KEY=VALUE credentials file, see LookupFromMap).
// Values are trimmed and empty values are treated as unset. A username of the
// form "tenant/user" is split, and the tenant claim of the token (if it can be
// parsed) takes precedence over the tenant setting.
func FromLookup(lookup LookupFunc) AuthOptions {
	auth := AuthOptions{
		Tenant:   lookupValue(lookup, EnvironmentTenant...),
		Username: lookupValue(lookup, EnvironmentUsername...),
		Password: lookupValue(lookup, EnvironmentPassword...),
		Token:    lookupValue(lookup, EnvironmentToken...),

		CertificateKey: lookupValue(lookup, EnvironmentCertificateKeyFile...),
		Certificate:    lookupValue(lookup, EnvironmentCertificateFile...),
	}

	if strings.Contains(auth.Username, "/") {
		if tenant, username, found := strings.Cut(auth.Username, "/"); found {
			if tenant != "" {
				auth.Tenant = tenant
			}
			if username != "" {
				auth.Username = username
			}
		}
	}

	if tok, err := ParseToken(auth.Token); err == nil && tok.Tenant != "" {
		auth.Tenant = tok.Tenant
	}

	return auth
}

// GetEnvValue returns the first non-empty (trimmed) environment variable of keys.
func GetEnvValue(key ...string) string {
	return lookupValue(os.LookupEnv, key...)
}

// lookupValue returns the first non-empty (trimmed) value of keys.
func lookupValue(lookup LookupFunc, keys ...string) string {
	for _, k := range keys {
		if v, ok := lookup(k); ok {
			if v = strings.TrimSpace(v); v != "" {
				return v
			}
		}
	}
	return ""
}
