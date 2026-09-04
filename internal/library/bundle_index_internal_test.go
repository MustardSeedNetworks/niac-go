package library

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/templates"
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

// The starter pack ships inside the binary and outranks the index: a bundle
// cannot claim a file that reappears on every bootstrap.
func TestStarterContentOutranksTheIndex(t *testing.T) {
	lib := newLibraryWithFile(t, KindNetworks, "enterprise.yaml")
	if err := RecordBundleInstall(lib.root, "v1", map[string]string{"networks/enterprise.yaml": "abc"}); err != nil {
		t.Fatalf("RecordBundleInstall: %v", err)
	}
	starterName := firstStarterNetwork(t)

	if got := lib.detectSource(starterName); got != SourceStarter {
		t.Errorf("starter %s reported as %q", starterName, got)
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

func firstStarterNetwork(t *testing.T) string {
	t.Helper()
	names := templates.ListNames()
	if len(names) == 0 {
		t.Fatal("the starter pack ships no network")
	}

	return names[0] + ".yaml"
}
