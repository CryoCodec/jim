package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// Custom completion command
var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate completion script",
	Long: `To load completions:

Bash:

  $ source <(jim completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ jim completion bash > /etc/bash_completion.d/jim
  # macOS:
  $ jim completion bash > /usr/local/etc/bash_completion.d/jim

Zsh:

  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ jim completion zsh > "${fpath[1]}/_jim"

  # You will need to start a new shell for this setup to take effect.

fish:

  $ jim completion fish | source

  # To load completions for each session, execute once:
  $ jim completion fish > ~/.config/fish/completions/jim.fish

PowerShell:

  PS> jim completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> jim completion powershell > jim.ps1
  # and source this file from your PowerShell profile.
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.ExactValidArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// completionCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// completionCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
