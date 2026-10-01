package daemon

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
)

// TestLoadSimulationConfigResolvesABuiltinScenarioFromTheLibrary is the installed-host
// case. `niac scenario list` advertises ten built-in scenarios, and every one of
// them answered "template not found" through the daemon: the loader walked
// on-disk scenario directories (cmd/niac/templates, examples,
// /usr/share/niac/templates) and the deb and rpm ship none of them —
// `dpkg -L niac | grep templ` is empty. What does ship is the library, which
// first run seeds with exactly those scenarios.
func TestLoadSimulationConfigResolvesABuiltinScenarioFromTheLibrary(t *testing.T) {
	root := t.TempDir()
	networks := filepath.Join(root, "networks")
	if err := os.Mkdir(networks, 0o750); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NIAC_LIBRARY_ROOT", root)
	t.Setenv("NIAC_CONFIGS_DIR", filepath.Join(root, "configs"))
	t.Setenv("NIAC_TEMPLATES_DIR", filepath.Join(root, "no-templates-here"))

	starter := writeManagedPathTestConfig(t, networks, "home-network.yaml")
	want, err := filepath.EvalSymlinks(starter)
	if err != nil {
		t.Fatal(err)
	}

	_, resolved, loadErr := loadSimulationConfig(
		api.SimulationRequest{ScenarioName: "home-network"}, false)
	if loadErr != nil {
		t.Fatalf("loadSimulationConfig() error = %v", loadErr)
	}
	if resolved != want {
		t.Fatalf("resolved path = %q, want %q", resolved, want)
	}
}

// TestLoadSimulationConfigRejectsAnUnknownScenario keeps the refusal: a name in
// neither a scenario directory nor the library is still not found, rather than
// resolving to whatever the working directory happens to hold.
func TestLoadSimulationConfigRejectsAnUnknownScenario(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "networks"), 0o750); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NIAC_LIBRARY_ROOT", root)
	t.Setenv("NIAC_CONFIGS_DIR", filepath.Join(root, "configs"))
	t.Setenv("NIAC_TEMPLATES_DIR", filepath.Join(root, "no-templates-here"))

	cwd := t.TempDir()
	writeManagedPathTestConfig(t, cwd, "sneaky.yaml")
	t.Chdir(cwd)

	if _, _, err := loadSimulationConfig(
		api.SimulationRequest{ScenarioName: "sneaky"}, false); err == nil {
		t.Fatal("loadSimulationConfig() resolved a scenario outside every root")
	}
}
