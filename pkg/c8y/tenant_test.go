package c8y

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

// mustParseURL parses a URL string and fails the test immediately on error.
func mustParseURL(t *testing.T, rawURL string) *url.URL {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("url.Parse(%q): %v", rawURL, err)
	}
	return u
}

// ---------------------------------------------------------------------------
// getAuthorizationEndpointFromURL
// ---------------------------------------------------------------------------

func TestGetAuthorizationEndpointFromURL_ParsesQueryParams(t *testing.T) {
	u := mustParseURL(t, "https://idp.example.com/auth?client_id=cid&audience=aud&scope=openid&scope=profile")

	endpoint := getAuthorizationEndpointFromURL(u)

	if endpoint.ClientID != "cid" {
		t.Errorf("ClientID: got %q, want %q", endpoint.ClientID, "cid")
	}
	if endpoint.Audience != "aud" {
		t.Errorf("Audience: got %q, want %q", endpoint.Audience, "aud")
	}
	if len(endpoint.Scopes) != 2 {
		t.Errorf("Scopes: got %v, want 2 entries", endpoint.Scopes)
	}
	if endpoint.URL != u {
		t.Error("URL pointer should be the same as the input")
	}
}

func TestGetAuthorizationEndpointFromURL_NoQueryParams(t *testing.T) {
	u := mustParseURL(t, "https://idp.example.com/auth")
	endpoint := getAuthorizationEndpointFromURL(u)

	if endpoint.ClientID != "" {
		t.Errorf("ClientID should be empty, got %q", endpoint.ClientID)
	}
	if endpoint.Audience != "" {
		t.Errorf("Audience should be empty, got %q", endpoint.Audience)
	}
	if len(endpoint.Scopes) != 0 {
		t.Errorf("Scopes should be empty, got %v", endpoint.Scopes)
	}
	if endpoint.URL != u {
		t.Error("URL pointer should be the same as the input")
	}
}

// ---------------------------------------------------------------------------
// getAuthorizationRequest – 3xx redirect (original behavior)
// ---------------------------------------------------------------------------

func TestGetAuthorizationRequest_302Redirect(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r,
			"https://idp.example.com/auth?client_id=my-client&audience=api&scope=openid+profile",
			http.StatusFound,
		)
	}))
	defer ts.Close()

	endpoint, err := getAuthorizationRequest(context.Background(), &http.Client{}, ts.URL, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if endpoint.ClientID != "my-client" {
		t.Errorf("ClientID: got %q, want %q", endpoint.ClientID, "my-client")
	}
	if endpoint.Audience != "api" {
		t.Errorf("Audience: got %q, want %q", endpoint.Audience, "api")
	}
	if len(endpoint.Scopes) == 0 {
		t.Error("Scopes should not be empty")
	}
	if endpoint.URL == nil {
		t.Error("URL should not be nil")
	}
}

func TestGetAuthorizationRequest_301Redirect(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r,
			"https://idp.example.com/auth?client_id=perm-client&scope=openid",
			http.StatusMovedPermanently,
		)
	}))
	defer ts.Close()

	endpoint, err := getAuthorizationRequest(context.Background(), &http.Client{}, ts.URL, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if endpoint.ClientID != "perm-client" {
		t.Errorf("ClientID: got %q, want %q", endpoint.ClientID, "perm-client")
	}
}

// ---------------------------------------------------------------------------
// getAuthorizationRequest – 200 with JSON redirectTo (new behavior)
// ---------------------------------------------------------------------------

func TestGetAuthorizationRequest_200WithRedirectTo(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"redirectTo":"https://idp.example.com/auth?client_id=app-client&audience=myapi&scope=openid"}`))
	}))
	defer ts.Close()

	endpoint, err := getAuthorizationRequest(context.Background(), &http.Client{}, ts.URL, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if endpoint.ClientID != "app-client" {
		t.Errorf("ClientID: got %q, want %q", endpoint.ClientID, "app-client")
	}
	if endpoint.Audience != "myapi" {
		t.Errorf("Audience: got %q, want %q", endpoint.Audience, "myapi")
	}
	if endpoint.URL == nil {
		t.Error("URL should not be nil")
	}
}

func TestGetAuthorizationRequest_200WithRedirectToAndRedirectURL(t *testing.T) {
	// When redirectURL is non-empty it must be injected as redirect_uri.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"redirectTo":"https://idp.example.com/auth?client_id=app-client&scope=openid"}`))
	}))
	defer ts.Close()

	const callbackURL = "http://127.0.0.1:5001/callback"
	endpoint, err := getAuthorizationRequest(context.Background(), &http.Client{}, ts.URL, callbackURL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := endpoint.URL.Query().Get("redirect_uri")
	if got != callbackURL {
		t.Errorf("redirect_uri: got %q, want %q", got, callbackURL)
	}
	// Other params must survive the round-trip.
	if endpoint.ClientID != "app-client" {
		t.Errorf("ClientID: got %q, want %q", endpoint.ClientID, "app-client")
	}
}

func TestGetAuthorizationRequest_200WithRedirectTo_NoRedirectURL(t *testing.T) {
	// When redirectURL is empty, redirect_uri must NOT be added.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"redirectTo":"https://idp.example.com/auth?client_id=app-client"}`))
	}))
	defer ts.Close()

	endpoint, err := getAuthorizationRequest(context.Background(), &http.Client{}, ts.URL, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if endpoint.URL.Query().Get("redirect_uri") != "" {
		t.Errorf("redirect_uri should be absent when redirectURL is empty")
	}
}

func TestGetAuthorizationRequest_200WithoutRedirectTo(t *testing.T) {
	// A 200 response with no "redirectTo" field should return an error.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"ok"}`))
	}))
	defer ts.Close()

	_, err := getAuthorizationRequest(context.Background(), &http.Client{}, ts.URL, "")
	if err == nil {
		t.Fatal("expected an error for 200 response without redirectTo, got nil")
	}
}

func TestGetAuthorizationRequest_404(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	_, err := getAuthorizationRequest(context.Background(), &http.Client{}, ts.URL, "")
	if err == nil {
		t.Fatal("expected an error for 404 response, got nil")
	}
}
