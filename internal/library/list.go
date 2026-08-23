package library

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/MustardSeedNetworks/niac-go/internal/templates"
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
	// Edited is true when a preserved pristine original exists for this
	// entry (see PreserveOriginal / HasOriginal in original.go) — the
	// entry has been overwritten at least once since it was first
	// written and can be restored via RevertToOriginal.
	Edited bool `json:"edited"`
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

// networkFilename validates name (bare filename, no separators, no
// "..", ASCII alnum/./_/- only via validateName) and returns the
// trimmed base name plus the "<name>.yaml" leaf that ReadNetwork /
// WriteNetwork / DeleteNetwork open relative to the networks/ os.Root.
// validateName is defense in depth on top of the OS-enforced
// containment the os.Root itself provides.
func networkFilename(name string) (string, string, error) {
	trimmed := trimYAMLExt(name)
	if err := validateName(trimmed); err != nil {
		return "", "", err
	}
	return trimmed, trimmed + ".yaml", nil
}

// openNetworksRoot opens the networks/ subdirectory as an os.Root. All
// file operations on the returned root are confined to that directory
// by the OS (openat2 RESOLVE_BENEATH semantics on Linux, equivalent
// checks elsewhere): any leaf that would escape via "..", an absolute
// path, or a symlink is rejected by the kernel/runtime, so path
// traversal is impossible by construction. Callers MUST Close the root.
func (l *Library) openNetworksRoot() (*os.Root, error) {
	root, err := os.OpenRoot(l.SubDir(KindNetworks))
	if err != nil {
		return nil, fmt.Errorf("open networks dir: %w", err)
	}
	return root, nil
}

// openKindRoot opens kind's subdirectory (walks/ or pcaps/) as an
// os.Root, giving the same OS-enforced containment as openNetworksRoot
// for every file op on a user-supplied relative path. Using the root
// (rather than filepath.Join + a bare os.* call guarded by #nosec) is
// what makes path traversal impossible by construction and is the
// structural safety CodeQL's go/path-injection query recognises.
// Callers MUST Close the root.
func (l *Library) openKindRoot(kind Kind) (*os.Root, error) {
	root, err := os.OpenRoot(l.SubDir(kind))
	if err != nil {
		return nil, fmt.Errorf("open %s dir: %w", kind, err)
	}
	return root, nil
}

// ReadNetwork returns the full content of a single network. The name is
// validated and then opened relative to the networks/ os.Root, which
// makes escaping the directory impossible.
func (l *Library) ReadNetwork(name string) (*NetworkContent, error) {
	trimmed, leaf, err := networkFilename(name)
	if err != nil {
		return nil, err
	}
	root, err := l.openNetworksRoot()
	if err != nil {
		return nil, err
	}
	defer func() { _ = root.Close() }()

	f, err := root.Open(leaf)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("%w: %s", ErrNotFound, trimmed)
		}
		return nil, err
	}
	defer func() { _ = f.Close() }()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return &NetworkContent{
		Name:    trimmed,
		Content: string(data),
		Format:  "yaml",
		Source:  l.detectSource(leaf),
	}, nil
}

// WriteNetwork creates or overwrites a network YAML. Used by upload
// endpoints. Marks user-created entries by NOT being part of the
// starter pack — detectSource picks that up on next list.
func (l *Library) WriteNetwork(name, content string) error {
	_, leaf, err := networkFilename(name)
	if err != nil {
		return err
	}
	if content == "" {
		return ErrEmptyContent
	}
	if !strings.Contains(content, "devices:") {
		return fmt.Errorf("%w: content must contain 'devices:' section", ErrEmptyContent)
	}
	root, err := l.openNetworksRoot()
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()

	f, err := root.OpenFile(leaf, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, libraryFileMode)
	if err != nil {
		return err
	}
	if _, writeErr := f.WriteString(content); writeErr != nil {
		_ = f.Close()
		return writeErr
	}
	return f.Close()
}

// DeleteNetwork removes a single network. Refuses to delete entries
// whose source is "starter" because they'd just come back on next
// bootstrap.
func (l *Library) DeleteNetwork(name string) error {
	trimmed, leaf, err := networkFilename(name)
	if err != nil {
		return err
	}
	if l.detectSource(leaf) == SourceStarter {
		return fmt.Errorf("starter networks cannot be deleted: %s", trimmed)
	}
	root, err := l.openNetworksRoot()
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()

	if removeErr := root.Remove(leaf); removeErr != nil {
		if errors.Is(removeErr, fs.ErrNotExist) {
			return fmt.Errorf("%w: %s", ErrNotFound, trimmed)
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
// crafted path can't escape the library kind directory. Existing files
// are overwritten without warning; callers that need the prior content
// preserved call PreserveOriginal (preserve-once ".orig") before writing.
func (l *Library) WriteFile(kind Kind, relPath string, content []byte) error {
	if kind != KindWalks && kind != KindPcaps {
		return fmt.Errorf("%w: %s", ErrUnsupportedKind, kind)
	}
	if len(content) == 0 {
		return ErrEmptyContent
	}
	if err := validateRelPath(relPath); err != nil {
		return err
	}

	dest := filepath.Join(l.SubDir(kind), filepath.FromSlash(relPath))
	if mkErr := os.MkdirAll(filepath.Dir(dest), libraryDirMode); mkErr != nil {
		return fmt.Errorf("create dir for %s: %w", dest, mkErr)
	}

	return os.WriteFile(dest, content, libraryFileMode)
}

// ReadFile returns the raw content of {kind}/{relPath}. Used by the
// walk-sanitize route (#950) to load a walk before transforming it; opened
// relative to the kind's os.Root, the same OS-enforced containment
// PreserveOriginal/RevertToOriginal use, so a crafted relPath can't escape
// the library directory.
func (l *Library) ReadFile(kind Kind, relPath string) ([]byte, error) {
	if kind != KindWalks && kind != KindPcaps {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedKind, kind)
	}
	if err := validateRelPath(relPath); err != nil {
		return nil, err
	}

	root, err := l.openKindRoot(kind)
	if err != nil {
		return nil, err
	}
	defer func() { _ = root.Close() }()

	leaf := filepath.ToSlash(relPath)
	src, err := root.Open(leaf)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("read %s: %w", leaf, err)
	}
	defer func() { _ = src.Close() }()

	content, err := io.ReadAll(src)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", leaf, err)
	}
	return content, nil
}

// ListFiles enumerates walks/ or pcaps/. Used by PR 3's browser tabs.
// One level of subdir nesting is allowed; names returned include the
// subdir for namespacing (e.g. "cisco/c3900.walk"). Preserve-once
// sidecars (.orig) and legacy rolling backups (.bak) are bookkeeping,
// never entries in their own right, so they're excluded here.
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
		if isOriginalOrBackup(d.Name()) {
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
		out = append(out, l.fileEntry(kind, filepath.ToSlash(rel), info))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// fileEntry builds the FileEntry row for relPath from its already-
// stat'd info, stamping Edited via HasOriginal so ListFiles and
// FileEntryByName share one source of truth for the row shape.
func (l *Library) fileEntry(kind Kind, relPath string, info fs.FileInfo) FileEntry {
	return FileEntry{
		Name:       relPath,
		SizeBytes:  info.Size(),
		ModifiedAt: info.ModTime().UTC(),
		Source:     l.detectFileSource(kind, relPath),
		Edited:     l.HasOriginal(kind, relPath),
	}
}

// detectFileSource mirrors detectSource for the walks/pcaps kinds. Only
// KindWalks has a starter pack today; both kinds can arrive in a bundle.
func (l *Library) detectFileSource(kind Kind, relPath string) Source {
	if kind == KindWalks {
		if _, err := starterWalks.Open(filepath.Join("starter", "walks", relPath)); err == nil {
			return SourceStarter
		}
	}
	if l.bundles.contains(l.root, filepath.Join(string(kind), relPath)) {
		return SourceBundle
	}

	return SourceUser
}

// FileEntryByName returns the single FileEntry for relPath within
// kind's subdir, without a full ListFiles scan. Used to return a
// freshly refreshed row after a mutating operation (e.g.
// RevertToOriginal).
func (l *Library) FileEntryByName(kind Kind, relPath string) (FileEntry, error) {
	if kind != KindWalks && kind != KindPcaps {
		return FileEntry{}, fmt.Errorf("%w: %s", ErrUnsupportedKind, kind)
	}
	if err := validateRelPath(relPath); err != nil {
		return FileEntry{}, err
	}
	root, err := l.openKindRoot(kind)
	if err != nil {
		return FileEntry{}, err
	}
	defer func() { _ = root.Close() }()

	leaf := filepath.ToSlash(relPath)
	info, err := root.Stat(leaf)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return FileEntry{}, fmt.Errorf("%w: %s", ErrNotFound, relPath)
		}
		return FileEntry{}, err
	}
	return l.fileEntry(kind, leaf, info), nil
}

// detectSource decides whether a given filename came from the starter pack, a
// content bundle, or the operator. Starter networks are the templates embedded
// in the binary, so consulting internal/templates is the source of truth for
// those; a bundle records what it installed, because afterwards its files are
// indistinguishable from anything else on disk.
func (l *Library) detectSource(filename string) Source {
	if templates.Exists(strings.TrimSuffix(filename, filepath.Ext(filename))) {
		return SourceStarter
	}
	if _, err := starterWalks.Open(filepath.Join("starter", "walks", filename)); err == nil {
		return SourceStarter
	}
	if l.bundles.contains(l.root, filepath.Join(string(KindNetworks), filename)) {
		return SourceBundle
	}

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
