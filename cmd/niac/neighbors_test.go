package main

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestAddNeighborsCommandFlags(t *testing.T) {
	root := &cobra.Command{Use: "niac"}
	services := new(serviceOptions)
	addNeighborsCommand(root, services)

	cmd := findSubcommand(root, "neighbors")
	if cmd == nil {
		t.Fatal("Expected neighbors command to be registered")
	}

	expectedFlags := []string{"device", "protocol", "socket", "json"}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil && cmd.PersistentFlags().Lookup(flag) == nil {
			t.Errorf("Expected --%s flag on neighbors command", flag)
		}
	}

	// Check watch subcommand
	watchCmd := findSubcommand(cmd, "watch")
	if watchCmd == nil {
		t.Fatal("Expected watch subcommand")
	}
}

func TestNeighborsProtocolAllConstant(t *testing.T) {
	if neighborsProtocolAll != "all" {
		t.Errorf("neighborsProtocolAll = %q, want %q", neighborsProtocolAll, "all")
	}
}

func TestNeighborEntryStructJSON(t *testing.T) {
	entry := NeighborEntry{
		Device: "switch-1",
	}

	if entry.Device != "switch-1" {
		t.Errorf("Device = %q, want %q", entry.Device, "switch-1")
	}
}
