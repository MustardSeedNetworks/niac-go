// Package templates provides template management for NIAC configurations.
// It includes built-in templates and functionality for loading, listing, and applying templates.
package templates

import (
	"embed"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed builtin/*.yaml
var builtinFS embed.FS

// Template represents a NIAC configuration template
type Template struct {
	Name        string // Template name (without .yaml extension)
	Description string // Human-readable description
	UseCase     string // Primary use case or scenario
	Content     string // Template YAML content
}

// TemplateMetadata contains template information without content
type TemplateMetadata struct {
	Name        string
	Description string
	UseCase     string
}

// templateRegistry holds metadata for all built-in templates
var templateRegistry = []TemplateMetadata{
	{
		Name:        "basic-network",
		Description: "Simple router and switch setup",
		UseCase:     "Small networks with basic routing and switching",
	},
	{
		Name:        "small-office",
		Description: "Router, switch, AP, DNS/DHCP server",
		UseCase:     "Small office or branch network (10-50 devices)",
	},
	{
		Name:        "data-center",
		Description: "Multiple routers, switches, and servers",
		UseCase:     "Data center or enterprise core network",
	},
	{
		Name:        "iot-network",
		Description: "IoT devices with constrained protocols",
		UseCase:     "IoT sensor networks and embedded devices",
	},
	{
		Name:        "enterprise-campus",
		Description: "Multi-building campus network",
		UseCase:     "Large enterprise campus with distribution layers",
	},
	{
		Name:        "service-provider",
		Description: "ISP-style topology with BGP",
		UseCase:     "Service provider or carrier network simulation",
	},
	{
		Name:        "home-network",
		Description: "Residential gateway, WiFi, devices",
		UseCase:     "Home network with consumer devices",
	},
	{
		Name:        "test-lab",
		Description: "Lab environment for protocol testing",
		UseCase:     "Network testing and protocol development",
	},
}

// List returns all available template metadata
func List() []TemplateMetadata {
	return templateRegistry
}

// ListNames returns just the template names
func ListNames() []string {
	names := make([]string, len(templateRegistry))
	for i, t := range templateRegistry {
		names[i] = t.Name
	}
	sort.Strings(names)
	return names
}

// Get retrieves a template by name with its full content
func Get(name string) (*Template, error) {
	// Normalize name (remove .yaml if present)
	name = strings.TrimSuffix(name, ".yaml")

	// Find in registry
	var metadata *TemplateMetadata
	for i := range templateRegistry {
		if templateRegistry[i].Name == name {
			metadata = &templateRegistry[i]
			break
		}
	}
	if metadata == nil {
		return nil, fmt.Errorf("template not found: %s", name)
	}

	// Load content from embedded FS
	content, err := loadTemplate(name)
	if err != nil {
		return nil, err
	}

	return &Template{
		Name:        metadata.Name,
		Description: metadata.Description,
		UseCase:     metadata.UseCase,
		Content:     content,
	}, nil
}

// Exists checks if a template with the given name exists
func Exists(name string) bool {
	name = strings.TrimSuffix(name, ".yaml")
	for _, t := range templateRegistry {
		if t.Name == name {
			return true
		}
	}
	return false
}

// loadTemplate loads template content from embedded filesystem
func loadTemplate(name string) (string, error) {
	// Try with .yaml extension
	path := filepath.Join("builtin", name+".yaml")

	file, err := builtinFS.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to load template %s: %w", name, err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read template %s: %w", name, err)
	}

	return string(content), nil
}

// Validate checks if template content is valid YAML
// This is a basic check - full validation requires config.Load()
func Validate(content string) error {
	if len(content) == 0 {
		return fmt.Errorf("template content is empty")
	}
	if !strings.Contains(content, "devices:") {
		return fmt.Errorf("template must contain 'devices:' section")
	}
	return nil
}
