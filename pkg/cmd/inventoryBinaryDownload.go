package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/spf13/cobra"
)

var inventoryBinaryDownloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Download a managed object",
	Long:  `Download a managed object`,
	Example: `
	Download a managed object
	c8y inventory download --id 12345
	`,
	Run: func(cmd *cobra.Command, args []string) {
		ids := GetIDs(cmd, args)
		output, _ := cmd.Flags().GetString("output")

		if v, err := filepath.Abs(output); err == nil {
			output = v
		}

		wg := new(sync.WaitGroup)
		wg.Add(len(ids))

		for i := range ids {
			go func(index int) {
				log.Printf("id: %s\n", ids[index])
				outputfile, err := client.Inventory.DownloadBinary(
					context.Background(),
					ids[index],
				)

				if err != nil {
					log.Printf("gID=%s, error`=%s", ids[index], err)
				} else {
					if output != "" {
						if err := os.Rename(outputfile, output); err != nil {
							log.Printf("Failed to rename file. %s", err)
						}
					} else {
						output = outputfile
					}

					fmt.Println(output)
				}
				wg.Done()
			}(i)
		}

		wg.Wait()
	},
}

func init() {
	// Flags
	addInventoryOptions(inventoryBinaryDownloadCmd)
	addIDFlag(inventoryBinaryDownloadCmd)
	inventoryBinaryDownloadCmd.Flags().StringP("output", "o", "", "Output file")
}
