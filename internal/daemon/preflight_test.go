package daemon

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

func TestPreflightSimulationDoesNotPersistInlineConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NIAC_CONFIGS_DIR", dir)
	d := &Daemon{}
	req := routedRequest(2)

	report, err := d.PreflightSimulation(req)
	if err != nil {
		t.Fatalf("PreflightSimulation() error = %v", err)
	}
	if !report.Safe {
		t.Fatalf("safe = false, diagnostics = %#v", report.Diagnostics)
	}
	if _, err = os.Stat(filepath.Join(dir, inlineConfigName)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("inline config stat error = %v, want not exist", err)
	}
}

func TestStartSimulationRecompilesUnsafeRoutedRequest(t *testing.T) {
	t.Setenv(e2eDryRunEnv, "true")
	t.Setenv("NIAC_CONFIGS_DIR", t.TempDir())
	d := &Daemon{}
	req := routedRequest(0)

	err := d.StartSimulation(req)

	if !errors.Is(err, ErrUnsafeTopology) {
		t.Fatalf("StartSimulation() error = %v, want ErrUnsafeTopology", err)
	}
}

func TestUnsafeReplacementLeavesRunningSimulationIntact(t *testing.T) {
	t.Setenv(e2eDryRunEnv, "true")
	t.Setenv("NIAC_CONFIGS_DIR", t.TempDir())
	running := &Simulation{Interface: "eth0"}
	d := &Daemon{simulation: running}

	err := d.StartSimulation(routedRequest(0))

	if !errors.Is(err, ErrUnsafeTopology) {
		t.Fatalf("StartSimulation() error = %v, want ErrUnsafeTopology", err)
	}
	if d.simulation != running {
		t.Fatal("unsafe replacement stopped the running simulation")
	}
}

func TestPreflightSimulationPreservesFlatScenarioWorkflow(t *testing.T) {
	d := &Daemon{}
	req := api.SimulationRequest{
		Interface: "eth0", Attachment: "tester", AttachmentMode: fabric.ModeAccess,
		AccessVLAN: 200, ConfigData: `
devices:
  - name: legacy-router
    type: router
    mac: 02:00:00:00:00:01
    ips:
      - 10.10.200.1
`,
	}

	report, err := d.PreflightSimulation(req)
	if err != nil {
		t.Fatalf("PreflightSimulation() error = %v", err)
	}
	if !report.Safe {
		t.Fatalf("safe = false, diagnostics = %#v", report.Diagnostics)
	}
}

func routedRequest(accessVLAN uint16) api.SimulationRequest {
	return api.SimulationRequest{
		Interface:      "eth0",
		Attachment:     "tester",
		AttachmentMode: fabric.ModeAccess,
		AccessVLAN:     accessVLAN,
		ConfigData: `
networks:
  - name: lab-access
    subnet: 10.10.200.0/24
attachments:
  - name: tester
    connect: lab-access
devices:
  - name: edge
    type: router
    mac: 02:00:00:00:00:01
    interfaces:
      - name: outside
        network: lab-access
        address: 10.10.200.1/24
`,
	}
}
