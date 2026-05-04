package main

import (
	"os"

	"github.com/spf13/cobra"
)

const completionLongHelp = `Generate shell completion script for NIAC.

To load completions:

Bash:
  $ source <(niac completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ niac completion bash > /etc/bash_completion.d/niac
  # macOS:
  $ niac completion bash > $(brew --prefix)/etc/bash_completion.d/niac

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ niac completion zsh > "${fpath[1]}/_niac"

  # You will need to start a new shell for this setup to take effect.

Fish:
  $ niac completion fish | source

  # To load completions for each session, execute once:
  $ niac completion fish > ~/.config/fish/completions/niac.fish

PowerShell:
  PS> niac completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> niac completion powershell > niac.ps1
  # and source this file from your PowerShell profile.
`

func addCompletionCommand(root *cobra.Command) {
	completionCmd := &cobra.Command{
		Use:                   "completion [bash|zsh|fish|powershell]",
		Short:                 "Generate completion script",
		Long:                  completionLongHelp,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE:                  runCompletion,
	}

	root.AddCommand(completionCmd)
}

// runCompletion dispatches to the shell-specific completion generator.
func runCompletion(cmd *cobra.Command, args []string) error {
	switch args[0] {
	case "bash":
		return cmd.Root().GenBashCompletion(os.Stdout)
	case "zsh":
		return cmd.Root().GenZshCompletion(os.Stdout)
	case "fish":
		return cmd.Root().GenFishCompletion(os.Stdout, true)
	case "powershell":
		return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
	}
	return nil
}
