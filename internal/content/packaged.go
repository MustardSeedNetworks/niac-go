package content

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
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
// essentials (≈18 starters). Adoption runs whenever the bundle on disk
// is not the one the library last recorded, so upgrading the
// niac-content package brings its new devices in. It used to run only
// while walks/ held nothing but embedded starters, which meant a single
// walk uploaded before the package was installed blocked the whole
// 138-device bundle for good.
//
// The bundle identifies itself by the SHA-256 of its own bytes rather
// than a version string inside the tar: the generator's layout is the
// manifest and stays that way, and any rebuild of the corpus is a
// different bundle whether or not someone remembered to bump a number.
// Re-adopting the recorded bundle is a no-op, so this costs one hash per
// daemon boot and nothing else.
//
// What lands is decided per file by Extract: a device the library does
// not have, or one this bundle shipped earlier and nobody has touched
// since, is written; the operator's own walks and any shipped walk they
// have edited are preserved. The embedded starters are never recorded as
// bundle content, so they fall on the preserved side too.
//
// bundlePath must be a local file (the packaged path or its env
// override); never a URL. If it doesn't exist (the niac-content package
// isn't installed) this is a no-op — installed=false, err=nil.
func AdoptPackagedBundle(lib *library.Library, bundlePath string) (Manifest, bool, error) {
	f, err := os.Open(bundlePath) // #nosec G304 -- operator-controlled packaged path, not user input
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Manifest{}, false, nil
		}
		return Manifest{}, false, fmt.Errorf("open packaged bundle %s: %w", bundlePath, err)
	}
	defer func() { _ = f.Close() }()

	version, err := bundleVersion(f)
	if err != nil {
		return Manifest{}, false, fmt.Errorf("read packaged bundle %s: %w", bundlePath, err)
	}
	index, err := library.ReadBundleIndex(lib.Root())
	if err != nil {
		return Manifest{}, false, fmt.Errorf("read bundle index: %w", err)
	}
	if index.Version == version {
		return Manifest{}, false, nil
	}

	if _, err = f.Seek(0, io.SeekStart); err != nil {
		return Manifest{}, false, fmt.Errorf("rewind packaged bundle %s: %w", bundlePath, err)
	}
	manifest, err := Extract(f, lib.Root(), ExtractOptions{BundleVersion: version})
	if err != nil {
		return Manifest{}, false, fmt.Errorf("extract packaged bundle %s: %w", bundlePath, err)
	}

	return manifest, true, nil
}

// bundleVersion identifies a bundle by the SHA-256 of its bytes.
func bundleVersion(r io.Reader) (string, error) {
	digest := sha256.New()
	if _, err := io.Copy(digest, r); err != nil {
		return "", err
	}

	return hex.EncodeToString(digest.Sum(nil)), nil
}
