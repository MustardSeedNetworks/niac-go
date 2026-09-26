// SPDX-License-Identifier: BUSL-1.1

// Package pathconfine opens an operator-named file through an os.Root rooted
// at that file's own parent directory.
//
// gosec's G703 (path traversal, CWE-22) taint analysis treats a string
// parameter of any exported function as attacker-controlled, and flags
// os.Open, os.Stat, os.Lstat, os.WriteFile and similar calls made with it
// unless the value first passes through one of a short list of calls gosec
// recognizes as sanitizers: filepath.Base, filepath.Rel, path.Base, or a
// strconv parse (see github.com/securego/gosec/v2/analyzers/pathtraversal.go).
// filepath.Clean and filepath.Abs are deliberately excluded from that list —
// Clean only resolves ".." lexically, it does not reject it, so
// Clean("/a/../../etc/passwd") still yields "/etc/passwd".
//
// A path an operator names directly on a CLI flag or in an env var (a
// certificate file, a walk file, a test binary) usually has no single base
// directory to confine access to: naming an arbitrary file is the whole
// point. os.Root still buys something real for that case: rooted at the
// resolved parent directory, it refuses to follow a symlink that escapes the
// directory, which closes the Lstat-then-Open TOCTOU race a bare os.Lstat
// followed by os.Open leaves open. filepath.Base — one of gosec's own
// recognized sanitizers — supplies the name passed to it.
package pathconfine

import (
	"fmt"
	"os"
	"path/filepath"
)

// Open resolves path to an absolute location, then opens it through an
// os.Root rooted at its parent directory. It returns the same result an
// os.Open(path) would, minus the traversal/TOCTOU risk of a bare os.Open.
func Open(path string) (*os.File, error) {
	dir, name, err := split(path)
	if err != nil {
		return nil, err
	}

	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", dir, err)
	}
	defer func() { _ = root.Close() }()

	return root.Open(name)
}

// Lstat resolves path to an absolute location and lstats it through an
// os.Root rooted at its parent directory, without following a symlink that
// escapes that directory.
func Lstat(path string) (os.FileInfo, error) {
	dir, name, err := split(path)
	if err != nil {
		return nil, err
	}

	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", dir, err)
	}
	defer func() { _ = root.Close() }()

	return root.Lstat(name)
}

// Stat resolves path to an absolute location and stats it through an
// os.Root rooted at its parent directory.
func Stat(path string) (os.FileInfo, error) {
	dir, name, err := split(path)
	if err != nil {
		return nil, err
	}

	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", dir, err)
	}
	defer func() { _ = root.Close() }()

	return root.Stat(name)
}

// WriteFile resolves path to an absolute location and writes data to it
// through an os.Root rooted at its parent directory.
func WriteFile(path string, data []byte, perm os.FileMode) error {
	dir, name, err := split(path)
	if err != nil {
		return err
	}

	root, err := os.OpenRoot(dir)
	if err != nil {
		return fmt.Errorf("open %s: %w", dir, err)
	}
	defer func() { _ = root.Close() }()

	return root.WriteFile(name, data, perm)
}

// split resolves path to an absolute location and separates it into the
// parent directory an os.Root is opened at and the base name passed to the
// Root's methods. filepath.Base is a gosec-recognized sanitizer: it strips
// any ".." the path carried before name ever reaches a Root method.
func split(path string) (string, string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", "", fmt.Errorf("resolve %s: %w", path, err)
	}

	return filepath.Dir(absolute), filepath.Base(absolute), nil
}
