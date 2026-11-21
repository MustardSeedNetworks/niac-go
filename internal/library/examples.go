package library

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
	"github.com/MustardSeedNetworks/niac-go/internal/templates"
)

// ErrReadOnlyExample is returned for a write or delete aimed at a shipped
// example. Examples live in the binary, so there is nothing on disk to change;
// Duplicate saves an editable copy into the library instead.
var ErrReadOnlyExample = errors.New("shipped examples are read-only; duplicate one to edit it")

// exampleCopySuffix names a duplicate when the caller gives no name.
const exampleCopySuffix = "-copy"

// example is one shipped scenario served from the binary: a hand-written
// template, or a scenario pack generated on first use.
type example struct {
	name        string
	description string
	useCase     string
	deviceCount int
	// sizeBytes is the YAML size for a template, and zero for a pack, whose
	// document is generated on demand (a second or more for the larger ones)
	// and so is not known when the library is listed.
	sizeBytes int64
	modified  time.Time
	content   func() ([]byte, error)
}

// exampleCatalog lists the shipped examples. It reads only metadata that is
// already in the binary, so listing is cheap: no pack is generated until
// something asks for its document.
func exampleCatalog() []example {
	built := binaryModTime()
	var out []example
	for _, meta := range templates.List() {
		tmpl, err := templates.Get(meta.Name)
		if err != nil {
			// The registry just listed this name from the same embedded tree.
			continue
		}
		data := []byte(tmpl.Content)
		out = append(out, example{
			name:        meta.Name,
			description: meta.Description,
			useCase:     meta.UseCase,
			deviceCount: countDevices(data),
			sizeBytes:   int64(len(data)),
			modified:    built,
			content:     func() ([]byte, error) { return data, nil },
		})
	}
	for _, pack := range scenario.Packs() {
		out = append(out, example{
			name:        pack.ID,
			description: pack.Description,
			useCase:     pack.Name,
			deviceCount: pack.Manifest.DeviceCount,
			modified:    built,
			content:     packContent(pack),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out
}

// packContent generates a pack's scenario document. Generation is deterministic
// and takes a second or more for the larger packs, which is why it happens on
// request rather than when the library is listed.
func packContent(pack scenario.Pack) func() ([]byte, error) {
	return func() ([]byte, error) {
		result, err := scenario.Generate(pack.Request)
		if err != nil {
			return nil, fmt.Errorf("generate example %s: %w", pack.ID, err)
		}
		return result.YAML, nil
	}
}

// binaryModTime is when the binary holding the examples was built, which is
// the only honest "modified" an embedded document has.
func binaryModTime() time.Time {
	exe, err := os.Executable()
	if err != nil {
		return time.Time{}
	}
	info, err := os.Stat(exe)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime().UTC()
}

func findExample(name string) (example, bool) {
	name = trimYAMLExt(name)
	for _, candidate := range exampleCatalog() {
		if candidate.name == name {
			return candidate, true
		}
	}
	return example{}, false
}

// IsExample reports whether name is a shipped example, whether or not a file
// in the library shadows it.
func IsExample(name string) bool {
	_, ok := findExample(name)
	return ok
}

// ExampleContent returns the unmodified document of a shipped example.
func ExampleContent(name string) ([]byte, error) {
	found, ok := findExample(name)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, trimYAMLExt(name))
	}
	return found.content()
}

// ExampleStartContent returns an example's document ready to be loaded from
// outside the networks/ directory, which is where the daemon stages it. A
// template that reads captures names them relative to the library (include_path:
// ../walks), so the path is made absolute against walksDir; without that the
// session's staged copy would look for the walks beside the staging directory
// and again on every recovery.
func ExampleStartContent(name, walksDir string) ([]byte, error) {
	data, err := ExampleContent(name)
	if err != nil {
		return nil, err
	}
	return withAbsoluteIncludePath(data, walksDir)
}

func withAbsoluteIncludePath(data []byte, walksDir string) ([]byte, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse example: %w", err)
	}
	if len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return data, nil
	}
	fields := doc.Content[0].Content
	for i := 0; i+1 < len(fields); i += 2 {
		if fields[i].Value != "include_path" || filepath.IsAbs(fields[i+1].Value) {
			continue
		}
		fields[i+1].Value = filepath.ToSlash(walksDir)
		out, err := yaml.Marshal(&doc)
		if err != nil {
			return nil, fmt.Errorf("render example: %w", err)
		}
		return out, nil
	}
	return data, nil
}

// exampleEntries lists the examples that no file in the library shadows. A
// file of the same name wins everywhere a bare name is resolved, so showing the
// example beside it would list one name twice.
func exampleEntries(onDisk map[string]bool) []NetworkEntry {
	var out []NetworkEntry
	for _, shipped := range exampleCatalog() {
		if onDisk[shipped.name] {
			continue
		}
		out = append(out, NetworkEntry{
			Name:        shipped.name,
			Description: shipped.description,
			UseCase:     shipped.useCase,
			DeviceCount: shipped.deviceCount,
			ModifiedAt:  shipped.modified,
			SizeBytes:   shipped.sizeBytes,
			Source:      SourceExample,
			Valid:       true,
		})
	}
	return out
}

// exampleOnly reports whether name is a shipped example that no library file
// stands behind: the case where there is nothing to write over or remove.
func (l *Library) exampleOnly(name string) bool {
	if !IsExample(name) {
		return false
	}
	root, err := l.openNetworksRoot()
	if err != nil {
		return true
	}
	defer func() { _ = root.Close() }()
	_, statErr := root.Stat(trimYAMLExt(name) + ".yaml")
	return statErr != nil
}

// DuplicateExample saves a copy of a shipped example into the library under
// newName, or "<example>-copy" when newName is empty, and returns the name it
// used. The copy is created exclusively: an existing network is never replaced.
func (l *Library) DuplicateExample(example, newName string) (string, error) {
	if !IsExample(example) {
		return "", fmt.Errorf("%w: %s", ErrNotFound, trimYAMLExt(example))
	}
	if newName == "" {
		newName = trimYAMLExt(example) + exampleCopySuffix
	}
	trimmed, leaf, err := networkFilename(newName)
	if err != nil {
		return "", err
	}
	if l.exampleOnly(trimmed) {
		return "", fmt.Errorf("%w: %s", ErrReadOnlyExample, trimmed)
	}
	data, err := ExampleContent(example)
	if err != nil {
		return "", err
	}
	if err = l.createNetworkExclusive(leaf, data); err != nil {
		if errors.Is(err, ErrAlreadyExists) {
			return "", fmt.Errorf("%w: %s", ErrAlreadyExists, trimmed)
		}
		return "", err
	}
	return trimmed, nil
}

func (l *Library) createNetworkExclusive(leaf string, data []byte) error {
	root, err := l.openNetworksRoot()
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()

	file, err := root.OpenFile(leaf, os.O_WRONLY|os.O_CREATE|os.O_EXCL, libraryFileMode)
	if errors.Is(err, os.ErrExist) {
		return ErrAlreadyExists
	}
	if err != nil {
		return fmt.Errorf("create network: %w", err)
	}
	_, writeErr := file.Write(data)
	if closeErr := file.Close(); writeErr == nil {
		writeErr = closeErr
	}
	if writeErr != nil {
		_ = root.Remove(leaf)
		return fmt.Errorf("write network: %w", writeErr)
	}
	return nil
}
