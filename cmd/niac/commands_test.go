package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func newTestRoot() (*cobra.Command, *serviceOptions) {
	info := versionInfo{version: "test", commit: "abc123", date: "2024-01-01"}
	services := new(serviceOptions)
	root := &cobra.Command{Use: "niac"}
	root.SetVersionTemplate("test")
	_ = info
	return root, services
}

func TestAddAnalyzeCommand(t *testing.T) {
	root, services := newTestRoot()
	addAnalyzeCommand(root, services)

	cmd := findSubcommand(root, "analyze-walk")
	if cmd == nil {
		t.Fatal("Expected analyze-walk command to be registered")
	}
	if cmd.Use != "analyze-walk <walk-file>" {
		t.Errorf("Unexpected Use: %q", cmd.Use)
	}

	// Verify flags
	if cmd.Flags().Lookup("output") == nil {
		t.Error("Expected --output flag")
	}
	if cmd.Flags().Lookup("show-neighbors") == nil {
		t.Error("Expected --show-neighbors flag")
	}
	if cmd.Flags().Lookup("graphviz") == nil {
		t.Error("Expected --graphviz flag")
	}
}

func TestAddAnalyzePcapCommand(t *testing.T) {
	root, services := newTestRoot()
	addAnalyzePcapCommand(root, services)

	cmd := findSubcommand(root, "analyze-pcap")
	if cmd == nil {
		t.Fatal("Expected analyze-pcap command to be registered")
	}
	if cmd.Flags().Lookup("output") == nil {
		t.Error("Expected --output flag")
	}
}

func TestAddDumpCommand(t *testing.T) {
	root, services := newTestRoot()
	addDumpCommand(root, services)

	cmd := findSubcommand(root, "dump")
	if cmd == nil {
		t.Fatal("Expected dump command to be registered")
	}
	if cmd.Flags().Lookup("device") == nil {
		t.Error("Expected --device flag")
	}
	if cmd.Flags().Lookup("interface") == nil {
		t.Error("Expected --interface flag")
	}
	if cmd.Flags().Lookup("count") == nil {
		t.Error("Expected --count flag")
	}
	// The socket transport is gone; the daemon is reached over its API.
	if cmd.Flags().Lookup("api") == nil {
		t.Error("Expected --api flag")
	}
	if cmd.Flags().Lookup("json") == nil {
		t.Error("Expected --json flag")
	}
}

func TestAddStatusCommand(t *testing.T) {
	root, services := newTestRoot()
	addStatusCommand(root, services)

	cmd := findSubcommand(root, "status")
	if cmd == nil {
		t.Fatal("Expected status command to be registered")
	}
	if cmd.Flags().Lookup("json") == nil {
		t.Error("Expected --json flag")
	}
	// The socket transport is gone; the daemon is reached over its API.
	if cmd.Flags().Lookup("api") == nil {
		t.Error("Expected --api flag")
	}
}

func TestAddTemplateCommand(t *testing.T) {
	root, services := newTestRoot()
	addTemplateCommand(root, services)

	cmd := findSubcommand(root, "template")
	if cmd == nil {
		t.Fatal("Expected template command to be registered")
	}

	// Check subcommands
	subcommands := []string{"list", "show", "use", "apply"}
	for _, name := range subcommands {
		sub := findSubcommand(cmd, name)
		if sub == nil {
			t.Errorf("Expected template subcommand %q", name)
		}
	}
}

func TestAddListCommand(t *testing.T) {
	root, services := newTestRoot()
	addListCommand(root, services)

	cmd := findSubcommand(root, "list")
	if cmd == nil {
		t.Fatal("Expected list command to be registered")
	}
	if cmd.PersistentFlags().Lookup("root") == nil {
		t.Error("Expected --root flag")
	}

	subcommands := []string{"interfaces", "scenarios", "walks", "captures"}
	for _, name := range subcommands {
		sub := findSubcommand(cmd, name)
		if sub == nil {
			t.Errorf("Expected list subcommand %q", name)
		}
	}
}

func TestRunListScenariosIncludesLibraryNetworks(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "networks", "lab.yaml"), []byte(`# Description: Lab scenario
devices:
  - name: router-1
`))

	output := captureStdout(t, func() {
		if err := runListScenarios(&listOptions{root: root}); err != nil {
			t.Fatalf("runListScenarios: %v", err)
		}
	})

	if !strings.Contains(output, "Built-in templates:") {
		t.Fatalf("output missing built-in templates section:\n%s", output)
	}
	if !strings.Contains(output, "lab") {
		t.Fatalf("output missing library scenario:\n%s", output)
	}
}

func TestRunListFilesFiltersWalkPrefix(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "walks", "cisco", "switch.walk"), []byte("oid = STRING: x\n"))
	writeTestFile(t, filepath.Join(root, "walks", "juniper", "router.walk"), []byte("oid = STRING: y\n"))

	output := captureStdout(t, func() {
		err := runListFiles(&listOptions{root: root}, "walks", "cisco")
		if err != nil {
			t.Fatalf("runListFiles: %v", err)
		}
	})

	if !strings.Contains(output, "cisco/switch.walk") {
		t.Fatalf("output missing cisco walk:\n%s", output)
	}
	if strings.Contains(output, "juniper/router.walk") {
		t.Fatalf("output should not include filtered walk:\n%s", output)
	}
}

func TestRunListFilesCaptures(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "pcaps", "sample.pcap"), []byte{0x01, 0x02})

	output := captureStdout(t, func() {
		err := runListFiles(&listOptions{root: root}, "pcaps", "")
		if err != nil {
			t.Fatalf("runListFiles: %v", err)
		}
	})

	if !strings.Contains(output, "sample.pcap") {
		t.Fatalf("output missing capture:\n%s", output)
	}
}

func writeTestFile(t *testing.T, path string, body []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	original := os.Stdout
	readEnd, writeEnd, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = writeEnd
	defer func() {
		os.Stdout = original
	}()

	fn()

	if closeErr := writeEnd.Close(); closeErr != nil {
		t.Fatalf("close stdout pipe: %v", closeErr)
	}
	var buf bytes.Buffer
	if _, readErr := buf.ReadFrom(readEnd); readErr != nil {
		t.Fatalf("read stdout pipe: %v", readErr)
	}
	return buf.String()
}

func TestAddTopologyCommand(t *testing.T) {
	root, services := newTestRoot()
	addTopologyCommand(root, services)

	cmd := findSubcommand(root, "topology")
	if cmd == nil {
		t.Fatal("Expected topology command to be registered")
	}

	exportCmd := findSubcommand(cmd, "export")
	if exportCmd == nil {
		t.Fatal("Expected topology export subcommand")
	}
	if exportCmd.Flags().Lookup("format") == nil {
		t.Error("Expected --format flag")
	}
	if exportCmd.Flags().Lookup("output") == nil {
		t.Error("Expected --output flag")
	}
	// The socket transport is gone; the daemon is reached over its API.
	if exportCmd.Flags().Lookup("api") == nil {
		t.Error("Expected --api flag")
	}
}

func TestAddValidateCommand(t *testing.T) {
	root, services := newTestRoot()
	addValidateCommand(root, services)

	cmd := findSubcommand(root, "validate")
	if cmd == nil {
		t.Fatal("Expected validate command to be registered")
	}
	if cmd.Flags().Lookup("verbose") == nil {
		t.Error("Expected --verbose flag")
	}
	if cmd.Flags().Lookup("json") == nil {
		t.Error("Expected --json flag")
	}
}

func TestAddSanitizeCommand(t *testing.T) {
	root, services := newTestRoot()
	addSanitizeCommand(root, services)

	cmd := findSubcommand(root, "sanitize")
	if cmd == nil {
		t.Fatal("Expected sanitize command to be registered")
	}
	expectedFlags := []string{
		"mapping-file", "domain", "location", "contact",
		"community", "batch", "input-dir", "output-dir",
	}
	for _, flag := range expectedFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("Expected --%s flag", flag)
		}
	}
}

func TestAddInitCommand(t *testing.T) {
	root, services := newTestRoot()
	addInitCommand(root, services)

	cmd := findSubcommand(root, "init")
	if cmd == nil {
		t.Fatal("Expected init command to be registered")
	}
}

func TestAddCompletionCommand(t *testing.T) {
	root := &cobra.Command{Use: "niac"}
	addCompletionCommand(root)

	cmd := findSubcommand(root, "completion")
	if cmd == nil {
		t.Fatal("Expected completion command to be registered")
	}
}

func TestAddServiceCommand(t *testing.T) {
	root, services := newTestRoot()
	addServiceCommand(root, services)

	cmd := findSubcommand(root, "service")
	if cmd == nil {
		// On non-Windows, service may not register or may behave differently
		t.Skip("Service command may not be available on this platform")
	}
}

func TestAddDaemonCommand(t *testing.T) {
	root := &cobra.Command{Use: "niac"}
	info := versionInfo{version: "test", commit: "abc", date: "now"}
	addDaemonCommand(root, info)

	cmd := findSubcommand(root, "daemon")
	if cmd == nil {
		t.Fatal("Expected daemon command to be registered")
	}
}

func TestAddLogsCommand(t *testing.T) {
	root, services := newTestRoot()
	addLogsCommand(root, services)

	cmd := findSubcommand(root, "logs")
	if cmd == nil {
		t.Fatal("Expected logs command to be registered")
	}
}

func TestAddMonitorCommand(t *testing.T) {
	root, services := newTestRoot()
	addMonitorCommand(root, services)

	cmd := findSubcommand(root, "monitor")
	if cmd == nil {
		t.Fatal("Expected monitor command to be registered")
	}
}

func TestAddNeighborsCommand(t *testing.T) {
	root, services := newTestRoot()
	addNeighborsCommand(root, services)

	cmd := findSubcommand(root, "neighbors")
	if cmd == nil {
		t.Fatal("Expected neighbors command to be registered")
	}
}

func TestNewRootCommand(t *testing.T) {
	info := versionInfo{version: "1.0.0", commit: "abc123", date: "2024-01-01"}
	services := new(serviceOptions)
	legacyCalled := false

	root := newRootCommand(info, services, func(_ []string) {
		legacyCalled = true
	}, []func(*cobra.Command, *serviceOptions){
		addConfigCommand,
		addValidateCommand,
	})

	if root == nil {
		t.Fatal("newRootCommand returned nil")
	}
	if root.Use != "niac" {
		t.Errorf("Expected root.Use = 'niac', got %q", root.Use)
	}

	// Check that subcommands were added
	configCmd := findSubcommand(root, "config")
	if configCmd == nil {
		t.Error("Expected config command to be added")
	}
	validateCmd := findSubcommand(root, "validate")
	if validateCmd == nil {
		t.Error("Expected validate command to be added")
	}

	// Verify persistent flags
	if root.PersistentFlags().Lookup("api-listen") != nil {
		t.Error("obsolete --api-listen persistent flag must not be registered")
	}
	if root.PersistentFlags().Lookup("api-token") != nil {
		t.Error("obsolete global --api-token persistent flag must not be registered")
	}
	if root.PersistentFlags().Lookup("metrics-listen") != nil {
		t.Error("obsolete --metrics-listen persistent flag must not be registered")
	}
	if root.PersistentFlags().Lookup("storage-path") == nil {
		t.Error("Expected --storage-path persistent flag")
	}

	_ = legacyCalled
}

func TestRootCommandForwardsLegacyArgs(t *testing.T) {
	info := versionInfo{version: "1.0.0", commit: "abc123", date: "2024-01-01"}
	services := new(serviceOptions)
	var gotArgs []string

	root := newRootCommand(info, services, func(args []string) {
		gotArgs = append([]string(nil), args...)
	}, nil)
	root.SetArgs([]string{"lo0", "config.yaml"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	wantArgs := []string{"lo0", "config.yaml"}
	if len(gotArgs) != len(wantArgs)+1 {
		t.Fatalf("legacy runner args length = %d, want %d; args=%v", len(gotArgs), len(wantArgs)+1, gotArgs)
	}
	for i, want := range wantArgs {
		if gotArgs[i+1] != want {
			t.Fatalf("legacy runner arg %d = %q, want %q; args=%v", i+1, gotArgs[i+1], want, gotArgs)
		}
	}
}

func TestShouldUseLegacyCommand(t *testing.T) {
	info := versionInfo{version: "1.0.0", commit: "abc123", date: "2024-01-01"}
	services := new(serviceOptions)
	root := newRootCommand(info, services, func(_ []string) {}, []func(*cobra.Command, *serviceOptions){
		func(root *cobra.Command, services *serviceOptions) { addRunCommand(root, services, info) },
		addValidateCommand,
	})

	tests := []struct {
		name string
		args []string
		want bool
	}{
		{name: "legacy positional", args: []string{"en0", "config.yaml"}, want: true},
		{name: "legacy dry run", args: []string{"--dry-run", "lo0", "config.yaml"}, want: true},
		{name: "legacy verbose with value flag", args: []string{"--debug", "2", "en0", "config.yaml"}, want: true},
		{name: "legacy informational flag", args: []string{"--list-interfaces"}, want: true},
		{name: "cobra subcommand", args: []string{"run", "en0", "config.yaml"}, want: false},
		{name: "help", args: []string{"help"}, want: false},
		{name: "double dash compatibility", args: []string{"--", "--dry-run", "lo0", "config.yaml"}, want: false},
		{name: "no args", args: nil, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldUseLegacyCommand(tt.args, root); got != tt.want {
				t.Fatalf("shouldUseLegacyCommand(%v) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}

func TestVersionInfoFunctions(t *testing.T) {
	t.Run("readVersionInfo", func(t *testing.T) {
		info := readVersionInfo()
		if info.version == "" {
			t.Error("Expected version to be non-empty")
		}
	})
}

// findSubcommand finds a subcommand by name.
func findSubcommand(cmd *cobra.Command, name string) *cobra.Command {
	for _, sub := range cmd.Commands() {
		if sub.Name() == name {
			return sub
		}
	}
	return nil
}
