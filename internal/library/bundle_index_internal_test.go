package library

import (
	"os"
	"path/filepath"
	"testing"
)

// A bundle's files land in the same directories as the operator's own and look
// identical afterwards, so without a record every installed file was reported
// as "user" - and the Bundle badge the UI already draws could never appear.
func TestBundleInstalledContentReportsItsSource(t *testing.T) {
	lib := newLibraryWithFile(t, KindNetworks, "campus.yaml")
	if err := RecordBundleInstall(lib.root, "v1", map[string]string{"networks/campus.yaml": "abc"}); err != nil {
		t.Fatalf("RecordBundleInstall: %v", err)
	}

	if got := lib.detectSource("campus.yaml"); got != SourceBundle {
		t.Errorf("source = %q, want %q", got, SourceBundle)
	}
}

// Anything the operator wrote is still theirs.
func TestUnrecordedContentIsStillTheOperatorsOwn(t *testing.T) {
	lib := newLibraryWithFile(t, KindNetworks, "mine.yaml")

	if got := lib.detectSource("mine.yaml"); got != SourceUser {
		t.Errorf("source = %q, want %q", got, SourceUser)
	}
}

// Walks and pcaps arrive in bundles too, and are asked about by kind.
func TestBundleInstalledWalksReportTheirSource(t *testing.T) {
	lib := newLibraryWithFile(t, KindWalks, "switch-01.walk")
	if err := RecordBundleInstall(lib.root, "v1", map[string]string{"walks/switch-01.walk": "abc"}); err != nil {
		t.Fatalf("RecordBundleInstall: %v", err)
	}

	if got := lib.detectFileSource(KindWalks, "switch-01.walk"); got != SourceBundle {
		t.Errorf("source = %q, want %q", got, SourceBundle)
	}
}

// The starter walks ship inside the binary and outrank the index: a bundle
// cannot claim a file that reappears on every bootstrap.
func TestStarterWalksOutrankTheIndex(t *testing.T) {
	starter := firstStarterWalk(t)
	lib := newLibraryWithFile(t, KindWalks, starter)
	if err := RecordBundleInstall(lib.root, "v1", map[string]string{"walks/" + starter: "abc"}); err != nil {
		t.Fatalf("RecordBundleInstall: %v", err)
	}

	if got := lib.detectFileSource(KindWalks, starter); got != SourceStarter {
		t.Errorf("starter walk %s reported as %q", starter, got)
	}
}

// A shipped scenario is an Example served from the binary, never a file, so a
// network file named like one is the operator's even when a bundle also wrote it.
func TestNetworkFileNamedLikeAnExampleIsNotStarter(t *testing.T) {
	lib := newLibraryWithFile(t, KindNetworks, "basic-network.yaml")
	if got := lib.detectSource("basic-network.yaml"); got != SourceUser {
		t.Errorf("source = %q, want %q", got, SourceUser)
	}
}

func newLibraryWithFile(t *testing.T, kind Kind, name string) *Library {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, string(kind))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte("devices: []\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	return &Library{root: root}
}

func firstStarterWalk(t *testing.T) string {
	t.Helper()
	entries, err := starterWalks.ReadDir("starter/walks")
	if err != nil || len(entries) == 0 {
		t.Fatalf("the starter pack ships no walk (err=%v)", err)
	}

	return entries[0].Name()
}
