// Package main demonstrates the Cumulocity SSO Authorization Code flow from a
// CLI.  The heavy lifting is done by client.AuthorizeWithBrowserFlow which:
//
//  1. Starts a local HTTP callback server on a random port.
//  2. Calls the Cumulocity initRequest endpoint to obtain the IdP auth URL.
//  3. Opens the system browser to the IdP login page.
//  4. Waits for the authorization code callback and exchanges it for a token.
//
// Required environment variable:
//
//	C8Y_BASEURL – e.g. https://mytenant.cumulocity.com
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/authentication"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/devices"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/tenants/currenttenant"
)

func main() {
	ctx := context.Background()

	baseURL := authentication.HostFromEnvironment()
	if baseURL == "" {
		fmt.Fprintln(os.Stderr, "error: C8Y_BASEURL is not set")
		os.Exit(1)
	}

	// -----------------------------------------------------------------------
	// 1. Discover SSO login option.
	// -----------------------------------------------------------------------
	client := api.NewClient(api.ClientOptions{BaseURL: baseURL})

	loginOption, found, err := client.HasExternalAuthProvider(ctx)
	if err != nil {
		slog.Error("Failed to retrieve Cumulocity login options", "err", err)
		os.Exit(1)
	}
	if !found {
		slog.Error("No external OAuth2 SSO provider is configured on this Cumulocity tenant")
		os.Exit(1)
	}
	slog.Info("Found SSO login option", "initRequest", loginOption.InitRequest())

	// -----------------------------------------------------------------------
	// 2. Run the Authorization Code flow in the browser.
	//    AuthorizeWithBrowserFlow starts the local callback server, opens the
	//    browser, waits for the code, exchanges it, and updates the client.
	// -----------------------------------------------------------------------
	fmt.Fprintln(os.Stderr, "🌐 Opening browser for SSO login …")
	if _, err := client.AuthorizeWithBrowserFlow(ctx, loginOption.InitRequest(), api.BrowserFlowOptions{
		ListenAddr: "localhost:5001",
	}); err != nil {
		slog.Error("SSO browser flow failed", "err", err)
		os.Exit(1)
	}

	// -----------------------------------------------------------------------
	// 3. Verify the token with a test API call.
	// -----------------------------------------------------------------------
	fmt.Fprintln(os.Stderr, "🔍 Verifying token …")
	result := client.Tenants.Current.Get(ctx, currenttenant.GetOptions{})
	if result.Err != nil {
		slog.Error("API call failed", "err", result.Err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "✅ Authenticated via SSO Authorization Code flow\n")
	fmt.Fprintf(os.Stderr, "   Tenant:  %s\n", result.Data.Name())
	fmt.Fprintf(os.Stderr, "   Domain:  %s\n", result.Data.DomainName())

	devicesResult := client.Devices.List(context.Background(), devices.ListOptions{
		PaginationOptions: pagination.PaginationOptions{
			WithTotalElements: true,
		},
	})
	if devicesResult.Err != nil {
		slog.Error("API call failed", "err", devicesResult.Err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "   Devices: %#v\n", devicesResult.Meta["totalElements"])
}

//
