// Package templates provides template management for NIAC configurations.
//
// Templates are YAML network definitions shipped as part of the binary.
// They live in internal/templates/builtin/ as a flat library and are
// auto-discovered at process start via the embedded FS — adding a new
// template means dropping a .yaml file in builtin/ with a sensible
// header comment, no Go-side registry edit required.
package templates

import (
	"embed"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Sentinel errors for template operations.
var (
	ErrTemplateNotFound       = errors.New("template not found")
	ErrTemplateContentEmpty   = errors.New("template content is empty")
	ErrTemplateMissingDevices = errors.New("template must contain 'devices:' section")
)

//go:embed builtin/*.yaml
var builtinFS embed.FS

// Template represents a NIAC configuration template.
type Template struct {
	Name        string // Template name (without .yaml extension)
	Description string // Human-readable description (from YAML header)
	UseCase     string // Primary use case or scenario (from YAML header)
	Content     string // Template YAML content
}

// TemplateMetadata contains template information without content.
type TemplateMetadata struct {
	Name        string
	Description string
	UseCase     string
}

const (
	// metadataMaxScanLines bounds how many lines we scan for the
	// # Description / # Use Case lines at the top of each template.
	// Scanning stops on the first non-comment, non-blank line anyway.
	metadataMaxScanLines = 40
	// metadataReadBytes caps the byte window we pull off the start of
	// the file for header parsing. The leading comment block is rarely
	// longer than 1 KB; 4 KB gives plenty of margin without slurping
	// the whole device table.
	metadataReadBytes = 4096
)

var (
	registryOnce    sync.Once
	registry        []TemplateMetadata
	errLoadRegistry error
)

// loadRegistry reads every file in builtin/, parses its header comment
// for description + use case, and returns the sorted metadata slice.
// Cached after the first call.
func loadRegistry() ([]TemplateMetadata, error) {
	registryOnce.Do(func() {
		entries, err := builtinFS.ReadDir("builtin")
		if err != nil {
			errLoadRegistry = fmt.Errorf("read builtin/: %w", err)
			return
		}

		out := make([]TemplateMetadata, 0, len(entries))
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
				continue
			}
			name := strings.TrimSuffix(entry.Name(), ".yaml")

			content, readErr := loadTemplate(name)
			if readErr != nil {
				errLoadRegistry = readErr
				return
			}

			desc, useCase := parseHeaderMetadata(content)
			out = append(out, TemplateMetadata{
				Name:        name,
				Description: desc,
				UseCase:     useCase,
			})
		}

		sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
		registry = out
	})
	return registry, errLoadRegistry
}

// parseHeaderMetadata reads the leading comment block of a template
// YAML file and pulls out a description and use-case string. Recognised
// conventions:
//
//   - A `# Description: …` line wins for description.
//   - Otherwise the first non-empty `# …` line is used (skipping shebang-
//     style or trivial markers).
//   - A `# Use Case: …` line is captured verbatim.
//
// Scanning stops at the first non-comment, non-blank line so we never
// read into the actual YAML body.
func parseHeaderMetadata(content string) (string, string) {
	scanner := strings.NewReader(content)
	buf := make([]byte, metadataReadBytes)
	n, _ := scanner.Read(buf)
	head := string(buf[:n])

	lines := strings.Split(head, "\n")
	limit := min(len(lines), metadataMaxScanLines)

	var description, useCase, firstComment string
	for i := range limit {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "#") {
			break
		}
		stripped := strings.TrimSpace(strings.TrimPrefix(line, "#"))
		if stripped == "" {
			continue
		}

		if desc, ok := matchPrefix(stripped, "Description:"); ok && description == "" {
			description = desc
		}
		if uc, ok := matchPrefix(stripped, "Use Case:"); ok && useCase == "" {
			useCase = uc
		}
		if firstComment == "" {
			firstComment = stripped
		}
	}

	if description == "" {
		description = firstComment
	}
	return description, useCase
}

func matchPrefix(line, prefix string) (string, bool) {
	if !strings.HasPrefix(strings.ToLower(line), strings.ToLower(prefix)) {
		return "", false
	}
	return strings.TrimSpace(line[len(prefix):]), true
}

// List returns all available template metadata.
func List() []TemplateMetadata {
	reg, _ := loadRegistry()
	// Defensive copy so callers can't mutate the cache.
	out := make([]TemplateMetadata, len(reg))
	copy(out, reg)
	return out
}

// ListNames returns just the template names, sorted.
func ListNames() []string {
	reg, _ := loadRegistry()
	names := make([]string, len(reg))
	for i, t := range reg {
		names[i] = t.Name
	}
	return names
}

// Get retrieves a template by name with its full content.
func Get(name string) (*Template, error) {
	name = strings.TrimSuffix(name, ".yaml")

	reg, err := loadRegistry()
	if err != nil {
		return nil, err
	}

	var metadata *TemplateMetadata
	for i := range reg {
		if reg[i].Name == name {
			metadata = &reg[i]
			break
		}
	}

	if metadata == nil {
		return nil, fmt.Errorf("%w: %s", ErrTemplateNotFound, name)
	}

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

// Exists checks if a template with the given name exists.
func Exists(name string) bool {
	name = strings.TrimSuffix(name, ".yaml")
	reg, _ := loadRegistry()
	for _, t := range reg {
		if t.Name == name {
			return true
		}
	}
	return false
}

// loadTemplate loads template content from embedded filesystem.
func loadTemplate(name string) (string, error) {
	path := filepath.Join("builtin", name+".yaml")

	file, err := builtinFS.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to load template %s: %w", name, err)
	}
	defer func() { _ = file.Close() }()

	content, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read template %s: %w", name, err)
	}

	return string(content), nil
}

// Validate checks if template content is valid YAML
// This is a basic check - full validation requires config.Load().
func Validate(content string) error {
	if len(content) == 0 {
		return ErrTemplateContentEmpty
	}

	if !strings.Contains(content, "devices:") {
		return ErrTemplateMissingDevices
	}

	return nil
}
