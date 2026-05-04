package main

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestCompletionCommandBash(t *testing.T) {
	root := &cobra.Command{Use: "niac"}
	addCompletionCommand(root)

	root.SetArgs([]string{"completion", "bash"})
	err := root.Execute()
	if err != nil {
		t.Errorf("completion bash failed: %v", err)
	}
}

func TestCompletionCommandZsh(t *testing.T) {
	root := &cobra.Command{Use: "niac"}
	addCompletionCommand(root)

	root.SetArgs([]string{"completion", "zsh"})
	err := root.Execute()
	if err != nil {
		t.Errorf("completion zsh failed: %v", err)
	}
}

func TestCompletionCommandFish(t *testing.T) {
	root := &cobra.Command{Use: "niac"}
	addCompletionCommand(root)

	root.SetArgs([]string{"completion", "fish"})
	err := root.Execute()
	if err != nil {
		t.Errorf("completion fish failed: %v", err)
	}
}

func TestCompletionCommandPowershell(t *testing.T) {
	root := &cobra.Command{Use: "niac"}
	addCompletionCommand(root)

	root.SetArgs([]string{"completion", "powershell"})
	err := root.Execute()
	if err != nil {
		t.Errorf("completion powershell failed: %v", err)
	}
}
