package cmd

import (
	"github.com/spf13/cobra"
)

type tenantOptionsCmd struct {
	*baseCmd
}

func newTenantOptionsRootCmd() *tenantOptionsCmd {
	ccmd := &tenantOptionsCmd{}

	cmd := &cobra.Command{
		Use:   "tenantOptions",
		Short: "Cumulocity tenantOptions",
		Long:  `REST endpoint to interact with Cumulocity tenantOptions`,
	}

	// Subcommands
	cmd.AddCommand(newGetTenantOptionCollectionCmd().getCommand())
	cmd.AddCommand(newNewTenantOptionCmd().getCommand())
	cmd.AddCommand(newGetTenantOptionCmd().getCommand())
	cmd.AddCommand(newDeleteTenantOptionCmd().getCommand())
	cmd.AddCommand(newUpdateTenantOptionCmd().getCommand())
	cmd.AddCommand(newUpdateTenantOptionBulkCmd().getCommand())
	cmd.AddCommand(newGetTenantOptionsForCategoryCmd().getCommand())
	cmd.AddCommand(newUpdateTenantOptionEditableCmd().getCommand())

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}
