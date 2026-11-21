package main

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestAddDaemonCommandFlags(t *testing.T) {
	root := &cobra.Command{Use: "niac"}
	info := versionInfo{version: "test", commit: "abc", date: "now"}
	addDaemonCommand(root, info)

	cmd := findSubcommand(root, "daemon")
	if cmd == nil {
		t.Fatal("Expected daemon command to be registered")
	}

	expectedFlags := []string{"attachment-policy", "listen", "api-token", "storage"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("Expected --%s flag on daemon command", flag)
		}
	}
	if flagType := cmd.Flags().Lookup("attachment-policy").Value.Type(); flagType != "stringArray" {
		t.Fatalf("--attachment-policy type = %q, want stringArray", flagType)
	}
}

func TestDaemonOptionsStruct(t *testing.T) {
	opts := &daemonOptions{
		listen:      ":9090",
		storagePath: "/custom/path.db",
	}

	if opts.listen != ":9090" {
		t.Errorf("listen = %q, want %q", opts.listen, ":9090")
	}
	if opts.storagePath != "/custom/path.db" {
		t.Errorf("storagePath = %q, want %q", opts.storagePath, "/custom/path.db")
	}
}
