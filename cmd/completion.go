package cmd

import (
	"fmt"
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
			handleErr(cmd.Root().GenBashCompletion(os.Stdout))
		case "zsh":
			handleErr(cmd.Root().GenZshCompletion(os.Stdout))
		case "fish":
			handleErr(cmd.Root().GenFishCompletion(os.Stdout, true))
		case "powershell":
			handleErr(cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout))
		}
	},
}

func handleErr(err error) {
	if err != nil {
		fmt.Printf("something went wrong: %s", err)
	}
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
