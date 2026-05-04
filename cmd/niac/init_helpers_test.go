package main

import (
	"bufio"
	"strings"
	"testing"

	"github.com/krisarmstrong/niac-go/internal/templates"
)

func TestPrintInitHeader(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("printInitHeader panicked: %v", r)
		}
	}()
	printInitHeader()
}

func TestPrintTemplateDetails(t *testing.T) {
	tmpl := &templates.Template{
		Name:        "test-template",
		Description: "Test description",
		UseCase:     "Testing",
	}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("printTemplateDetails panicked: %v", r)
		}
	}()
	printTemplateDetails(tmpl)
}

func TestPrintInitSuccess(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("printInitSuccess panicked: %v", r)
		}
	}()
	printInitSuccess("test-config.yaml", "basic-network")
}

func TestPrintNextStep(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("printNextStep panicked: %v", r)
		}
	}()
	printNextStep("1", "Test step:", "test command")
}

func TestPromptOutputFileWithArgs(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader(""))
	result := promptOutputFile(reader, []string{"custom-output.yaml"})
	if result != "custom-output.yaml" {
		t.Errorf("promptOutputFile() = %q, want %q", result, "custom-output.yaml")
	}
}

func TestConfirmOverwriteIfExistsNoFile(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader(""))
	// Non-existent file should return true without prompting
	result := confirmOverwriteIfExists(reader, "/tmp/nonexistent-niac-test-file-999.yaml")
	if !result {
		t.Error("Expected true for non-existent file")
	}
}

func TestMapNetworkTypeToTemplate(t *testing.T) {
	tests := []struct {
		name         string
		networkType  string
		expectedName string
		expectDesc   bool
	}{
		{"basic network", "a", "basic-network", true},
		{"small office", "b", "small-office", true},
		{"data center", "c", "data-center", true},
		{"iot network", "d", "iot-network", true},
		{"enterprise campus", "e", "enterprise-campus", true},
		{"service provider", "f", "service-provider", true},
		{"home network", "g", "home-network", true},
		{"test lab", "h", "test-lab", true},
		{"invalid defaults to basic", "z", "basic-network", true},
		{"empty defaults to basic", "", "basic-network", true},
		{"uppercase invalid", "A", "basic-network", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, desc := mapNetworkTypeToTemplate(tt.networkType)
			if name != tt.expectedName {
				t.Errorf("mapNetworkTypeToTemplate(%q) name = %q, want %q", tt.networkType, name, tt.expectedName)
			}
			if tt.expectDesc && desc == "" {
				t.Errorf("mapNetworkTypeToTemplate(%q) desc is empty", tt.networkType)
			}
		})
	}
}

func TestReadLine(t *testing.T) {
	t.Run("normal input", func(t *testing.T) {
		reader := bufio.NewReader(strings.NewReader("hello world\n"))
		line, err := readLine(reader)
		if err != nil {
			t.Fatalf("readLine() error = %v", err)
		}
		if line != "hello world" {
			t.Errorf("readLine() = %q, want %q", line, "hello world")
		}
	})

	t.Run("input with whitespace", func(t *testing.T) {
		reader := bufio.NewReader(strings.NewReader("  trimmed  \n"))
		line, err := readLine(reader)
		if err != nil {
			t.Fatalf("readLine() error = %v", err)
		}
		if line != "trimmed" {
			t.Errorf("readLine() = %q, want %q", line, "trimmed")
		}
	})

	t.Run("EOF with content", func(t *testing.T) {
		reader := bufio.NewReader(strings.NewReader("no-newline"))
		line, err := readLine(reader)
		if err != nil {
			t.Fatalf("readLine() error = %v", err)
		}
		if line != "no-newline" {
			t.Errorf("readLine() = %q, want %q", line, "no-newline")
		}
	})

	t.Run("EOF empty", func(t *testing.T) {
		reader := bufio.NewReader(strings.NewReader(""))
		_, err := readLine(reader)
		if err == nil {
			t.Error("Expected error for empty EOF")
		}
	})
}
