package daemon

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
	"github.com/MustardSeedNetworks/niac-go/internal/library"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

const exampleTestInterface = "missing-e2e-interface"

// exampleTestHost isolates every directory the daemon reads or writes for a
// start by name, and opens the library so the starter walks a captured example
// replays are in place.
func exampleTestHost(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv("NIAC_LIBRARY_ROOT", filepath.Join(root, "library"))
	t.Setenv("NIAC_CONFIGS_DIR", filepath.Join(root, "configs"))
	t.Setenv("NIAC_TEMPLATES_DIR", filepath.Join(root, "no-templates-here"))
	t.Setenv(e2eDryRunEnv, "1")
	t.Chdir(t.TempDir())
	if _, err := library.Open(filepath.Join(root, "library")); err != nil {
		t.Fatalf("open library: %v", err)
	}
	return filepath.Join(root, "library")
}

func newExampleTestDaemon(t *testing.T) *Daemon {
	t.Helper()
	d, err := NewDaemon(Config{
		StoragePath: "disabled",
		AttachmentPolicies: []fabric.PhysicalAttachmentPolicy{{
			Interface: exampleTestInterface, Mode: fabric.ModeAccess, AccessVLAN: 200,
		}},
	})
	if err != nil {
		t.Fatalf("NewDaemon(): %v", err)
	}
	d.apiServer = api.NewServer(api.ServerConfig{})
	return d
}

func startExample(t *testing.T, d *Daemon, req api.SimulationRequest) api.SimulationStatus {
	t.Helper()
	req.Interface = exampleTestInterface
	if err := d.StartSimulation(req); err != nil {
		t.Fatalf("StartSimulation(%+v): %v", req, err)
	}
	t.Cleanup(func() { _ = d.StopSimulation("") })
	return d.GetStatus()
}

func TestStartAcceptsAnExampleByName(t *testing.T) {
	spellings := map[string]api.SimulationRequest{
		"scenario name":        {ScenarioName: "basic-network"},
		"bare config path":     {ConfigPath: "basic-network.yaml"},
		"extensionless path":   {ConfigPath: "basic-network"},
		"scenario with ext":    {ScenarioName: "basic-network.yaml"},
		"yml bare config path": {ConfigPath: "basic-network.yml"},
	}
	for label, req := range spellings {
		t.Run(label, func(t *testing.T) {
			exampleTestHost(t)
			d := newExampleTestDaemon(t)

			status := startExample(t, d, req)
			if !status.Running || status.DeviceCount == 0 {
				t.Fatalf("status = %+v, want running with devices", status)
			}
			if !strings.Contains(filepath.Base(status.ConfigPath), ".inline.yaml") {
				t.Errorf("config path = %q, want the session's staged copy", status.ConfigPath)
			}
		})
	}
}

// A captured example names walks relative to the library, and the session
// records the staged copy for recovery, so the recorded file has to resolve
// them on its own: a second load of the recorded path is what recovery does.
func TestStartedCapturedExampleReloadsFromItsRecordedPath(t *testing.T) {
	exampleTestHost(t)
	d := newExampleTestDaemon(t)

	status := startExample(t, d, api.SimulationRequest{ScenarioName: "captured-switch"})
	if status.DeviceCount == 0 {
		t.Fatalf("no devices: %+v", status)
	}
	cfg, _, err := config.LoadYAMLManaged(status.ConfigPath, simulationConfigRoots())
	if err != nil {
		t.Fatalf("recorded path %s does not reload: %v", status.ConfigPath, err)
	}
	walkBacked := false
	for _, device := range cfg.Devices {
		walkBacked = walkBacked || device.SNMPConfig.WalkFile != ""
	}
	if !walkBacked {
		t.Error("the reloaded session lost its captured walk")
	}
}

func TestStartAcceptsAScenarioPackByName(t *testing.T) {
	exampleTestHost(t)
	d := newExampleTestDaemon(t)

	var campus scenario.Pack
	for _, pack := range scenario.Packs() {
		if pack.ID == "campus" {
			campus = pack
		}
	}
	req := api.SimulationRequest{
		ScenarioName:   "campus",
		Interface:      exampleTestInterface,
		Attachment:     campus.Request.AttachmentName,
		AttachmentMode: fabric.ModeAccess,
		AccessVLAN:     200,
	}
	report, err := d.PreflightSimulation(req)
	if err != nil {
		t.Fatalf("PreflightSimulation: %v", err)
	}
	if !report.Safe {
		t.Fatalf("preflight diagnostics = %#v", report.Diagnostics)
	}
	status := startExample(t, d, req)
	if status.DeviceCount != campus.Manifest.DeviceCount {
		t.Errorf("devices = %d, want the manifest's %d", status.DeviceCount, campus.Manifest.DeviceCount)
	}
}

// A file in the library answers a bare name before the binary's example does,
// the order every other bare name already resolves in.
func TestLibraryFileShadowsAnExampleOnStart(t *testing.T) {
	libraryRoot := exampleTestHost(t)
	own := filepath.Join(libraryRoot, "networks", "basic-network.yaml")
	if err := os.WriteFile(own, []byte(managedPathTestYAML), 0o600); err != nil {
		t.Fatal(err)
	}
	d := newExampleTestDaemon(t)

	status := startExample(t, d, api.SimulationRequest{ScenarioName: "basic-network"})
	want, err := filepath.EvalSymlinks(own)
	if err != nil {
		t.Fatal(err)
	}
	if status.ConfigPath != want {
		t.Errorf("config path = %q, want the library file %q", status.ConfigPath, want)
	}
}

func TestResolveExampleRequestLeavesEverythingElseAlone(t *testing.T) {
	exampleTestHost(t)

	tests := map[string]api.SimulationRequest{
		"unknown name":       {ScenarioName: "no-such-scenario"},
		"removed built-in":   {ScenarioName: "small-office"},
		"path, not a name":   {ConfigPath: "sub/basic-network.yaml"},
		"traversal":          {ScenarioName: "../basic-network"},
		"inline data wins":   {ConfigData: "devices: []", ConfigPath: "basic-network.yaml"},
		"hidden file":        {ScenarioName: ".basic-network"},
		"empty":              {},
		"example by pathish": {ConfigPath: "/etc/basic-network.yaml"},
	}
	for label, req := range tests {
		t.Run(label, func(t *testing.T) {
			if got := resolveExampleRequest(req); got != req {
				t.Errorf("request was rewritten: %+v -> %+v", req, got)
			}
		})
	}
}

func TestStartRejectsAnUnknownScenarioName(t *testing.T) {
	exampleTestHost(t)
	d := newExampleTestDaemon(t)

	err := d.StartSimulation(api.SimulationRequest{
		Interface: exampleTestInterface, ScenarioName: "small-office",
	})
	if !errors.Is(err, ErrScenarioNotFound) {
		t.Fatalf("want ErrScenarioNotFound for a removed built-in, got %v", err)
	}
}

// A content pack installs its networks into the library and names its captures
// relative to the library (`include_path: ../walks`). Sent by name, the
// daemon loads the file in place and the walk resolves; sent as inline data,
// the same text resolves against the configs directory and the walk is "not
// found". The Simulation page used to send pack networks inline, so no pack
// scenario could start from it (#3154).
func TestStartResolvesAPackNetworkWalkOnlyByName(t *testing.T) {
	libraryRoot := exampleTestHost(t)
	starter, err := os.ReadDir(filepath.Join(libraryRoot, "walks"))
	if err != nil || len(starter) == 0 {
		t.Fatalf("starter walks: %v (%d entries)", err, len(starter))
	}
	pack := "include_path: ../walks\ndevices:\n  - name: pack-switch\n    type: switch\n" +
		"    mac: \"02:00:00:00:00:09\"\n    ips: [\"192.0.2.9\"]\n    snmp_agent:\n" +
		"      community: public\n      walk_file: " + starter[0].Name() + "\n"

	for _, extension := range []string{".yaml", ".yml"} {
		t.Run("by name "+extension, func(t *testing.T) {
			networks := filepath.Join(libraryRoot, "networks")
			if mkErr := os.MkdirAll(networks, 0o750); mkErr != nil {
				t.Fatal(mkErr)
			}
			file := filepath.Join(networks, "pack-net"+extension)
			if writeErr := os.WriteFile(file, []byte(pack), 0o600); writeErr != nil {
				t.Fatal(writeErr)
			}
			t.Cleanup(func() { _ = os.Remove(file) })
			d := newExampleTestDaemon(t)

			status := startExample(t, d, api.SimulationRequest{ScenarioName: "pack-net"})
			if !status.Running || status.DeviceCount != 1 {
				t.Fatalf("status = %+v, want the pack network running", status)
			}
		})
	}

	t.Run("inline loses the library context", func(t *testing.T) {
		d := newExampleTestDaemon(t)
		startErr := d.StartSimulation(api.SimulationRequest{Interface: exampleTestInterface, ConfigData: pack})
		if !errors.Is(startErr, config.ErrWalkFileNotFound) {
			t.Fatalf("StartSimulation(inline) = %v, want ErrWalkFileNotFound", startErr)
		}
	})
}
