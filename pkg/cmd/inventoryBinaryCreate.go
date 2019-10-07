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

var inventoryBinaryCreateCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload a binary managed object",
	Long:  `Upload a binary managed object`,
	Example: `
	Upload a binary managed object
	c8y inventory binary upload --file ./mybinary.zip

	c8y inventory binary upload --file ./test.zip --data "name=test,type=application/json"

	`,
	Run: func(cmd *cobra.Command, args []string) {

		var filenames []string
		if v, err := cmd.Flags().GetStringArray(inventoryFlagFile); err == nil {
			filenames = v
		}

		wg := new(sync.WaitGroup)
		wg.Add(len(filenames))

		for i := range filenames {
			go func(index int) {

				data := getDataFlag(cmd)

				// Set type if not already set
				if _, exists := data["type"]; !exists {
					// Guess the file type by reading the first 512 bytes
					// https://golangcode.com/get-the-content-type-of-file/
					f, err := os.Open(filenames[index])
					if err != nil {
						panic(err)
					}
					defer f.Close()

					// Get the content
					contentType, err := GetFileContentType(f)
					if err != nil {
						panic(err)
					}
					data["type"] = contentType
				}

				// Set name if blank
				if _, exists := data["name"]; !exists {
					data["name"] = filepath.Base(filenames[index])
				}

				_, resp, err := client.Inventory.CreateBinary(
					context.Background(),
					filenames[index],
					data,
				)

				if err != nil {
					log.Printf("file=%s, error`=%s", filenames[index], err)
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
	inventoryBinaryCreateCmd.Flags().StringArrayP(inventoryFlagFile, "f", []string{}, "Input file to upload as a Cumulocity binary")
	inventoryBinaryCreateCmd.MarkFlagRequired(inventoryFlagFile)
	addDataFlag(inventoryBinaryCreateCmd)
}
