package library

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// originalSuffix names the pristine-copy sidecar preserved on the
// first edit of a walk/pcap entry. See PreserveOriginal.
const originalSuffix = ".orig"

// PreserveOriginal snapshots {relPath} to {relPath}.orig the FIRST
// time it is called for a given entry. This is a preserve-ONCE
// operation, not a rolling backup: once {relPath}.orig exists it is
// never overwritten by a later call, so it always holds the content
// exactly as it was before ANY edit — the basis RevertToOriginal
// restores from.
//
// No-ops (returns nil, does not error) when:
//   - {relPath}.orig already exists — never clobber a preserved original, or
//   - {relPath} itself does not exist yet — nothing to preserve.
func (l *Library) PreserveOriginal(kind Kind, relPath string) error {
	if kind != KindWalks && kind != KindPcaps {
		return fmt.Errorf("%w: %s", ErrUnsupportedKind, kind)
	}
	if err := validateRelPath(relPath); err != nil {
		return err
	}

	root, err := l.openKindRoot(kind)
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()

	leaf := filepath.ToSlash(relPath)
	origLeaf := leaf + originalSuffix

	if _, statErr := root.Stat(origLeaf); statErr == nil {
		return nil // already preserved — never overwrite the pristine copy
	} else if !errors.Is(statErr, fs.ErrNotExist) {
		return fmt.Errorf("stat %s: %w", origLeaf, statErr)
	}

	src, err := root.Open(leaf)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil // nothing to preserve yet
		}
		return fmt.Errorf("read %s: %w", leaf, err)
	}
	content, readErr := io.ReadAll(src)
	_ = src.Close()
	if readErr != nil {
		return fmt.Errorf("read %s: %w", leaf, readErr)
	}

	dst, err := root.OpenFile(origLeaf, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, libraryFileMode)
	if err != nil {
		return fmt.Errorf("write %s: %w", origLeaf, err)
	}
	if _, writeErr := dst.Write(content); writeErr != nil {
		_ = dst.Close()
		return fmt.Errorf("write %s: %w", origLeaf, writeErr)
	}
	return dst.Close()
}

// HasOriginal reports whether {relPath}.orig exists — i.e. whether
// relPath has been edited at least once since it was first written
// and can be restored with RevertToOriginal.
func (l *Library) HasOriginal(kind Kind, relPath string) bool {
	if kind != KindWalks && kind != KindPcaps {
		return false
	}
	if err := validateRelPath(relPath); err != nil {
		return false
	}
	root, err := l.openKindRoot(kind)
	if err != nil {
		return false
	}
	defer func() { _ = root.Close() }()
	_, statErr := root.Stat(filepath.ToSlash(relPath) + originalSuffix)
	return statErr == nil
}

// RevertToOriginal restores {relPath} from {relPath}.orig and then
// removes the .orig sidecar, returning the entry to a clean "no
// edits" state where the next PreserveOriginal call starts fresh.
//
// Returns ErrNoOriginal if there is nothing to revert from.
func (l *Library) RevertToOriginal(kind Kind, relPath string) error {
	if kind != KindWalks && kind != KindPcaps {
		return fmt.Errorf("%w: %s", ErrUnsupportedKind, kind)
	}
	if err := validateRelPath(relPath); err != nil {
		return err
	}

	root, err := l.openKindRoot(kind)
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()

	leaf := filepath.ToSlash(relPath)
	origLeaf := leaf + originalSuffix

	src, err := root.Open(origLeaf)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return ErrNoOriginal
		}
		return fmt.Errorf("read %s: %w", origLeaf, err)
	}
	content, readErr := io.ReadAll(src)
	_ = src.Close()
	if readErr != nil {
		return fmt.Errorf("read %s: %w", origLeaf, readErr)
	}

	dst, err := root.OpenFile(leaf, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, libraryFileMode)
	if err != nil {
		return fmt.Errorf("restore %s: %w", leaf, err)
	}
	if _, writeErr := dst.Write(content); writeErr != nil {
		_ = dst.Close()
		return fmt.Errorf("restore %s: %w", leaf, writeErr)
	}
	if closeErr := dst.Close(); closeErr != nil {
		return fmt.Errorf("restore %s: %w", leaf, closeErr)
	}
	if rmErr := root.Remove(origLeaf); rmErr != nil {
		return fmt.Errorf("remove %s: %w", origLeaf, rmErr)
	}
	return nil
}

// isOriginalOrBackup reports whether filename is a bookkeeping
// sidecar — a preserve-once pristine copy (.orig) or a legacy rolling
// backup (.bak) — rather than a real library entry. ListFiles
// excludes these so they never appear as their own walk/pcap rows.
func isOriginalOrBackup(filename string) bool {
	return strings.HasSuffix(filename, originalSuffix) || strings.HasSuffix(filename, ".bak")
}
