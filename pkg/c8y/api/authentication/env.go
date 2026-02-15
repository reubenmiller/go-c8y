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

func HostFromEnvironment() string {
	// Prefer host instead of the token's Issuer, so it works well in situations
	// where the issuer might be from a URL which is not reachable for the device
	host := GetEnvValue(EnvironmentHost...)
	if host != "" {
		return host
	}
	if token := GetEnvValue(EnvironmentToken...); token != "" {
		if tok, err := ParseToken(token); err == nil && tok.XSRFToken != "" && tok.Issuer != "" {
			slog.Info("Setting host from issuer", "value", tok.Issuer)
			return tok.Issuer
		}
	}
	return ""
}

func FromEnvironment() AuthOptions {
	auth := AuthOptions{
		Tenant:   GetEnvValue(EnvironmentTenant...),
		Username: GetEnvValue(EnvironmentUsername...),
		Password: GetEnvValue(EnvironmentPassword...),
		Token:    GetEnvValue(EnvironmentToken...),

		CertificateKey: GetEnvValue(EnvironmentCertificateKeyFile...),
		Certificate:    GetEnvValue(EnvironmentCertificateFile...),
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

func GetEnvValue(key ...string) string {
	for _, k := range key {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	return ""
}
