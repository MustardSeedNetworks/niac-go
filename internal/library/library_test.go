package library_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/library"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp"
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
//
// walks/ is pre-seeded with a placeholder before Open so the starter
// walks bootstrap sees a non-empty dir and skips — most callers plant
// their own walk fixtures and expect walks/ to start pristine. The
// starter-walks bootstrap itself is covered by
// TestOpenBootstrapsStarterWalks / TestOpenSkipsWalksBootstrapWhenWalksHasContent.
func openTempLibrary(t *testing.T) *library.Library {
	t.Helper()
	root := t.TempDir()
	t.Setenv("NIAC_LIBRARY_ROOT", root)

	walksDir := filepath.Join(root, "walks")
	if err := os.MkdirAll(walksDir, 0o755); err != nil {
		t.Fatalf("pre-create walks dir: %v", err)
	}
	placeholder := filepath.Join(walksDir, ".keep")
	if err := os.WriteFile(placeholder, nil, 0o644); err != nil {
		t.Fatalf("write walks placeholder: %v", err)
	}

	lib, err := library.Open(root)
	if err != nil {
		t.Fatalf("open library: %v", err)
	}
	if removeErr := os.Remove(placeholder); removeErr != nil {
		t.Fatalf("remove walks placeholder: %v", removeErr)
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

func TestOpenBootstrapsStarterWalks(t *testing.T) {
	root := t.TempDir()
	t.Setenv("NIAC_LIBRARY_ROOT", root)
	lib, err := library.Open(root)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	entries, err := lib.ListFiles(library.KindWalks)
	if err != nil {
		t.Fatalf("list walks: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected starter walks to unpack into walks/, got 0 entries")
	}

	starterCount := 0
	for _, e := range entries {
		if e.SizeBytes == 0 {
			t.Errorf("starter walk %s is empty", e.Name)
		}
		if e.Source == library.SourceStarter {
			starterCount++
		}
	}
	if starterCount != len(entries) {
		t.Errorf("expected every bootstrapped walk to be SourceStarter, got %d/%d", starterCount, len(entries))
	}

	for _, e := range entries {
		path := filepath.Join(lib.SubDir(library.KindWalks), e.Name)
		walkEntries, parseErr := snmp.ParseWalkFile(path)
		if parseErr != nil {
			t.Errorf("starter walk %s does not parse: %v", e.Name, parseErr)
			continue
		}
		if len(walkEntries) == 0 {
			t.Errorf("starter walk %s parsed with 0 entries", e.Name)
		}
	}
}

func TestOpenSkipsWalksBootstrapWhenWalksHasContent(t *testing.T) {
	root := t.TempDir()
	// Pre-create walks/ with a user file before Open. Bootstrap must
	// NOT touch that dir.
	if err := os.MkdirAll(filepath.Join(root, "walks"), 0o755); err != nil {
		t.Fatal(err)
	}
	userWalk := filepath.Join(root, "walks", "user.walk")
	if err := os.WriteFile(userWalk, []byte("1.3.6.1 = STRING: x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	lib, err := library.Open(root)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	entries, err := lib.ListFiles(library.KindWalks)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected exactly 1 entry (the pre-existing user file), got %d: %v", len(entries), entries)
	}
	if entries[0].Name != "user.walk" {
		t.Errorf("expected name=user.walk, got %s", entries[0].Name)
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

// networkPathTraversalNames is the shared set of malicious names used by
// TestReadNetworkPathContainment, TestWriteNetworkPathContainment, and
// TestDeleteNetworkPathContainment — all three entry points route
// through the shared resolveNetworkPath containment helper (CodeQL
// go/path-injection #42/#43/#44). Every one must be rejected with
// ErrInvalidName before any filesystem call happens.
func networkPathTraversalNames() []string {
	return []string{
		"../escape",
		"../../etc/passwd",
		"..",
		"a/../../b",
		"with/slash",
		`with\backslash`,
		"/etc/passwd",
		"/absolute/path",
		".hidden",
		"",
	}
}

func TestReadNetworkPathContainment(t *testing.T) {
	lib := openTempLibrary(t)
	for _, name := range networkPathTraversalNames() {
		t.Run(name, func(t *testing.T) {
			_, err := lib.ReadNetwork(name)
			if !errors.Is(err, library.ErrInvalidName) {
				t.Errorf("ReadNetwork(%q): want ErrInvalidName, got %v", name, err)
			}
		})
	}
}

func TestWriteNetworkPathContainment(t *testing.T) {
	lib := openTempLibrary(t)
	for _, name := range networkPathTraversalNames() {
		t.Run(name, func(t *testing.T) {
			err := lib.WriteNetwork(name, sampleYAML)
			if !errors.Is(err, library.ErrInvalidName) {
				t.Errorf("WriteNetwork(%q): want ErrInvalidName, got %v", name, err)
			}
		})
	}
}

func TestDeleteNetworkPathContainment(t *testing.T) {
	lib := openTempLibrary(t)
	for _, name := range networkPathTraversalNames() {
		t.Run(name, func(t *testing.T) {
			err := lib.DeleteNetwork(name)
			if !errors.Is(err, library.ErrInvalidName) {
				t.Errorf("DeleteNetwork(%q): want ErrInvalidName, got %v", name, err)
			}
		})
	}
}

func TestNetworkPathLegitimateNameStillResolves(t *testing.T) {
	lib := openTempLibrary(t)
	if err := lib.WriteNetwork("safe-name_1.0", sampleYAML); err != nil {
		t.Fatalf("write: %v", err)
	}
	doc, readErr := lib.ReadNetwork("safe-name_1.0")
	if readErr != nil {
		t.Fatalf("read: %v", readErr)
	}
	if doc.Name != "safe-name_1.0" {
		t.Errorf("name: got %q want safe-name_1.0", doc.Name)
	}
	if deleteErr := lib.DeleteNetwork("safe-name_1.0"); deleteErr != nil {
		t.Fatalf("delete: %v", deleteErr)
	}
}

// TestNetworkPathNoEscapeOntoDisk is belt-and-suspenders: even if a
// future change to validateName regressed the character allowlist,
// resolveNetworkPath's absolute-prefix containment check must still
// refuse to resolve outside the networks/ base directory. This proves
// the escaped file was never created on disk.
func TestNetworkPathNoEscapeOntoDisk(t *testing.T) {
	lib := openTempLibrary(t)
	outsideMarker := filepath.Join(t.TempDir(), "outside.yaml")
	if err := lib.WriteNetwork("../"+filepath.Base(outsideMarker), sampleYAML); err == nil {
		t.Fatalf("expected traversal write to be rejected")
	}
	if _, statErr := os.Stat(outsideMarker); !os.IsNotExist(statErr) {
		t.Errorf("traversal write leaked a file outside the library root: %v", statErr)
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

func TestReadFileRoundTrip(t *testing.T) {
	lib := openTempLibrary(t)
	root := lib.Root()

	walkPath := filepath.Join(root, "walks", "router.walk")
	if err := os.WriteFile(walkPath, []byte("1.3.6.1 = STRING: x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := lib.ReadFile(library.KindWalks, "router.walk")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "1.3.6.1 = STRING: x\n" {
		t.Errorf("ReadFile content = %q, want %q", got, "1.3.6.1 = STRING: x\n")
	}
}

func TestReadFileMissing(t *testing.T) {
	lib := openTempLibrary(t)
	_, err := lib.ReadFile(library.KindWalks, "missing.walk")
	if !errors.Is(err, library.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestReadFileInvalidName(t *testing.T) {
	lib := openTempLibrary(t)
	_, err := lib.ReadFile(library.KindWalks, "../../etc/passwd")
	if !errors.Is(err, library.ErrInvalidName) {
		t.Errorf("want ErrInvalidName, got %v", err)
	}
}

func TestReadFileRejectsNetworksKind(t *testing.T) {
	lib := openTempLibrary(t)
	_, err := lib.ReadFile(library.KindNetworks, "anything.yaml")
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
