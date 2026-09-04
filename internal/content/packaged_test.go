package content_test

import (
	"maps"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/content"
	"github.com/MustardSeedNetworks/niac-go/internal/library"
)

// newDeviceWalk is a device name deliberately absent from the embedded
// starter walk pack, so it can only be on disk because the bundle put it
// there.
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

// TestAdoptPackagedBundleIgnoresUnrelatedUserContent pins the fix: a walk
// the operator uploaded before the niac-content package was installed used
// to mark the library "seeded" and block the full bundle for good. Adoption
// is gated on the recorded bundle version now, so the bundle still lands and
// the operator's walk is untouched.
func TestAdoptPackagedBundleIgnoresUnrelatedUserContent(t *testing.T) {
	lib, err := library.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open library: %v", err)
	}
	userWalk := filepath.Join(lib.SubDir(library.KindWalks), "user-supplied-01.walk")
	const userBody = "1.3.6.1 = STRING: \"u\"\n"
	if writeErr := os.WriteFile(userWalk, []byte(userBody), 0o644); writeErr != nil {
		t.Fatalf("plant user walk: %v", writeErr)
	}

	bundle := writePackagedBundle(t)
	_, installed, err := content.AdoptPackagedBundle(lib, bundle)
	if err != nil {
		t.Fatalf("adopt: %v", err)
	}
	if !installed {
		t.Fatal("one uploaded walk blocked the packaged bundle")
	}
	if _, statErr := os.Stat(filepath.Join(lib.SubDir(library.KindWalks), newDeviceWalk)); statErr != nil {
		t.Errorf("bundle walk not on disk: %v", statErr)
	}
	if got, _ := os.ReadFile(userWalk); string(got) != userBody {
		t.Errorf("operator's walk disturbed: got %q want %q", got, userBody)
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

// writeBundle synthesises a niac-content bundle carrying the given
// walks/ entries and returns its path on disk. Each bundle version gets
// its own file so a test can adopt v1 and v2 in turn.
func writeBundle(t *testing.T, name string, walks map[string]string) string {
	t.Helper()
	entries := make([]tarballEntry, 0, len(walks))
	for _, device := range slices.Sorted(maps.Keys(walks)) {
		entries = append(entries, tarballEntry{name: "walks/" + device, body: walks[device]})
	}
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, buildTarball(t, entries), 0o644); err != nil {
		t.Fatalf("write bundle %s: %v", name, err)
	}
	return path
}

func readWalk(t *testing.T, lib *library.Library, device string) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(lib.SubDir(library.KindWalks), device))
	if err != nil {
		t.Fatalf("read walk %s: %v", device, err)
	}
	return string(body)
}

// TestAdoptPackagedBundleUpgrade is the P1c-5 acceptance: install v1,
// upload a walk of your own, edit one of v1's files, then install v2.
//
// Every clause here failed before the bundle carried a version: adoption
// was gated on "walks/ holds nothing but starters", so the operator's own
// walk permanently blocked the upgrade, and the one adoption that did run
// overwrote edited files without asking.
func TestAdoptPackagedBundleUpgrade(t *testing.T) {
	lib, err := library.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open library: %v", err)
	}
	walksDir := lib.SubDir(library.KindWalks)

	const deviceA, deviceB, deviceC = "zzz-a-01.walk", "zzz-b-01.walk", "zzz-c-01.walk"
	v1 := writeBundle(t, "v1.tar.gz", map[string]string{
		deviceA: "walk A v1\n",
		deviceB: "walk B v1\n",
	})
	if _, installed, adoptErr := content.AdoptPackagedBundle(lib, v1); adoptErr != nil || !installed {
		t.Fatalf("adopt v1: installed=%v err=%v", installed, adoptErr)
	}

	// The operator adds a walk of their own and edits one the bundle shipped.
	const userWalk = "zzz-user-01.walk"
	const userBody = "walk from the operator\n"
	if writeErr := os.WriteFile(filepath.Join(walksDir, userWalk), []byte(userBody), 0o644); writeErr != nil {
		t.Fatalf("plant user walk: %v", writeErr)
	}
	const editedBody = "walk A, edited by the operator\n"
	if writeErr := os.WriteFile(filepath.Join(walksDir, deviceA), []byte(editedBody), 0o644); writeErr != nil {
		t.Fatalf("edit bundle walk: %v", writeErr)
	}

	// v2 revises both shipped walks and adds a third.
	v2 := writeBundle(t, "v2.tar.gz", map[string]string{
		deviceA: "walk A v2\n",
		deviceB: "walk B v2\n",
		deviceC: "walk C v2\n",
	})
	_, installed, err := content.AdoptPackagedBundle(lib, v2)
	if err != nil {
		t.Fatalf("adopt v2: %v", err)
	}
	if !installed {
		t.Fatal("v2 was not adopted: a different bundle version must re-sync, " +
			"even though the operator's own walk is present")
	}

	if got := readWalk(t, lib, deviceC); got != "walk C v2\n" {
		t.Errorf("new walk from v2 missing: got %q", got)
	}
	if got := readWalk(t, lib, deviceB); got != "walk B v2\n" {
		t.Errorf("untouched shipped walk not upgraded: got %q want v2 content", got)
	}
	if got := readWalk(t, lib, deviceA); got != editedBody {
		t.Errorf("operator's edit was overwritten: got %q want %q", got, editedBody)
	}
	if got := readWalk(t, lib, userWalk); got != userBody {
		t.Errorf("operator's own walk was disturbed: got %q want %q", got, userBody)
	}

	// A third adoption of the same bundle is a no-op: the recorded version
	// already matches, so re-syncing costs nothing on every daemon boot.
	if _, again, adoptErr := content.AdoptPackagedBundle(lib, v2); adoptErr != nil || again {
		t.Errorf("re-adopting the same version: installed=%v err=%v, want false/nil", again, adoptErr)
	}
}
