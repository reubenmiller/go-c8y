package authentication_test

import (
	"fmt"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/authentication"
)

// TokenSourceFunc wraps any function as a TokenSource.
// This is the simplest way to supply a custom bearer token.
func ExampleTokenSourceFunc() {
	// A function that returns a hard-coded token — swap the body for any
	// real credential-fetch logic (HTTP call, vault lookup, etc.)
	src := authentication.TokenSourceFunc(func() (*authentication.Token, error) {
		return &authentication.Token{
			AccessToken: "my-bearer-token",
			Expiry:      time.Now().Add(1 * time.Hour),
		}, nil
	})

	tok, err := src.Token()
	if err != nil {
		panic(err)
	}
	fmt.Println(tok.AccessToken)
	// Output:
	// my-bearer-token
}

// CachedTokenSource caches the result of the underlying source and only calls
// it again once the token has (nearly) expired.
func ExampleNewCachedTokenSource() {
	calls := 0
	base := authentication.TokenSourceFunc(func() (*authentication.Token, error) {
		calls++
		return &authentication.Token{
			AccessToken: fmt.Sprintf("token-%d", calls),
			Expiry:      time.Now().Add(1 * time.Hour), // long-lived
		}, nil
	})

	cached := authentication.NewCachedTokenSource(base)

	// First call — fetches from base.
	t1, _ := cached.Token()
	// Second call — returns cached value; base is NOT called again.
	t2, _ := cached.Token()

	fmt.Println(t1.AccessToken)
	fmt.Println(t2.AccessToken) // same token
	fmt.Println("base calls:", calls)
	// Output:
	// token-1
	// token-1
	// base calls: 1
}

// After a 401 the client calls Invalidate so the next Token() call fetches a
// brand-new token instead of reusing the revoked one.
func ExampleCachedTokenSource_Invalidate() {
	calls := 0
	base := authentication.TokenSourceFunc(func() (*authentication.Token, error) {
		calls++
		return &authentication.Token{
			AccessToken: fmt.Sprintf("token-%d", calls),
			Expiry:      time.Now().Add(1 * time.Hour),
		}, nil
	})

	cached := authentication.NewCachedTokenSource(base)

	t1, _ := cached.Token() // fetches token-1
	cached.Invalidate()     // simulates a 401 — discard the cached token
	t2, _ := cached.Token() // fetches token-2

	fmt.Println(t1.AccessToken)
	fmt.Println(t2.AccessToken)
	// Output:
	// token-1
	// token-2
}

// Seed pre-populates the cache with a token the caller already holds (e.g.
// immediately after performing a device-flow or password login) so the first
// real API request does not pay an extra network round-trip.
func ExampleCachedTokenSource_Seed() {
	calls := 0
	base := authentication.TokenSourceFunc(func() (*authentication.Token, error) {
		calls++
		return &authentication.Token{
			AccessToken: "refreshed-token",
			Expiry:      time.Now().Add(1 * time.Hour),
		}, nil
	})

	cached := authentication.NewCachedTokenSource(base)

	// We already have a valid token from a previous login; seed it so the
	// base source is not called until this token actually expires.
	cached.Seed(&authentication.Token{
		AccessToken: "initial-token",
		Expiry:      time.Now().Add(55 * time.Minute),
	})

	tok, _ := cached.Token()
	fmt.Println(tok.AccessToken)
	fmt.Println("base calls:", calls)
	// Output:
	// initial-token
	// base calls: 0
}
