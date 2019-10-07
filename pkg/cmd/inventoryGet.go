package cmd

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
)

var inventoryGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Delete a managed object",
	Long:  `Delete a managed object`,
	Example: `
	Remove a list of managed objects
	c8y inventory delete --id 12345 --id 67891
	`,
	Run: func(cmd *cobra.Command, args []string) {
		ids := GetIDs(cmd, args)
		log.Printf("args: len=%d, %v\n", len(ids), ids)
		wg := new(sync.WaitGroup)

		wg.Add(len(ids))

		withParents, _ := cmd.Flags().GetBool("withParents")
		filterArg, _ := cmd.Flags().GetString("filter")
		filter := strings.Split(filterArg, ",")

		if filterArg != "" && len(filter) > 0 {
			// Print csv header
			fmt.Println(filterArg)
		}

		for i := range ids {
			go func(index int, filter []string) {
				log.Printf("id: %s\n", ids[index])
				_, resp, err := client.Inventory.GetManagedObject(
					context.Background(),
					ids[index],
					&c8y.ManagedObjectOptions{
						WithParents:       withParents,
						PaginationOptions: *c8y.NewPaginationOptions(1), // TODO: This should not be required as it is not supported by the api!
					},
				)

				if err != nil {
					log.Printf("gID=%s, error`=%s", ids[index], err)
				} else {
					if filterArg != "" && len(filter) > 0 {
						log.Printf("Filtering results: len=%d\n", len(filter))
						selectedOutput := FilterJSON(*resp.JSON, filter)
						fmt.Println(strings.Join(selectedOutput, ","))
					} else {
						fmt.Println(*resp.JSONData)
					}
				}
				wg.Done()
			}(i, filter)
		}

		wg.Wait()
	},
}

func init() {
	// Flags
	addInventoryOptions(inventoryGetCmd)
	addResultFilterFlags(inventoryGetCmd)
	addIDFlag(inventoryGetCmd)
}
