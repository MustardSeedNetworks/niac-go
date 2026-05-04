package main

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestAddManCommand(t *testing.T) {
	root := &cobra.Command{Use: "niac"}
	info := versionInfo{version: "test", commit: "abc", date: "now"}
	addManCommand(root, info)

	cmd := findSubcommand(root, "man")
	if cmd == nil {
		t.Fatal("Expected man command to be registered")
	}

	if !cmd.Hidden {
		t.Error("Expected man command to be hidden")
	}
}
