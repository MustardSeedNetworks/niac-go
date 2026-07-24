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
	if startErr := first.StartSimulation(request, fullSimulationEntitlements()); startErr != nil {
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
	second.recoverActiveSimulation(fullSimulationEntitlements())
	status := second.GetStatus()
	if !status.Running || status.Interface != request.Interface {
		t.Fatalf("recovered status = %#v", status)
	}
	if status.Recovery == nil || status.Recovery.State != recoveryStateRecovered {
		t.Fatalf("recovery status = %#v", status.Recovery)
	}
	if stopErr := second.StopSimulation(); stopErr != nil {
		t.Fatalf("StopSimulation() error = %v", stopErr)
	}
	if _, statErr := os.Stat(recoveryPath); !os.IsNotExist(statErr) {
		t.Fatalf("explicit stop did not clear recovery state: %v", statErr)
	}
}

func TestRecoverActiveSimulationFailsClosedForStaleConfig(t *testing.T) {
	t.Setenv(e2eDryRunEnv, "true")
	recoveryPath := filepath.Join(t.TempDir(), activeSimulationFileName)
	state := activeSimulationState{
		SchemaVersion: activeSimulationSchemaVersion,
		Request: api.SimulationRequest{
			Interface:  "recovery0",
			ConfigPath: filepath.Join(t.TempDir(), "missing.yaml"),
		},
		SavedAt: time.Now().UTC(),
	}
	writeActiveSimulationFixture(t, recoveryPath, state)

	daemon := recoveryTestDaemon(t, recoveryPath)
	daemon.recoverActiveSimulation(fullSimulationEntitlements())
	status := daemon.GetStatus()
	if status.Running {
		t.Fatalf("stale state started a simulation: %#v", status)
	}
	if status.Recovery == nil || status.Recovery.State != recoveryStateFailed ||
		!strings.Contains(status.Recovery.Message, "configuration") {
		t.Fatalf("recovery status = %#v", status.Recovery)
	}
}

func TestRecoverActiveSimulationRechecksEntitlements(t *testing.T) {
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
		Request: api.SimulationRequest{
			Interface:  "recovery0",
			ConfigPath: configPath,
		},
		SavedAt: time.Now().UTC(),
	})

	daemon := recoveryTestDaemon(t, recoveryPath)
	daemon.recoverActiveSimulation(api.SimulationEntitlements{})
	status := daemon.GetStatus()
	if status.Running || status.Recovery == nil ||
		!strings.Contains(status.Recovery.Message, api.ErrRoutedLabsLicenseRequired.Error()) {
		t.Fatalf("recovery status = %#v", status)
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
		Request: api.SimulationRequest{
			Interface:      "recovery0",
			Attachment:     "tester",
			AttachmentMode: fabric.ModeDirect,
			ConfigPath:     configPath,
		},
		SavedAt: time.Now().UTC(),
	})

	daemon := recoveryTestDaemon(t, recoveryPath)
	daemon.recoverActiveSimulation(fullSimulationEntitlements())
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
	daemon.recoverActiveSimulation(fullSimulationEntitlements())
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
	daemon.recoverActiveSimulation(fullSimulationEntitlements())
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
