package daemon

import (
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
)

func TestStartSimulationE2EDryRunDoesNotOpenInterface(t *testing.T) {
	t.Setenv(e2eDryRunEnv, "1")

	d, err := NewDaemon(Config{StoragePath: "disabled"})
	if err != nil {
		t.Fatalf("NewDaemon() error = %v", err)
	}
	d.apiServer = api.NewServer(api.ServerConfig{})

	err = d.StartSimulation(api.SimulationRequest{
		Interface: "missing-e2e-interface",
		ConfigData: `devices:
  - name: dry-run-router
    type: router
    mac: "02:00:00:00:00:01"
    ips:
      - "192.0.2.10"
`,
	}, fullSimulationEntitlements())
	if err != nil {
		t.Fatalf("StartSimulation() error = %v", err)
	}

	status := d.GetStatus()
	if !status.Running {
		t.Fatal("status.Running = false, want true")
	}
	if status.Interface != "missing-e2e-interface" {
		t.Fatalf("status.Interface = %q", status.Interface)
	}
	if status.DeviceCount != 1 {
		t.Fatalf("status.DeviceCount = %d, want 1", status.DeviceCount)
	}

	if stopErr := d.StopSimulation(); stopErr != nil {
		t.Fatalf("StopSimulation() error = %v", stopErr)
	}
}
