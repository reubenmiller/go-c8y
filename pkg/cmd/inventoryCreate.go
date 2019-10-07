package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var inventoryCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a managed object",
	Long:  `Create a managed object`,
	Example: `
	c8y inventory create --data '{\"name\":\"hello-go\"}'

	or using the json shorthand form (to create non nested data)

	c8y inventory create --data "name=hello-go,c8y_IsDevice={}"
	`,
	Run: func(cmd *cobra.Command, args []string) {
		// Parse JSON data
		data := getDataFlag(cmd)

		_, resp, err := client.Inventory.Create(
			context.Background(),
			&data,
		)
		if err != nil {
			panic(fmt.Errorf("%s", err))
		}
		fmt.Println(*resp.JSONData)
	},
}

func init() {
	// Flags
	addDataFlag(inventoryCreateCmd)
}
