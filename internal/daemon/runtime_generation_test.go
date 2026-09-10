package daemon

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestReplacementRecoveryDoesNotRestorePreviousFaults(t *testing.T) {
	t.Setenv(e2eDryRunEnv, "true")
	t.Setenv("NIAC_CONFIGS_DIR", t.TempDir())
	path := filepath.Join(t.TempDir(), activeSimulationFileName)
	d := recoveryTestDaemon(t, path)
	request := api.SimulationRequest{Interface: "recovery0", ConfigData: runtimeStateConfig}
	if err := d.StartSimulation(request); err != nil {
		t.Fatal(err)
	}
	stopRuntimeStateWriter(d.simulation)
	stack := runtimeTestStack(t, d)
	if err := stack.SetDeviceFault("runtime-switch", devicestate.FaultLatency, 777); err != nil {
		t.Fatal(err)
	}
	if err := d.writeRuntimeState(defaultSessionID, d.simulation.runtimeGeneration, stack); err != nil {
		t.Fatal(err)
	}
	if err := d.StartSimulation(request); err != nil {
		t.Fatal(err)
	}
	stopRuntimeStateWriter(d.simulation)
	restarted := recoveryTestDaemon(t, path)
	restarted.recoverActiveSimulation()
	if status := restarted.GetStatus(); status.Recovery == nil || status.Recovery.State != recoveryStateRecovered {
		t.Fatalf("recovery failed: %+v", status.Recovery)
	}
	states := runtimeTestStack(t, restarted).ExportDeviceStates()
	if faults := states["runtime-switch"].DeviceFaults; len(faults) != 0 {
		t.Fatalf("replacement restored previous faults: %+v", faults)
	}
}

func TestUncommittedRuntimeGenerationIsIgnored(t *testing.T) {
	t.Setenv(e2eDryRunEnv, "true")
	t.Setenv("NIAC_CONFIGS_DIR", t.TempDir())
	path := filepath.Join(t.TempDir(), activeSimulationFileName)
	d := recoveryTestDaemon(t, path)
	request := api.SimulationRequest{Interface: "recovery0", ConfigData: runtimeStateConfig}
	if err := d.StartSimulation(request); err != nil {
		t.Fatal(err)
	}
	stopRuntimeStateWriter(d.simulation)
	stack := runtimeTestStack(t, d)
	if err := stack.SetDeviceFault("runtime-switch", devicestate.FaultLatency, 777); err != nil {
		t.Fatal(err)
	}
	if err := d.writeRuntimeState(defaultSessionID, d.simulation.runtimeGeneration, stack); err != nil {
		t.Fatal(err)
	}
	if err := stack.SetDeviceFault("runtime-switch", devicestate.FaultLatency, 888); err != nil {
		t.Fatal(err)
	}
	if err := d.writeRuntimeState(defaultSessionID, "00000000000000000000000000000001", stack); err != nil {
		t.Fatal(err)
	}
	restarted := recoveryTestDaemon(t, path)
	restarted.recoverActiveSimulation()
	states := runtimeTestStack(t, restarted).ExportDeviceStates()
	faults := states["runtime-switch"].DeviceFaults
	if len(faults) != 1 || faults[0].Value != 777 {
		t.Fatalf("recovery used uncommitted snapshot: %+v", faults)
	}
}

func TestRecoveryRejectsEmptyCommittedRuntimeState(t *testing.T) {
	t.Setenv(e2eDryRunEnv, "true")
	t.Setenv("NIAC_CONFIGS_DIR", t.TempDir())
	path := filepath.Join(t.TempDir(), activeSimulationFileName)
	d := recoveryTestDaemon(t, path)
	request := api.SimulationRequest{Interface: "recovery0", ConfigData: runtimeStateConfig}
	if err := d.StartSimulation(request); err != nil {
		t.Fatal(err)
	}
	stopRuntimeStateWriter(d.simulation)
	generation := d.simulation.runtimeGeneration
	record := runtimeStateRecord{
		SchemaVersion: runtimeStateSchemaVersion,
		SessionID:     defaultSessionID,
		Generation:    generation,
	}
	data, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if err = writeStateFile(d.runtimeStateFile(defaultSessionID, generation), data); err != nil {
		t.Fatal(err)
	}
	restarted := recoveryTestDaemon(t, path)
	restarted.recoverActiveSimulation()
	status := restarted.GetStatus()
	if status.Running || status.Recovery == nil || status.Recovery.State != recoveryStateFailed {
		t.Fatalf("empty committed state recovered healthy: %+v", status)
	}
}

func TestManifestFailurePreservesActiveInlineConfig(t *testing.T) {
	t.Setenv(e2eDryRunEnv, "true")
	t.Setenv("NIAC_CONFIGS_DIR", t.TempDir())
	d := recoveryTestDaemon(t, filepath.Join(t.TempDir(), activeSimulationFileName))
	request := api.SimulationRequest{Interface: "recovery0", ConfigData: runtimeStateConfig}
	if err := d.StartSimulation(request); err != nil {
		t.Fatal(err)
	}
	active := d.simulation
	stopRuntimeStateWriter(active)
	before, err := os.ReadFile(active.ConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	d.cfg.RecoveryPath = t.TempDir()
	request.ConfigData = strings.ReplaceAll(runtimeStateConfig, "runtime-switch", "replacement-switch")
	if err = d.StartSimulation(request); err == nil {
		t.Fatal("replacement succeeded with an invalid manifest path")
	}
	after, err := os.ReadFile(active.ConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("failed replacement overwrote the active recovery configuration")
	}
	if d.simulation != active {
		t.Fatal("failed replacement changed active simulation")
	}
}

func TestInitialSnapshotFailurePreservesManifest(t *testing.T) {
	t.Setenv(e2eDryRunEnv, "true")
	t.Setenv("NIAC_CONFIGS_DIR", t.TempDir())
	path := filepath.Join(t.TempDir(), activeSimulationFileName)
	d := recoveryTestDaemon(t, path)
	request := api.SimulationRequest{Interface: "recovery0", ConfigData: runtimeStateConfig}
	if err := d.StartSimulation(request); err != nil {
		t.Fatal(err)
	}
	active := d.simulation
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	const generation = "00000000000000000000000000000001"
	if err = os.Mkdir(d.runtimeStateFile(defaultSessionID, generation), 0o700); err != nil {
		t.Fatal(err)
	}
	if err = d.startGeneration(request, generation, false); err == nil {
		t.Fatal("replacement committed without an initial snapshot")
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) || d.simulation != active {
		t.Fatal("failed initial snapshot changed the active recovery intent")
	}
}
