package applications_test

import (
	"context"
	"fmt"

	"github.com/reubenmiller/go-c8y/v2/internal/pkg/examplesetup"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/applications"
)

// Create a hosted web application, find it by name and delete it.
func Example() {
	client, stop := examplesetup.New()
	defer stop()
	ctx := context.Background()

	created := client.Applications.Create(ctx, map[string]any{
		"name": "example-app", "key": "example-app-key", "type": "HOSTED", "contextPath": "example-app",
	})
	if created.Err != nil {
		fmt.Println("create failed:", created.Err)
		return
	}
	fmt.Println("created:", created.Data.Name(), created.Data.Type())

	byName := client.Applications.ListByName(ctx, applications.ListByNameOptions{Name: "example-app"})
	for app, err := range byName.Items() {
		fmt.Println("found:", app.Name(), err)
	}

	deleted := client.Applications.Delete(ctx, created.Data.ID(), applications.DeleteOptions{})
	fmt.Println("deleted:", deleted.HTTPStatus, deleted.Err)
	// Output:
	// created: example-app HOSTED
	// found: example-app <nil>
	// deleted: 204 <nil>
}
