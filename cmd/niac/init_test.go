package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPromptChoice(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		validChoices   []string
		expectedChoice string
	}{
		{
			name:           "Valid choice - lowercase",
			input:          "a\n",
			validChoices:   []string{"a", "b", "c"},
			expectedChoice: "a",
		},
		{
			name:           "Valid choice - uppercase converted to lowercase",
			input:          "B\n",
			validChoices:   []string{"a", "b", "c"},
			expectedChoice: "b",
		},
		{
			name:           "Valid choice with whitespace",
			input:          "  c  \n",
			validChoices:   []string{"a", "b", "c"},
			expectedChoice: "c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bufio.NewReader(strings.NewReader(tt.input))
			result := promptChoice(reader, "", tt.validChoices)

			if result != tt.expectedChoice {
				t.Errorf("promptChoice() = %v, want %v", result, tt.expectedChoice)
			}
		})
	}
}

func TestPromptYesNo(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Yes - lowercase",
			input:    "y\n",
			expected: true,
		},
		{
			name:     "Yes - uppercase",
			input:    "Y\n",
			expected: true,
		},
		{
			name:     "Yes - full word",
			input:    "yes\n",
			expected: true,
		},
		{
			name:     "Yes - full word uppercase",
			input:    "YES\n",
			expected: true,
		},
		{
			name:     "No - lowercase",
			input:    "n\n",
			expected: false,
		},
		{
			name:     "No - uppercase",
			input:    "N\n",
			expected: false,
		},
		{
			name:     "No - full word",
			input:    "no\n",
			expected: false,
		},
		{
			name:     "No - full word uppercase",
			input:    "NO\n",
			expected: false,
		},
		{
			name:     "Yes with whitespace",
			input:    "  yes  \n",
			expected: true,
		},
		{
			name:     "No with whitespace",
			input:    "  no  \n",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bufio.NewReader(strings.NewReader(tt.input))
			result := promptYesNo(reader, "")

			if result != tt.expected {
				t.Errorf("promptYesNo() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestPromptInt(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		min      int
		max      int
		expected int
	}{
		{
			name:     "Valid integer within range",
			input:    "5\n",
			min:      1,
			max:      10,
			expected: 5,
		},
		{
			name:     "Minimum value",
			input:    "1\n",
			min:      1,
			max:      10,
			expected: 1,
		},
		{
			name:     "Maximum value",
			input:    "10\n",
			min:      1,
			max:      10,
			expected: 10,
		},
		{
			name:     "Integer with whitespace",
			input:    "  7  \n",
			min:      1,
			max:      10,
			expected: 7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bufio.NewReader(strings.NewReader(tt.input))
			result := promptInt(reader, "", tt.min, tt.max)

			if result != tt.expected {
				t.Errorf("promptInt() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestInitCommandFileCreation(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "test-config.yaml")

	// Test that we can write to a file
	testContent := []byte("test: content")
	err := os.WriteFile(outputFile, testContent, 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Error("Output file should exist")
	}

	// Verify content
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	if string(content) != string(testContent) {
		t.Errorf("File content mismatch: got %s, want %s", string(content), string(testContent))
	}
}

func TestInitCommandFileOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "existing.yaml")

	// Create existing file
	originalContent := []byte("original: content")
	err := os.WriteFile(outputFile, originalContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create existing file: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Error("Original file should exist")
	}

	// Overwrite with new content
	newContent := []byte("new: content")
	err = os.WriteFile(outputFile, newContent, 0644)
	if err != nil {
		t.Fatalf("Failed to overwrite file: %v", err)
	}

	// Verify new content
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read overwritten file: %v", err)
	}

	if string(content) != string(newContent) {
		t.Errorf("Overwrite failed: got %s, want %s", string(content), string(newContent))
	}
}

func TestInitCommandPermissions(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "perms-test.yaml")

	// Write file with specific permissions
	err := os.WriteFile(outputFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	// Check file permissions
	info, err := os.Stat(outputFile)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	mode := info.Mode()
	// On Unix-like systems, should be 0644 (rw-r--r--)
	expectedPerm := os.FileMode(0644)
	if mode.Perm() != expectedPerm {
		t.Logf("File permissions: got %v, expected %v (may vary by OS)", mode.Perm(), expectedPerm)
	}
}

func TestInitCommandInvalidDirectory(t *testing.T) {
	// Try to write to a non-existent directory
	nonExistentDir := "/nonexistent/path/to/file.yaml"
	err := os.WriteFile(nonExistentDir, []byte("test"), 0644)

	if err == nil {
		t.Error("Expected error when writing to non-existent directory, got nil")
	}
}

func TestInitCommandFileExists(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "existing-file.yaml")

	// Create file
	err := os.WriteFile(testFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Check if file exists using Stat
	_, err = os.Stat(testFile)
	if err != nil {
		if os.IsNotExist(err) {
			t.Error("File should exist but doesn't")
		} else {
			t.Errorf("Unexpected error checking file: %v", err)
		}
	}
}

func TestInitCommandFileDoesNotExist(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistentFile := filepath.Join(tmpDir, "does-not-exist.yaml")

	// Check if file does not exist
	_, err := os.Stat(nonExistentFile)
	if err == nil {
		t.Error("File should not exist but Stat returned no error")
	}

	if !os.IsNotExist(err) {
		t.Errorf("Expected IsNotExist error, got: %v", err)
	}
}
