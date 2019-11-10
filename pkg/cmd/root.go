package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
	globalFlagPrettyPrint    bool
	globalFlagDryRun         bool
	globalFlagSessionFile    string
	globalFlagUseEnv         bool
	globalFlagRaw            bool
)

func Execute() {
	// config file
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&globalFlagSessionFile, "session", "", "Session configuration")

	// Global flags
	rootCmd.PersistentFlags().BoolVarP(&globalFlagVerbose, "verbose", "v", false, "Verbose logging")
	rootCmd.PersistentFlags().IntVar(&globalFlagPageSize, "pageSize", 5, "Maximum results per page")
	rootCmd.PersistentFlags().BoolVar(&globalFlagWithTotalPages, "withTotalPages", false, "Include all results")
	rootCmd.PersistentFlags().BoolVar(&globalFlagPrettyPrint, "pretty", true, "Pretty print the json responses")
	rootCmd.PersistentFlags().BoolVar(&globalFlagDryRun, "dry", false, "Dry run. Don't send any data to the server")
	rootCmd.PersistentFlags().BoolVar(&globalFlagUseEnv, "useEnv", false, "Allow loading Cumulocity session setting from environment variables")
	rootCmd.PersistentFlags().BoolVar(&globalFlagRaw, "raw", false, "Raw values")

	rootCmd.PersistentFlags().StringSlice("filter", nil, "filter")
	rootCmd.PersistentFlags().StringSlice("select", nil, "select")
	rootCmd.PersistentFlags().String("format", "", "format")

	// TODO: Make flags case-insensitive
	// rootCmd.PersistentFlags().SetNormalizeFunc(flagNormalizeFunc)

	rootCmd.AddCommand(newCompletionsCmd().getCommand())
	rootCmd.AddCommand(newVersionCmd().getCommand())

	// rootCmd.AddCommand(newInventoryCmd().getCommand())
	rootCmd.AddCommand(newDeviceRootCmd().getCommand())
	rootCmd.AddCommand(newRealtimeCmd().getCommand())
	rootCmd.AddCommand(newSessionsRootCmd().getCommand())

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

	// inventory commands
	rootCmd.AddCommand(newInventoryRootCmd().getCommand())

	// inventoryReferences commands
	rootCmd.AddCommand(newInventoryReferencesRootCmd().getCommand())

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

func initConfig() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	if globalFlagVerbose || globalFlagDryRun {
		log.SetPrefix("VERBOSE: ")
	} else {
		// Disable log messages
		log.SetOutput(ioutil.Discard)
	}

	if globalFlagSessionFile == "" && os.Getenv("C8Y_SESSION") != "" {
		globalFlagSessionFile = os.Getenv("C8Y_SESSION")
		log.Printf("Using session environment variable: %s\n", globalFlagSessionFile)
	}

	if os.Getenv("C8Y_USE_ENVIRONMENT") != "" {
		globalFlagUseEnv = true
	}

	// only parse env variables if no explict config file is given
	if globalFlagUseEnv {
		log.Println("C8Y_USE_ENVIRONMENT is set. Environment variables can be used to override config settings")
		viper.SetEnvPrefix("C8Y")
		viper.AutomaticEnv()
	}

	if _, err := os.Stat(globalFlagSessionFile); err == nil {
		// Use config file from the flag.
		viper.SetConfigFile(globalFlagSessionFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			log.Panic(err)
		}

		// Search config in home directory with name ".cobra" (without extension).
		viper.AddConfigPath(".")
		viper.AddConfigPath(path.Join(home, ".cumulocity"))

		if globalFlagSessionFile != "" {
			viper.SetConfigName(globalFlagSessionFile)
		} else {
			viper.SetConfigName("session")
		}
	}

	if err := viper.ReadInConfig(); err == nil {
		log.Println("Using config file:", viper.ConfigFileUsed())
		client = c8y.NewClient(
			nil,
			formatHost(viper.GetString("host")),
			viper.GetString("tenant"),
			viper.GetString("username"),
			viper.GetString("password"),
			true,
		)
		return
	}

	// get session from environment variables
	client = c8y.NewClientFromEnvironment(nil, true)
}
