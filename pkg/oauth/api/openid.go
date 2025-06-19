package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

type OpenIDConfiguration struct {
	Issuer                      string   `json:"issuer"`
	AuthorizationEndpoint       string   `json:"authorization_endpoint"`
	DeviceAuthorizationEndpoint string   `json:"device_authorization_endpoint"`
	TokenEndpoint               string   `json:"token_endpoint"`
	UserInfoEndpoint            string   `json:"userinfo_endpoint"`
	JwksUri                     string   `json:"jwks_uri"`
	RegistrationEndpoint        string   `json:"registration_endpoint"`
	RevocationEndpoint          string   `json:"revocation_endpoint"`
	ResponseTypesSupported      []string `json:"response_types_supported"`
}

func getOpenIDConnectConfigurationURL(u *url.URL) string {
	path := "/"
	if strings.Contains(u.Host, "login.microsoftonline.com") {
		// Microsoft
		path = "/oauth2/v2.0/"
	} else if strings.Contains(u.Path, "/realms/") {
		// Keycloak
		r := regexp.MustCompile("(/realms/[^/]+/)")
		if matches := r.FindStringSubmatch(u.Path); len(matches) > 0 {
			path = matches[0]
		}
	}
	return path + ".well-known/openid-configuration"
}

func GetOpenIDConfiguration(ctx context.Context, client *http.Client, oauthUrl *url.URL, oidc_url string, data any) error {
	if oidc_url == "" {
		oidc_url = getOpenIDConnectConfigurationURL(oauthUrl)
	}
	u, err := oauthUrl.Parse(oidc_url)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return err
	}

	if client == nil {
		client = http.DefaultClient
	}

	// Disable redirects so we can capture the first redirect location
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 399 {
		return fmt.Errorf("openid-configuration request failed. status_code=%s", resp.Status)
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(data)
}
