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

// InstallPackagedIfEmpty installs the bundle at bundlePath into lib's
// walks/ directory, but only as a one-time first-run convenience: if
// walks/ already has content (a prior bundle install, or user files),
// this is a no-op so it never clobbers what's there. If bundlePath
// doesn't exist (the niac-content package isn't installed), this is
// also a no-op — installed=false, err=nil. Never touches the network;
// bundlePath must already be a local file.
func InstallPackagedIfEmpty(lib *library.Library, bundlePath string) (Manifest, bool, error) {
	files, err := lib.ListFiles(library.KindWalks)
	if err != nil {
		return Manifest{}, false, fmt.Errorf("check walks dir: %w", err)
	}
	if len(files) > 0 {
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
