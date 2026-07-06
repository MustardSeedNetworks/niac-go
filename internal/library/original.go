package library

import (
	"errors"
	"fmt"
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

	dest := filepath.Join(l.SubDir(kind), filepath.FromSlash(relPath))
	origPath := dest + originalSuffix

	_, statErr := os.Stat(origPath)
	if statErr == nil {
		return nil // already preserved — never overwrite the pristine copy
	}
	if !errors.Is(statErr, fs.ErrNotExist) {
		return fmt.Errorf("stat %s: %w", origPath, statErr)
	}

	// #nosec G304 -- relPath validated by validateRelPath above; dest is bounded to l.SubDir(kind).
	content, err := os.ReadFile(dest)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil // nothing to preserve yet
		}
		return fmt.Errorf("read %s: %w", dest, err)
	}

	// #nosec G703 -- origPath is dest+originalSuffix; dest is bounded to l.SubDir(kind)
	// via relPath validated by validateRelPath above. gosec's taint tracker can't
	// follow that provenance through the .orig suffix concatenation (same class of
	// false positive as internal/truststore/truststore_linux.go's writeAnchorFile).
	if writeErr := os.WriteFile(origPath, content, libraryFileMode); writeErr != nil {
		return fmt.Errorf("write %s: %w", origPath, writeErr)
	}
	return nil
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
	dest := filepath.Join(l.SubDir(kind), filepath.FromSlash(relPath))
	_, err := os.Stat(dest + originalSuffix)
	return err == nil
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

	dest := filepath.Join(l.SubDir(kind), filepath.FromSlash(relPath))
	origPath := dest + originalSuffix

	// #nosec G304 -- relPath validated by validateRelPath above; origPath is bounded to l.SubDir(kind).
	content, err := os.ReadFile(origPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return ErrNoOriginal
		}
		return fmt.Errorf("read %s: %w", origPath, err)
	}

	// #nosec G703 -- dest is bounded to l.SubDir(kind) via relPath validated by
	// validateRelPath above; gosec's taint tracker can't follow that provenance
	// through the content read from origPath earlier in this function (same
	// class of false positive as truststore_linux.go's writeAnchorFile).
	if writeErr := os.WriteFile(dest, content, libraryFileMode); writeErr != nil {
		return fmt.Errorf("restore %s: %w", dest, writeErr)
	}
	if rmErr := os.Remove(origPath); rmErr != nil {
		return fmt.Errorf("remove %s: %w", origPath, rmErr)
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
