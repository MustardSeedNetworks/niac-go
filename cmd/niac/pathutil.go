// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/pathconfine"
)

// ErrPathTraversal is returned when a user-provided path contains parent-directory traversal.
var ErrPathTraversal = errors.New("path contains parent directory traversal")

// validateCLIPath cleans and validates a user-provided CLI file path.
// It rejects paths that still contain parent-directory traversal after cleaning.
// The returned path is the cleaned form, safe to pass to os file APIs.
func validateCLIPath(p string) (string, error) {
	cleaned := filepath.Clean(p)
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") || strings.HasPrefix(cleaned, `..\`) {
		return "", fmt.Errorf("%w: %s", ErrPathTraversal, p)
	}

	if strings.Contains(cleaned, string(filepath.Separator)+"..") {
		return "", fmt.Errorf("%w: %s", ErrPathTraversal, p)
	}

	return cleaned, nil
}

// writeSafeFile writes data to a CLI-validated path with 0600 permissions,
// through an os.Root rooted at the path's own parent directory so the write
// cannot follow a symlink escaping that directory.
func writeSafeFile(path string, data []byte) error {
	return pathconfine.WriteFile(path, data, 0o600)
}

// statSafeFile checks whether a CLI-validated path exists, through an
// os.Root rooted at the path's own parent directory.
func statSafeFile(path string) (os.FileInfo, error) {
	return pathconfine.Stat(path)
}
