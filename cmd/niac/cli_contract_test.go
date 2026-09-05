package main

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// The legacy positional runtime (`niac en0 config.yaml`) and cobra `run` both
// built their own protocol stack, bypassing the session registry, admission
// budgets and preflight. `daemon --once` replaced them. These tests pin what a
// caller of the old forms now gets, so the deletion cannot quietly come back as
// a silent success.

func newContractRoot(t *testing.T) *cobra.Command {
	t.Helper()

	info := versionInfo{version: "test", commit: "abc123", date: "2024-01-01"}
	root := newRootCommand(info, new(serviceOptions), commandBuilders(info))
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)

	return root
}

func TestLegacyPositionalFormIsAUsageError(t *testing.T) {
	root := newContractRoot(t)
	root.SetArgs([]string{"en0", "config.yaml"})

	err := root.Execute()
	if err == nil {
		t.Fatal("`niac en0 config.yaml` succeeded; the legacy runtime is gone and it must report a usage error")
	}
	if got := exitCodeForError(err); got != exitUsage {
		t.Errorf("exit code = %d, want %d for %v", got, exitUsage, err)
	}
}

func TestLegacyFlagIsAUsageError(t *testing.T) {
	for _, flag := range []string{"--list-interfaces", "--dry-run", "--debug-lldp"} {
		t.Run(flag, func(t *testing.T) {
			root := newContractRoot(t)
			root.SetArgs([]string{flag, "en0", "config.yaml"})

			err := root.Execute()
			if err == nil {
				t.Fatalf("`niac %s en0 config.yaml` succeeded; legacy flags are gone", flag)
			}
			if got := exitCodeForError(err); got != exitUsage {
				t.Errorf("exit code = %d, want %d for %v", got, exitUsage, err)
			}
		})
	}
}

func TestRunCommandIsGone(t *testing.T) {
	root := newContractRoot(t)

	for _, cmd := range root.Commands() {
		if cmd.Name() == "run" {
			t.Fatal("`niac run` is still registered; `daemon --once` is the one foreground runtime")
		}
	}
}

func TestExitCodeForError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"nil", nil, 0},
		{"unknown command", errors.New(`unknown command "bogus" for "niac"`), exitUsage},
		{"unknown flag", errors.New("unknown flag: --list-interfaces"), exitUsage},
		{"unknown shorthand", errors.New(`unknown shorthand flag: 'd' in -d`), exitUsage},
		{"coded", withExitCode(3, errors.New("boom")), 3},
		{"plain failure", errors.New("the simulation stopped"), 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := exitCodeForError(tt.err); got != tt.want {
				t.Errorf("exitCodeForError(%v) = %d, want %d", tt.err, got, tt.want)
			}
		})
	}
}

func TestRootHelpDoesNotAdvertiseALegacyForm(t *testing.T) {
	root := newContractRoot(t)

	if strings.Contains(root.Example, "legacy") || strings.Contains(root.Example, "niac run ") {
		t.Errorf("root Example still advertises a deleted runtime:\n%s", root.Example)
	}
}
