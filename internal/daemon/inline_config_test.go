package daemon

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
)

// TestStageInlineSessionConfigRejectsEscapingIDs is why
// the session id is validated here rather than trusted from the caller.
//
// The id becomes part of the inline config's filename. The HTTP handler checks
// it, but the crash-recovery path calls StartSimulation with a request read back
// from active-simulation.json and never reaches that validator, so the guarantee
// has to live at the sink. Each id below either escapes the configs directory or
// puts a character in the filename that has no business there; none may reach
// the filesystem.
func TestStageInlineSessionConfigRejectsEscapingIDs(t *testing.T) {
	ids := []string{
		"../../etc/evil",
		"..",
		"a/../../b",
		"/absolute",
		"has space",
		"UPPER",
		"-leading-hyphen",
		"trailing-hyphen-",
		"",
		strings.Repeat("a", 41),
	}

	for _, id := range ids {
		t.Run(id, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("NIAC_CONFIGS_DIR", dir)

			path, finish, err := stageInlineSessionConfig("devices: []\n", id, testInlineGeneration)
			if finish != nil {
				finish(false)
			}
			if err == nil {
				t.Fatalf("stageInlineSessionConfig(%q) wrote %q, want a refusal", id, path)
			}
			if !errors.Is(err, errInvalidInlineSessionID) {
				t.Errorf("error = %v, want errInvalidInlineSessionID", err)
			}
			if path != "" {
				t.Errorf("path = %q, want empty alongside the error", path)
			}

			assertNothingWritten(t, dir)
		})
	}
}

// TestStageInlineSessionConfigWritesValidSessions is the positive half: the
// ids the API accepts produce independently persisted launches.
func TestStageInlineSessionConfigWritesValidSessions(t *testing.T) {
	tests := []struct {
		sessionID string
		wantName  string
	}{
		{defaultSessionID, "_running.default."},
		{"lab-01", "_running.lab-01."},
		{"a", "_running.a."},
	}

	for _, tt := range tests {
		t.Run(tt.sessionID, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("NIAC_CONFIGS_DIR", dir)

			const content = "devices: []\n"

			path, finish, err := stageInlineSessionConfig(content, tt.sessionID, testInlineGeneration)
			if err != nil {
				t.Fatalf("stageInlineSessionConfig(%q): %v", tt.sessionID, err)
			}
			finish(true)

			if got, want := filepath.Base(path), tt.wantName+testInlineGeneration+".inline.yaml"; got != want {
				t.Errorf("filename = %q, want the launch's generation %q", got, want)
			}
			if !filepath.IsAbs(path) {
				t.Errorf("path = %q, want an absolute path", path)
			}

			written, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatalf("read back %s: %v", path, readErr)
			}
			if string(written) != content {
				t.Errorf("file contains %q, want %q", written, content)
			}
		})
	}
}

const testInlineGeneration = "0123456789abcdef0123456789abcdef"

// TestInlineConfigLifecycleRemovesSupersededFiles pins #2444: every inline
// start wrote a file and only a failed start removed it, so replacing,
// stopping or deleting a session left its file behind for good.
func TestInlineConfigLifecycleRemovesSupersededFiles(t *testing.T) {
	t.Setenv(e2eDryRunEnv, "true")
	configs := t.TempDir()
	t.Setenv("NIAC_CONFIGS_DIR", configs)
	d := recoveryTestDaemon(t, filepath.Join(t.TempDir(), activeSimulationFileName))
	request := api.SimulationRequest{Interface: "recovery0", ConfigData: validRecoveryConfig}

	if err := d.StartSimulation(request); err != nil {
		t.Fatal(err)
	}
	first := d.simulation.ConfigPath
	if err := d.StartSimulation(request); err != nil {
		t.Fatal(err)
	}
	second := d.simulation.ConfigPath
	if first == second {
		t.Fatalf("replacement reused the superseded file %s", first)
	}
	if got := sessionInlineConfigs(t, configs, defaultSessionID); !slices.Equal(got, []string{second}) {
		t.Fatalf("after replacement inline files = %v, want only the replacement's %s", got, second)
	}

	if err := d.StopSimulation(""); err != nil {
		t.Fatal(err)
	}
	if got := sessionInlineConfigs(t, configs, defaultSessionID); len(got) != 0 {
		t.Fatalf("after stop inline files = %v, want none", got)
	}
}

// The DELETE route stops one named session: only that session's file goes.
func TestStoppingNamedSessionRemovesOnlyItsInlineConfig(t *testing.T) {
	t.Setenv(e2eDryRunEnv, "true")
	configs := t.TempDir()
	t.Setenv("NIAC_CONFIGS_DIR", configs)
	d := recoveryTestDaemon(t, filepath.Join(t.TempDir(), activeSimulationFileName))
	for _, id := range []string{"hospital", "warehouse"} {
		request := api.SimulationRequest{SessionID: id, Interface: "recovery-" + id, ConfigData: validRecoveryConfig}
		if err := d.StartSimulation(request); err != nil {
			t.Fatalf("StartSimulation(%s): %v", id, err)
		}
	}
	if err := d.StopSimulation("hospital"); err != nil {
		t.Fatal(err)
	}
	if got := sessionInlineConfigs(t, configs, "hospital"); len(got) != 0 {
		t.Fatalf("stopped session kept %v", got)
	}
	if got := sessionInlineConfigs(t, configs, "warehouse"); len(got) != 1 {
		t.Fatalf("running session's inline files = %v, want its one file", got)
	}
}

// A failed replacement removes its own uncommitted file and leaves the active
// generation's, which recovery still needs.
func TestFailedReplacementKeepsOnlyActiveInlineConfig(t *testing.T) {
	t.Setenv(e2eDryRunEnv, "true")
	configs := t.TempDir()
	t.Setenv("NIAC_CONFIGS_DIR", configs)
	d := recoveryTestDaemon(t, filepath.Join(t.TempDir(), activeSimulationFileName))
	request := api.SimulationRequest{Interface: "recovery0", ConfigData: runtimeStateConfig}
	if err := d.StartSimulation(request); err != nil {
		t.Fatal(err)
	}
	active := d.simulation.ConfigPath
	stopRuntimeStateWriter(d.simulation)
	d.cfg.RecoveryPath = t.TempDir()
	if err := d.StartSimulation(request); err == nil {
		t.Fatal("replacement succeeded with an unwritable recovery path")
	}
	if got := sessionInlineConfigs(t, configs, defaultSessionID); !slices.Equal(got, []string{active}) {
		t.Fatalf("inline files = %v, want only the active %s", got, active)
	}
}

// A shutdown keeps the file; the restarted daemon recovers from it and its
// explicit stop is what removes it.
func TestRecoveredSessionFindsAndThenRemovesItsInlineConfig(t *testing.T) {
	t.Setenv(e2eDryRunEnv, "true")
	configs := t.TempDir()
	t.Setenv("NIAC_CONFIGS_DIR", configs)
	recoveryPath := filepath.Join(t.TempDir(), activeSimulationFileName)
	first := recoveryTestDaemon(t, recoveryPath)
	if err := first.StartSimulation(api.SimulationRequest{
		Interface: "recovery0", ConfigData: validRecoveryConfig,
	}); err != nil {
		t.Fatal(err)
	}
	path := first.simulation.ConfigPath
	first.mu.Lock()
	stopErr := first.stopSimulationLocked(false)
	first.mu.Unlock()
	if stopErr != nil {
		t.Fatal(stopErr)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("shutdown removed the file recovery needs: %v", err)
	}

	second := recoveryTestDaemon(t, recoveryPath)
	second.recoverActiveSimulation()
	if status := second.GetStatus(); !status.Running ||
		status.Recovery == nil || status.Recovery.State != recoveryStateRecovered {
		t.Fatalf("recovered status = %#v", status)
	}
	if second.simulation.ConfigPath != path {
		t.Fatalf("recovered from %s, want %s", second.simulation.ConfigPath, path)
	}
	if err := second.StopSimulation(""); err != nil {
		t.Fatal(err)
	}
	if got := sessionInlineConfigs(t, configs, defaultSessionID); len(got) != 0 {
		t.Fatalf("after recovered stop inline files = %v, want none", got)
	}
}

// Cleanup owns only the file the daemon wrote, never a request's ConfigPath.
func TestStopKeepsRequestConfigPath(t *testing.T) {
	t.Setenv(e2eDryRunEnv, "true")
	configs := t.TempDir()
	t.Setenv("NIAC_CONFIGS_DIR", configs)
	authored := filepath.Join(configs, "authored.yaml")
	if err := os.WriteFile(authored, []byte(validRecoveryConfig), 0o600); err != nil {
		t.Fatal(err)
	}
	d := recoveryTestDaemon(t, filepath.Join(t.TempDir(), activeSimulationFileName))
	request := api.SimulationRequest{Interface: "recovery0", ConfigPath: authored}
	for range 2 {
		if err := d.StartSimulation(request); err != nil {
			t.Fatal(err)
		}
	}
	if err := d.StopSimulation(""); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(authored); err != nil {
		t.Fatalf("stop removed the request's own config: %v", err)
	}
}

// The configs directory is a config root, so a client can start a session from
// another session's inline file by path. Stopping the writer must not delete
// what the reader still runs on.
func TestInlineConfigInUseByAnotherSessionSurvivesStop(t *testing.T) {
	t.Setenv(e2eDryRunEnv, "true")
	configs := t.TempDir()
	t.Setenv("NIAC_CONFIGS_DIR", configs)
	d := recoveryTestDaemon(t, filepath.Join(t.TempDir(), activeSimulationFileName))
	if err := d.StartSimulation(api.SimulationRequest{
		SessionID: "hospital", Interface: "recovery-hospital", ConfigData: validRecoveryConfig,
	}); err != nil {
		t.Fatal(err)
	}
	inline := d.sessions.get("hospital").ConfigPath
	if err := d.StartSimulation(api.SimulationRequest{
		SessionID: "warehouse", Interface: "recovery-warehouse", ConfigPath: inline,
	}); err != nil {
		t.Fatal(err)
	}
	if err := d.StopSimulation("hospital"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(inline); err != nil {
		t.Fatalf("stop removed a file another running session uses: %v", err)
	}
}

func sessionInlineConfigs(t *testing.T, directory, sessionID string) []string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(directory, "_running."+sessionID+".*.inline.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for i, match := range matches {
		if matches[i], err = filepath.Abs(match); err != nil {
			t.Fatal(err)
		}
	}
	return matches
}

// assertNothingWritten fails if the configs directory gained any file, which is
// what distinguishes a refusal from a write that merely reported an error.
func assertNothingWritten(t *testing.T, dir string) {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read configs dir: %v", err)
	}

	if len(entries) != 0 {
		names := make([]string, len(entries))
		for i, entry := range entries {
			names[i] = entry.Name()
		}

		t.Errorf("configs dir contains %v, want nothing written", names)
	}
}
