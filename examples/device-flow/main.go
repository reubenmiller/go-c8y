package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/reubenmiller/go-c8y/pkg/oauth/api"
)

func main() {
	c8y.SilenceLogger()

	// Create the client from the following environment variables
	client := c8y.NewClientFromOptions(nil, c8y.ClientOptions{
		BaseURL:  os.Getenv("C8Y_HOST"),
		Realtime: false,
	})

	loginOption, found, err := client.Tenant.HasExternalAuthProvider(context.Background())
	if err != nil {
		log.Fatalf("Could not get Cumulocity login options. %s", err)
	}
	if !found {
		log.Fatalf("Cumulocity instance does not have an external OAUTH2 provider configured")
	}

	// Request token using device flow
	fmt.Fprintf(os.Stderr, "🏄 Signing in using OAuth2 device flow\n\n")
	_, err = client.Tenant.AuthorizeWithDeviceFlow(context.Background(), loginOption.InitRequest, api.AuthEndpoints{
		DeviceAuthorizationURL: "/oauth/device/code",
		TokenURL:               "/oauth/token",
	}, nil)
	if err != nil {
		log.Fatalf("Failed to get access token. %s", err)
	}

	fmt.Fprintf(os.Stderr, "🔍 Checking if the token can be used to make API calls\n")
	_, _, err = client.Alarm.GetAlarms(
		context.Background(),
		&c8y.AlarmCollectionOptions{
			Severity: c8y.AlarmSeverityMajor,
		},
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "🚫 API call failed. %s\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "✅ API call was successful using the token from oAuth2\n")
	os.Exit(0)
}
