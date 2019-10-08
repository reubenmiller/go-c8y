package cmd

import (
	"github.com/spf13/cobra"
)

var inventoryCmd = &cobra.Command{
	Use:   "inventory",
	Short: "Inventory rest endpoint",
	Long:  `Inventory rest endpoint to interact with Cumulocity managed objects`,
}

func init() {
	rootCmd.AddCommand(inventoryCmd)
	inventoryCmd.AddCommand(inventoryQueryCmd)
	inventoryCmd.AddCommand(newNewManagedObjectCmd().getCommand())
	inventoryCmd.AddCommand(inventoryCreateCmd)
	inventoryCmd.AddCommand(inventoryUpdateCmd)
	inventoryCmd.AddCommand(inventoryDeleteCmd)
	inventoryCmd.AddCommand(inventoryBinaryCmd)
}
