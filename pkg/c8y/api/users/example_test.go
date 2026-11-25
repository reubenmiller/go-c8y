package users_test

import (
	"context"
	"fmt"

	"github.com/reubenmiller/go-c8y/v2/internal/pkg/examplesetup"
)

// ResolveID turns a user name into a user id. A name without a wildcard is
// returned as is. A name with "*" is matched against the users of the tenant.
func Example() {
	client, stop := examplesetup.New()
	defer stop()
	ctx := context.Background()

	client.Users.Create(ctx, map[string]any{"userName": "alice.smith", "email": "alice@example.com", "password": "Passw0rd!2345"})
	client.Users.Create(ctx, map[string]any{"userName": "bob.jones", "email": "bob@example.com", "password": "Passw0rd!2345"})

	id, err := client.Users.ResolveID(ctx, "alice*", nil)
	fmt.Println(id, err)

	// More than one match is an error, so use a pattern which selects a single user.
	_, err = client.Users.ResolveID(ctx, "*.*", nil)
	fmt.Println(err != nil)

	// Output:
	// alice.smith <nil>
	// true
}
