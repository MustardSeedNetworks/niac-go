package content

import (
	"errors"
	"fmt"
	"os"

	"github.com/MustardSeedNetworks/niac-go/internal/library"
)

// PackagedBundlePathEnv overrides the path the daemon checks for a
// package-installed content bundle. Local file only — never a network
// fetch (see the package doc comment on extract.go).
const PackagedBundlePathEnv = "NIAC_PACKAGED_CONTENT"

// DefaultPackagedBundlePath is where the niac-content deb/rpm package
// installs its bundle (see the "niac-content" nfpm entry in
// .goreleaser.yml).
const DefaultPackagedBundlePath = "/usr/share/niac/content/niac-content.tar.gz"

// PackagedBundlePath resolves the path the daemon checks for a
// package-installed bundle: PackagedBundlePathEnv if set, else
// DefaultPackagedBundlePath.
func PackagedBundlePath() string {
	if p := os.Getenv(PackagedBundlePathEnv); p != "" {
		return p
	}
	return DefaultPackagedBundlePath
}

// AdoptPackagedBundle installs the niac-content deb/rpm bundle at
// bundlePath into lib's walks/ directory, giving the full packaged
// device set precedence over the small embedded "essentials" starter
// pack that library.Open seeds on first run.
//
// Precedence: the deb full bundle (≈138 devices) wins over the embedded
// essentials (≈18 starters). It runs while the library is still
// "unseeded" — walks/ is empty, or holds only embedded-starter files
// (Source==SourceStarter) with no real/user content yet. The moment any
// non-starter walk is present the library counts as seeded and adoption
// is skipped, so this never fights user content.
//
// Extraction is additive (ExtractOptions.SkipExisting): the embedded
// starters and any user files are preserved untouched; the extract only
// fills in the ≈120 devices the embed lacks. That also gives a natural
// once-guard — after adoption walks/ holds non-starter bundle files
// (extracted files aren't in the embedded starter set, so
// library.detectFileSource stamps them SourceUser), so the "only
// starters" check is false on every later boot and it won't re-extract.
//
// bundlePath must be a local file (the packaged path or its env
// override); never a URL. If it doesn't exist (the niac-content package
// isn't installed) this is a no-op — installed=false, err=nil.
func AdoptPackagedBundle(lib *library.Library, bundlePath string) (Manifest, bool, error) {
	seeded, err := librarySeeded(lib)
	if err != nil {
		return Manifest{}, false, err
	}
	if seeded {
		return Manifest{}, false, nil
	}

	f, err := os.Open(bundlePath) // #nosec G304 -- operator-controlled packaged path, not user input
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Manifest{}, false, nil
		}
		return Manifest{}, false, fmt.Errorf("open packaged bundle %s: %w", bundlePath, err)
	}
	defer func() { _ = f.Close() }()

	manifest, err := Extract(f, lib.Root(), ExtractOptions{SkipExisting: true})
	if err != nil {
		return Manifest{}, false, fmt.Errorf("extract packaged bundle %s: %w", bundlePath, err)
	}
	return manifest, true, nil
}

// librarySeeded reports whether walks/ already holds real content. An
// empty walks/ or one holding only embedded starters is "unseeded"
// (false); the first non-starter (user- or bundle-sourced) walk flips
// it to seeded (true).
func librarySeeded(lib *library.Library) (bool, error) {
	files, err := lib.ListFiles(library.KindWalks)
	if err != nil {
		return false, fmt.Errorf("list walks: %w", err)
	}
	for _, f := range files {
		if f.Source != library.SourceStarter {
			return true, nil
		}
	}
	return false, nil
}
