package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/templates"
)

func TestTemplateList(t *testing.T) {
	// Test that we can list templates
	templateList := templates.List()

	if len(templateList) == 0 {
		t.Error("Expected at least one template, got none")
	}

	// Verify template structure
	for _, tmpl := range templateList {
		if tmpl.Name == "" {
			t.Error("Template name should not be empty")
		}
		if tmpl.Description == "" {
			t.Error("Template description should not be empty")
		}
	}
}

func TestTemplateGet(t *testing.T) {
	t.Run("Get basic-network template", func(t *testing.T) {
		tmpl, err := templates.Get("basic-network")
		if err != nil {
			t.Fatalf("Unexpected error getting template: %v", err)
		}

		assertTemplateValid(t, tmpl)
	})

	t.Run("Get non-existent template", func(t *testing.T) {
		_, err := templates.Get("nonexistent-template-xyz")
		if err == nil {
			t.Error("Expected error for non-existent template, got nil")
		}
	})
}

// assertTemplateValid validates that a template has required fields.
func assertTemplateValid(t *testing.T, tmpl *templates.Template) {
	t.Helper()

	if tmpl == nil {
		t.Fatal("Expected template, got nil")
	}

	if tmpl.Name == "" {
		t.Error("Template name should not be empty")
	}

	if tmpl.Content == "" {
		t.Error("Template content should not be empty")
	}
}

func TestTemplateUseFileCreation(t *testing.T) {
	t.Run("Create basic-network config", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputFile := filepath.Join(tmpDir, "basic.yaml")

		tmpl, err := templates.Get("basic-network")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		assertTemplateWriteable(t, tmpl, outputFile)
	})

	t.Run("Non-existent template", func(t *testing.T) {
		_, err := templates.Get("invalid-template")
		if err == nil {
			t.Error("Expected error for invalid template, got nil")
		}
	})
}

// assertTemplateWriteable writes a template to file and validates it.
func assertTemplateWriteable(t *testing.T, tmpl *templates.Template, outputFile string) {
	t.Helper()

	err := os.WriteFile(outputFile, []byte(tmpl.Content), 0o644)
	if err != nil {
		t.Fatalf("Failed to write template: %v", err)
	}

	if _, statErr := os.Stat(outputFile); os.IsNotExist(statErr) {
		t.Error("Output file should exist")
	}

	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	if len(content) == 0 {
		t.Error("Template content should not be empty")
	}
}

func TestTemplateFileOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "overwrite-test.yaml")

	// Create initial file
	initialContent := []byte("initial: content")
	err := os.WriteFile(outputFile, initialContent, 0o644)
	if err != nil {
		t.Fatalf("Failed to create initial file: %v", err)
	}

	// Get template
	tmpl, err := templates.Get("basic-network")
	if err != nil {
		t.Fatalf("Failed to get template: %v", err)
	}

	// Overwrite with template
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
		t.Error("File should have been overwritten with template content")
	}

	if string(content) != tmpl.Content {
		t.Error("File content does not match template")
	}
}

func TestTemplateContentValidity(t *testing.T) {
	// Get a template and verify its content is valid YAML
	tmpl, err := templates.Get("basic-network")
	if err != nil {
		t.Fatalf("Failed to get template: %v", err)
	}

	// Basic check - should contain 'devices:'
	if tmpl.Content == "" {
		t.Error("Template content is empty")
	}

	// Templates should be YAML format
	// This is a simple check - actual validation happens in config package
	if !strings.Contains(tmpl.Content, "devices:") && !strings.Contains(tmpl.Content, "device:") {
		t.Error("Template should contain 'devices:' key")
	}
}

func TestAllTemplatesLoadable(t *testing.T) {
	// Test that all available templates can be loaded
	templateList := templates.List()

	for _, info := range templateList {
		t.Run("Load_"+info.Name, func(t *testing.T) {
			tmpl, err := templates.Get(info.Name)
			if err != nil {
				t.Errorf("Failed to load template %s: %v", info.Name, err)
				return
			}

			if tmpl.Name != info.Name {
				t.Errorf("Template name mismatch: got %s, want %s", tmpl.Name, info.Name)
			}

			if tmpl.Content == "" {
				t.Errorf("Template %s has empty content", info.Name)
			}
		})
	}
}

func TestTemplateInvalidDirectory(t *testing.T) {
	// Try to write template to invalid directory
	invalidPath := "/nonexistent/directory/config.yaml"

	tmpl, err := templates.Get("basic-network")
	if err != nil {
		t.Fatalf("Failed to get template: %v", err)
	}

	err = os.WriteFile(invalidPath, []byte(tmpl.Content), 0o644)
	if err == nil {
		t.Error("Expected error when writing to invalid directory, got nil")
	}
}

func TestTemplateFilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "perms-test.yaml")

	tmpl, err := templates.Get("basic-network")
	if err != nil {
		t.Fatalf("Failed to get template: %v", err)
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
