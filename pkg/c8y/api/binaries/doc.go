// Package binaries covers binaries stored in the inventory: upload, download
// and delete files. Uploads take a path or an io.Reader and report progress
// with UploadFileOptions.Progress.
//
// Reach it from a client with client.Binaries.
//
// Example, with a client and ctx in scope:
//
//	result := client.Binaries.Create(ctx, binaries.UploadFileOptions{
//		Reader: strings.NewReader("file contents"),
//		Name:   "notes.txt",
//	})
//	fmt.Println(result.Data.Get("id").String(), result.Err)
package binaries
