package user

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// TokenResponse represents the OAuth2 token response from Cumulocity
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// GetToken retrieves an OAuth2 access token for Cumulocity using the resource owner password credentials flow
func GetToken(ctx context.Context, baseURL, tenant, username, password string) (string, error) {
	return GetTokenWithFlow(ctx, baseURL, tenant, username, password, "")
}

// GetTokenWithFlow retrieves an OAuth2 access token for Cumulocity
// If code is provided, uses authorization code flow; otherwise uses password flow
func GetTokenWithFlow(ctx context.Context, baseURL, tenant, username, password, code string) (string, error) {
	return GetTokenWithCode(ctx, baseURL, tenant, username, password, code, "", "")
}

// GetTokenWithCode retrieves an OAuth2 access token for Cumulocity with full authorization code flow parameters
func GetTokenWithCode(ctx context.Context, baseURL, tenant, username, password, code, clientID, clientSecret string) (string, error) {
	return GetTokenWithCodeAndRedirect(ctx, baseURL, tenant, username, password, code, clientID, clientSecret, "")
}

// GetTokenWithCodeAndRedirect retrieves an OAuth2 access token for Cumulocity with full authorization code flow parameters including redirect URI
func GetTokenWithCodeAndRedirect(ctx context.Context, baseURL, tenant, username, password, code, clientID, clientSecret, redirectURI string) (string, error) {
	// Ensure baseURL has a trailing slash
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	// Prepare form data
	data := url.Values{}

	if code != "" {
		// Authorization Code Flow
		data.Set("grant_type", "AUTHORIZATION_CODE")
		data.Set("code", code)
		if clientID != "" {
			data.Set("client_id", clientID)
		}
		if clientSecret != "" {
			data.Set("client_secret", clientSecret)
		}
		if redirectURI != "" {
			data.Set("redirect_uri", redirectURI)
		}
	} else {
		// Resource Owner Password Credentials Flow
		data.Set("grant_type", "PASSWORD")
		data.Set("username", username)
		data.Set("password", password)
		data.Set("tfa_code", "undefined") // Default TFA code
	}

	// Create the request
	tokenURL := baseURL + "tenant/oauth/token"
	if tenant != "" {
		tokenURL += "?tenant_id=" + url.QueryEscape(tenant)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=UTF-8")
	req.Header.Set("Accept", "application/json")

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OAuth2 request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse the response
	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("access_token not found in response")
	}

	return tokenResp.AccessToken, nil
}
