package daemon

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestRuntimeStateRecoversAfterAbruptProcessExit(t *testing.T) {
	if os.Getenv("NIAC_TEST_CRASH_RECOVERY_CHILD") == "1" {
		runRuntimeCrashChild(t)
		return
	}
	t.Setenv(e2eDryRunEnv, "true")
	directory := t.TempDir()
	t.Setenv("NIAC_CONFIGS_DIR", filepath.Join(directory, "configs"))
	path := filepath.Join(directory, activeSimulationFileName)
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	child := exec.CommandContext(ctx, executable, "-test.run=^TestRuntimeStateRecoversAfterAbruptProcessExit$")
	child.Dir = directory
	child.Env = append(os.Environ(), "NIAC_TEST_CRASH_RECOVERY_CHILD=1")
	if output, runErr := child.CombinedOutput(); runErr != nil {
		t.Fatalf("crash subprocess: %v\n%s", runErr, output)
	}
	d := recoveryTestDaemon(t, path)
	d.recoverActiveSimulation()
	faults := runtimeTestState(t, runtimeTestStack(t, d)).DeviceFaults
	if len(faults) != 1 || faults[0].Type != devicestate.FaultLatency || faults[0].Value != 250 {
		t.Fatalf("crash recovery lost persisted faults: %+v", faults)
	}
	if err = d.StopSimulation(""); err != nil {
		t.Fatal(err)
	}
	if err = d.StartSimulation(
		api.SimulationRequest{Interface: "recovery0", ConfigData: runtimeStateConfig},
	); err != nil {
		t.Fatal(err)
	}
	if state := runtimeTestState(t, runtimeTestStack(t, d)); len(state.DeviceFaults)+len(state.InterfaceFaults) != 0 {
		t.Fatal("explicit Stop/Start restored faults from the previous run")
	}
}

func runRuntimeCrashChild(t *testing.T) {
	t.Helper()
	directory, directoryErr := os.Getwd()
	if directoryErr != nil {
		t.Fatal(directoryErr)
	}
	path := filepath.Join(directory, activeSimulationFileName)
	d := recoveryTestDaemon(t, path)
	if err := d.StartSimulation(
		api.SimulationRequest{Interface: "recovery0", ConfigData: runtimeStateConfig},
	); err != nil {
		t.Fatal(err)
	}
	if err := runtimeTestStack(t, d).SetDeviceFault("runtime-switch", devicestate.FaultLatency, 250); err != nil {
		t.Fatal(err)
	}
	generation := d.simulation.runtimeGeneration
	ticker := time.NewTicker(10 * time.Millisecond)
	deadline := time.NewTimer(10 * time.Second)
	for {
		select {
		case <-deadline.C:
			ticker.Stop()
			t.Fatal("periodic writer did not persist the armed fault")
		case <-ticker.C:
			states, err := d.loadRuntimeState(defaultSessionID, generation)
			faults := states["runtime-switch"].DeviceFaults
			if err == nil && len(faults) == 1 && faults[0].Value == 250 {
				// Bypass shutdown, deferred cleanup and the final flush.
				ticker.Stop()
				deadline.Stop()
				os.Exit(0)
			}
		}
	}
}
