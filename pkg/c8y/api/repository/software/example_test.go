package software_test

import (
	"bytes"
	"context"
	"fmt"

	"github.com/reubenmiller/go-c8y/v2/internal/pkg/examplesetup"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/repository/software/softwareversions"
)

// Create a software item together with a version and its binary.
// CreateVersion finds or creates the software item by name, then uploads the file.
func Example() {
	client, stop := examplesetup.New()
	defer stop()
	ctx := context.Background()

	version := client.Repository.Software.Versions.CreateVersion(ctx, softwareversions.CreateVersionOptions{
		SoftwareName: "example-agent",
		SoftwareType: "apt",
		Version:      "1.0.0",
		File: softwareversions.UploadFileOptions{
			Reader: bytes.NewReader([]byte("package contents")),
			Name:   "example-agent_1.0.0.deb",
		},
	})
	if version.Err != nil {
		fmt.Println("create failed:", version.Err)
		return
	}
	fmt.Println("version:", version.Data.Get("c8y_Software.version").String())

	// The software item was created by name. Resolve it back to its id.
	id, err := client.Repository.Software.ResolveID(ctx, "example-agent", nil)
	fmt.Println("item found:", id != "", err)
	// Output:
	// version: 1.0.0
	// item found: true <nil>
}
