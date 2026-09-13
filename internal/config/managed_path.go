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
// roots hold the shipped scenarios, and where they live is the daemon's own
// business. Without this a bare name was made absolute against the daemon's
// working directory, so on an installed host (cwd /var/lib/niac, library under
// the service user's home) nothing shipped could be named at all and `--config
// basic-network.yaml` was refused as outside managed storage.
//
// Anything that is not a plain filename is left alone for the caller's own
// path handling to resolve and the containment checks to judge. Roots are
// searched in order, so a user config shadows a starter of the same name. The
// match is made against each root's own directory entries, so the path handed
// back is one the root already held rather than one built from the request.
func resolveBareNameInRoots(name string, roots []string) (string, bool) {
	if !isPlainFileName(name) {
		return "", false
	}
	for _, root := range roots {
		if root == "" {
			continue
		}
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.Name() != name || !entry.Type().IsRegular() {
				continue
			}
			// Joined with the directory entry's own name rather than the
			// caller's string: what is returned is a file this root was
			// already holding, which is what "selected from NIAC-managed
			// storage" means, and no path is ever built out of the request.
			candidate, absErr := filepath.Abs(filepath.Join(root, entry.Name()))
			if absErr != nil {
				continue
			}
			return candidate, true
		}
	}
	return "", false
}

// isPlainFileName reports whether name is a bare filename that cannot leave the
// directory it is joined to: no separator, no "." or ".." spelling, no leading
// dot, and only the characters the library already allows a stored file to
// carry (internal/library.validateName). Stated as one predicate so the barrier
// is visible to a reader and to static analysis, rather than inferred from the
// join.
func isPlainFileName(name string) bool {
	if name == "" || name == "." || name == ".." {
		return false
	}
	if strings.HasPrefix(name, ".") {
		return false
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '.' || r == '_' || r == '-':
		default:
			return false
		}
	}
	return true
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
