// Package childassets covers the child assets of a managed object: assign,
// unassign and read the references.
//
// Reach it from a client with client.ManagedObjects.ChildAssets.
//
// Example, with a client and ctx in scope:
//
//	result := client.ManagedObjects.ChildAssets.Assign(ctx, "12345", map[string]any{"id": "67890"})
//	fmt.Println(result.Err)
package childassets
