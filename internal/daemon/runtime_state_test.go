package daemon

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

// Orderly restart retains armed faults and their event history.
func TestRuntimeStateSurvivesDaemonRestart(t *testing.T) {
	t.Setenv(e2eDryRunEnv, "true")
	t.Setenv("NIAC_CONFIGS_DIR", t.TempDir())
	stateDir := t.TempDir()
	recoveryPath := filepath.Join(stateDir, activeSimulationFileName)

	first := recoveryTestDaemon(t, recoveryPath)
	request := api.SimulationRequest{Interface: "recovery0", ConfigData: runtimeStateConfig}
	if startErr := first.StartSimulation(request); startErr != nil {
		t.Fatalf("StartSimulation() error = %v", startErr)
	}
	stack := runtimeTestStack(t, first)
	generation := first.simulation.runtimeGeneration
	if err := stack.SetInterfaceFault(
		"runtime-switch",
		"GigabitEthernet1/0/1",
		devicestate.FaultLinkDown,
		1,
	); err != nil {
		t.Fatalf("SetInterfaceFault() error = %v", err)
	}
	if err := stack.SetDeviceFault("runtime-switch", devicestate.FaultLatency, 250); err != nil {
		t.Fatalf("SetDeviceFault() error = %v", err)
	}
	before := runtimeTestState(t, stack)

	// Orderly daemon shutdown preserves launch intent and flushes runtime state.
	first.mu.Lock()
	if stopErr := first.stopSimulationLocked(false); stopErr != nil {
		first.mu.Unlock()
		t.Fatalf("shutdown stop error = %v", stopErr)
	}
	first.mu.Unlock()

	second := recoveryTestDaemon(t, recoveryPath)
	second.recoverActiveSimulation()
	status := second.GetStatus()
	if status.Recovery == nil || status.Recovery.State != recoveryStateRecovered {
		t.Fatalf("recovery status = %#v", status.Recovery)
	}
	restored := runtimeTestStack(t, second)
	after := runtimeTestState(t, restored)

	assertSameFaults(t, before, after)
	assertHistoryPreserved(t, before.Events, after.Events)

	// The hazard the raw export exists to avoid: a link-down fault is
	// projected onto a snapshot, so exporting a snapshot as running state
	// would restore an interface latched operationally down with no fault
	// left to clear. Clearing the restored fault must return the carrier.
	if err := restored.SetInterfaceFault(
		"runtime-switch", "GigabitEthernet1/0/1", devicestate.FaultLinkDown, 0,
	); err != nil {
		t.Fatalf("clear restored fault error = %v", err)
	}
	assertCarrierRecovered(t, runtimeTestState(t, restored))

	if stopErr := second.StopSimulation(""); stopErr != nil {
		t.Fatalf("StopSimulation() error = %v", stopErr)
	}
	if _, statErr := os.Stat(runtimeStatePath(stateDir, defaultSessionID, generation)); !os.IsNotExist(statErr) {
		t.Fatalf("explicit stop left runtime state behind: %v", statErr)
	}
}

func runtimeTestStack(t *testing.T, daemon *Daemon) *protocols.Stack {
	t.Helper()
	daemon.mu.RLock()
	defer daemon.mu.RUnlock()
	if daemon.simulation == nil || daemon.simulation.stack == nil {
		t.Fatalf("daemon has no running simulation stack")
	}
	return daemon.simulation.stack
}

func runtimeTestState(t *testing.T, stack *protocols.Stack) devicestate.State {
	t.Helper()
	states := stack.ExportDeviceStates()
	state, ok := states["runtime-switch"]
	if !ok {
		t.Fatalf("device state export = %#v, want recovery-router", states)
	}
	return state
}

const runtimeStateConfig = `devices:
  - name: runtime-switch
    type: switch
    mac: "02:00:00:00:00:11"
    ips: ["192.0.2.11"]
    interfaces:
      - name: "GigabitEthernet1/0/1"
        type: ethernet
        speed: 1000
        admin_status: up
        oper_status: up
    snmp_agent:
      enabled: true
      community: "public"
      sysname: "runtime-switch"
`

// The periodic writer is what covers a hard crash: nothing calls the orderly
// shutdown path there, so a tick has to have written the record already.
func TestRuntimeStateWriterWritesOnlyWhatChanged(t *testing.T) {
	t.Setenv(e2eDryRunEnv, "true")
	t.Setenv("NIAC_CONFIGS_DIR", t.TempDir())
	stateDir := t.TempDir()
	daemon := recoveryTestDaemon(t, filepath.Join(stateDir, activeSimulationFileName))
	if err := daemon.StartSimulation(api.SimulationRequest{
		Interface: "recovery0", ConfigData: runtimeStateConfig,
	}); err != nil {
		t.Fatalf("StartSimulation() error = %v", err)
	}
	stack := runtimeTestStack(t, daemon)
	generation := daemon.simulation.runtimeGeneration

	writes := make(chan error, 8)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	defer func() { cancel(); <-done }()
	go func() {
		defer close(done)
		runRuntimeStateWriter(ctx, defaultSessionID, stack, func(sessionID string, target *protocols.Stack) error {
			err := daemon.writeRuntimeState(sessionID, generation, target)
			writes <- err
			return err
		}, time.Millisecond)
	}()

	select {
	case err := <-writes:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the writer never persisted the session")
	}
	// Nothing has changed since, so a settled writer stops writing.
	drainWrites(writes, 200*time.Millisecond)
	if quiet := drainWrites(writes, 100*time.Millisecond); quiet != 0 {
		t.Fatalf("writer wrote %d times with no state change", quiet)
	}
	if _, err := os.Stat(runtimeStatePath(stateDir, defaultSessionID, generation)); err != nil {
		t.Fatalf("runtime state was not written: %v", err)
	}

	if err := stack.SetDeviceFault("runtime-switch", devicestate.FaultLatency, 120); err != nil {
		t.Fatalf("SetDeviceFault() error = %v", err)
	}
	select {
	case err := <-writes:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("a state change did not reach the writer")
	}
	saved, err := daemon.loadRuntimeState(defaultSessionID, generation)
	if err != nil {
		t.Fatal(err)
	}
	faults := saved["runtime-switch"].DeviceFaults
	if len(faults) != 1 || faults[0].Type != devicestate.FaultLatency || faults[0].Value != 120 {
		t.Fatalf("writer did not save the changed fault: %+v", faults)
	}
}

// A corrupt record fails recovery closed. Starting healthy instead would
// report a successful recovery of a scenario that is not the one that crashed.
func TestRecoveryFailsOnCorruptRuntimeState(t *testing.T) {
	t.Setenv(e2eDryRunEnv, "true")
	t.Setenv("NIAC_CONFIGS_DIR", t.TempDir())
	stateDir := t.TempDir()
	recoveryPath := filepath.Join(stateDir, activeSimulationFileName)

	first := recoveryTestDaemon(t, recoveryPath)
	if err := first.StartSimulation(api.SimulationRequest{
		Interface: "recovery0", ConfigData: runtimeStateConfig,
	}); err != nil {
		t.Fatalf("StartSimulation() error = %v", err)
	}
	first.mu.Lock()
	generation := first.simulation.runtimeGeneration
	if err := first.stopSimulationLocked(false); err != nil {
		first.mu.Unlock()
		t.Fatalf("shutdown stop error = %v", err)
	}
	first.mu.Unlock()

	runtimePath := runtimeStatePath(stateDir, defaultSessionID, generation)
	if err := os.WriteFile(runtimePath, []byte(`{"schemaVersion":1`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	second := recoveryTestDaemon(t, recoveryPath)
	second.recoverActiveSimulation()
	status := second.GetStatus()
	if status.Recovery == nil || status.Recovery.State != recoveryStateFailed ||
		!strings.Contains(status.Recovery.Message, "runtime state") {
		t.Fatalf("recovery status = %#v", status.Recovery)
	}
}

func drainWrites(writes <-chan error, window time.Duration) int {
	deadline := time.After(window)
	count := 0
	for {
		select {
		case <-writes:
			count++
		case <-deadline:
			return count
		}
	}
}

func assertSameFaults(t *testing.T, before, after devicestate.State) {
	t.Helper()
	if len(after.InterfaceFaults) != len(before.InterfaceFaults) {
		t.Fatalf("interface faults after restart = %#v, want %#v",
			after.InterfaceFaults, before.InterfaceFaults)
	}
	for index, fault := range before.InterfaceFaults {
		if after.InterfaceFaults[index] != fault {
			t.Errorf("interface fault %d = %#v, want %#v", index, after.InterfaceFaults[index], fault)
		}
	}
	if len(after.DeviceFaults) != len(before.DeviceFaults) {
		t.Fatalf("device faults after restart = %#v, want %#v",
			after.DeviceFaults, before.DeviceFaults)
	}
	for index, fault := range before.DeviceFaults {
		if after.DeviceFaults[index] != fault {
			t.Errorf("device fault %d = %#v, want %#v", index, after.DeviceFaults[index], fault)
		}
	}
}

// The restored history is a prefix of what the device reports afterwards, not
// necessarily a copy: anything the session does after the restart appends.
func assertHistoryPreserved(t *testing.T, before, after []devicestate.Event) {
	t.Helper()
	if len(after) < len(before) {
		t.Fatalf("event history shrank across the restart: %d < %d", len(after), len(before))
	}
	for index, event := range before {
		restored := after[index]
		if restored.Version != event.Version || restored.Kind != event.Kind ||
			restored.Target != event.Target {
			t.Fatalf("event %d after restart = %v/%s/%q, want %v/%s/%q",
				index, restored.Version, restored.Kind, restored.Target,
				event.Version, event.Kind, event.Target)
		}
	}
}

func assertCarrierRecovered(t *testing.T, cleared devicestate.State) {
	t.Helper()
	if len(cleared.InterfaceFaults) != 0 {
		t.Fatalf("faults after clear = %#v", cleared.InterfaceFaults)
	}
	for _, iface := range cleared.Running.Network.Interfaces {
		if iface.Name == "GigabitEthernet1/0/1" {
			if !iface.OperUp || !iface.CarrierUp {
				t.Fatalf("clearing the restored fault left %s down: %#v", iface.Name, iface)
			}
			return
		}
	}
	t.Fatal("restored interface GigabitEthernet1/0/1 is missing")
}
