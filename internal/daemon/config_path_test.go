package daemon

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
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

func TestLoadSimulationConfigRestrictsInlineSegmentPathsToManagedRoots(t *testing.T) {
	base := t.TempDir()
	configsDir := filepath.Join(base, "configs")
	if err := os.Mkdir(configsDir, 0o750); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NIAC_LIBRARY_ROOT", filepath.Join(base, "library"))
	t.Setenv("NIAC_CONFIGS_DIR", configsDir)

	managed := writeManagedPathTestConfig(t, configsDir, "managed.yaml")
	outside := writeManagedPathTestConfig(t, base, "outside.yaml")
	escape := filepath.Join(configsDir, "escape.yaml")
	if err := os.Symlink(outside, escape); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		childPath string
		ok        bool
	}{
		{name: "managed absolute path", childPath: managed, ok: true},
		{name: "absolute outside root", childPath: outside},
		{name: "relative escape", childPath: "../outside.yaml"},
		{name: "symlink escape", childPath: escape},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configData := "segments:\n  - tag: 200\n    config: " + tt.childPath + "\n"
			_, _, err := loadSimulationConfig(api.SimulationRequest{ConfigData: configData}, false)
			if tt.ok {
				if err != nil {
					t.Fatalf("loadSimulationConfig() error = %v", err)
				}
				return
			}
			if !errors.Is(err, config.ErrPathOutsideManagedRoots) {
				t.Fatalf("error = %v, want ErrPathOutsideManagedRoots", err)
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

func TestSimulationConfigRootsHonoursAnExplicitConfigsDir(t *testing.T) {
	// NIAC_CONFIGS_DIR names the operator's configs directory. Keeping the
	// three built-in locations behind it let a daemon under isolated env
	// dirs resolve a config out of the invoking user's $HOME (P1-16). The
	// library root and the shipped templates are governed separately and
	// stay in the search path.
	libraryRoot := t.TempDir()
	custom := t.TempDir()
	t.Setenv("NIAC_LIBRARY_ROOT", libraryRoot)
	t.Setenv("NIAC_CONFIGS_DIR", custom)

	roots := simulationConfigRoots()

	if len(roots) == 0 || roots[0] != custom {
		t.Fatalf("roots = %v, want %q first", roots, custom)
	}
	home, homeErr := os.UserHomeDir()
	if homeErr != nil {
		t.Fatal(homeErr)
	}
	for _, unwanted := range []string{
		"configs",
		"/var/lib/niac/configs",
		filepath.Join(home, ".niac", "configs"),
	} {
		if slices.Contains(roots, unwanted) {
			t.Errorf("roots still contains the built-in location %q: %v", unwanted, roots)
		}
	}
	if !slices.Contains(roots, filepath.Join(libraryRoot, "networks")) {
		t.Errorf("roots dropped the library networks dir: %v", roots)
	}
}

func TestSimulationConfigRootsFallsBackToTheBuiltInLocations(t *testing.T) {
	t.Setenv("NIAC_CONFIGS_DIR", "")

	roots := simulationConfigRoots()

	if !slices.Contains(roots, "configs") || !slices.Contains(roots, "/var/lib/niac/configs") {
		t.Fatalf("roots = %v, want the built-in locations when the override is unset", roots)
	}
}
