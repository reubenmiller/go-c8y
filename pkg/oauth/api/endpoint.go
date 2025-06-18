package api

import (
	"fmt"
	"net/url"
	"strings"
)

// AuthorizationRequest OAuth2 authorization request data
type AuthorizationRequest struct {
	// The token value, typically a 40-character random string.
	ClientID string

	// Audience
	Audience string

	// Scopes
	Scopes []string

	// The refresh token value, associated with the access token.
	URL *url.URL
}

// AuthEndpoints OAuth2 endpoints used to get retrieve the device code and access token
type AuthEndpoints struct {
	// Device Authorization URL e.g. /oauth/device/code
	DeviceAuthorizationURL string

	// Token Authorization URL, e.g. /oauth/token
	TokenURL string
}

// GetEndpointUrl get the full url related to a given oauth endpoint
func GetEndpointUrl(endpoint *AuthorizationRequest, u string) string {
	if endpoint == nil {
		return u
	}
	return fmt.Sprintf("%s://%s/%s", endpoint.URL.Scheme, endpoint.URL.Host, strings.TrimLeft(u, "/"))
}
