package cmd

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/authentication"
	"github.com/spf13/cobra"
)

// ---------------------------------------------------------------------------
// Shared client factory
// ---------------------------------------------------------------------------

var (
	sharedClient *api.Client
	clientOnce   sync.Once

	// auth flags - each falls back to the corresponding environment variable
	hostFlag     string
	tenantFlag   string
	userFlag     string
	passwordFlag string
	tokenFlag    string

	verboseFlag bool
	dryRunFlag  bool
)

// orEnv returns flagVal if non-empty, otherwise the first non-empty env variable.
func orEnv(flagVal string, envKeys ...string) string {
	if flagVal != "" {
		return flagVal
	}
	return authentication.GetEnvValue(envKeys...)
}

// clientFactory returns the lazily-initialised API client.
// Commands must only call this after PersistentPreRunE has run (inside RunE).
func clientFactory() *api.Client { return sharedClient }

// requestContext returns a context pre-configured with CLI-level settings.
// Pass it to every SDK call so flags like --dry-run take effect.
func requestContext() context.Context {
	return api.WithDryRun(context.Background(), dryRunFlag)
}

// ---------------------------------------------------------------------------
// Root command
// ---------------------------------------------------------------------------

// rootCmd is the top-level cobra command.
//
// DESIGN: The root command initialises the shared API client once in
// PersistentPreRunE so all sub-commands share one authenticated session.
// Running --help never triggers PersistentPreRunE, so no credentials are
// required just to read the help text.
var rootCmd = &cobra.Command{
	Use:           "c8y-demo",
	Short:         "Minimal Cumulocity CLI powered by go-c8y SDK",
	SilenceErrors: true,
	SilenceUsage:  true,
	Long: `c8y-demo - a minimalist CLI demonstrating how the go-c8y SDK drives a Cobra CLI.

CORE DESIGN: SDK options structs are the single source of truth.

  Field names    -> flag names  (url struct tag: url:"severity" -> --severity)
  Field comments -> help text   (copied verbatim into flag descriptions)
  Field types    -> flag types  (string, []string, bool, time.Time via adapter)
  Embedded types -> flag groups (PaginationOptions gives --page-size, --max-items)

When the SDK adds a new query parameter, the CLI picks it up with one new flag
binding: no spec, no generated code, no extra docs to keep in sync.

AUTHENTICATION - flags take priority over environment variables:
  --host      / C8Y_HOST (or C8Y_BASEURL, C8Y_URL)
  --tenant    / C8Y_TENANT
  --user      / C8Y_USERNAME (or C8Y_USER)
  --password  / C8Y_PASSWORD
  --token     / C8Y_TOKEN  (bearer token; skips username/password)

EXAMPLES:
  c8y-demo alarms list --severity MAJOR --dateFrom "-1h"
  c8y-demo alarms list --device "name:myDevice" --status ACTIVE --max-items 50
  c8y-demo alarms count --status ACTIVE
  c8y-demo alarms get 12345
  c8y-demo alarms create --device 12345 --type c8y_TestAlarm --text "Test" --severity MINOR`,

	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		host := orEnv(hostFlag, authentication.EnvironmentHost...)
		if host == "" {
			return fmt.Errorf("host is required: use --host or set C8Y_HOST")
		}
		clientOnce.Do(func() {
			auth := authentication.AuthOptions{
				Tenant:   orEnv(tenantFlag, authentication.EnvironmentTenant...),
				Username: orEnv(userFlag, authentication.EnvironmentUsername...),
				Password: orEnv(passwordFlag, authentication.EnvironmentPassword...),
				Token:    orEnv(tokenFlag, authentication.EnvironmentToken...),
			}
			sharedClient = api.NewClient(api.ClientOptions{
				BaseURL: host,
				Auth:    auth,
			})
			if verboseFlag {
				sharedClient.Client.SetDebug(true)
			}
		})
		return nil
	},
}

// Execute is called from main.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func init() {
	f := rootCmd.PersistentFlags()

	// Authentication
	f.StringVar(&hostFlag, "host", "", "Cumulocity base URL, e.g. https://mytenant.cumulocity.com (env: C8Y_HOST)")
	f.StringVar(&tenantFlag, "tenant", "", "Tenant ID (env: C8Y_TENANT)")
	f.StringVar(&userFlag, "user", "", "Username (env: C8Y_USERNAME)")
	f.StringVar(&passwordFlag, "password", "", "Password (env: C8Y_PASSWORD)")
	f.StringVar(&tokenFlag, "token", "", "Bearer token; skips username/password auth (env: C8Y_TOKEN)")

	// Behaviour
	f.BoolVar(&verboseFlag, "verbose", false, "Enable verbose/debug output")
	f.BoolVar(&dryRunFlag, "dry-run", false, "Print the HTTP request without actually sending it")

	rootCmd.AddCommand(newAlarmsCmd())
}
