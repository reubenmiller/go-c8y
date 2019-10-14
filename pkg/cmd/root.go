package cmd

import (
	"fmt"
	"os"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
)

type baseCmd struct {
	cmd *cobra.Command
}

func (c *baseCmd) getCommand() *cobra.Command {
	return c.cmd
}

func newBaseCmd(cmd *cobra.Command) *baseCmd {
	return &baseCmd{cmd: cmd}
}

var rootCmd = &cobra.Command{
	Use:   "c8y",
	Short: "Cumulocity command line interface",
	Long: `A Fast and Flexible Static Site Generator built with
                love by spf13 and friends in Go.
                Complete documentation is available at http://hugo.spf13.com`,
	Run: func(cmd *cobra.Command, args []string) {
		// Do Stuff Here
	},
}

var (
	client                   *c8y.Client
	globalFlagPageSize       int
	globalFlagVerbose        bool
	globalFlagWithTotalPages bool
)

func init() {
	client = c8y.NewClientFromEnvironment(nil, false)
}

func Execute() {
	// Global flags
	rootCmd.PersistentFlags().BoolVarP(&globalFlagVerbose, "verbose", "v", false, "Verbose logging")
	rootCmd.PersistentFlags().IntVar(&globalFlagPageSize, "pageSize", 5, "Maximum results per page")
	rootCmd.PersistentFlags().BoolVar(&globalFlagWithTotalPages, "withTotalPages", false, "Include all results")

	// Make flags case-insensitive
	rootCmd.PersistentFlags().SetNormalizeFunc(flagNormalizeFunc)

	rootCmd.AddCommand(newCompletionsCmd().getCommand())
	rootCmd.AddCommand(newVersionCmd().getCommand())

	rootCmd.AddCommand(newInventoryCmd().getCommand())
	rootCmd.AddCommand(newDeviceCmd().getCommand())
	rootCmd.AddCommand(newRealtimeCmd().getCommand())

	// generic commands
	rootCmd.AddCommand(newGetGenericRestCmd().getCommand())

	// Auto generated commands
	// alarms commands
	rootCmd.AddCommand(newAlarmsRootCmd().getCommand())

	// applications commands
	rootCmd.AddCommand(newApplicationsRootCmd().getCommand())

	// auditRecords commands
	rootCmd.AddCommand(newAuditRecordsRootCmd().getCommand())

	// binaries commands
	rootCmd.AddCommand(newBinariesRootCmd().getCommand())

	// currentApplication commands
	rootCmd.AddCommand(newCurrentApplicationRootCmd().getCommand())

	// operations commands
	rootCmd.AddCommand(newOperationsRootCmd().getCommand())

	// events commands
	rootCmd.AddCommand(newEventsRootCmd().getCommand())

	// identity commands
	rootCmd.AddCommand(newIdentityRootCmd().getCommand())

	// measurements commands
	rootCmd.AddCommand(newMeasurementsRootCmd().getCommand())

	// retentionRules commands
	rootCmd.AddCommand(newRetentionRulesRootCmd().getCommand())

	// systemOptions commands
	rootCmd.AddCommand(newSystemOptionsRootCmd().getCommand())

	// tenantOptions commands
	rootCmd.AddCommand(newTenantOptionsRootCmd().getCommand())

	// tenants commands
	rootCmd.AddCommand(newTenantsRootCmd().getCommand())

	// tenantStatistics commands
	rootCmd.AddCommand(newTenantStatisticsRootCmd().getCommand())

	// users commands
	rootCmd.AddCommand(newUsersRootCmd().getCommand())

	// userGroups commands
	rootCmd.AddCommand(newUserGroupsRootCmd().getCommand())

	// userReferences commands
	rootCmd.AddCommand(newUserReferencesRootCmd().getCommand())

	// userRoles commands
	rootCmd.AddCommand(newUserRolesRootCmd().getCommand())

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
