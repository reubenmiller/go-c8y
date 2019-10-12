package cmd

import (
	"github.com/spf13/cobra"
)

type alarmsCmd struct {
	*baseCmd
}

func newAlarmsCmd() *alarmsCmd {
	ccmd := &alarmsCmd{}

	cmd := &cobra.Command{
		Use:   "alarms",
		Short: "Alarms rest endpoint",
		Long:  `Alarms rest endpoint to interact with Cumulocity managed objects`,
	}

	// Subcommands
	// new
	cmd.AddCommand(newNewAlarmCmd().getCommand())

	// get
	cmd.AddCommand(newGetAlarmCollectionCmd().getCommand())

	// updte
	cmd.AddCommand(newUpdateAlarmCmd().getCommand())
	cmd.AddCommand(newUpdateAlarmCollectionCmd().getCommand())

	// delete
	cmd.AddCommand(newDeleteAlarmCmd().getCommand())
	cmd.AddCommand(newDeleteAlarmCollectionCmd().getCommand())

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}
