package cmd

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/spf13/cobra"
)

var inventoryUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update a managed object",
	Long:  `Update a managed object`,
	Example: `
	c8y inventory update --id 12345 --data '{\"name\":\"hello-go\"}'

	or using the json shorthand form (to create non nested data)

	c8y inventory update --data "name=hello-go,c8y_IsDevice={}"
	`,
	Run: func(cmd *cobra.Command, args []string) {
		// Parse JSON data
		data := getDataFlag(cmd)
		ids := GetIDs(cmd, args)

		wg := new(sync.WaitGroup)
		wg.Add(len(ids))

		for i := range ids {
			go func(index int) {
				_, resp, err := client.Inventory.Update(
					context.Background(),
					ids[index],
					data,
				)
				if err != nil {
					log.Printf("id=%s, error`=%s", ids[index], err)
				} else {
					fmt.Println(*resp.JSONData)
				}
				wg.Done()
			}(i)
		}

		wg.Wait()
	},
}

func init() {
	// Flags
	addIDFlag(inventoryUpdateCmd)
	addDataFlag(inventoryUpdateCmd)
}
