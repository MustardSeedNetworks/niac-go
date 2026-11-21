package library_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/library"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
	"github.com/MustardSeedNetworks/niac-go/internal/templates"
)

func entryByName(t *testing.T, lib *library.Library, name string) (library.NetworkEntry, bool) {
	t.Helper()
	entries, err := lib.ListNetworks()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, entry := range entries {
		if entry.Name == name {
			return entry, true
		}
	}
	return library.NetworkEntry{}, false
}

func TestListNetworksIncludesEveryShippedScenarioAsExample(t *testing.T) {
	lib := openTempLibrary(t)

	want := templates.ListNames()
	for _, pack := range scenario.Packs() {
		want = append(want, pack.ID)
	}
	for _, name := range want {
		entry, ok := entryByName(t, lib, name)
		if !ok {
			t.Errorf("%s: missing from the listing", name)
			continue
		}
		if entry.Source != library.SourceExample || !entry.Valid || entry.DeviceCount == 0 {
			t.Errorf("%s: want a valid example with devices, got %+v", name, entry)
		}
		if entry.Description == "" {
			t.Errorf("%s: no description", name)
		}
	}
	if _, ok := entryByName(t, lib, "small-office"); ok {
		t.Error("small-office was removed from the built-ins and must not be listed")
	}
}

func TestReadNetworkServesExample(t *testing.T) {
	lib := openTempLibrary(t)

	doc, err := lib.ReadNetwork("basic-network")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	tmpl, err := templates.Get("basic-network")
	if err != nil {
		t.Fatal(err)
	}
	if doc.Source != library.SourceExample || doc.Content != tmpl.Content {
		t.Errorf("want the embedded template as an example, got source=%q (%d bytes)", doc.Source, len(doc.Content))
	}
	if _, readErr := lib.ReadNetwork("basic-network.yaml"); readErr != nil {
		t.Errorf("the .yaml spelling must resolve too: %v", readErr)
	}
}

func TestExamplesAreReadOnly(t *testing.T) {
	lib := openTempLibrary(t)

	if err := lib.DeleteNetwork("basic-network"); !errors.Is(err, library.ErrReadOnlyExample) {
		t.Errorf("delete of an example: want ErrReadOnlyExample, got %v", err)
	}
	if err := lib.WriteNetwork("basic-network", sampleYAML); !errors.Is(err, library.ErrReadOnlyExample) {
		t.Errorf("write over an example: want ErrReadOnlyExample, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(lib.SubDir(library.KindNetworks), "basic-network.yaml")); !os.IsNotExist(err) {
		t.Errorf("a refused write must leave nothing behind, stat err = %v", err)
	}
}

func TestDuplicateExampleSavesEditableCopy(t *testing.T) {
	lib := openTempLibrary(t)

	name, err := lib.DuplicateExample("basic-network", "")
	if err != nil {
		t.Fatalf("duplicate: %v", err)
	}
	if name != "basic-network-copy" {
		t.Errorf("default name = %q, want basic-network-copy", name)
	}
	entry, ok := entryByName(t, lib, name)
	if !ok || entry.Source != library.SourceUser {
		t.Fatalf("the copy must be listed as the operator's own, got %+v (found=%v)", entry, ok)
	}
	tmpl, _ := templates.Get("basic-network")
	if got, readErr := lib.ReadNetwork(name); readErr != nil || got.Content != tmpl.Content {
		t.Errorf("copy content differs from the example (err=%v)", readErr)
	}
	if err = lib.WriteNetwork(name, sampleYAML); err != nil {
		t.Errorf("the copy must be editable: %v", err)
	}
	if err = lib.DeleteNetwork(name); err != nil {
		t.Errorf("the copy must be deletable: %v", err)
	}
}

func TestDuplicateExampleRefusals(t *testing.T) {
	lib := openTempLibrary(t)
	if err := lib.WriteNetwork("taken", sampleYAML); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		example string
		newName string
		want    error
	}{
		{"not an example", "no-such-example", "", library.ErrNotFound},
		{"existing network is never replaced", "basic-network", "taken", library.ErrAlreadyExists},
		{"another example's name is reserved", "basic-network", "captured-switch", library.ErrReadOnlyExample},
		{"invalid name", "basic-network", "../escape", library.ErrInvalidName},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := lib.DuplicateExample(tt.example, tt.newName); !errors.Is(err, tt.want) {
				t.Errorf("want %v, got %v", tt.want, err)
			}
		})
	}
	got, err := lib.ReadNetwork("taken")
	if err != nil || got.Content != sampleYAML {
		t.Errorf("the existing network was modified (err=%v)", err)
	}
}

// A file named like an example is the operator's, wins every lookup, and keeps
// the list to one row per name; deleting it lets the example show through.
func TestFileShadowsExampleOfSameName(t *testing.T) {
	lib := openTempLibrary(t)
	writeFile(t, filepath.Join(lib.SubDir(library.KindNetworks), "basic-network.yaml"), sampleYAML)

	entries, err := lib.ListNetworks()
	if err != nil {
		t.Fatal(err)
	}
	rows := 0
	for _, entry := range entries {
		if entry.Name == "basic-network" {
			rows++
			if entry.Source != library.SourceUser {
				t.Errorf("shadowing file source = %q, want user", entry.Source)
			}
		}
	}
	if rows != 1 {
		t.Errorf("basic-network listed %d times, want once", rows)
	}
	if doc, readErr := lib.ReadNetwork("basic-network"); readErr != nil || doc.Content != sampleYAML {
		t.Errorf("the file must win a read (err=%v)", readErr)
	}
	if err = lib.WriteNetwork("basic-network", sampleYAML+"# edited\n"); err != nil {
		t.Errorf("an existing file stays editable: %v", err)
	}
	if err = lib.DeleteNetwork("basic-network"); err != nil {
		t.Fatalf("delete shadowing file: %v", err)
	}
	if entry, ok := entryByName(t, lib, "basic-network"); !ok || entry.Source != library.SourceExample {
		t.Errorf("after delete the example should show through, got %+v (found=%v)", entry, ok)
	}
}

func TestExampleStartContentMakesIncludePathAbsolute(t *testing.T) {
	walks := filepath.Join(t.TempDir(), "walks")

	staged, err := library.ExampleStartContent("captured-switch", walks)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(staged), "include_path: "+walks) {
		t.Errorf("include_path not made absolute against %s:\n%.400s", walks, staged)
	}
	pristine, err := library.ExampleContent("captured-switch")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(pristine), "include_path: ../walks") {
		t.Error("the served document must keep its relative include_path")
	}

	plain, err := library.ExampleStartContent("basic-network", walks)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := library.ExampleContent("basic-network")
	if string(plain) != string(want) {
		t.Error("a template with no include_path must pass through byte for byte")
	}
}
