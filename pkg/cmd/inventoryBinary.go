package cmd

import (
	"github.com/spf13/cobra"
)

var inventoryBinaryCmd = &cobra.Command{
	Use:   "binary",
	Short: "Inventory rest endpoint",
	Long:  `Inventory rest endpoint to interact with Cumulocity managed objects`,
}

func init() {
	inventoryBinaryCmd.AddCommand(inventoryBinaryDownloadCmd)
	inventoryBinaryCmd.AddCommand(inventoryBinaryDeleteCmd)
	inventoryBinaryCmd.AddCommand(inventoryBinaryCreateCmd)
}
