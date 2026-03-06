package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/alarms"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/authentication"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/model"
	oauth2_api "github.com/reubenmiller/go-c8y/pkg/oauth/api"
)

func main() {
	// Create the client from the following environment variables
	client := api.NewClient(api.ClientOptions{
		BaseURL: authentication.HostFromEnvironment(),
	})

	// Request token using device flow
	fmt.Fprintf(os.Stderr, "🏄 Signing in using OAuth2 device flow\n\n")
	_, err := client.AuthorizeWithDeviceFlow(context.Background(), "", oauth2_api.AuthEndpoints{}, nil)
	if err != nil {
		slog.Error("Failed to get access token", "err", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "🔍 Checking if the token can be used to make API calls\n")
	result := client.Alarms.List(
		context.Background(),
		alarms.ListOptions{
			Severity: []model.AlarmSeverity{
				model.AlarmSeverityMajor,
			},
		},
	)
	if result.Err != nil {
		fmt.Fprintf(os.Stderr, "🚫 API call failed. %s\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "✅ API call was successful using the token from oAuth2\n")
	os.Exit(0)
}
