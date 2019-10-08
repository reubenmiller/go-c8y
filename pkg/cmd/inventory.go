package cmd

import (
	"github.com/spf13/cobra"
)

type inventoryCmd struct {
	*baseCmd
}

func newInventoryCmd() *inventoryCmd {
	ccmd := &inventoryCmd{}

	cmd := &cobra.Command{
		Use:   "inventory",
		Short: "Inventory rest endpoint",
		Long:  `Inventory rest endpoint to interact with Cumulocity managed objects`,
	}

	// Subcommands
	cmd.AddCommand(newQueryManagedObjectCmd().getCommand())
	cmd.AddCommand(newGetManagedObjectCmd().getCommand())
	cmd.AddCommand(newNewManagedObjectCmd().getCommand())
	cmd.AddCommand(newUpdateManagedObjectCmd().getCommand())
	cmd.AddCommand(newDeleteManagedObjectCmd().getCommand())
	cmd.AddCommand(newBinaryGetManagedObjectCmd().getCommand())

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}
