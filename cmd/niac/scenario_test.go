package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/templates"
)

func TestScenarioList(t *testing.T) {
	// Test that we can list scenarios
	scenarioList := templates.List()

	if len(scenarioList) == 0 {
		t.Error("Expected at least one scenario, got none")
	}

	// Verify scenario structure
	for _, tmpl := range scenarioList {
		if tmpl.Name == "" {
			t.Error("Scenario name should not be empty")
		}
		if tmpl.Description == "" {
			t.Error("Scenario description should not be empty")
		}
	}
}

func TestScenarioGet(t *testing.T) {
	t.Run("Get basic-network scenario", func(t *testing.T) {
		tmpl, err := templates.Get("basic-network")
		if err != nil {
			t.Fatalf("Unexpected error getting scenario: %v", err)
		}

		assertScenarioValid(t, tmpl)
	})

	t.Run("Get non-existent scenario", func(t *testing.T) {
		_, err := templates.Get("nonexistent-scenario-xyz")
		if err == nil {
			t.Error("Expected error for non-existent scenario, got nil")
		}
	})
}

// assertScenarioValid validates that a scenario has required fields.
func assertScenarioValid(t *testing.T, tmpl *templates.Template) {
	t.Helper()

	if tmpl == nil {
		t.Fatal("Expected scenario, got nil")
	}

	if tmpl.Name == "" {
		t.Error("Scenario name should not be empty")
	}

	if tmpl.Content == "" {
		t.Error("Scenario content should not be empty")
	}
}

func TestScenarioUseFileCreation(t *testing.T) {
	t.Run("Create basic-network config", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputFile := filepath.Join(tmpDir, "basic.yaml")

		tmpl, err := templates.Get("basic-network")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		assertScenarioWriteable(t, tmpl, outputFile)
	})

	t.Run("Non-existent scenario", func(t *testing.T) {
		_, err := templates.Get("invalid-scenario")
		if err == nil {
			t.Error("Expected error for invalid scenario, got nil")
		}
	})
}

// assertScenarioWriteable writes a scenario to file and validates it.
func assertScenarioWriteable(t *testing.T, tmpl *templates.Template, outputFile string) {
	t.Helper()

	err := os.WriteFile(outputFile, []byte(tmpl.Content), 0o644)
	if err != nil {
		t.Fatalf("Failed to write scenario: %v", err)
	}

	if _, statErr := os.Stat(outputFile); os.IsNotExist(statErr) {
		t.Error("Output file should exist")
	}

	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	if len(content) == 0 {
		t.Error("Scenario content should not be empty")
	}
}

func TestScenarioFileOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "overwrite-test.yaml")

	// Create initial file
	initialContent := []byte("initial: content")
	err := os.WriteFile(outputFile, initialContent, 0o644)
	if err != nil {
		t.Fatalf("Failed to create initial file: %v", err)
	}

	// Get scenario
	tmpl, err := templates.Get("basic-network")
	if err != nil {
		t.Fatalf("Failed to get scenario: %v", err)
	}

	// Overwrite with scenario
	err = os.WriteFile(outputFile, []byte(tmpl.Content), 0o644)
	if err != nil {
		t.Fatalf("Failed to overwrite file: %v", err)
	}

	// Verify new content
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(content) == string(initialContent) {
		t.Error("File should have been overwritten with scenario content")
	}

	if string(content) != tmpl.Content {
		t.Error("File content does not match scenario")
	}
}

func TestScenarioContentValidity(t *testing.T) {
	// Get a scenario and verify its content is valid YAML
	tmpl, err := templates.Get("basic-network")
	if err != nil {
		t.Fatalf("Failed to get scenario: %v", err)
	}

	// Basic check - should contain 'devices:'
	if tmpl.Content == "" {
		t.Error("Scenario content is empty")
	}

	// Scenarios should be YAML format
	// This is a simple check - actual validation happens in config package
	if !strings.Contains(tmpl.Content, "devices:") && !strings.Contains(tmpl.Content, "device:") {
		t.Error("Scenario should contain 'devices:' key")
	}
}

func TestAllScenariosLoadable(t *testing.T) {
	// Test that all available scenarios can be loaded
	scenarioList := templates.List()

	for _, info := range scenarioList {
		t.Run("Load_"+info.Name, func(t *testing.T) {
			tmpl, err := templates.Get(info.Name)
			if err != nil {
				t.Errorf("Failed to load scenario %s: %v", info.Name, err)
				return
			}

			if tmpl.Name != info.Name {
				t.Errorf("Scenario name mismatch: got %s, want %s", tmpl.Name, info.Name)
			}

			if tmpl.Content == "" {
				t.Errorf("Scenario %s has empty content", info.Name)
			}
		})
	}
}

func TestScenarioInvalidDirectory(t *testing.T) {
	// Try to write scenario to invalid directory
	invalidPath := "/nonexistent/directory/config.yaml"

	tmpl, err := templates.Get("basic-network")
	if err != nil {
		t.Fatalf("Failed to get scenario: %v", err)
	}

	err = os.WriteFile(invalidPath, []byte(tmpl.Content), 0o644)
	if err == nil {
		t.Error("Expected error when writing to invalid directory, got nil")
	}
}

func TestScenarioFilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "perms-test.yaml")

	tmpl, err := templates.Get("basic-network")
	if err != nil {
		t.Fatalf("Failed to get scenario: %v", err)
	}

	// Write with 0644 permissions
	err = os.WriteFile(outputFile, []byte(tmpl.Content), 0o644)
	if err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	// Check permissions
	info, err := os.Stat(outputFile)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	mode := info.Mode()
	expectedPerm := os.FileMode(0o644)
	if mode.Perm() != expectedPerm {
		t.Logf("File permissions: got %v, expected %v (may vary by OS)", mode.Perm(), expectedPerm)
	}
}
