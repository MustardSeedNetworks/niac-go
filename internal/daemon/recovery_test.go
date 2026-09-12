package daemon

import (
	"encoding/json"
	"errors"
	"io/fs"
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
		}, Generation: "00000000000000000000000000000001"}},
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
	recoveryPath := filepath.Join(t.TempDir(), activeSimulationFileName)
	first := recoveryTestDaemon(t, recoveryPath)
	if err := first.StartSimulation(api.SimulationRequest{
		SessionID: "valid", Interface: "recovery0", ConfigData: validRecoveryConfig,
	}); err != nil {
		t.Fatal(err)
	}
	stopRuntimeStateWriter(first.simulation)
	state, err := readRecoveryState(recoveryPath)
	if err != nil {
		t.Fatal(err)
	}
	missing := activeSimulationEntry{Request: api.SimulationRequest{
		SessionID: "missing", Interface: "recovery1", ConfigPath: filepath.Join(configDir, "missing.yaml"),
	}, Generation: "00000000000000000000000000000001"}
	state.Sessions = append(state.Sessions, missing)
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
	if persisted.Sessions[0].Generation != missing.Generation {
		t.Fatal("stopping another session lost the failed recovery generation")
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
		}, Generation: "00000000000000000000000000000001"}},
		SavedAt: time.Now().UTC(),
	})

	daemon := recoveryTestDaemon(t, recoveryPath)
	daemon.recoverActiveSimulation()
	status := daemon.GetStatus()
	if status.Running || status.Recovery == nil ||
		!strings.Contains(status.Recovery.Message, fabric.ErrUnsafeTopology.Error()) {
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
		!strings.Contains(status.Recovery.Message, "decode") {
		t.Fatalf("recovery status = %#v", status.Recovery)
	}
	// Truncated state is unreadable, so it is set aside for the same reason a
	// stale schema is: whatever is left behind would poison every later start.
	if _, statErr := os.Stat(recoveryPath); !errors.Is(statErr, fs.ErrNotExist) {
		t.Errorf("undecodable state left in place: stat err = %v", statErr)
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
	daemon.apiServer = api.NewServer(api.ServerConfig{LibraryRoot: t.TempDir()})
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

// An upgrade that bumps the recovery schema must carry the running sessions
// forward, not strand them (#2092, owner 2026-09-12: "schema updates should
// update"). Schema 3 added a per-session runtime generation; a schema 2 file
// has everything else, so it migrates by minting one.
func TestRecoveryStateMigratesSchemaTwoForward(t *testing.T) {
	recoveryPath := filepath.Join(t.TempDir(), activeSimulationFileName)
	stale := `{"schemaVersion":2,"savedAt":"2026-09-01T00:00:00Z","sessions":[` +
		`{"request":{"sessionId":"campus","interface":"eth0","configPath":"/tmp/campus.yaml"}}]}`
	if writeErr := os.WriteFile(recoveryPath, []byte(stale), 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}

	state, err := readRecoveryState(recoveryPath)
	if err != nil {
		t.Fatalf("readRecoveryState() error = %v, want a migrated state", err)
	}
	if state.SchemaVersion != activeSimulationSchemaVersion {
		t.Errorf("SchemaVersion = %d, want %d", state.SchemaVersion, activeSimulationSchemaVersion)
	}
	if len(state.Sessions) != 1 {
		t.Fatalf("got %d sessions, want 1", len(state.Sessions))
	}
	if state.Sessions[0].Request.SessionID != "campus" {
		t.Errorf("SessionID = %q, want \"campus\"", state.Sessions[0].Request.SessionID)
	}
	// The migration has to mint a generation, because validation rejects an
	// empty one and the whole point is that the session survives.
	if !validRuntimeGeneration(state.Sessions[0].Generation) {
		t.Errorf("Generation = %q, want a freshly minted one", state.Sessions[0].Generation)
	}
}

// State this build cannot migrate is stale, not recoverable. It must be set
// aside so the daemon carries on: leaving it in place made every later start
// fail, and deploy-validate passed throughout that outage.
func TestRecoverActiveSimulationQuarantinesUnmigratableState(t *testing.T) {
	recoveryDir := t.TempDir()
	recoveryPath := filepath.Join(recoveryDir, activeSimulationFileName)
	stale := []byte(`{"schemaVersion":99,"savedAt":"2026-09-01T00:00:00Z","sessions":[]}`)
	if writeErr := os.WriteFile(recoveryPath, stale, 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}

	daemon := recoveryTestDaemon(t, recoveryPath)
	daemon.recoverActiveSimulation()

	if _, statErr := os.Stat(recoveryPath); !errors.Is(statErr, fs.ErrNotExist) {
		t.Fatalf("stale state left in place: stat err = %v", statErr)
	}
	matches, globErr := filepath.Glob(recoveryPath + ".unusable-*")
	if globErr != nil {
		t.Fatal(globErr)
	}
	if len(matches) != 1 {
		t.Fatalf("got %d quarantined copies, want 1", len(matches))
	}
	kept, readErr := os.ReadFile(matches[0])
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(kept) != string(stale) {
		t.Error("quarantined copy does not match the original bytes")
	}

	status := daemon.GetStatus()
	if status.Recovery == nil || status.Recovery.State != recoveryStateFailed {
		t.Fatalf("recovery status = %#v", status.Recovery)
	}
	// The operator's next question is "where did it go", so the message answers
	// it. The old wording told them to remove a file themselves.
	for _, want := range []string{"set aside", filepath.Base(matches[0])} {
		if !strings.Contains(status.Recovery.Message, want) {
			t.Errorf("message %q does not mention %q", status.Recovery.Message, want)
		}
	}
}

// The defect in #2092 was not the refusal to recover — that part is correct —
// but that the rejected file then poisoned the write path: persisting a new
// session reads the existing file first, so every POST /api/v1/simulation
// failed with a generic 500 until an operator moved the file by hand.
func TestPersistingASessionSurvivesUnusableState(t *testing.T) {
	recoveryPath := filepath.Join(t.TempDir(), activeSimulationFileName)
	if writeErr := os.WriteFile(
		recoveryPath,
		[]byte(`{"schemaVersion":99,"savedAt":"2026-09-01T00:00:00Z","sessions":[]}`),
		0o600,
	); writeErr != nil {
		t.Fatal(writeErr)
	}

	daemon := recoveryTestDaemon(t, recoveryPath)
	requests, err := daemon.persistedSimulationRequests()
	if err != nil {
		t.Fatalf("persistedSimulationRequests() error = %v, want a fresh start", err)
	}
	if len(requests) != 0 {
		t.Fatalf("got %d carried-over requests, want 0", len(requests))
	}
}
