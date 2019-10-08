package cmd

import (
	"github.com/spf13/cobra"
)

type deviceCmd struct {
	*baseCmd
}

func newDeviceCmd() *deviceCmd {
	ccmd := &deviceCmd{}

	cmd := &cobra.Command{
		Use:   "device",
		Short: "Devices Endpoint",
		Long:  `Devices Endpoint`,
	}

	// Subcommands
	cmd.AddCommand(newQueryDeviceCmd().getCommand())

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}
