package library_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/library"
	"github.com/MustardSeedNetworks/niac-go/internal/templates"
)

// staleSchemaNetwork is a network written by an earlier release's schema: a
// device carried a single `ip:` where the current schema has `ips:`. The
// current strict decoder rejects it, which is what #2202's library held.
const staleSchemaNetwork = `# Description: written by an earlier release
devices:
  - name: office-router
    type: router
    mac: "00:50:56:A1:01:01"
    ip: "192.168.10.1"
`

// seedUpgradedLibrary lays out a library as an earlier release left it: a
// starter network in the old schema, a user network in the old schema, a
// walk so neither directory is empty, and returns the root.
func seedUpgradedLibrary(t *testing.T, starterContent string) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "networks", "small-office.yaml"), starterContent)
	writeFile(t, filepath.Join(root, "networks", "my-lab.yaml"), staleSchemaNetwork)
	writeFile(t, filepath.Join(root, "walks", "user.walk"), "1.3.6.1 = STRING: x\n")
	return root
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// TestOpenRefreshesStarterWrittenByEarlierSchema guards #2202: a starter
// network the current release cannot load is replaced by the embedded
// template on open, the replaced bytes are kept beside it, and a user
// network is never rewritten.
func TestOpenRefreshesStarterWrittenByEarlierSchema(t *testing.T) {
	root := seedUpgradedLibrary(t, staleSchemaNetwork)
	networks := filepath.Join(root, "networks")

	if _, err := library.Open(root); err != nil {
		t.Fatalf("open: %v", err)
	}

	assertTemplateLoads(t, "small-office", filepath.Join(networks, "small-office.yaml"))
	tmpl, err := templates.Get("small-office")
	if err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(networks, "small-office.yaml")); got != tmpl.Content {
		t.Error("stale starter was not replaced by the embedded template")
	}
	if got := readFile(t, filepath.Join(networks, "small-office.yaml.orig")); got != staleSchemaNetwork {
		t.Errorf("replaced starter bytes not preserved in .orig, got:\n%s", got)
	}
	if got := readFile(t, filepath.Join(networks, "my-lab.yaml")); got != staleSchemaNetwork {
		t.Errorf("user network was rewritten:\n%s", got)
	}
}

// A starter the operator customised and that still loads is theirs: open
// must leave it exactly as it is.
func TestOpenKeepsCustomisedStarterThatLoads(t *testing.T) {
	tmpl, err := templates.Get("small-office")
	if err != nil {
		t.Fatal(err)
	}
	customised := tmpl.Content + "\n# operator note\n"
	root := seedUpgradedLibrary(t, customised)

	if _, openErr := library.Open(root); openErr != nil {
		t.Fatalf("open: %v", openErr)
	}

	networks := filepath.Join(root, "networks")
	if got := readFile(t, filepath.Join(networks, "small-office.yaml")); got != customised {
		t.Error("a customised starter that loads was overwritten")
	}
	if _, statErr := os.Stat(filepath.Join(networks, "small-office.yaml.orig")); !os.IsNotExist(statErr) {
		t.Errorf("no .orig expected for an untouched starter, stat err = %v", statErr)
	}
}

// ListNetworks reports what `niac validate` would: a network the strict
// loader rejects is invalid, with its first error on one line.
func TestListNetworksMarksUnloadableNetworkInvalid(t *testing.T) {
	root := seedUpgradedLibrary(t, staleSchemaNetwork)
	lib, err := library.Open(root)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	entries, err := lib.ListNetworks()
	if err != nil {
		t.Fatal(err)
	}
	byName := make(map[string]library.NetworkEntry, len(entries))
	for _, entry := range entries {
		byName[entry.Name] = entry
	}

	user := byName["my-lab"]
	if user.Valid {
		t.Fatal("my-lab is listed valid although the loader rejects it")
	}
	if !strings.Contains(user.Error, "field ip not found") {
		t.Errorf("error does not name the rejected field: %q", user.Error)
	}
	if strings.Contains(user.Error, "\n") {
		t.Errorf("error should be the first error on one line: %q", user.Error)
	}
	if user.DeviceCount != 1 {
		t.Errorf("an invalid row still reports its devices, got %d", user.DeviceCount)
	}

	starter := byName["small-office"]
	if !starter.Valid || starter.Error != "" {
		t.Errorf("refreshed starter listed invalid: %+v", starter)
	}
}

// A routed network the semantic validator passes but the fabric compiler
// refuses is invalid in the list too, as it is in `niac validate`: the
// interface address sits outside its own network.
func TestListNetworksMarksFabricDefectInvalid(t *testing.T) {
	lib := openTempLibrary(t)
	const routed = `networks:
  - name: clinical
    subnet: 10.20.0.0/24
devices:
  - name: core
    type: router
    mac: "00:11:22:33:44:55"
    interfaces:
      - name: eth0
        network: clinical
        address: 10.20.9.1/24
`
	if err := lib.WriteNetwork("routed", routed); err != nil {
		t.Fatal(err)
	}

	entries, err := lib.ListNetworks()
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name != "routed" {
			continue
		}
		if entry.Valid || entry.Error == "" {
			t.Fatalf("fabric defect listed valid: %+v", entry)
		}
		return
	}
	t.Fatal("routed network missing from the list")
}
