package library

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

// BundleIndexName is the file recording which library entries arrived in a
// content bundle, and what they looked like when they did.
//
// Nothing else can tell: a bundle's files land in the same directories as the
// operator's own and look identical afterwards, so without this every installed
// file was reported as "user" and the Bundle badge the UI already draws could
// never appear.
//
// It holds the version of the bundle last adopted and, per library-relative
// path, the SHA-256 of the file as that bundle shipped it. The hash is what
// lets an upgrade tell a file the operator has edited from one it may safely
// replace. A missing or unreadable index means nothing is known to have come
// from a bundle: every file on disk is then treated as the operator's and left
// alone, which is the safe direction.
const BundleIndexName = ".bundle-installed"

// bundleIndexMode keeps the index readable only by the owner, as the
// append-only list it replaces was.
const bundleIndexMode = 0o600

// BundleIndex is what the library records about the content bundle it holds.
type BundleIndex struct {
	// Version identifies the bundle last adopted. Adoption re-runs when the
	// bundle on disk no longer matches it; content.AdoptPackagedBundle uses
	// the bundle file's SHA-256.
	Version string `json:"version"`
	// Files maps a library-relative path (slash-separated, kind-prefixed, e.g.
	// "walks/cisco-2960.walk") to the SHA-256 of the bytes the bundle shipped
	// there.
	Files map[string]string `json:"files"`
}

// bundleIndex is the index as read from disk, loaded once per library handle.
// Listing a directory asks for every entry's source in turn, and each answer
// should not cost a file read.
type bundleIndex struct {
	once   sync.Once
	loaded BundleIndex
}

func (b *bundleIndex) contains(root, relative string) bool {
	b.once.Do(func() { b.loaded, _ = ReadBundleIndex(root) })

	_, ok := b.loaded.Files[filepath.ToSlash(relative)]

	return ok
}

// ReadBundleIndex loads the library's bundle index. A missing index is not an
// error: it reports an empty index, meaning nothing is known to have come from
// a bundle.
func ReadBundleIndex(root string) (BundleIndex, error) {
	index := BundleIndex{Files: map[string]string{}}
	raw, err := os.ReadFile(filepath.Join(root, BundleIndexName))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return index, nil
		}
		return index, err
	}
	if err = json.Unmarshal(raw, &index); err != nil {
		return BundleIndex{Files: map[string]string{}}, err
	}
	if index.Files == nil {
		index.Files = map[string]string{}
	}

	return index, nil
}

// RecordBundleInstall folds what a bundle just wrote into the library's index,
// so those files are reported as bundle content rather than as the operator's
// own and a later bundle can tell which of them are still untouched.
//
// Paths are library-relative, as the extractor resolves them, mapped to the
// SHA-256 of the bytes just written. Entries from earlier installs are kept: a
// path this bundle happens not to ship is still bundle content on disk, and
// keeping its hash means a bundle that reintroduces it later does not mistake
// it for the operator's file. A non-empty version replaces the recorded one;
// an empty version leaves it alone, for installs that do not identify
// themselves (an operator's `niac content install`).
func RecordBundleInstall(root, version string, files map[string]string) error {
	if len(files) == 0 && version == "" {
		return nil
	}
	index, err := ReadBundleIndex(root)
	if err != nil {
		// An index we cannot parse is replaced rather than merged into: the
		// alternative is refusing every future install because of one corrupt
		// file. Nothing on disk is touched, only the record of it.
		index = BundleIndex{Files: map[string]string{}}
	}
	if version != "" {
		index.Version = version
	}
	for relative, sum := range files {
		index.Files[filepath.ToSlash(relative)] = sum
	}

	encoded, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(root, BundleIndexName), append(encoded, '\n'), bundleIndexMode)
}
