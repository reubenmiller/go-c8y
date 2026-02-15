package main

import (
	"context"
	"fmt"
	"log"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/authentication"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/inventory/managedobjects"
)

func main() {
	// Create a client
	client := api.NewClient(api.ClientOptions{
		BaseURL: "https://example.c8y.io",
		Auth: authentication.AuthOptions{
			Tenant:   "t12345",
			Username: "user",
			Password: "pass",
		},
	})

	// Example 1: Normal request (will be sent to server)
	fmt.Println("=== Normal Request ===")
	result := client.ManagedObjects.Get(context.Background(), "12345", managedobjects.GetOptions{})
	if result.Err != nil {
		log.Printf("Normal request error: %v\n", result.Err)
	} else {
		fmt.Printf("Response: ID=%s, Name=%s\n", result.Data.ID(), result.Data.Name())
	}

	// Example 2: Dry run request (returns mock response)
	fmt.Println("\n=== Dry Run Request (Mock Response) ===")
	dryRunCtx := api.WithDryRun(context.Background(), true)
	result = client.ManagedObjects.Get(dryRunCtx, "12345", managedobjects.GetOptions{})
	if result.Err != nil {
		log.Printf("Dry run request error: %v\n", result.Err)
	} else {
		fmt.Printf("Mock Response: ID=%s, Name=%s\n", result.Data.ID(), result.Data.Name())
		fmt.Printf("HTTP Status: %d\n", result.HTTPStatus)
	}

	// Example 3: Dry run POST request
	fmt.Println("\n=== Dry Run POST Request (Mock Create) ===")
	createResult := client.ManagedObjects.Create(dryRunCtx, map[string]any{
		"name": "Test Device",
		"type": "c8y_Device",
	})
	if createResult.Err != nil {
		log.Printf("Dry run create error: %v\n", createResult.Err)
	} else {
		fmt.Printf("Mock Created: ID=%s\n", createResult.Data.ID())
		fmt.Printf("HTTP Status: %d (Created)\n", createResult.HTTPStatus)
	}

	// Example 4: Check if dry run is enabled
	fmt.Println("\n=== Check Dry Run Status ===")
	fmt.Printf("Normal context dry run: %v\n", api.IsDryRun(context.Background()))
	fmt.Printf("Dry run context dry run: %v\n", api.IsDryRun(dryRunCtx))
}
