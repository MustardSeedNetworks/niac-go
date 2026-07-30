package daemon

import (
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
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

func TestScenarioPacksStartInRuntime(t *testing.T) {
	t.Setenv(e2eDryRunEnv, "1")

	for _, pack := range scenario.Packs() {
		t.Run(pack.ID, func(t *testing.T) {
			const interfaceName = "missing-e2e-interface"
			d, err := NewDaemon(Config{
				StoragePath: "disabled",
				AttachmentPolicies: []fabric.PhysicalAttachmentPolicy{{
					Interface: interfaceName, Mode: fabric.ModeAccess, AccessVLAN: 200,
				}},
			})
			if err != nil {
				t.Fatalf("NewDaemon(): %v", err)
			}
			d.apiServer = api.NewServer(api.ServerConfig{})

			result, err := scenario.Generate(pack.Request)
			if err != nil {
				t.Fatalf("Generate(): %v", err)
			}
			err = d.StartSimulation(api.SimulationRequest{
				Interface:      interfaceName,
				Attachment:     pack.Request.AttachmentName,
				AttachmentMode: fabric.ModeAccess,
				AccessVLAN:     200,
				ConfigData:     string(result.YAML),
			}, fullSimulationEntitlements())
			if err != nil {
				t.Fatalf("StartSimulation(): %v", err)
			}

			status := d.GetStatus()
			if !status.Running || status.DeviceCount != result.Manifest.DeviceCount {
				t.Fatalf("status = %#v, want running with %d devices", status, result.Manifest.DeviceCount)
			}
			if err = d.StopSimulation(); err != nil {
				t.Fatalf("StopSimulation(): %v", err)
			}
		})
	}
}
