package cmd

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/reubenmiller/go-c8y/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var Logger *logger.Logger

// Build data
// These variables should be set using the -ldflags "-X github.com/reubenmiller/go-c8y/pkg/cmd.version=1.0.0" when running go build
var buildVersion string
var buildBranch string

const (
	module = "c8yapi"
)

func init() {
	Logger = logger.NewDummyLogger(module)
}

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
	globalFlagOutputFile     string
	globalFlagUseEnv         bool
	globalFlagRaw            bool
	globalFlagProxy          string
	globalFlagNoProxy        bool
	globalFlagTimeout        uint
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
	rootCmd.PersistentFlags().StringVar(&globalFlagProxy, "proxy", "", "Proxy setting")
	rootCmd.PersistentFlags().BoolVar(&globalFlagNoProxy, "noProxy", false, "Ignore the proxy settings")

	rootCmd.PersistentFlags().StringVar(&globalFlagOutputFile, "outputFile", "", "Output file")

	rootCmd.PersistentFlags().StringSlice("filter", nil, "filter")
	rootCmd.PersistentFlags().StringSlice("select", nil, "select")
	rootCmd.PersistentFlags().String("format", "", "format")
	rootCmd.PersistentFlags().UintVarP(&globalFlagTimeout, "timeout", "t", 10*60*1000, "Timeout in milliseconds")

	// TODO: Make flags case-insensitive
	// rootCmd.PersistentFlags().SetNormalizeFunc(flagNormalizeFunc)

	rootCmd.AddCommand(newCompletionsCmd().getCommand())
	rootCmd.AddCommand(newVersionCmd().getCommand())

	rootCmd.AddCommand(newDeviceRootCmd().getCommand())
	rootCmd.AddCommand(newRealtimeCmd().getCommand())
	rootCmd.AddCommand(newSessionsRootCmd().getCommand())

	// generic commands
	rootCmd.AddCommand(newGetGenericRestCmd().getCommand())

	// Auto generated commands
	// alarms commands
	alarms := newAlarmsRootCmd().getCommand()
	alarms.AddCommand(newSubscribeAlarmCmd().getCommand())
	rootCmd.AddCommand(alarms)

	// applications commands
	rootCmd.AddCommand(newApplicationsRootCmd().getCommand())

	// auditRecords commands
	rootCmd.AddCommand(newAuditRecordsRootCmd().getCommand())

	// binaries commands
	rootCmd.AddCommand(newBinariesRootCmd().getCommand())

	// currentApplication commands
	rootCmd.AddCommand(newCurrentApplicationRootCmd().getCommand())

	// operations commands
	operations := newOperationsRootCmd().getCommand()
	operations.AddCommand(newSubscribeOperationCmd().getCommand())
	rootCmd.AddCommand(operations)

	// events commands
	events := newEventsRootCmd().getCommand()
	events.AddCommand(newSubscribeEventCmd().getCommand())
	rootCmd.AddCommand(events)

	// identity commands
	rootCmd.AddCommand(newIdentityRootCmd().getCommand())

	// inventory commands
	inventory := newInventoryRootCmd().getCommand()
	inventory.AddCommand(newSubscribeManagedObjectCmd().getCommand())
	rootCmd.AddCommand(inventory)

	// inventoryReferences commands
	rootCmd.AddCommand(newInventoryReferencesRootCmd().getCommand())

	// measurements commands
	measurements := newMeasurementsRootCmd().getCommand()
	measurements.AddCommand(newSubscribeMeasurementCmd().getCommand())
	rootCmd.AddCommand(measurements)

	// microservices commands
	microservices := newMicroservicesRootCmd().getCommand()
	microservices.AddCommand(newNewMicroserviceCmd().getCommand())
	rootCmd.AddCommand(microservices)

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
	// Set logging
	if globalFlagVerbose || globalFlagDryRun {
		Logger = logger.NewLogger(module)
	} else {
		// Disable log messages
		Logger = logger.NewDummyLogger(module)
		c8y.SilenceLogger()
	}

	if globalFlagSessionFile == "" && os.Getenv("C8Y_SESSION") != "" {
		globalFlagSessionFile = os.Getenv("C8Y_SESSION")
		Logger.Printf("Using session environment variable: %s\n", globalFlagSessionFile)
	}

	// global session flag has precendence over use environment
	if globalFlagSessionFile != "" && os.Getenv("C8Y_USE_ENVIRONMENT") != "" {
		globalFlagUseEnv = true
	}

	// only parse env variables if no explict config file is given
	if globalFlagUseEnv {
		Logger.Println("C8Y_USE_ENVIRONMENT is set. Environment variables can be used to override config settings")
		viper.SetEnvPrefix("C8Y")
		viper.AutomaticEnv()
	}

	// Proxy settings
	// Either use explicit proxy, ignore proxy, or use existing env variables
	// --proxy "http://10.0.0.1:8080"
	// --noProxy
	// HTTP_PROXY=http://10.0.0.1:8080
	// NO_PROXY=localhost,127.0.0.1
	explictProxy := rootCmd.Flags().Changed("proxy")
	explictNoProxy := rootCmd.Flags().Changed("noProxy")
	if explictNoProxy {
		Logger.Debug("using explicit noProxy setting")
		os.Setenv("HTTP_PROXY", "")
		os.Setenv("HTTPS_PROXY", "")
		os.Setenv("http_proxy", "")
		os.Setenv("https_proxy", "")
	} else {
		if explictProxy {
			Logger.Debugf("using explicit proxy [%s]", globalFlagProxy)

			globalFlagProxy = strings.TrimSpace(globalFlagProxy)

			os.Setenv("HTTP_PROXY", globalFlagProxy)
			os.Setenv("HTTPS_PROXY", globalFlagProxy)
			os.Setenv("http_proxy", globalFlagProxy)
			os.Setenv("https_proxy", globalFlagProxy)

		} else {
			Logger.Debugf(
				"using existing env variables. HTTP_PROXY [%s], http_proxy [%s], HTTPS_PROXY [%s], https_proxy [%s], NO_PROXY [%s], no_proxy [%s]",
				os.Getenv("HTTP_PROXY"),
				os.Getenv("http_proxy"),
				os.Getenv("HTTPS_PROXY"),
				os.Getenv("https_proxy"),
				os.Getenv("NO_PROXY"),
				os.Getenv("no_proxy"),
			)
		}
	}

	if _, err := os.Stat(globalFlagSessionFile); err == nil {
		// Use config file from the flag.
		viper.SetConfigFile(globalFlagSessionFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			Logger.Panic(err)
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

	httpClient := newHTTPClient(globalFlagNoProxy)

	// Try reading session from file
	if err := viper.ReadInConfig(); err == nil {
		Logger.Println("Using config file:", viper.ConfigFileUsed())
		client = c8y.NewClient(
			httpClient,
			formatHost(viper.GetString("host")),
			viper.GetString("tenant"),
			viper.GetString("username"),
			viper.GetString("password"),
			true,
		)
	} else {
		// Fallback to reading session from environment variables
		client = c8y.NewClientFromEnvironment(httpClient, true)
	}

	// Add the realtime client
	client.Realtime = c8y.NewRealtimeClient(
		client.BaseURL.String(),
		newWebsocketDialer(globalFlagNoProxy),
		client.TenantName,
		client.Username,
		client.Password,
	)
}

func newWebsocketDialer(ignoreProxySettings bool) *websocket.Dialer {
	dialer := &websocket.Dialer{
		Proxy:             http.ProxyFromEnvironment,
		HandshakeTimeout:  10 * time.Second,
		EnableCompression: false,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	if ignoreProxySettings {
		dialer.Proxy = nil
	}

	return dialer
}

func newHTTPClient(ignoreProxySettings bool) *http.Client {
	// Default client ignores self signed certificates (to enable compatibility to the edge which uses self signed certs)
	defaultTransport := http.DefaultTransport.(*http.Transport)
	tr := &http.Transport{
		Proxy:                 defaultTransport.Proxy,
		DialContext:           defaultTransport.DialContext,
		MaxIdleConns:          defaultTransport.MaxIdleConns,
		IdleConnTimeout:       defaultTransport.IdleConnTimeout,
		ExpectContinueTimeout: defaultTransport.ExpectContinueTimeout,
		TLSHandshakeTimeout:   defaultTransport.TLSHandshakeTimeout,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	if ignoreProxySettings {
		tr.Proxy = nil
	}

	return &http.Client{
		Transport: tr,
	}
}
