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
	Use:   "cy",
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

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
