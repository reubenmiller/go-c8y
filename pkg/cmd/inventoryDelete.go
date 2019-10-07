package cmd

import (
	"context"
	"log"
	"sync"

	"github.com/spf13/cobra"
)

var inventoryDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a managed object",
	Long:  `Delete a managed object`,
	Example: `
	Remove a list of managed objects
	c8y inventory delete --id 12345 --id 67891
	`,
	Run: func(cmd *cobra.Command, args []string) {

		ids := GetIDs(cmd, args)
		wg := new(sync.WaitGroup)
		wg.Add(len(ids))

		for i := range ids {
			go func(index int) {
				_, err := client.Inventory.Delete(
					context.Background(),
					ids[index],
				)

				if err != nil {
					log.Printf("gID=%s, error`=%s", ids[index], err)
				}
				wg.Done()
			}(i)
		}

		wg.Wait()
	},
}

func init() {
	// Flags
	addIDFlag(inventoryDeleteCmd)
}
