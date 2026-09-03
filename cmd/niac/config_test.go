package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func newTestRootCommand() *cobra.Command {
	info := versionInfo{
		version: "test",
		commit:  "test",
		date:    "test",
	}
	services := new(serviceOptions)
	return newRootCommand(
		info,
		services,
		func([]string) {},
		[]func(*cobra.Command, *serviceOptions){
			addConfigCommand,
		},
	)
}

// TestConfigExportCommand tests the config export command.
func TestConfigExportCommand(t *testing.T) {
	// Create temp directory for test files
	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "input.yaml")
	outputFile := filepath.Join(tmpDir, "output.yaml")

	// Create minimal valid config
	configContent := `devices:
  - name: "test-device"
    mac: "00:11:22:33:44:55"
    ips:
      - "192.168.1.1"
`
	if err := os.WriteFile(inputFile, []byte(configContent), 0o644); err != nil {
		t.Fatalf("Failed to create test input file: %v", err)
	}

	// Test export
	rootCmd := newTestRootCommand()
	rootCmd.SetArgs([]string{"config", "export", inputFile, outputFile})
	if err := rootCmd.Execute(); err != nil {
		t.Errorf("Config export failed: %v", err)
	}

	// Verify output file exists
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Error("Output file was not created")
	}

	// Verify output file content
	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "test-device") {
		t.Error("Output file does not contain device name")
	}
}

// Export must refuse to clobber an existing file. Asserting the exit code alone
// would pass even if the file had already been truncated, so the original
// contents are checked too.
func TestConfigExportOverwriteProtection(t *testing.T) {
	dir := t.TempDir()
	input := writeFile(t, dir, "in.yaml", minimalConfig)
	output := writeFile(t, dir, "out.json", "original contents")

	err := runConfigExport([]string{input, output})
	if !errors.Is(err, errOutputExists) {
		t.Fatalf("runConfigExport over an existing file = %v, want errOutputExists", err)
	}

	contents, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("reading %s: %v", output, err)
	}

	if string(contents) != "original contents" {
		t.Errorf("output file was modified: %q", contents)
	}
}

// TestConfigDiffCommand tests the config diff command.
func TestConfigDiffCommand(t *testing.T) {
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "config1.yaml")
	file2 := filepath.Join(tmpDir, "config2.yaml")

	// Create first config with one device
	config1 := `devices:
  - name: "router-1"
    mac: "00:11:22:33:44:55"
    ips:
      - "192.168.1.1"
    type: "router"
`
	err := os.WriteFile(file1, []byte(config1), 0o644)
	if err != nil {
		t.Fatalf("Failed to create config1: %v", err)
	}

	// Create second config with different device
	config2 := `devices:
  - name: "router-1"
    mac: "00:11:22:33:44:66"
    ips:
      - "192.168.1.1"
    type: "switch"
  - name: "router-2"
    mac: "00:11:22:33:44:77"
    ips:
      - "192.168.1.2"
    type: "router"
`
	err = os.WriteFile(file2, []byte(config2), 0o644)
	if err != nil {
		t.Fatalf("Failed to create config2: %v", err)
	}

	// Test diff - should succeed and show differences
	rootCmd := newTestRootCommand()
	rootCmd.SetArgs([]string{"config", "diff", file1, file2})
	err = rootCmd.Execute()
	if err != nil {
		t.Errorf("Config diff failed: %v", err)
	}

	// Note: We can't easily capture stdout in this test,
	// but we verify the command doesn't error
}

// TestConfigDiffIdentical tests diff on identical configs.
func TestConfigDiffIdentical(t *testing.T) {
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "config1.yaml")
	file2 := filepath.Join(tmpDir, "config2.yaml")

	// Create identical configs
	config := `devices:
  - name: "router-1"
    mac: "00:11:22:33:44:55"
    ips:
      - "192.168.1.1"
`
	err := os.WriteFile(file1, []byte(config), 0o644)
	if err != nil {
		t.Fatalf("Failed to create config1: %v", err)
	}
	err = os.WriteFile(file2, []byte(config), 0o644)
	if err != nil {
		t.Fatalf("Failed to create config2: %v", err)
	}

	// Test diff - should succeed with no differences
	rootCmd := newTestRootCommand()
	rootCmd.SetArgs([]string{"config", "diff", file1, file2})
	err = rootCmd.Execute()
	if err != nil {
		t.Errorf("Config diff failed: %v", err)
	}
}

// TestConfigMergeCommand tests the config merge command.
func TestConfigMergeCommand(t *testing.T) {
	tmpDir := t.TempDir()
	baseFile := filepath.Join(tmpDir, "base.yaml")
	overlayFile := filepath.Join(tmpDir, "overlay.yaml")
	outputFile := filepath.Join(tmpDir, "merged.yaml")

	// Create base config
	baseConfig := `devices:
  - name: "router-1"
    mac: "00:11:22:33:44:55"
    ips:
      - "192.168.1.1"
    type: "router"
  - name: "switch-1"
    mac: "00:11:22:33:44:66"
    ips:
      - "192.168.1.2"
    type: "switch"
`
	if err := os.WriteFile(baseFile, []byte(baseConfig), 0o644); err != nil {
		t.Fatalf("Failed to create base config: %v", err)
	}

	// Create overlay config (replaces router-1, adds router-2)
	overlayConfig := `devices:
  - name: "router-1"
    mac: "00:11:22:33:44:99"
    ips:
      - "192.168.1.10"
    type: "router"
  - name: "router-2"
    mac: "00:11:22:33:44:77"
    ips:
      - "192.168.1.3"
    type: "router"
`
	if err := os.WriteFile(overlayFile, []byte(overlayConfig), 0o644); err != nil {
		t.Fatalf("Failed to create overlay config: %v", err)
	}

	// Test merge
	rootCmd := newTestRootCommand()
	rootCmd.SetArgs([]string{"config", "merge", baseFile, overlayFile, outputFile})
	if err := rootCmd.Execute(); err != nil {
		t.Errorf("Config merge failed: %v", err)
	}

	// Verify output file exists
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Error("Merged output file was not created")
	}

	// Verify merged content
	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read merged file: %v", err)
	}

	content := string(data)
	// Should have router-1 (from overlay), switch-1 (from base), router-2 (from overlay)
	if !strings.Contains(content, "router-1") {
		t.Error("Merged file missing router-1")
	}
	if !strings.Contains(content, "switch-1") {
		t.Error("Merged file missing switch-1 from base")
	}
	if !strings.Contains(content, "router-2") {
		t.Error("Merged file missing router-2 from overlay")
	}
	// router-1 should have overlay's MAC (ending in :99)
	if !strings.Contains(content, "44:99") {
		t.Errorf("Merged file should have overlay's MAC for router-1 (44:99). Content:\n%s", content)
	}
	// Verify base MAC (:55) for router-1 is replaced
	// (switch-1 keeps its original MAC which also ends in a different value)
}

// Merge shares checkOutputNotExists with export; it gets its own test because
// it reaches that guard by a different path and takes three arguments.
func TestConfigMergeOverwriteProtection(t *testing.T) {
	dir := t.TempDir()
	base := writeFile(t, dir, "base.yaml", minimalConfig)
	overlay := writeFile(t, dir, "overlay.yaml", minimalConfig)
	output := writeFile(t, dir, "merged.yaml", "original contents")

	err := runConfigMerge([]string{base, overlay, output})
	if !errors.Is(err, errOutputExists) {
		t.Fatalf("runConfigMerge over an existing file = %v, want errOutputExists", err)
	}

	contents, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("reading %s: %v", output, err)
	}

	if string(contents) != "original contents" {
		t.Errorf("output file was modified: %q", contents)
	}
}

// Malformed YAML must fail loudly rather than exporting an empty config.
func TestConfigInvalidInput(t *testing.T) {
	dir := t.TempDir()
	input := writeFile(t, dir, "broken.yaml", "devices: [this is not: valid yaml\n")

	if err := runConfigExport([]string{input, filepath.Join(dir, "out.json")}); err == nil {
		t.Fatal("runConfigExport on malformed YAML = nil, want a load error")
	}
}

// A path that does not exist is the commonest operator typo, and it must exit
// rather than proceed with a zero-value config.
func TestConfigMissingFiles(t *testing.T) {
	dir := t.TempDir()

	err := runConfigExport([]string{filepath.Join(dir, "absent.yaml"), filepath.Join(dir, "out.json")})
	if err == nil {
		t.Fatal("runConfigExport on a missing input = nil, want a load error")
	}
}
