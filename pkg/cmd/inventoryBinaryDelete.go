package cmd

import (
	"context"
	"log"
	"sync"

	"github.com/spf13/cobra"
)

var inventoryBinaryDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a binary managed object",
	Long:  `Delete a binary managed object`,
	Example: `
	Delete a binary managed object
	c8y inventory binary download --id 12345
	`,
	Run: func(cmd *cobra.Command, args []string) {
		ids := GetIDs(cmd, args)

		wg := new(sync.WaitGroup)
		wg.Add(len(ids))

		for i := range ids {
			go func(index int) {
				_, err := client.Inventory.DeleteBinary(
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
	addIDFlag(inventoryBinaryDeleteCmd)
}
