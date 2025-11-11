package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/krisarmstrong/niac-go/pkg/templates"
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
	tests := []struct {
		name         string
		templateName string
		expectError  bool
	}{
		{
			name:         "Get basic-network template",
			templateName: "basic-network",
			expectError:  false,
		},
		{
			name:         "Get non-existent template",
			templateName: "nonexistent-template-xyz",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := templates.Get(tt.templateName)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error for non-existent template, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error getting template: %v", err)
				}
				if tmpl == nil {
					t.Error("Expected template, got nil")
				}
				if tmpl != nil {
					if tmpl.Name == "" {
						t.Error("Template name should not be empty")
					}
					if tmpl.Content == "" {
						t.Error("Template content should not be empty")
					}
				}
			}
		})
	}
}

func TestTemplateUseFileCreation(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name         string
		templateName string
		outputFile   string
		expectError  bool
	}{
		{
			name:         "Create basic-network config",
			templateName: "basic-network",
			outputFile:   filepath.Join(tmpDir, "basic.yaml"),
			expectError:  false,
		},
		{
			name:         "Non-existent template",
			templateName: "invalid-template",
			outputFile:   filepath.Join(tmpDir, "invalid.yaml"),
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get template
			tmpl, err := templates.Get(tt.templateName)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error for invalid template, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Write template to file
			err = os.WriteFile(tt.outputFile, []byte(tmpl.Content), 0644)
			if err != nil {
				t.Errorf("Failed to write template: %v", err)
				return
			}

			// Verify file exists
			if _, err := os.Stat(tt.outputFile); os.IsNotExist(err) {
				t.Error("Output file should exist")
			}

			// Verify content
			content, err := os.ReadFile(tt.outputFile)
			if err != nil {
				t.Errorf("Failed to read output file: %v", err)
				return
			}

			if len(content) == 0 {
				t.Error("Template content should not be empty")
			}
		})
	}
}

func TestTemplateFileOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "overwrite-test.yaml")

	// Create initial file
	initialContent := []byte("initial: content")
	err := os.WriteFile(outputFile, initialContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create initial file: %v", err)
	}

	// Get template
	tmpl, err := templates.Get("basic-network")
	if err != nil {
		t.Fatalf("Failed to get template: %v", err)
	}

	// Overwrite with template
	err = os.WriteFile(outputFile, []byte(tmpl.Content), 0644)
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
	if !contains(tmpl.Content, "devices:") && !contains(tmpl.Content, "device:") {
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

	err = os.WriteFile(invalidPath, []byte(tmpl.Content), 0644)
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
	err = os.WriteFile(outputFile, []byte(tmpl.Content), 0644)
	if err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	// Check permissions
	info, err := os.Stat(outputFile)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	mode := info.Mode()
	expectedPerm := os.FileMode(0644)
	if mode.Perm() != expectedPerm {
		t.Logf("File permissions: got %v, expected %v (may vary by OS)", mode.Perm(), expectedPerm)
	}
}

// Note: contains function is defined in generate_test.go
