package main

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestAddRunCommand(t *testing.T) {
	root := &cobra.Command{Use: "niac"}
	services := new(serviceOptions)
	info := versionInfo{version: "test", commit: "abc", date: "now"}

	addRunCommand(root, services, info)

	cmd := findSubcommand(root, "run")
	if cmd == nil {
		t.Fatal("Expected run command to be registered")
	}

	expectedFlags := []string{"debug", "verbose", "quiet", "no-color", "tui", "dry-run"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("Expected --%s flag on run command", flag)
		}
	}
	for _, obsolete := range []string{"web", "port"} {
		if cmd.Flags().Lookup(obsolete) != nil {
			t.Errorf("obsolete --%s flag must not be registered", obsolete)
		}
	}
}

func TestRunOptionsStruct(t *testing.T) {
	opts := &runOptions{
		debugLevel: 2,
		verbose:    true,
		tui:        true,
	}

	if opts.debugLevel != 2 {
		t.Errorf("debugLevel = %d, want 2", opts.debugLevel)
	}
	if !opts.verbose {
		t.Error("Expected verbose=true")
	}
	if !opts.tui {
		t.Error("Expected tui=true")
	}
}
