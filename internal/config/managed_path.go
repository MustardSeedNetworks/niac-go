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
	if bare, ok := resolveBareNameInRoots(path, roots); ok {
		absPath = bare
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

// resolveBareNameInRoots searches the managed roots for a configuration named
// without any directory part, which is the only spelling an operator has: the
// roots hold the shipped scenarios, and their location is the daemon's own
// business. Without this a bare name was made absolute against the daemon's
// working directory, so on an installed host (cwd /var/lib/niac, library under
// the service user's home) nothing shipped could be named and `--config
// basic-network.yaml` was refused as outside managed storage.
//
// A name containing a separator is left alone: the caller meant a path, and
// resolving it here would let a root's name prefix reach into a sibling. Roots
// are searched in order, so a user config shadows a starter of the same name.
func resolveBareNameInRoots(name string, roots []string) (string, bool) {
	if name == "" || strings.ContainsAny(name, `/\`) {
		return "", false
	}
	for _, root := range roots {
		if root == "" {
			continue
		}
		candidate := filepath.Join(root, name)
		if info, err := os.Stat(candidate); err == nil && info.Mode().IsRegular() {
			abs, absErr := filepath.Abs(candidate)
			if absErr != nil {
				continue
			}
			return abs, true
		}
	}
	return "", false
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
