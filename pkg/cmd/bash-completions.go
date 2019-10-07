package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

func init() {
	var completionCmd = &cobra.Command{
		Use:   "completion",
		Short: "Generates bash completion scripts",
		Long: `To load completion run

. <(bitbucket completion)

To configure your bash shell to load completions for each session add to your bashrc

# ~/.bashrc or ~/.profile
. <(bitbucket completion)
`,
		Run: func(cmd *cobra.Command, args []string) {
			rootCmd.GenBashCompletion(os.Stdout)
		},
	}
	var completionBashCmd = &cobra.Command{
		Use:   "bash",
		Short: "Generates bash completion scripts",
		Long: `To load completion run

. <(bitbucket completion)

To configure your bash shell to load completions for each session add to your bashrc

# ~/.bashrc or ~/.profile
. <(bitbucket completion)
`,
		Run: func(cmd *cobra.Command, args []string) {
			rootCmd.GenBashCompletion(os.Stdout)
		},
	}

	var completionPowershellCmd = &cobra.Command{
		Use:   "powershell",
		Short: "Generates powershell completion scripts",
		Long: `To load completion run

. <(bitbucket completion)

To configure your bash shell to load completions for each session add to your bashrc

# ~/.bashrc or ~/.profile
. <(bitbucket completion)
`,
		Run: func(cmd *cobra.Command, args []string) {
			rootCmd.GenPowerShellCompletion(os.Stdout)
		},
	}

	completionCmd.AddCommand(completionBashCmd)
	completionCmd.AddCommand(completionPowershellCmd)
	rootCmd.AddCommand(completionCmd)
}
