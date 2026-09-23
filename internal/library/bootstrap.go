package library

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"github.com/MustardSeedNetworks/niac-go/internal/templates"
)

// starterWalks holds a small, curated set of SNMP walk files (one
// representative device per vendor) bundled into the binary. On first
// run they're copied into walks/ so replay has something to work with
// before any content tarball is installed. Keep the set small — for
// the full corpus users install the content bundle.
//
//go:embed starter/walks/*.walk
var starterWalks embed.FS

// starterPackEntries marks every starter file with Source==starter via
// a sentinel filename suffix recognised by detectSource(); we keep the
// upstream filenames identical to make user customization (overwrite a
// starter with a same-named user file) work via the same code path.

// bootstrapStarterPack copies the embedded starter pack into the
// networks/ subdir, but only if that directory is empty. A library
// that already has user content is left strictly alone.
func (l *Library) bootstrapStarterPack() error {
	networksDir := l.SubDir(KindNetworks)
	empty, err := dirIsEmpty(networksDir)
	if err != nil {
		return fmt.Errorf("check networks dir: %w", err)
	}
	if !empty {
		return nil
	}

	// Seeded from internal/templates, the maintained template tree, rather than
	// a second embedded copy of the same eight files.
	//
	// There used to be a starter/*.yaml duplicate here. It was never migrated
	// when the config schema changed, and nothing validated it, so every
	// template the New Simulation wizard offered failed to load while the two
	// other template trees stayed green in CI (D2). One tree, one place to fix.
	for _, name := range templates.ListNames() {
		tmpl, getErr := templates.Get(name)
		if getErr != nil {
			return fmt.Errorf("read starter template %s: %w", name, getErr)
		}
		dst := filepath.Join(networksDir, name+".yaml")
		if writeErr := os.WriteFile(dst, []byte(tmpl.Content), libraryFileMode); writeErr != nil {
			return fmt.Errorf("write starter file %s: %w", dst, writeErr)
		}
	}

	return nil
}

// refreshStaleStarters replaces each starter network on disk that the current
// release can no longer load with its embedded template.
//
// bootstrapStarterPack writes the starters once, into an empty library, so a
// library written by an earlier schema kept starters that failed to load after
// every upgrade while they still looked runnable (#2202). A starter the
// operator customised and that still loads is theirs and stays. One that no
// longer loads is replaced, and its bytes are kept once in "<name>.yaml.orig"
// so a customisation is never silently lost. A missing starter is not
// re-created: a library that already has content is otherwise left alone.
func (l *Library) refreshStaleStarters() error {
	root, err := l.openNetworksRoot()
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()

	for _, name := range templates.ListNames() {
		tmpl, getErr := templates.Get(name)
		if getErr != nil {
			return fmt.Errorf("read starter template %s: %w", name, getErr)
		}
		leaf := name + ".yaml"
		current, readErr := root.ReadFile(leaf)
		if errors.Is(readErr, fs.ErrNotExist) {
			continue
		}
		if readErr != nil {
			return fmt.Errorf("read starter %s: %w", leaf, readErr)
		}
		if string(current) == tmpl.Content || networkProblem(filepath.Join(root.Name(), leaf)) == "" {
			continue
		}
		if _, statErr := root.Stat(leaf + originalSuffix); errors.Is(statErr, fs.ErrNotExist) {
			if writeErr := root.WriteFile(leaf+originalSuffix, current, libraryFileMode); writeErr != nil {
				return fmt.Errorf("preserve starter %s: %w", leaf, writeErr)
			}
		}
		if writeErr := root.WriteFile(leaf, []byte(tmpl.Content), libraryFileMode); writeErr != nil {
			return fmt.Errorf("refresh starter %s: %w", leaf, writeErr)
		}
	}
	return nil
}

// bootstrapWalks copies the embedded starter walks into the walks/
// subdir, but only if that directory is empty. A library that already
// has user content is left strictly alone.
func (l *Library) bootstrapWalks() error {
	walksDir := l.SubDir(KindWalks)
	empty, err := dirIsEmpty(walksDir)
	if err != nil {
		return fmt.Errorf("check walks dir: %w", err)
	}
	if !empty {
		return nil
	}

	entries, err := starterWalks.ReadDir("starter/walks")
	if err != nil {
		return fmt.Errorf("read starter walks: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !isWalkFilename(entry.Name()) {
			continue
		}
		// embed.FS is always slash-separated regardless of GOOS, so this must be
		// path.Join and not filepath.Join: on Windows the latter produced
		// "starter\\walks\\x.walk", which the embedded FS cannot resolve, and the
		// whole content library failed to open with /api/v1/library disabled.
		src := path.Join("starter", "walks", entry.Name())
		data, readErr := fs.ReadFile(starterWalks, src)
		if readErr != nil {
			return fmt.Errorf("read starter walk %s: %w", entry.Name(), readErr)
		}
		dst := filepath.Join(walksDir, entry.Name())
		if writeErr := os.WriteFile(dst, data, libraryFileMode); writeErr != nil {
			return fmt.Errorf("write starter walk %s: %w", dst, writeErr)
		}
	}

	return nil
}

func dirIsEmpty(path string) (bool, error) {
	// #nosec G304 -- caller passes Library.SubDir(kind) which is
	// always rooted under the resolved Library.root we own.
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer func() { _ = f.Close() }()

	names, err := f.Readdirnames(1)
	if err != nil && err.Error() != "EOF" {
		// io.EOF on an empty dir is fine; surfaces as len(names)==0.
		return len(names) == 0, nil
	}
	return len(names) == 0, nil
}

func isYAMLFilename(name string) bool {
	ext := filepath.Ext(name)
	return ext == ".yaml" || ext == ".yml"
}

func isWalkFilename(name string) bool {
	return filepath.Ext(name) == ".walk"
}
