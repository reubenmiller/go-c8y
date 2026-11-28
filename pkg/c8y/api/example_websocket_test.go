package api_test

import (
	"context"
	"fmt"
	"net/http"

	"github.com/reubenmiller/go-c8y/v2/internal/pkg/testclient"

	api "github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/authentication"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/realtime"
)

// With dry run enabled a realtime subscription does not connect. The upgrade
// request goes to the dry-run handler like an HTTP request, and the stream
// ends immediately without an error. The channels which would have been
// subscribed to are in the result's metadata.
func ExampleClient_Realtime() {
	client := testclient.Must(api.ClientOptions{
		BaseURL:          "https://example.invalid",
		SkipTenantLookup: true,
		Auth: authentication.AuthOptions{
			Tenant: "t1", Username: "user", Password: "pass",
			AuthType: []authentication.AuthType{authentication.AuthTypeBasic},
		},
	})

	ctx := api.WithDryRun(context.Background(), true)
	ctx = api.WithDryRunHandler(ctx, func(req *http.Request) {
		fmt.Println(req.Method, req.URL)
		fmt.Println("Upgrade:", req.Header.Get("Upgrade"))
	})

	result := client.Alarms.SubscribeStream(ctx, "12345")
	fmt.Println("status:", result.Status, "error:", result.Err)
	fmt.Println("channels:", result.Meta[realtime.DryRunMetaChannels])

	stream := result.Data
	defer stream.Close()
	for range stream.Seq() {
		fmt.Println("never reached")
	}
	fmt.Println("stream error:", stream.Err())

	// Output:
	// GET https://example.invalid/cep/realtime
	// Upgrade: websocket
	// status: Skipped error: <nil>
	// channels: [/alarms/12345]
	// stream error: <nil>
}
