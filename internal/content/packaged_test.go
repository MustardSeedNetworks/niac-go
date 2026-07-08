package content_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/content"
	"github.com/MustardSeedNetworks/niac-go/internal/library"
)

// newDeviceWalk is a device name deliberately absent from the embedded
// starter walk pack, so library.detectFileSource stamps it SourceUser
// after extraction — the marker that the library is now "seeded".
const newDeviceWalk = "zzz-new-device-99.walk"

// writePackagedBundle synthesises a niac-content bundle carrying a
// single non-starter walk and returns its path on disk.
func writePackagedBundle(t *testing.T) string {
	t.Helper()
	tarball := buildTarball(t, []tarballEntry{
		{name: "walks/" + newDeviceWalk, body: "1.3.6.1.2.1.1.1.0 = STRING: \"new\"\n"},
	})
	path := filepath.Join(t.TempDir(), "niac-content.tar.gz")
	if err := os.WriteFile(path, tarball, 0o644); err != nil {
		t.Fatalf("write bundle: %v", err)
	}
	return path
}

// countWalks returns how many walks/ entries carry each Source.
func countWalks(t *testing.T, lib *library.Library) (int, int) {
	t.Helper()
	files, err := lib.ListFiles(library.KindWalks)
	if err != nil {
		t.Fatalf("list walks: %v", err)
	}
	var starters, others int
	for _, f := range files {
		if f.Source == library.SourceStarter {
			starters++
		} else {
			others++
		}
	}
	return starters, others
}

// TestAdoptPackagedBundleOverEssentials covers the L1/L3a integration:
// library.Open seeds the embedded starter walks, and the packaged deb
// bundle must still be adopted on top of them (precedence: full bundle
// > embedded essentials), additively.
func TestAdoptPackagedBundleOverEssentials(t *testing.T) {
	lib, err := library.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open library: %v", err)
	}
	starters, others := countWalks(t, lib)
	if starters == 0 {
		t.Fatalf("expected embedded starter walks seeded by Open, got none")
	}
	if others != 0 {
		t.Fatalf("expected only starter walks before adoption, got %d non-starter", others)
	}

	bundle := writePackagedBundle(t)
	manifest, installed, err := content.AdoptPackagedBundle(lib, bundle)
	if err != nil {
		t.Fatalf("adopt: %v", err)
	}
	if !installed {
		t.Fatal("expected adoption when only starter walks present")
	}
	if manifest.Files != 1 {
		t.Errorf("manifest files: got %d want 1", manifest.Files)
	}

	// Additive: the starters survive and the bundle device is added.
	startersAfter, othersAfter := countWalks(t, lib)
	if startersAfter != starters {
		t.Errorf("starters clobbered: got %d want %d", startersAfter, starters)
	}
	if othersAfter != 1 {
		t.Errorf("bundle device not added: got %d non-starter walks want 1", othersAfter)
	}
	if _, statErr := os.Stat(filepath.Join(lib.SubDir(library.KindWalks), newDeviceWalk)); statErr != nil {
		t.Errorf("bundle walk not on disk: %v", statErr)
	}
}

// TestAdoptPackagedBundleSkipsWhenSeeded proves the once-guard: with a
// real (non-starter) walk already present the library is seeded, so a
// second adoption is a no-op. This is also the natural re-boot guard —
// after the first adoption walks/ holds SourceUser bundle files.
func TestAdoptPackagedBundleSkipsWhenSeeded(t *testing.T) {
	lib, err := library.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open library: %v", err)
	}
	// Plant a user walk (not in the embedded starter set) → seeded.
	userWalk := filepath.Join(lib.SubDir(library.KindWalks), "user-supplied-01.walk")
	if writeErr := os.WriteFile(userWalk, []byte("1.3.6.1 = STRING: \"u\"\n"), 0o644); writeErr != nil {
		t.Fatalf("plant user walk: %v", writeErr)
	}

	bundle := writePackagedBundle(t)
	_, installed, err := content.AdoptPackagedBundle(lib, bundle)
	if err != nil {
		t.Fatalf("adopt: %v", err)
	}
	if installed {
		t.Fatal("expected adoption to be skipped when user content is present")
	}
	if _, statErr := os.Stat(filepath.Join(lib.SubDir(library.KindWalks), newDeviceWalk)); statErr == nil {
		t.Error("bundle was extracted despite library already being seeded")
	}
}

// TestAdoptPackagedBundleEmptyWalks confirms adoption still runs when
// walks/ is empty (no starters, no user files).
func TestAdoptPackagedBundleEmptyWalks(t *testing.T) {
	lib, err := library.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open library: %v", err)
	}
	// Strip the seeded starters so walks/ is genuinely empty.
	files, err := lib.ListFiles(library.KindWalks)
	if err != nil {
		t.Fatalf("list walks: %v", err)
	}
	walksDir := lib.SubDir(library.KindWalks)
	for _, f := range files {
		if rmErr := os.Remove(filepath.Join(walksDir, f.Name)); rmErr != nil {
			t.Fatalf("remove starter %s: %v", f.Name, rmErr)
		}
	}

	bundle := writePackagedBundle(t)
	_, installed, err := content.AdoptPackagedBundle(lib, bundle)
	if err != nil {
		t.Fatalf("adopt: %v", err)
	}
	if !installed {
		t.Fatal("expected adoption when walks/ is empty")
	}
	if _, statErr := os.Stat(filepath.Join(walksDir, newDeviceWalk)); statErr != nil {
		t.Errorf("bundle walk not on disk: %v", statErr)
	}
}

// TestAdoptPackagedBundleMissingFileIsNoop confirms a missing bundle
// path (niac-content package not installed) is a silent no-op.
func TestAdoptPackagedBundleMissingFileIsNoop(t *testing.T) {
	lib, err := library.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open library: %v", err)
	}
	// Empty walks/ so the seeded-check doesn't short-circuit first.
	files, _ := lib.ListFiles(library.KindWalks)
	walksDir := lib.SubDir(library.KindWalks)
	for _, f := range files {
		_ = os.Remove(filepath.Join(walksDir, f.Name))
	}

	_, installed, err := content.AdoptPackagedBundle(lib, filepath.Join(t.TempDir(), "absent.tar.gz"))
	if err != nil {
		t.Fatalf("adopt with missing bundle should not error: %v", err)
	}
	if installed {
		t.Fatal("expected no adoption when bundle file is absent")
	}
}
