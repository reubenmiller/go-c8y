package main

import (
	"context"
	"log"
	"os"
	"strings"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/reubenmiller/go-c8y/pkg/oauth/api"
)

func main() {
	// Create the client from the following environment variables
	// C8Y_HOST, C8Y_TENANT, C8Y_USER, C8Y_PASSWORD
	// os.env
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

	log.Printf("loginOptions: %#v", loginOption)
	parts := strings.Split(strings.TrimPrefix(strings.TrimPrefix(loginOption.Self, "http://"), "https://"), ".")
	if len(parts) > 0 {
		client.TenantName = parts[0]
	}

	callbackURL := "http://127.0.0.1:5001/callback"
	if v := os.Getenv("C8Y_CALLBACK_URL"); v != "" {
		callbackURL = v
	}

	_, loginErr := client.Tenant.AuthorizeWithAuthorizationFlow(
		context.TODO(),
		loginOption.InitRequest,
		api.AuthEndpoints{},
		callbackURL,
		nil,
	)

	if loginErr != nil {
		log.Fatalf("login failed. %s", loginErr)
	}

	paging := c8y.NewPaginationOptions(1)
	paging.WithTotalElements = true

	devices, _, err := client.Inventory.GetDevices(
		context.Background(),
		paging,
	)

	if err != nil {
		log.Fatalf("Could not retrieve devices. %s", err)
	}

	log.Printf("Total devices: %d", *devices.Statistics.TotalElements)
}
