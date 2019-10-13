package cmd

import (
	"github.com/spf13/cobra"
)

type identityCmd struct {
	*baseCmd
}

func newIdentityRootCmd() *identityCmd {
	ccmd := &identityCmd{}

	cmd := &cobra.Command{
		Use:   "identity",
		Short: "Identity REST endpoint",
		Long:  `REST endpoint to interact with Cumulocity identities (external ids)`,
	}

	// Subcommands
	cmd.AddCommand(newGetExternalIDCmd().getCommand())
	cmd.AddCommand(newGetExternalIDCollectionCmd().getCommand())
	cmd.AddCommand(newNewExternalIDCmd().getCommand())
	cmd.AddCommand(newDeleteExternalIDCmd().getCommand())

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}
