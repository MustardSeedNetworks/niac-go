package daemon

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

const managedPathTestYAML = `devices:
  - name: edge
    type: router
    mac: 02:00:00:00:00:01
    ips: [10.254.200.1]
`

func TestLoadSimulationConfigRestrictsPathsToManagedRoots(t *testing.T) {
	root := t.TempDir()
	networks := filepath.Join(root, "networks")
	if err := os.Mkdir(networks, 0o750); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NIAC_LIBRARY_ROOT", root)
	t.Setenv("NIAC_CONFIGS_DIR", filepath.Join(root, "configs"))

	managed := writeManagedPathTestConfig(t, networks, "managed.yaml")
	managedReal, realErr := filepath.EvalSymlinks(managed)
	if realErr != nil {
		t.Fatal(realErr)
	}
	outside := writeManagedPathTestConfig(t, t.TempDir(), "outside.yaml")
	escape := filepath.Join(networks, "escape.yaml")
	if symlinkErr := os.Symlink(outside, escape); symlinkErr != nil {
		t.Fatal(symlinkErr)
	}

	tests := []struct {
		name string
		path string
		ok   bool
	}{
		{name: "managed absolute path", path: managed, ok: true},
		{name: "absolute outside root", path: outside},
		{name: "symlink escape", path: escape},
		{
			name: "traversal",
			path: networks + string(filepath.Separator) + ".." +
				string(filepath.Separator) + filepath.Base(outside),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, resolved, loadErr := loadSimulationConfig(api.SimulationRequest{ConfigPath: tt.path}, false)
			if tt.ok {
				if loadErr != nil {
					t.Fatalf("loadSimulationConfig() error = %v", loadErr)
				}
				if resolved != managedReal {
					t.Fatalf("resolved path = %q, want %q", resolved, managedReal)
				}
				return
			}
			if !errors.Is(loadErr, config.ErrPathOutsideManagedRoots) {
				t.Fatalf("error = %v, want ErrPathOutsideManagedRoots", loadErr)
			}
		})
	}
}

func writeManagedPathTestConfig(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(managedPathTestYAML), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
