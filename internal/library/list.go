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

	// #nosec G304 -- filename comes from os.ReadDir of the library
	// subdir we own + isYAMLFilename() filter; not user-supplied path.
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

// resolveNetworkPath validates name (bare filename, no separators, no
// "..", ASCII alnum/./_/- only via validateName) and resolves it to an
// absolute path under the networks/ subdirectory. It then re-verifies,
// via an absolute-path prefix check against the resolved networks/ base
// directory, that the result cannot have escaped that directory —
// defense in depth on top of the character allowlist, and the single
// containment helper shared by ReadNetwork/WriteNetwork/DeleteNetwork so
// the base-dir check lives in exactly one place.
func (l *Library) resolveNetworkPath(name string) (string, string, error) {
	trimmed := trimYAMLExt(name)
	if err := validateName(trimmed); err != nil {
		return "", "", err
	}

	baseDir, err := filepath.Abs(l.SubDir(KindNetworks))
	if err != nil {
		return "", "", fmt.Errorf("resolve networks dir: %w", err)
	}

	candidate := filepath.Join(baseDir, trimmed+".yaml")
	if candidate != baseDir && !strings.HasPrefix(candidate, baseDir+string(os.PathSeparator)) {
		return "", "", ErrInvalidName
	}

	return trimmed, candidate, nil
}

// ReadNetwork returns the full content of a single network. Validates
// the requested name so a malicious caller can't escape networks/.
func (l *Library) ReadNetwork(name string) (*NetworkContent, error) {
	name, path, err := l.resolveNetworkPath(name)
	if err != nil {
		return nil, err
	}
	// #nosec G304 -- path was resolved by resolveNetworkPath above,
	// which rejects path traversal/non-alphanumeric names and verifies
	// the result stays within the networks/ base directory.
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
	_, path, err := l.resolveNetworkPath(name)
	if err != nil {
		return err
	}
	if content == "" {
		return ErrEmptyContent
	}
	if !strings.Contains(content, "devices:") {
		return fmt.Errorf("%w: content must contain 'devices:' section", ErrEmptyContent)
	}
	// #nosec G304 -- path was resolved by resolveNetworkPath above; see
	// ReadNetwork for the containment rationale.
	return os.WriteFile(path, []byte(content), libraryFileMode)
}

// DeleteNetwork removes a single network. Refuses to delete entries
// whose source is "starter" because they'd just come back on next
// bootstrap.
func (l *Library) DeleteNetwork(name string) error {
	name, path, err := l.resolveNetworkPath(name)
	if err != nil {
		return err
	}
	if l.detectSource(name+".yaml") == SourceStarter {
		return fmt.Errorf("starter networks cannot be deleted: %s", name)
	}
	// #nosec G304 -- path was resolved by resolveNetworkPath above; see
	// ReadNetwork for the containment rationale.
	if removeErr := os.Remove(path); removeErr != nil {
		if errors.Is(removeErr, fs.ErrNotExist) {
			return fmt.Errorf("%w: %s", ErrNotFound, name)
		}
		return removeErr
	}
	return nil
}

// WriteFile writes content to {kind}/{relPath}, creating
// intermediate directories as needed. Used by the synthesised-walk
// path (#546 part 2) to drop "walks/synthesized/{host}.walk" into
// the library without going through the YAML-specific WriteNetwork
// validation.
//
// relPath may include one level of subdirectory ("synthesized/foo.walk",
// "cisco/c3900.walk") — components are validated individually so a
// crafted path can't escape the library kind directory. Existing
// files are overwritten without warning; backupExisting=true moves
// the prior copy to "{relPath}.bak" first (one-step rollback per the
// design-doc resolution).
func (l *Library) WriteFile(kind Kind, relPath string, content []byte, backupExisting bool) error {
	if kind != KindWalks && kind != KindPcaps {
		return fmt.Errorf("%w: %s", ErrUnsupportedKind, kind)
	}
	if len(content) == 0 {
		return ErrEmptyContent
	}
	for segment := range strings.SplitSeq(filepath.ToSlash(relPath), "/") {
		if err := validateName(segment); err != nil {
			return err
		}
	}

	dest := filepath.Join(l.SubDir(kind), filepath.FromSlash(relPath))
	if mkErr := os.MkdirAll(filepath.Dir(dest), libraryDirMode); mkErr != nil {
		return fmt.Errorf("create dir for %s: %w", dest, mkErr)
	}

	if backupExisting {
		if _, statErr := os.Stat(dest); statErr == nil {
			// Move the prior file to .bak. Overwrites any older .bak,
			// matching the one-step rollback contract from the design
			// doc resolution.
			if renameErr := os.Rename(dest, dest+".bak"); renameErr != nil {
				return fmt.Errorf("backup %s: %w", dest, renameErr)
			}
		}
	}

	return os.WriteFile(dest, content, libraryFileMode)
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
		if before, ok := strings.CutSuffix(name, ext); ok {
			return before
		}
	}
	return name
}
