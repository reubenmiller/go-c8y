package main

import (
	"context"
	"fmt"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
)

func main() {
	client := c8y.NewClientFromEnvironment(nil, false)

	mor, resp, _ := client.Inventory.GetChildAdditions(context.Background(), "25636204",
		&c8y.ManagedObjectOptions{
			Query: "type eq myTest",
		})

	// Option 1: Using the Items which is a gjson array (this was only added in v0.28.0)
	fmt.Println("\n\nOption 1: Accessing data from the gjson array .Items")
	for i, mor := range mor.Items {
		fmt.Printf("\nChild %d\n", i)
		fmt.Printf("managedObject.id: %s\n", mor.Get("managedObject.id").String())
		if helloValue := mor.Get("managedObject.hello"); helloValue.Exists() {
			fmt.Printf("managedObject.hello: %s\n", helloValue)
		} else {
			fmt.Printf("managedObject.hello: <unset>\n")
		}
	}

	// Option 2: Using resp object directly to handle your own serialization
	fmt.Println("\n\nOption 2: Accessing data from the raw response using gjson")
	references := resp.JSON("references").Array()
	for i, mor := range references {
		fmt.Printf("\nChild %d\n", i)
		fmt.Printf("managedObject.id: %s\n", mor.Get("managedObject.id").String())
		if helloValue := mor.Get("managedObject.hello"); helloValue.Exists() {
			fmt.Printf("managedObject.hello: %s\n", helloValue)
		} else {
			fmt.Printf("managedObject.hello: <unset>\n")
		}
	}
}
