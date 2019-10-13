package cmd

import (
	"github.com/spf13/cobra"
)

type eventCmd struct {
	*baseCmd
}

func newEventRootCmd() *eventCmd {
	ccmd := &eventCmd{}

	cmd := &cobra.Command{
		Use:   "event",
		Short: "Event REST endpoint",
		Long:  `REST endpoint to interact with Cumulocity events`,
	}

	// Subcommands
	cmd.AddCommand(newGetEventBinaryCmd().getCommand())
	cmd.AddCommand(newGetEventCmd().getCommand())
	cmd.AddCommand(newGetEventCollectionCmd().getCommand())
	cmd.AddCommand(newNewEventBinaryCmd().getCommand())
	cmd.AddCommand(newNewEventCmd().getCommand())
	cmd.AddCommand(newDeleteEventBinaryCmd().getCommand())
	cmd.AddCommand(newDeleteEventCmd().getCommand())
	cmd.AddCommand(newDeleteEventCollectionCmd().getCommand())
	cmd.AddCommand(newUpdateEventBinaryCmd().getCommand())
	cmd.AddCommand(newUpdateEventCmd().getCommand())

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}
