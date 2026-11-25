package fakeserver

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

// The tenant offers OAI-Secure (OAUTH2_INTERNAL) like a Cumulocity tenant, and
// its token carries the expiry claims which clients use to decide whether a
// stored token can be reused
func TestOAuthInternal_LoginOptionAndToken(t *testing.T) {
	fs := New(t)

	status, body := doRequest(t, fs, http.MethodGet, "/tenant/loginOptions", "", "application/json")
	if status != http.StatusOK {
		t.Fatalf("GET loginOptions: status=%d", status)
	}
	var options struct {
		LoginOptions []struct {
			Type        string `json:"type"`
			InitRequest string `json:"initRequest"`
		} `json:"loginOptions"`
	}
	if err := json.Unmarshal([]byte(body), &options); err != nil {
		t.Fatal(err)
	}
	initRequest := ""
	for _, option := range options.LoginOptions {
		if option.Type == "OAUTH2_INTERNAL" {
			initRequest = option.InitRequest
		}
	}
	if initRequest == "" {
		t.Fatalf("no OAUTH2_INTERNAL login option with an initRequest: %s", body)
	}

	form := url.Values{"grant_type": {"PASSWORD"}, "username": {"admin"}, "password": {"admin-pass"}}
	resp, err := http.Post(initRequest, "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST %s: status=%d body=%s", initRequest, resp.StatusCode, out)
	}
	var token struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(out, &token); err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(token.AccessToken, ".")
	if len(parts) != 3 {
		t.Fatalf("token is not a JWT: %q", token.AccessToken)
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatal(err)
	}
	var claims struct {
		Sub string `json:"sub"`
		Ten string `json:"ten"`
		Exp int64  `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		t.Fatal(err)
	}
	if claims.Sub != "admin" || claims.Ten != "t12345" {
		t.Errorf("claims sub=%q ten=%q, want admin and t12345", claims.Sub, claims.Ten)
	}
	if expiresAt := time.Unix(claims.Exp, 0); !expiresAt.After(time.Now().Add(time.Hour)) {
		t.Errorf("token expires at %s, want a future expiry", expiresAt)
	}
}
