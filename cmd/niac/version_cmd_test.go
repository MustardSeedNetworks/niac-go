package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func newVersionTestRoot(t *testing.T) (*cobra.Command, *bytes.Buffer) {
	t.Helper()
	info := versionInfo{
		version:     "v1.2.3",
		commit:      "abc1234",
		date:        "2026-09-03T00:00:00Z",
		uiBuildHash: "deadbeef",
	}
	root := newRootCommand(info, new(serviceOptions), func(_ []string) {
		t.Fatal("legacy runner must not run for a cobra subcommand")
	}, []func(*cobra.Command, *serviceOptions){
		func(root *cobra.Command, _ *serviceOptions) { addVersionCommand(root, info) },
	})
	out := new(bytes.Buffer)
	root.SetOut(out)
	root.SetErr(out)

	return root, out
}

// `niac version` is what people type. Without a command by that name it fell
// through to the legacy runner, which printed the usage banner and exited 0 --
// so a script asking the binary what it was got success and no version.
func TestVersionCommandPrintsTheVersion(t *testing.T) {
	root, out := newVersionTestRoot(t)
	root.SetArgs([]string{"version"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if got := out.String(); !strings.Contains(got, "v1.2.3") || !strings.Contains(got, "abc1234") {
		t.Fatalf("output = %q, want the version and commit", got)
	}
}

func TestVersionCommandJSONMatchesTheVersionEndpointKeys(t *testing.T) {
	root, out := newVersionTestRoot(t)
	root.SetArgs([]string{"version", "--json"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	var payload map[string]string
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("version --json is not JSON: %v (%q)", err, out.String())
	}
	// The same keys /__version serves, so a deployment check reads the binary
	// and the running daemon the same way.
	for _, key := range []string{"version", "commit", "buildTime", "uiBuildHash"} {
		if payload[key] == "" {
			t.Errorf("key %q missing from %v", key, payload)
		}
	}
}

// A mistyped subcommand is a usage error. It used to reach the legacy runner,
// which printed the usage banner -- indistinguishable from a run that had been
// asked to do something.
func TestRootRejectsAMistypedSubcommand(t *testing.T) {
	root, _ := newVersionTestRoot(t)
	root.SetArgs([]string{"totallybogus"})

	err := root.Execute()
	if err == nil {
		t.Fatal("Execute() = nil, want an unknown-command error")
	}
	if !isUnknownCommandError(err) {
		t.Fatalf("error = %v, want one isUnknownCommandError recognises (it selects exit %d)", err, exitUsage)
	}
}

// The legacy positional form still reaches the legacy runner: an interface and
// a config file is a run, not a typo.
func TestRootStillRoutesTheLegacyPositionalForm(t *testing.T) {
	info := versionInfo{version: "v1.2.3"}
	ran := false
	root := newRootCommand(info, new(serviceOptions), func(_ []string) { ran = true },
		[]func(*cobra.Command, *serviceOptions){
			func(root *cobra.Command, _ *serviceOptions) { addVersionCommand(root, info) },
		})
	root.SetOut(new(bytes.Buffer))
	root.SetArgs([]string{"lo0", "config.yaml"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !ran {
		t.Fatal("legacy runner was not called for the positional form")
	}
}
