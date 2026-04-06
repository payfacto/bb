package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion <shell>",
	Short: "Generate shell completion script",
	Long: `Generate a shell completion script for bb.

Bash:
  bb completion bash > /etc/bash_completion.d/bb
  source /etc/bash_completion.d/bb

Zsh:
  bb completion zsh > "${fpath[1]}/_bb"

Fish:
  bb completion fish > ~/.config/fish/completions/bb.fish

PowerShell:
  bb completion powershell | Out-String | Invoke-Expression`,
	ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
	Args:      cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletion(os.Stdout)
		case "zsh":
			return rootCmd.GenZshCompletion(os.Stdout)
		case "fish":
			return rootCmd.GenFishCompletion(os.Stdout, true)
		case "powershell":
			return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
		default:
			return cmd.Help()
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
