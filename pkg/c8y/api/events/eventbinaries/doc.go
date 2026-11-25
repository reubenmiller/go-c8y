// Package eventbinaries provides the binary file attached to an event.
//
// Reach it from a client with client.Events.Binaries.
//
// Example, with a client and ctx in scope:
//
//	result := client.Events.Binaries.Create(ctx, "12345", eventbinaries.UploadFileOptions{
//		Reader: strings.NewReader("log lines"),
//		Name:   "device.log",
//	})
//	fmt.Println(result.Err)
package eventbinaries
