package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// ErrPathOutsideManagedRoots identifies an API/runtime path outside NIAC-owned storage.
var ErrPathOutsideManagedRoots = errors.New("configuration path is outside managed roots")

// ResolveManagedConfigPath resolves a configuration file and permits it only
// when its real path remains inside one of the supplied managed roots.
func ResolveManagedConfigPath(path string, roots []string) (string, error) {
	if hasParentTraversal(path) {
		return "", fmt.Errorf("%w: traversal is not allowed", ErrPathOutsideManagedRoots)
	}
	absPath, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", fmt.Errorf("resolve configuration path: %w", err)
	}
	if !pathWithinAnyRoot(absPath, roots, false) {
		return "", ErrPathOutsideManagedRoots
	}
	realPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		return "", fmt.Errorf("resolve configuration path: %w", err)
	}
	// Containment is re-checked here, after symlink resolution and before any
	// further filesystem call. The earlier check saw only the literal path; a
	// symlink inside a managed root can point anywhere, so until this passes the
	// resolved path is still untrusted. Statting first would touch an escaped
	// path and let its result -- "not a regular file" versus a stat error --
	// describe something outside the roots.
	if !pathWithinAnyRoot(realPath, roots, true) {
		return "", ErrPathOutsideManagedRoots
	}
	info, err := os.Stat(realPath)
	if err != nil {
		return "", fmt.Errorf("stat configuration path: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("%w: configuration must be a regular file", ErrPathOutsideManagedRoots)
	}

	return realPath, nil
}

func hasParentTraversal(path string) bool {
	components := strings.FieldsFunc(path, func(r rune) bool {
		return r == '/' || r == '\\'
	})
	return slices.Contains(components, "..")
}

func pathWithinAnyRoot(path string, roots []string, resolveSymlinks bool) bool {
	for _, root := range roots {
		if pathWithinManagedRoot(path, root, resolveSymlinks) {
			return true
		}
	}
	return false
}

func pathWithinManagedRoot(path, root string, resolveSymlinks bool) bool {
	absRoot, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		return false
	}
	checkedRoot := absRoot
	if resolveSymlinks {
		checkedRoot, err = filepath.EvalSymlinks(absRoot)
		if err != nil {
			return false
		}
	}
	relative, err := filepath.Rel(checkedRoot, path)
	if err != nil {
		return false
	}
	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}
