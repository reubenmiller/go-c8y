package cmd

import (
	"github.com/spf13/cobra"
)

type deviceRootCmd struct {
	*baseCmd
}

func newDeviceRootCmd() *deviceRootCmd {
	ccmd := &deviceRootCmd{}

	cmd := &cobra.Command{
		Use:   "devices",
		Short: "Cumulocity devices",
		Long:  `Cumulocity devices`,
	}

	// Subcommands
	cmd.AddCommand(newQueryDeviceCmd().getCommand())
	cmd.AddCommand(newGetDeviceCollectionCmd().getCommand())

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}
