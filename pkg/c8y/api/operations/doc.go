// Package operations covers operations: send commands to devices (restart,
// configuration, software update) and follow their status. Create takes a
// typed device reference and fragments which describe the command.
//
// Reach it from a client with client.Operations.
//
// Example, with a client and ctx in scope:
//
//	created := client.Operations.Create(ctx, operations.CreateOptions{
//		DeviceID:    managedobjects.ByName("my-device"),
//		Description: "Restart device",
//		Fragments:   []model.Fragment{model.Frag("c8y_Restart", map[string]any{})},
//	})
//	fmt.Println(created.Data.ID(), created.Err)
package operations
