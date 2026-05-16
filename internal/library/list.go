package library

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// NetworkEntry is one row in the networks list.
type NetworkEntry struct {
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	UseCase     string    `json:"useCase,omitempty"`
	DeviceCount int       `json:"deviceCount"`
	ModifiedAt  time.Time `json:"modifiedAt"`
	SizeBytes   int64     `json:"sizeBytes"`
	Source      Source    `json:"source"`
	Valid       bool      `json:"valid"`
	Error       string    `json:"error,omitempty"`
}

// NetworkContent is a single network YAML with its content body.
type NetworkContent struct {
	Name    string `json:"name"`
	Content string `json:"content"`
	Format  string `json:"format"`
	Source  Source `json:"source"`
}

// FileEntry is the lightweight row used for walks and pcaps (binary
// blobs, no per-row content parsing).
type FileEntry struct {
	Name       string    `json:"name"`
	SizeBytes  int64     `json:"sizeBytes"`
	ModifiedAt time.Time `json:"modifiedAt"`
	Source     Source    `json:"source"`
}

// ListNetworks enumerates every YAML in networks/, parses metadata
// from each file's header comment block, and returns the rows sorted
// by name. Files that fail to parse get Valid=false with an error
// message instead of crashing the whole list.
func (l *Library) ListNetworks() ([]NetworkEntry, error) {
	dir := l.SubDir(KindNetworks)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", dir, err)
	}

	out := make([]NetworkEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !isYAMLFilename(entry.Name()) {
			continue
		}
		row, rowErr := l.networkEntryFor(entry.Name())
		if rowErr != nil {
			// Surface as a row-level error so the UI can show a badge
			// rather than failing the entire list call.
			row = NetworkEntry{Name: trimYAMLExt(entry.Name()), Valid: false, Error: rowErr.Error()}
		}
		out = append(out, row)
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (l *Library) networkEntryFor(filename string) (NetworkEntry, error) {
	path := filepath.Join(l.SubDir(KindNetworks), filename)
	info, err := os.Stat(path)
	if err != nil {
		return NetworkEntry{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return NetworkEntry{}, err
	}

	desc, useCase := parseHeaderMetadata(data)
	deviceCount, parseErr := countDevices(data)
	if parseErr != nil {
		// Returning the parse error in-band (Valid=false + Error=…) is
		// deliberate — the row still appears in the list and the UI
		// renders a "broken" badge instead of the whole call 5xxing.
		return NetworkEntry{ //nolint:nilerr // structured surfacing
			Name:        trimYAMLExt(filename),
			Description: desc,
			UseCase:     useCase,
			ModifiedAt:  info.ModTime().UTC(),
			SizeBytes:   info.Size(),
			Source:      l.detectSource(filename),
			Valid:       false,
			Error:       parseErr.Error(),
		}, nil
	}

	return NetworkEntry{
		Name:        trimYAMLExt(filename),
		Description: desc,
		UseCase:     useCase,
		DeviceCount: deviceCount,
		ModifiedAt:  info.ModTime().UTC(),
		SizeBytes:   info.Size(),
		Source:      l.detectSource(filename),
		Valid:       true,
	}, nil
}

// ReadNetwork returns the full content of a single network. Validates
// the requested name so a malicious caller can't escape networks/.
func (l *Library) ReadNetwork(name string) (*NetworkContent, error) {
	name = trimYAMLExt(name)
	if err := validateName(name); err != nil {
		return nil, err
	}
	path := filepath.Join(l.SubDir(KindNetworks), name+".yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("%w: %s", ErrNotFound, name)
		}
		return nil, err
	}
	return &NetworkContent{
		Name:    name,
		Content: string(data),
		Format:  "yaml",
		Source:  l.detectSource(name + ".yaml"),
	}, nil
}

// WriteNetwork creates or overwrites a network YAML. Used by upload
// endpoints. Marks user-created entries by NOT being part of the
// starter pack — detectSource picks that up on next list.
func (l *Library) WriteNetwork(name, content string) error {
	name = trimYAMLExt(name)
	if err := validateName(name); err != nil {
		return err
	}
	if content == "" {
		return ErrEmptyContent
	}
	if !strings.Contains(content, "devices:") {
		return fmt.Errorf("%w: content must contain 'devices:' section", ErrEmptyContent)
	}
	path := filepath.Join(l.SubDir(KindNetworks), name+".yaml")
	return os.WriteFile(path, []byte(content), libraryFileMode)
}

// DeleteNetwork removes a single network. Refuses to delete entries
// whose source is "starter" because they'd just come back on next
// bootstrap.
func (l *Library) DeleteNetwork(name string) error {
	name = trimYAMLExt(name)
	if err := validateName(name); err != nil {
		return err
	}
	filename := name + ".yaml"
	if l.detectSource(filename) == SourceStarter {
		return fmt.Errorf("starter networks cannot be deleted: %s", name)
	}
	path := filepath.Join(l.SubDir(KindNetworks), filename)
	if err := os.Remove(path); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("%w: %s", ErrNotFound, name)
		}
		return err
	}
	return nil
}

// ListFiles enumerates walks/ or pcaps/. Used by PR 3's browser tabs.
// One level of subdir nesting is allowed; names returned include the
// subdir for namespacing (e.g. "cisco/c3900.walk").
func (l *Library) ListFiles(kind Kind) ([]FileEntry, error) {
	if kind != KindWalks && kind != KindPcaps {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedKind, kind)
	}
	dir := l.SubDir(kind)
	out := make([]FileEntry, 0)
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(dir, path)
		if relErr != nil {
			return relErr
		}
		info, infoErr := d.Info()
		if infoErr != nil {
			return infoErr
		}
		out = append(out, FileEntry{
			Name:       filepath.ToSlash(rel),
			SizeBytes:  info.Size(),
			ModifiedAt: info.ModTime().UTC(),
			Source:     SourceUser,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// detectSource decides whether a given filename came from the starter
// pack or from user/bundle install. Starter-pack files are embedded
// in the binary, so consulting starterPack is the source of truth.
func (l *Library) detectSource(filename string) Source {
	if _, err := starterPack.Open(filepath.Join("starter", filename)); err == nil {
		return SourceStarter
	}
	// Bundle vs user split is deferred to PR 2 (the bundle installer
	// writes a marker file). For now, anything non-starter is "user".
	return SourceUser
}

// countDevices parses just enough of the YAML to count entries under
// devices:. Falls back to a 0 with a clear error if the document
// shape is wrong, so the row still appears in the list.
func countDevices(data []byte) (int, error) {
	var doc struct {
		Devices []map[string]any `yaml:"devices"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return 0, fmt.Errorf("yaml parse: %w", err)
	}
	return len(doc.Devices), nil
}

func trimYAMLExt(name string) string {
	for _, ext := range []string{".yaml", ".yml"} {
		if strings.HasSuffix(name, ext) {
			return strings.TrimSuffix(name, ext)
		}
	}
	return name
}
