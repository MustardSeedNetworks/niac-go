package daemon

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

func TestRecoverActiveSimulationAfterDaemonRestart(t *testing.T) {
	t.Setenv(e2eDryRunEnv, "true")
	t.Setenv("NIAC_CONFIGS_DIR", t.TempDir())
	recoveryPath := filepath.Join(t.TempDir(), activeSimulationFileName)

	first := recoveryTestDaemon(t, recoveryPath)
	request := api.SimulationRequest{
		Interface:  "recovery0",
		ConfigData: validRecoveryConfig,
	}
	if startErr := first.StartSimulation(request); startErr != nil {
		t.Fatalf("StartSimulation() error = %v", startErr)
	}
	if _, statErr := os.Stat(recoveryPath); statErr != nil {
		t.Fatalf("recovery state was not persisted: %v", statErr)
	}

	first.mu.Lock()
	if stopErr := first.stopSimulationLocked(false); stopErr != nil {
		first.mu.Unlock()
		t.Fatalf("shutdown stop error = %v", stopErr)
	}
	first.mu.Unlock()

	second := recoveryTestDaemon(t, recoveryPath)
	second.recoverActiveSimulation()
	status := second.GetStatus()
	if !status.Running || status.Interface != request.Interface {
		t.Fatalf("recovered status = %#v", status)
	}
	if status.Recovery == nil || status.Recovery.State != recoveryStateRecovered {
		t.Fatalf("recovery status = %#v", status.Recovery)
	}
	if stopErr := second.StopSimulation(""); stopErr != nil {
		t.Fatalf("StopSimulation() error = %v", stopErr)
	}
	if _, statErr := os.Stat(recoveryPath); !os.IsNotExist(statErr) {
		t.Fatalf("explicit stop did not clear recovery state: %v", statErr)
	}
}

func TestRecoverConcurrentTrunkSessionsAfterDaemonRestart(t *testing.T) {
	t.Setenv(e2eDryRunEnv, "true")
	t.Setenv("NIAC_CONFIGS_DIR", t.TempDir())
	recoveryPath := filepath.Join(t.TempDir(), activeSimulationFileName)
	policy := fabric.PhysicalAttachmentPolicy{
		Interface: "eth0", Mode: fabric.ModeTrunk, AllowedVLANs: []uint16{200, 201},
	}
	first, err := NewDaemon(
		Config{
			RecoveryPath:       recoveryPath,
			AttachmentPolicies: []fabric.PhysicalAttachmentPolicy{policy},
		},
	)
	if err != nil {
		t.Fatalf("NewDaemon(first): %v", err)
	}
	first.apiServer = api.NewServer(api.ServerConfig{})
	if err = first.StartSimulation(trunkSessionRequest("hospital", 200)); err != nil {
		t.Fatalf("StartSimulation(hospital): %v", err)
	}
	if err = first.StartSimulation(trunkSessionRequest("warehouse", 201)); err != nil {
		t.Fatalf("StartSimulation(warehouse): %v", err)
	}
	state, err := readRecoveryState(recoveryPath)
	if err != nil || len(state.Sessions) != 2 {
		t.Fatalf("readRecoveryState() state = %#v, error = %v", state, err)
	}

	first.mu.Lock()
	for first.sessions.len() > 0 {
		first.simulation = first.sessions.first()
		if err = first.stopSimulationLocked(false); err != nil {
			first.mu.Unlock()
			t.Fatalf("shutdown stop: %v", err)
		}
	}
	first.mu.Unlock()

	second, err := NewDaemon(
		Config{
			RecoveryPath:       recoveryPath,
			AttachmentPolicies: []fabric.PhysicalAttachmentPolicy{policy},
		},
	)
	if err != nil {
		t.Fatalf("NewDaemon(second): %v", err)
	}
	second.apiServer = api.NewServer(api.ServerConfig{})
	second.recoverActiveSimulation()
	status := second.GetStatus()
	if len(status.Sessions) != 2 || status.Recovery == nil ||
		status.Recovery.State != recoveryStateRecovered {
		t.Fatalf("recovered status = %#v", status)
	}
}

func TestStoppingOneSessionPreservesOtherRecoveryIntent(t *testing.T) {
	t.Setenv(e2eDryRunEnv, "true")
	t.Setenv("NIAC_CONFIGS_DIR", t.TempDir())
	recoveryPath := filepath.Join(t.TempDir(), activeSimulationFileName)
	d, err := NewDaemon(Config{
		RecoveryPath: recoveryPath,
		AttachmentPolicies: []fabric.PhysicalAttachmentPolicy{{
			Interface: "eth0", Mode: fabric.ModeTrunk, AllowedVLANs: []uint16{200, 201},
		}},
	})
	if err != nil {
		t.Fatalf("NewDaemon(): %v", err)
	}
	d.apiServer = api.NewServer(api.ServerConfig{})
	if err = d.StartSimulation(trunkSessionRequest("hospital", 200)); err != nil {
		t.Fatalf("StartSimulation(hospital): %v", err)
	}
	if err = d.StartSimulation(trunkSessionRequest("warehouse", 201)); err != nil {
		t.Fatalf("StartSimulation(warehouse): %v", err)
	}
	if err = d.StopSimulation("hospital"); err != nil {
		t.Fatalf("StopSimulation(hospital): %v", err)
	}
	state, err := readRecoveryState(recoveryPath)
	if err != nil {
		t.Fatalf("readRecoveryState(): %v", err)
	}
	if len(state.Sessions) != 1 || state.Sessions[0].Request.SessionID != "warehouse" {
		t.Fatalf("recovery sessions = %#v", state.Sessions)
	}
}

func TestRecoverActiveSimulationFailsClosedForStaleConfig(t *testing.T) {
	t.Setenv(e2eDryRunEnv, "true")
	recoveryPath := filepath.Join(t.TempDir(), activeSimulationFileName)
	state := activeSimulationState{
		SchemaVersion: activeSimulationSchemaVersion,
		Sessions: []activeSimulationEntry{{Request: api.SimulationRequest{
			SessionID:  "default",
			Interface:  "recovery0",
			ConfigPath: filepath.Join(t.TempDir(), "missing.yaml"),
		}}},
		SavedAt: time.Now().UTC(),
	}
	writeActiveSimulationFixture(t, recoveryPath, state)

	daemon := recoveryTestDaemon(t, recoveryPath)
	daemon.recoverActiveSimulation()
	status := daemon.GetStatus()
	if status.Running {
		t.Fatalf("stale state started a simulation: %#v", status)
	}
	if status.Recovery == nil || status.Recovery.State != recoveryStateFailed ||
		!strings.Contains(status.Recovery.Message, "configuration") {
		t.Fatalf("recovery status = %#v", status.Recovery)
	}
}

func TestRecoverActiveSimulationPreservesFailedSessionIntent(t *testing.T) {
	t.Setenv(e2eDryRunEnv, "true")
	configDir := t.TempDir()
	t.Setenv("NIAC_CONFIGS_DIR", configDir)
	validPath := filepath.Join(configDir, "valid.yaml")
	if err := os.WriteFile(validPath, []byte(validRecoveryConfig), 0o600); err != nil {
		t.Fatal(err)
	}
	recoveryPath := filepath.Join(t.TempDir(), activeSimulationFileName)
	state := activeSimulationState{
		SchemaVersion: activeSimulationSchemaVersion,
		Sessions: []activeSimulationEntry{
			{Request: api.SimulationRequest{SessionID: "valid", Interface: "recovery0", ConfigPath: validPath}},
			{Request: api.SimulationRequest{
				SessionID: "missing", Interface: "recovery1",
				ConfigPath: filepath.Join(configDir, "missing.yaml"),
			}},
		},
		SavedAt: time.Now().UTC(),
	}
	writeActiveSimulationFixture(t, recoveryPath, state)
	daemon := recoveryTestDaemon(t, recoveryPath)
	daemon.recoverActiveSimulation()

	persisted, err := readRecoveryState(recoveryPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(persisted.Sessions) != 2 {
		t.Fatalf("persisted sessions = %#v, want both original intents", persisted.Sessions)
	}
	if err = daemon.StopSimulation("valid"); err != nil {
		t.Fatal(err)
	}
	persisted, err = readRecoveryState(recoveryPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(persisted.Sessions) != 1 || persisted.Sessions[0].Request.SessionID != "missing" {
		t.Fatalf("persisted sessions after stop = %#v, want failed intent", persisted.Sessions)
	}
}

func TestRecoverActiveSimulationRechecksAttachmentPolicy(t *testing.T) {
	t.Setenv(e2eDryRunEnv, "true")
	configDir := t.TempDir()
	t.Setenv("NIAC_CONFIGS_DIR", configDir)
	configPath := filepath.Join(configDir, "routed.yaml")
	if writeErr := os.WriteFile(configPath, []byte(routedRecoveryConfig), 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}
	recoveryPath := filepath.Join(t.TempDir(), activeSimulationFileName)
	writeActiveSimulationFixture(t, recoveryPath, activeSimulationState{
		SchemaVersion: activeSimulationSchemaVersion,
		Sessions: []activeSimulationEntry{{Request: api.SimulationRequest{
			SessionID:      "default",
			Interface:      "recovery0",
			Attachment:     "tester",
			AttachmentMode: fabric.ModeDirect,
			ConfigPath:     configPath,
		}}},
		SavedAt: time.Now().UTC(),
	})

	daemon := recoveryTestDaemon(t, recoveryPath)
	daemon.recoverActiveSimulation()
	status := daemon.GetStatus()
	if status.Running || status.Recovery == nil ||
		!strings.Contains(status.Recovery.Message, ErrUnsafeTopology.Error()) {
		t.Fatalf("recovery status = %#v", status)
	}
}

func TestRecoverActiveSimulationIgnoresInterruptedTempWrite(t *testing.T) {
	recoveryDir := t.TempDir()
	recoveryPath := filepath.Join(recoveryDir, activeSimulationFileName)
	if writeErr := os.WriteFile(
		filepath.Join(recoveryDir, ".active-simulation-interrupted"),
		[]byte(`{"schemaVersion":1`),
		0o600,
	); writeErr != nil {
		t.Fatal(writeErr)
	}

	daemon := recoveryTestDaemon(t, recoveryPath)
	daemon.recoverActiveSimulation()
	status := daemon.GetStatus()
	if status.Running || status.Recovery != nil {
		t.Fatalf("interrupted temp write affected recovery: %#v", status)
	}
}

func TestRecoverActiveSimulationReportsInvalidState(t *testing.T) {
	recoveryPath := filepath.Join(t.TempDir(), activeSimulationFileName)
	if writeErr := os.WriteFile(recoveryPath, []byte(`{"schemaVersion":1`), 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}

	daemon := recoveryTestDaemon(t, recoveryPath)
	daemon.recoverActiveSimulation()
	status := daemon.GetStatus()
	if status.Recovery == nil || status.Recovery.State != recoveryStateFailed ||
		!strings.Contains(status.Recovery.Message, "decode recovery state") {
		t.Fatalf("recovery status = %#v", status.Recovery)
	}
}

func recoveryTestDaemon(t *testing.T, recoveryPath string) *Daemon {
	t.Helper()
	daemon, newErr := NewDaemon(Config{
		StoragePath:  "disabled",
		RecoveryPath: recoveryPath,
	})
	if newErr != nil {
		t.Fatalf("NewDaemon() error = %v", newErr)
	}
	daemon.apiServer = api.NewServer(api.ServerConfig{})
	t.Cleanup(func() {
		daemon.mu.Lock()
		if daemon.simulation != nil {
			_ = daemon.stopSimulationLocked(false)
		}
		daemon.mu.Unlock()
	})
	return daemon
}

func writeActiveSimulationFixture(t *testing.T, path string, state activeSimulationState) {
	t.Helper()
	data, marshalErr := json.Marshal(state)
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	if writeErr := writeRecoveryState(path, data); writeErr != nil {
		t.Fatal(writeErr)
	}
}

const validRecoveryConfig = `devices:
  - name: recovery-router
    type: router
    mac: "02:00:00:00:00:01"
    ips: ["192.0.2.10"]
`

const routedRecoveryConfig = `networks:
  - name: lab
    subnet: 192.0.2.0/24
attachments:
  - name: tester
    connect: lab
devices:
  - name: recovery-router
    type: router
    mac: "02:00:00:00:00:01"
    interfaces:
      - name: lab0
        network: lab
        address: 192.0.2.10/24
`
