// Package library is the on-disk content store the daemon serves to
// the UI. It owns three sibling directories:
//
//	<root>/networks/   YAML network configs (replaces "templates" + "user configs")
//	<root>/walks/      SNMP walk files
//	<root>/pcaps/      PCAP captures
//
// The library is the single source of truth at runtime — every entry
// the picker / browser sees comes from here. Content arrives via three
// optional ingress paths:
//
//  1. Embedded starter packs — unpacked into networks/ and walks/ on
//     first run if each dir is empty, so the daemon works out of the
//     box.
//  2. CLI: `niac content install` downloads a versioned tarball from
//     the GitHub release and extracts it here (PR 2).
//  3. UI upload: POST /api/v1/library/install accepts a tarball via
//     multipart and unpacks it (PR 2).
//
// See issue #548 for the full architecture.
package library

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Sentinel errors.
var (
	ErrNotFound        = errors.New("library entry not found")
	ErrInvalidName     = errors.New("invalid name: only [a-zA-Z0-9._-] allowed, no path traversal")
	ErrUnsupportedKind = errors.New("unsupported library kind")
	ErrEmptyContent    = errors.New("content is empty")
	ErrAlreadyExists   = errors.New("library entry already exists")
	ErrNoOriginal      = errors.New("no preserved original to revert to")
)

// Kind is one of the three top-level subdirectories of the library.
type Kind string

// The three library subdirectory kinds.
const (
	KindNetworks Kind = "networks"
	KindWalks    Kind = "walks"
	KindPcaps    Kind = "pcaps"
)

// AllKinds returns the canonical kind list in stable order.
func AllKinds() []Kind { return []Kind{KindNetworks, KindWalks, KindPcaps} }

// Source describes where a given entry came from. Stamped onto every
// returned record so the UI can render small "Starter" / "User" /
// "Bundle" badges and so the delete handler can refuse to remove
// starter-pack entries (which would just reappear on next bootstrap).
type Source string

// The three entry sources tracked by Source.
const (
	SourceStarter Source = "starter"
	SourceBundle  Source = "bundle"
	SourceUser    Source = "user"
)

// Library is the runtime handle for the on-disk store. Create one per
// daemon process via Open(); the root path is resolved once and reused.
// Methods are concurrency-safe via simple read-the-disk semantics —
// every call to List or Read hits the filesystem so user-dropped files
// appear immediately without an explicit watcher.
type Library struct {
	root      string
	draftRoot *os.Root
	draftMu   draftNameLocks
	// bundles records which entries a content bundle installed, so they are
	// reported as bundle content instead of as the operator's own.
	bundles bundleIndex
}

// DefaultRoot resolves the standard library root following:
//
//  1. NIAC_LIBRARY_ROOT environment variable (highest)
//  2. /var/lib/niac/library if that directory exists (packaged install)
//  3. $XDG_DATA_HOME/niac/library if XDG_DATA_HOME is set
//  4. ~/.niac/library (user-default)
//
// Callers can override with an explicit path passed to Open().
func DefaultRoot() string {
	if env := os.Getenv("NIAC_LIBRARY_ROOT"); env != "" {
		return env
	}
	const packaged = "/var/lib/niac/library"
	if info, err := os.Stat(packaged); err == nil && info.IsDir() {
		return packaged
	}
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "niac", "library")
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".niac", "library")
	}
	// Last-resort fallback so the daemon doesn't crash; the cwd-relative
	// path won't be ideal but it's better than panicking.
	return ".niac-library"
}

// Open resolves the library root, ensures the three subdirectories
// exist, and unpacks the embedded starter packs into networks/ and
// walks/ if those directories are brand new. The returned Library is
// ready for List/Read calls; callers MUST handle the returned error
// before using it.
func Open(root string) (*Library, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve library root: %w", err)
	}

	lib := &Library{root: abs}

	if layoutErr := lib.ensureLayout(); layoutErr != nil {
		return nil, layoutErr
	}

	if bootstrapErr := lib.bootstrapStarterPack(); bootstrapErr != nil {
		_ = lib.draftRoot.Close()
		return nil, fmt.Errorf("bootstrap starter pack: %w", bootstrapErr)
	}

	if bootstrapErr := lib.bootstrapWalks(); bootstrapErr != nil {
		_ = lib.draftRoot.Close()
		return nil, fmt.Errorf("bootstrap starter walks: %w", bootstrapErr)
	}

	return lib, nil
}

// Root returns the absolute path the library is rooted at.
func (l *Library) Root() string { return l.root }

// SubDir returns the absolute path of the directory backing a kind.
// Used by handlers that need to stream raw bytes back to the client.
func (l *Library) SubDir(kind Kind) string {
	return filepath.Join(l.root, string(kind))
}

// ensureLayout makes sure all three subdirectories exist with safe
// permissions. Existing dirs are left alone.
func (l *Library) ensureLayout() error {
	if err := os.MkdirAll(l.root, libraryDirMode); err != nil {
		return fmt.Errorf("create library root %s: %w", l.root, err)
	}
	root, err := os.OpenRoot(l.root)
	if err != nil {
		return fmt.Errorf("open library root %s: %w", l.root, err)
	}
	defer func() { _ = root.Close() }()

	for _, kind := range AllKinds() {
		if mkdirErr := root.MkdirAll(string(kind), libraryDirMode); mkdirErr != nil {
			return fmt.Errorf("create library subdir %s: %w", l.SubDir(kind), mkdirErr)
		}
	}

	info, statErr := root.Lstat(draftsDirName)
	if errors.Is(statErr, os.ErrNotExist) {
		if mkdirErr := root.Mkdir(draftsDirName, draftDirMode); mkdirErr != nil && !errors.Is(mkdirErr, os.ErrExist) {
			return fmt.Errorf("create library drafts dir %s: %w", l.draftsDir(), mkdirErr)
		}
		info, statErr = root.Lstat(draftsDirName)
	}
	if statErr != nil {
		return fmt.Errorf("inspect library drafts dir %s: %w", l.draftsDir(), statErr)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("library drafts path must be a real directory: %s", l.draftsDir())
	}

	draftRoot, openErr := root.OpenRoot(draftsDirName)
	if openErr != nil {
		return fmt.Errorf("open library drafts dir %s: %w", l.draftsDir(), openErr)
	}
	if chmodErr := draftRoot.Chmod(".", draftDirMode); chmodErr != nil {
		_ = draftRoot.Close()
		return fmt.Errorf("protect library drafts dir %s: %w", l.draftsDir(), chmodErr)
	}
	// No lock file is pre-created here: draft cross-process locks are
	// per-name (see acquireDraftLock) and created lazily on first use of
	// that name, so bootstrap has nothing to reserve up front.
	l.draftRoot = draftRoot
	return nil
}

// libraryDirMode is the default mode for created library directories.
// 0o755 lets the user own it while still letting the daemon read it
// when it runs as a different user via /var/lib/niac/library.
const libraryDirMode = 0o755

// libraryFileMode is the default mode for files written to the
// library (uploads, starter-pack extracts).
const libraryFileMode = 0o644

// validateName rejects anything that could escape its kind directory.
// Names must be a bare filename (no slashes, no '..', no leading dot)
// composed of letters, digits, '.', '_', or '-'.
func validateName(name string) error {
	if name == "" {
		return ErrInvalidName
	}
	if strings.ContainsAny(name, "/\\") {
		return ErrInvalidName
	}
	if name == "." || name == ".." || strings.HasPrefix(name, "..") {
		return ErrInvalidName
	}
	if strings.HasPrefix(name, ".") {
		return ErrInvalidName
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '.' || r == '_' || r == '-':
		default:
			return ErrInvalidName
		}
	}
	return nil
}

// validateRelPath validates a walks/pcaps relative name that may
// include one level of vendor subdirectory (e.g.
// "cisco/cisco-c3900-01.walk"): each '/'-separated segment is checked
// individually via validateName, so a crafted ".." segment anywhere
// in the path is rejected the same way a bare invalid filename would
// be. Shared by WriteFile and the PreserveOriginal/HasOriginal/
// RevertToOriginal family in original.go.
func validateRelPath(relPath string) error {
	for segment := range strings.SplitSeq(filepath.ToSlash(relPath), "/") {
		if err := validateName(segment); err != nil {
			return err
		}
	}
	return nil
}
