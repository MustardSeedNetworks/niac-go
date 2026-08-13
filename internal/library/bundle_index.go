package library

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// BundleIndexName is the file recording which library entries arrived in a
// content bundle.
//
// Nothing else can tell: a bundle's files land in the same directories as the
// operator's own and look identical afterwards, so without this every installed
// file was reported as "user" and the Bundle badge the UI already draws could
// never appear.
//
// It is a plain list of library-relative paths, one per line, appended as
// bundles install. A missing or unreadable index means nothing is known to have
// come from a bundle, which is the pre-existing behaviour.
const BundleIndexName = ".bundle-installed"

// bundleIndex is the set of paths a bundle installed, read once per library
// handle. Listing a directory asks for every entry's source in turn, and each
// answer should not cost a file read.
type bundleIndex struct {
	once  sync.Once
	paths map[string]bool
}

func (b *bundleIndex) contains(root, relative string) bool {
	b.once.Do(func() { b.paths = readBundleIndex(root) })

	return b.paths[filepath.ToSlash(relative)]
}

func readBundleIndex(root string) map[string]bool {
	paths := map[string]bool{}
	file, err := os.Open(filepath.Join(root, BundleIndexName))
	if err != nil {
		return paths
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if line := strings.TrimSpace(scanner.Text()); line != "" {
			paths[filepath.ToSlash(line)] = true
		}
	}

	return paths
}

// RecordBundleInstall adds the paths a bundle wrote to the library's index, so
// they are reported as bundle content rather than as the operator's own.
//
// Paths are library-relative, as the extractor resolves them. Appending keeps
// earlier installs, and duplicates are harmless because the index is read as a
// set.
func RecordBundleInstall(root string, relativePaths []string) error {
	if len(relativePaths) == 0 {
		return nil
	}
	file, err := os.OpenFile(
		filepath.Join(root, BundleIndexName),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0o600,
	)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	writer := bufio.NewWriter(file)
	for _, path := range relativePaths {
		if _, err = writer.WriteString(filepath.ToSlash(path) + "\n"); err != nil {
			return err
		}
	}

	return writer.Flush()
}
