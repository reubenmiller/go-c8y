package cmd

import (
	"github.com/spf13/cobra"
)

type devicesCmd struct {
	*baseCmd
}

func newDevicesRootCmd() *devicesCmd {
	ccmd := &devicesCmd{}

	cmd := &cobra.Command{
		Use:   "devices",
		Short: "Cumulocity devices",
		Long:  `REST endpoint to interact with Cumulocity devices`,
	}

	// Subcommands
	cmd.AddCommand(newGetSupportedMeasurementsCmd().getCommand())
	cmd.AddCommand(newGetSupportedSeriesCmd().getCommand())
	cmd.AddCommand(newGetSupportedOperationsCmd().getCommand())
	cmd.AddCommand(newSetDeviceRequiredAvailabilityCmd().getCommand())

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}
