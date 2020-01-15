package cmd

import (
	"github.com/spf13/cobra"
)

type inventoryCmd struct {
	*baseCmd
}

func newInventoryRootCmd() *inventoryCmd {
	ccmd := &inventoryCmd{}

	cmd := &cobra.Command{
		Use:   "inventory",
		Short: "Cumulocity managed objects",
		Long:  `REST endpoint to interact with Cumulocity managed objects`,
	}

	// Subcommands
	cmd.AddCommand(newGetManagedObjectCollectionCmd().getCommand())
	cmd.AddCommand(newQueryManagedObjectCollectionCmd().getCommand())
	cmd.AddCommand(newNewManagedObjectCmd().getCommand())
	cmd.AddCommand(newGetManagedObjectCmd().getCommand())
	cmd.AddCommand(newUpdateManagedObjectCmd().getCommand())
	cmd.AddCommand(newDeleteManagedObjectCmd().getCommand())
	cmd.AddCommand(newGetSupportedMeasurementsCmd().getCommand())
	cmd.AddCommand(newGetSupportedSeriesCmd().getCommand())

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}
