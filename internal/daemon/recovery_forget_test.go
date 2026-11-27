package daemon

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
)

const (
	forgetGenMissing = "00000000000000000000000000000001"
	forgetGenOther   = "00000000000000000000000000000002"
)

// savedSessionsDaemon builds a daemon whose recovery file holds one session per
// name, none startable (the configuration is missing), then runs recovery.
func savedSessionsDaemon(t *testing.T, names ...string) (*Daemon, string) {
	t.Helper()
	t.Setenv(e2eDryRunEnv, "true")
	configDir := t.TempDir()
	t.Setenv("NIAC_CONFIGS_DIR", configDir)
	recoveryPath := filepath.Join(t.TempDir(), activeSimulationFileName)
	state := activeSimulationState{SchemaVersion: activeSimulationSchemaVersion, SavedAt: time.Now().UTC()}
	for i, name := range names {
		generation := forgetGenMissing
		if i > 0 {
			generation = forgetGenOther
		}
		state.Sessions = append(state.Sessions, activeSimulationEntry{
			Request: api.SimulationRequest{
				SessionID: name, Interface: "en16", ConfigPath: filepath.Join(configDir, name+".yaml"),
			},
			Generation: generation,
		})
	}
	writeActiveSimulationFixture(t, recoveryPath, state)
	daemon := recoveryTestDaemon(t, recoveryPath)
	daemon.recoverActiveSimulation()
	return daemon, recoveryPath
}

func failedIDs(status api.SimulationStatus) []string {
	if status.Recovery == nil {
		return nil
	}
	ids := make([]string, 0, len(status.Recovery.Failed))
	for _, failed := range status.Recovery.Failed {
		ids = append(ids, failed.SessionID)
	}
	return ids
}

func TestRecoveryReportsFailedSessionsStructurally(t *testing.T) {
	daemon, _ := savedSessionsDaemon(t, "modern")
	status := daemon.GetStatus()
	if status.Recovery == nil || len(status.Recovery.Failed) != 1 {
		t.Fatalf("recovery = %#v, want one failed session", status.Recovery)
	}
	failed := status.Recovery.Failed[0]
	if failed.SessionID != "modern" || failed.Interface != "en16" || failed.Reason == "" {
		t.Fatalf("failed = %#v", failed)
	}
	if strings.Contains(status.Recovery.Message, "Correct or remove") {
		t.Fatalf("message still tells the operator to edit the file: %q", status.Recovery.Message)
	}
}

func TestFailedRecoveryDoesNotBlockOtherSessions(t *testing.T) {
	daemon, recoveryPath := savedSessionsDaemon(t, "modern")
	if err := daemon.StartSimulation(api.SimulationRequest{
		SessionID: "t1", Interface: "recovery0", ConfigData: validRecoveryConfig,
	}); err != nil {
		t.Fatalf("start beside a failed saved session: %v", err)
	}
	state, err := readRecoveryState(recoveryPath)
	if err != nil || len(state.Sessions) != 2 {
		t.Fatalf("state = %#v, err = %v; want the failed record kept beside the new one", state, err)
	}
	if got := failedIDs(daemon.GetStatus()); len(got) != 1 || got[0] != "modern" {
		t.Fatalf("failed after a fresh start = %v", got)
	}
}

func TestForgetSavedSession(t *testing.T) {
	tests := []struct {
		name       string
		saved      []string
		forget     string
		wantErr    error
		wantFailed []string
		wantFile   []string
	}{
		{"only session removes the file", []string{"modern"}, "modern", nil, nil, nil},
		{
			"one of two keeps the other",
			[]string{"modern", "clinic"},
			"modern", nil,
			[]string{"clinic"},
			[]string{"clinic"},
		},
		{
			"unknown session",
			[]string{"modern"},
			"nope", api.ErrSavedSessionNotFound,
			[]string{"modern"},
			[]string{"modern"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			daemon, recoveryPath := savedSessionsDaemon(t, tt.saved...)
			if err := daemon.ForgetSavedSession(tt.forget); !errors.Is(err, tt.wantErr) {
				t.Fatalf("ForgetSavedSession() = %v, want %v", err, tt.wantErr)
			}
			if got := failedIDs(daemon.GetStatus()); !slices.Equal(got, tt.wantFailed) {
				t.Fatalf("failed = %v, want %v", got, tt.wantFailed)
			}
			if got := savedIDs(t, recoveryPath); !slices.Equal(got, tt.wantFile) {
				t.Fatalf("on disk = %v, want %v", got, tt.wantFile)
			}
		})
	}
}

// savedIDs lists the sessions in the recovery file; none when it is gone.
func savedIDs(t *testing.T, recoveryPath string) []string {
	t.Helper()
	state, err := readRecoveryState(recoveryPath)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]string, 0, len(state.Sessions))
	for _, session := range state.Sessions {
		ids = append(ids, session.Request.SessionID)
	}
	return ids
}

func TestForgetSavedSessionRefusesRunningSession(t *testing.T) {
	daemon, _ := savedSessionsDaemon(t, "modern")
	if err := daemon.StartSimulation(api.SimulationRequest{
		SessionID: "t1", Interface: "recovery0", ConfigData: validRecoveryConfig,
	}); err != nil {
		t.Fatal(err)
	}
	if err := daemon.ForgetSavedSession("t1"); !errors.Is(err, api.ErrSavedSessionNotFound) {
		t.Fatalf("forgetting a running session = %v, want ErrSavedSessionNotFound", err)
	}
	if !daemon.GetStatus().Running {
		t.Fatal("the running session was disturbed")
	}
}

func TestForgetSavedSessionRemovesItsRuntimeState(t *testing.T) {
	daemon, recoveryPath := savedSessionsDaemon(t, "modern")
	stateFile := daemon.runtimeStateFile("modern", forgetGenMissing)
	if err := writeStateFile(
		filepath.Dir(recoveryPath),
		runtimeStateName("modern", forgetGenMissing),
		[]byte("{}\n"),
	); err != nil {
		t.Fatal(err)
	}
	if err := daemon.ForgetSavedSession("modern"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stateFile); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("runtime state survived the forget: %v", err)
	}
}

func TestFreshStartUnderSavedSessionIDReplacesIt(t *testing.T) {
	daemon, _ := savedSessionsDaemon(t, "modern")
	if err := daemon.StartSimulation(api.SimulationRequest{
		SessionID: "modern", Interface: "recovery0", ConfigData: validRecoveryConfig,
	}); err != nil {
		t.Fatal(err)
	}
	if got := failedIDs(daemon.GetStatus()); len(got) != 0 {
		t.Fatalf("failed = %v after the session was started afresh", got)
	}
	if err := daemon.ForgetSavedSession("modern"); !errors.Is(err, api.ErrSavedSessionNotFound) {
		t.Fatalf("forget after a fresh start = %v, want ErrSavedSessionNotFound", err)
	}
}
