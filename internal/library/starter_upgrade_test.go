package library_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/library"
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

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// seedUpgradedLibrary lays out a library as an earlier release left it: a user
// network in the old schema and a walk so the walks directory is not empty.
func seedUpgradedLibrary(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "networks", "my-lab.yaml"), staleSchemaNetwork)
	writeFile(t, filepath.Join(root, "walks", "user.walk"), "1.3.6.1 = STRING: x\n")
	return root
}

// ListNetworks reports what `niac validate` would: a network the strict
// loader rejects is invalid, with its first error on one line.
func TestListNetworksMarksUnloadableNetworkInvalid(t *testing.T) {
	root := seedUpgradedLibrary(t)
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

	example := byName["basic-network"]
	if !example.Valid || example.Error != "" {
		t.Errorf("example listed invalid: %+v", example)
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
