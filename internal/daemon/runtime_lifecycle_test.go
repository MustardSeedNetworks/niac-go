package daemon

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"testing/synctest"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestRuntimeStopJoinsInflightWriter(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		release := make(chan struct{})
		done := make(chan struct{})
		sim := &Simulation{runtimeCancel: cancel, runtimeDone: done}
		go func() {
			<-ctx.Done()
			<-release
			close(done)
		}()
		settled := make(chan struct{})
		go func() {
			(&Daemon{}).settleRuntimeState(sim, true)
			close(settled)
		}()
		synctest.Wait()
		select {
		case <-settled:
			t.Error("stop completed while a writer still owned the record")
		default:
		}
		close(release)
		<-settled
	})
}

func TestReplacementCancelsRuntimeWriter(t *testing.T) {
	t.Setenv(e2eDryRunEnv, "true")
	t.Setenv("NIAC_CONFIGS_DIR", t.TempDir())
	d := replacementTestDaemon(t)
	startReplacementTestSimulation(t, d, "active-router")
	canceled := false
	d.simulation.runtimeCancel = func() { canceled = true }
	if err := d.StartSimulation(replacementRequest("replacement-router")); err != nil {
		t.Fatal(err)
	}
	if !canceled {
		t.Fatal("replacement left the old runtime writer alive")
	}
}

func TestShutdownPersistsFinalProducerMutation(t *testing.T) {
	t.Setenv(e2eDryRunEnv, "true")
	t.Setenv("NIAC_CONFIGS_DIR", t.TempDir())
	d := recoveryTestDaemon(t, filepath.Join(t.TempDir(), activeSimulationFileName))
	if err := d.StartSimulation(
		api.SimulationRequest{Interface: "recovery0", ConfigData: runtimeStateConfig},
	); err != nil {
		t.Fatal(err)
	}
	stack := runtimeTestStack(t, d)
	generation := d.simulation.runtimeGeneration
	d.simulation.cancel = func() {
		if err := stack.SetDeviceFault("runtime-switch", devicestate.FaultLatency, 777); err != nil {
			t.Error(err)
		}
	}
	d.mu.Lock()
	err := d.stopSimulationLocked(false)
	d.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	saved, err := d.loadRuntimeState(defaultSessionID, generation)
	if err != nil {
		t.Fatal(err)
	}
	faults := saved["runtime-switch"].DeviceFaults
	if len(faults) != 1 || faults[0].Value != 777 {
		t.Fatalf("shutdown lost final producer mutation: %+v", faults)
	}
}

func TestRuntimeStateRejectsWrongSession(t *testing.T) {
	const generation = "00000000000000000000000000000001"
	d := recoveryTestDaemon(t, filepath.Join(t.TempDir(), activeSimulationFileName))
	data, err := json.Marshal(
		runtimeStateRecord{
			SchemaVersion: runtimeStateSchemaVersion,
			SessionID:     "another-session",
			Generation:    generation,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.MkdirAll(d.runtimeStateDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(d.runtimeStateFile(defaultSessionID, generation), data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err = d.loadRuntimeState(defaultSessionID, generation); err == nil {
		t.Fatal("accepted runtime state belonging to another session")
	}
}
