package main

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestAddInteractiveCommand(t *testing.T) {
	root := &cobra.Command{Use: "niac"}
	services := new(serviceOptions)
	addInteractiveCommand(root, services)

	cmd := findSubcommand(root, "interactive")
	if cmd == nil {
		t.Fatal("Expected interactive command to be registered")
	}

	expectedFlags := []string{"debug", "verbose", "quiet", "no-color"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("Expected --%s flag", flag)
		}
	}
}

func TestInteractiveOptionsStruct(t *testing.T) {
	opts := &interactiveOptions{
		debugLevel: 3,
	}

	if opts.debugLevel != 3 {
		t.Errorf("debugLevel = %d, want 3", opts.debugLevel)
	}
}
