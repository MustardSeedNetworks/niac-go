package library_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/library"
)

// sampleYAML is the minimum YAML the WriteNetwork validator accepts:
// it must contain a "devices:" section. Body is otherwise irrelevant.
const sampleYAML = `# Description: example test network
# Use Case: integration testing
devices:
  - name: r1
    type: router
`

// openTempLibrary returns a fresh Library rooted at t.TempDir(). The
// starter pack auto-unpacks into networks/ on first Open. Tests that
// want a fully empty library should call ListNetworks → DeleteNetwork
// for every entry, or simply work around the starter content.
func openTempLibrary(t *testing.T) *library.Library {
	t.Helper()
	root := t.TempDir()
	t.Setenv("NIAC_LIBRARY_ROOT", root)
	lib, err := library.Open(root)
	if err != nil {
		t.Fatalf("open library: %v", err)
	}
	return lib
}

func TestOpenCreatesLayout(t *testing.T) {
	root := t.TempDir()
	if _, err := library.Open(root); err != nil {
		t.Fatalf("open: %v", err)
	}
	for _, sub := range []string{"networks", "walks", "pcaps"} {
		info, err := os.Stat(filepath.Join(root, sub))
		if err != nil {
			t.Errorf("expected subdir %s: %v", sub, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%s is not a directory", sub)
		}
	}
}

func TestOpenBootstrapsStarterPack(t *testing.T) {
	lib := openTempLibrary(t)
	entries, err := lib.ListNetworks()
	if err != nil {
		t.Fatalf("list networks: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected starter pack to unpack into networks/, got 0 entries")
	}
	starterCount := 0
	for _, e := range entries {
		if e.Source == library.SourceStarter {
			starterCount++
		}
	}
	if starterCount == 0 {
		t.Errorf("expected at least one SourceStarter entry, got %d entries with no starter source", len(entries))
	}
}

func TestOpenSkipsBootstrapWhenNetworksHasContent(t *testing.T) {
	root := t.TempDir()
	// Pre-create networks/ with a user file before Open. Bootstrap
	// must NOT touch that dir.
	if err := os.MkdirAll(filepath.Join(root, "networks"), 0o755); err != nil {
		t.Fatal(err)
	}
	userFile := filepath.Join(root, "networks", "user.yaml")
	if err := os.WriteFile(userFile, []byte(sampleYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	lib, err := library.Open(root)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	entries, err := lib.ListNetworks()
	if err != nil {
		t.Fatal(err)
	}
	// Only the user file should be present; the starter pack should
	// have been skipped.
	if len(entries) != 1 {
		t.Fatalf("expected exactly 1 entry (the pre-existing user file), got %d: %v", len(entries), entries)
	}
	if entries[0].Name != "user" {
		t.Errorf("expected name=user, got %s", entries[0].Name)
	}
	if entries[0].Source != library.SourceUser {
		t.Errorf("expected source=user, got %s", entries[0].Source)
	}
}

func TestWriteNetworkRoundTrip(t *testing.T) {
	lib := openTempLibrary(t)
	if err := lib.WriteNetwork("my-net", sampleYAML); err != nil {
		t.Fatalf("write: %v", err)
	}
	doc, err := lib.ReadNetwork("my-net")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if doc.Name != "my-net" {
		t.Errorf("name: got %q want my-net", doc.Name)
	}
	if !strings.Contains(doc.Content, "devices:") {
		t.Errorf("content roundtrip lost devices: section: %q", doc.Content)
	}
	if doc.Format != "yaml" {
		t.Errorf("format: got %q want yaml", doc.Format)
	}
}

func TestWriteNetworkRejectsInvalidName(t *testing.T) {
	lib := openTempLibrary(t)
	cases := []string{
		"../escape",
		"with/slash",
		".hidden",
		"",
		"has space",
	}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			err := lib.WriteNetwork(name, sampleYAML)
			if !errors.Is(err, library.ErrInvalidName) {
				t.Errorf("want ErrInvalidName for %q, got %v", name, err)
			}
		})
	}
}

func TestWriteNetworkRejectsEmptyOrBrokenContent(t *testing.T) {
	lib := openTempLibrary(t)
	if err := lib.WriteNetwork("empty", ""); !errors.Is(err, library.ErrEmptyContent) {
		t.Errorf("empty: want ErrEmptyContent, got %v", err)
	}
	// Missing the required `devices:` section is treated as
	// effectively empty so the picker doesn't surface broken rows.
	if err := lib.WriteNetwork("no-devices", "# just a comment\n"); !errors.Is(err, library.ErrEmptyContent) {
		t.Errorf("no devices: want ErrEmptyContent, got %v", err)
	}
}

func TestReadNetworkMissing(t *testing.T) {
	lib := openTempLibrary(t)
	_, err := lib.ReadNetwork("does-not-exist")
	if !errors.Is(err, library.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestReadNetworkInvalidName(t *testing.T) {
	lib := openTempLibrary(t)
	_, err := lib.ReadNetwork("../etc/passwd")
	if !errors.Is(err, library.ErrInvalidName) {
		t.Errorf("want ErrInvalidName, got %v", err)
	}
}

func TestDeleteNetworkRefusesStarter(t *testing.T) {
	lib := openTempLibrary(t)
	entries, err := lib.ListNetworks()
	if err != nil {
		t.Fatal(err)
	}
	var starter string
	for _, e := range entries {
		if e.Source == library.SourceStarter {
			starter = e.Name
			break
		}
	}
	if starter == "" {
		t.Skip("no starter networks present; nothing to test")
	}
	err = lib.DeleteNetwork(starter)
	if err == nil {
		t.Fatalf("expected starter delete to be refused, got nil")
	}
	if !strings.Contains(err.Error(), "starter") {
		t.Errorf("error should mention starter, got %v", err)
	}
}

func TestDeleteNetworkRemovesUserFile(t *testing.T) {
	lib := openTempLibrary(t)
	if err := lib.WriteNetwork("doomed", sampleYAML); err != nil {
		t.Fatal(err)
	}
	if err := lib.DeleteNetwork("doomed"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := lib.ReadNetwork("doomed"); !errors.Is(err, library.ErrNotFound) {
		t.Errorf("after delete, expected ErrNotFound, got %v", err)
	}
}

func TestListFilesWalksAndPcaps(t *testing.T) {
	lib := openTempLibrary(t)
	root := lib.Root()

	// Plant fixtures under walks/ (with one vendor subdir) and pcaps/.
	if err := os.MkdirAll(filepath.Join(root, "walks", "cisco"), 0o755); err != nil {
		t.Fatal(err)
	}
	walks := map[string]string{
		"walks/cisco/c3900.walk": "1.3.6.1 = STRING: x\n",
		"walks/router.walk":      "1.3.6.1 = STRING: y\n",
	}
	for rel, body := range walks {
		path := filepath.Join(root, rel)
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	pcap := filepath.Join(root, "pcaps", "sample.pcap")
	if err := os.WriteFile(pcap, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}

	walkEntries, err := lib.ListFiles(library.KindWalks)
	if err != nil {
		t.Fatalf("list walks: %v", err)
	}
	if len(walkEntries) != 2 {
		t.Errorf("walks count = %d, want 2 (%v)", len(walkEntries), walkEntries)
	}
	// Sorted by name; cisco/ subdir comes before bare files.
	if walkEntries[0].Name != "cisco/c3900.walk" {
		t.Errorf("walks[0] = %s, want cisco/c3900.walk", walkEntries[0].Name)
	}

	pcapEntries, err := lib.ListFiles(library.KindPcaps)
	if err != nil {
		t.Fatalf("list pcaps: %v", err)
	}
	if len(pcapEntries) != 1 || pcapEntries[0].Name != "sample.pcap" {
		t.Errorf("pcaps = %v, want [sample.pcap]", pcapEntries)
	}
}

func TestListFilesRejectsNetworksKind(t *testing.T) {
	lib := openTempLibrary(t)
	_, err := lib.ListFiles(library.KindNetworks)
	if !errors.Is(err, library.ErrUnsupportedKind) {
		t.Errorf("want ErrUnsupportedKind for KindNetworks, got %v", err)
	}
}

func TestDefaultRootHonoursEnv(t *testing.T) {
	t.Setenv("NIAC_LIBRARY_ROOT", "/tmp/custom-niac-library-root")
	got := library.DefaultRoot()
	if got != "/tmp/custom-niac-library-root" {
		t.Errorf("DefaultRoot = %q, want /tmp/custom-niac-library-root", got)
	}
}

func TestSubDir(t *testing.T) {
	lib := openTempLibrary(t)
	root := lib.Root()

	walksDir := lib.SubDir(library.KindWalks)
	if !strings.HasSuffix(walksDir, "walks") {
		t.Errorf("SubDir(walks) = %q, want suffix walks", walksDir)
	}
	if !strings.HasPrefix(walksDir, root) {
		t.Errorf("SubDir(walks) = %q must be under root %q", walksDir, root)
	}
}

func TestAllKindsStability(t *testing.T) {
	got := library.AllKinds()
	want := []library.Kind{library.KindNetworks, library.KindWalks, library.KindPcaps}
	if len(got) != len(want) {
		t.Fatalf("AllKinds len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("AllKinds[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
