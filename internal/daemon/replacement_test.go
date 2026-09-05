package daemon

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

func TestFailedSimulationReplacementPreservesActiveRun(t *testing.T) {
	t.Setenv(e2eDryRunEnv, "true")
	t.Setenv("NIAC_CONFIGS_DIR", t.TempDir())
	daemon := replacementTestDaemon(t)
	startReplacementTestSimulation(t, daemon, "active-router")

	active := daemon.simulation
	activeStatus := daemon.GetStatus()
	activeYAML, err := os.ReadFile(active.ConfigPath)
	if err != nil {
		t.Fatalf("ReadFile(active config) error = %v", err)
	}
	cleanupCalls := 0
	daemon.startSimulation = func(
		string,
		*config.Config,
		*fabric.Topology,
		bool,
		int,
	) (simulationResources, error) {
		return simulationResources{cancel: func() { cleanupCalls++ }}, errors.New("injected startup failure")
	}

	err = daemon.StartSimulation(replacementRequest("replacement-router"))
	if err == nil || !strings.Contains(err.Error(), "injected startup failure") {
		t.Fatalf("StartSimulation() error = %v, want injected startup failure", err)
	}
	if daemon.simulation != active {
		t.Fatal("failed replacement changed the active simulation")
	}
	if got := daemon.GetStatus(); got.Interface != activeStatus.Interface ||
		got.ConfigPath != activeStatus.ConfigPath ||
		got.ConfigName != activeStatus.ConfigName ||
		got.DeviceCount != activeStatus.DeviceCount ||
		!reflect.DeepEqual(got.Fabric, activeStatus.Fabric) {
		t.Fatalf("status after failed replacement = %#v, want %#v", got, activeStatus)
	}
	persisted, readErr := os.ReadFile(active.ConfigPath)
	if readErr != nil {
		t.Fatalf("ReadFile(active config after failure) error = %v", readErr)
	}
	if string(persisted) != string(activeYAML) {
		t.Fatalf("active inline config changed after failed replacement:\n%s", persisted)
	}
	if cleanupCalls != 1 {
		t.Fatalf("partial replacement cleanup calls = %d, want 1", cleanupCalls)
	}
}

func TestSuccessfulSimulationReplacementPublishesThenStopsOldRun(t *testing.T) {
	t.Setenv(e2eDryRunEnv, "true")
	t.Setenv("NIAC_CONFIGS_DIR", t.TempDir())
	daemon := replacementTestDaemon(t)
	startReplacementTestSimulation(t, daemon, "active-router")

	active := daemon.simulation
	oldStopped := false
	active.cancel = func() {
		oldStopped = true
		if daemon.simulation == active {
			t.Error("old simulation stopped before replacement was published")
		}
	}

	if err := daemon.StartSimulation(replacementRequest("replacement-router")); err != nil {
		t.Fatalf("StartSimulation(replacement) error = %v", err)
	}

	if !oldStopped {
		t.Fatal("successful replacement did not stop the old simulation")
	}
	if daemon.simulation == active {
		t.Fatal("successful replacement did not publish a new simulation")
	}
	status := daemon.GetStatus()
	if !status.Running || status.Interface != "replacement0" || status.DeviceCount != 1 ||
		status.Fabric == nil ||
		len(status.Fabric.Topology.Interfaces) != 1 ||
		status.Fabric.Topology.Interfaces[0].Device != "replacement-router" {
		t.Fatalf("replacement status = %#v", status)
	}
	persisted, err := os.ReadFile(status.ConfigPath)
	if err != nil {
		t.Fatalf("ReadFile(replacement config) error = %v", err)
	}
	if !strings.Contains(string(persisted), "replacement-router") {
		t.Fatalf("replacement config = %q", persisted)
	}
}

func replacementTestDaemon(t *testing.T) *Daemon {
	t.Helper()
	daemon, err := NewDaemon(Config{
		StoragePath: "disabled",
		AttachmentPolicies: []fabric.PhysicalAttachmentPolicy{{
			Interface: "replacement0", Mode: fabric.ModeAccess, AccessVLAN: 200,
		}},
	})
	if err != nil {
		t.Fatalf("NewDaemon() error = %v", err)
	}
	daemon.apiServer = api.NewServer(api.ServerConfig{})
	t.Cleanup(func() {
		if daemon.simulation != nil {
			_ = daemon.StopSimulation("")
		}
	})
	return daemon
}

func startReplacementTestSimulation(t *testing.T, daemon *Daemon, name string) {
	t.Helper()
	if err := daemon.StartSimulation(replacementRequest(name)); err != nil {
		t.Fatalf("StartSimulation(%s) error = %v", name, err)
	}
}

func replacementRequest(name string) api.SimulationRequest {
	return api.SimulationRequest{
		Interface:      "replacement0",
		Attachment:     "tester",
		AttachmentMode: fabric.ModeAccess,
		AccessVLAN:     200,
		ConfigData: "networks:\n" +
			"  - name: lab-access\n" +
			"    subnet: 192.0.2.0/24\n" +
			"attachments:\n" +
			"  - name: tester\n" +
			"    connect: lab-access\n" +
			"devices:\n" +
			"  - name: " + name + "\n" +
			"    type: router\n" +
			"    mac: \"02:00:00:00:00:01\"\n" +
			"    interfaces:\n" +
			"      - name: outside\n" +
			"        network: lab-access\n" +
			"        address: 192.0.2.10/24\n",
	}
}
