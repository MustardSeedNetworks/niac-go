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

	expectedFlags := []string{"debug", "verbose", "quiet", "no-color", "tui", "web", "port", "dry-run"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("Expected --%s flag on run command", flag)
		}
	}
}

func TestRunOptionsStruct(t *testing.T) {
	opts := &runOptions{
		debugLevel: 2,
		verbose:    true,
		tui:        true,
		web:        true,
		webPort:    "9000",
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
	if !opts.web {
		t.Error("Expected web=true")
	}
	if opts.webPort != "9000" {
		t.Errorf("webPort = %q, want %q", opts.webPort, "9000")
	}
}
