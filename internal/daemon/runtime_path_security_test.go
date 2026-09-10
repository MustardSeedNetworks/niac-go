package daemon

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
)

func TestRuntimeSnapshotRejectsEscapedRuntimeDirectory(t *testing.T) {
	t.Setenv(e2eDryRunEnv, "true")
	t.Setenv("NIAC_CONFIGS_DIR", t.TempDir())
	directory := t.TempDir()
	outside := t.TempDir()
	if err := makeTestDirectoryLink(outside, filepath.Join(directory, runtimeStateDirName)); err != nil {
		t.Fatal(err)
	}
	d := recoveryTestDaemon(t, filepath.Join(directory, activeSimulationFileName))
	if err := d.StartSimulation(
		api.SimulationRequest{Interface: "recovery0", ConfigData: runtimeStateConfig},
	); err == nil {
		t.Fatal("committed runtime state through an escaped directory symlink")
	}
	entries, err := os.ReadDir(outside)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("runtime write escaped into external directory: %v", entries)
	}
}

func TestStateWriteCannotTraverseOutsideRoot(t *testing.T) {
	directory := t.TempDir()
	victim := filepath.Join(directory, "external.json")
	if err := os.WriteFile(victim, []byte("external data"), 0o600); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(directory, "state")
	for _, name := range []string{"../external.json", victim} {
		if err := writeStateFile(root, name, []byte("overwrite")); err == nil {
			t.Fatalf("accepted escaped relative name %q", name)
		}
	}
	if data, err := os.ReadFile(victim); err != nil || string(data) != "external data" {
		t.Fatalf("state write touched external data: %q %v", data, err)
	}
}

func TestInlineFinishRetainsOriginalDirectoryCapability(t *testing.T) {
	for _, committed := range []bool{false, true} {
		t.Run(map[bool]string{false: "rollback", true: "commit"}[committed], func(t *testing.T) {
			checkInlineFinishCapability(t, committed)
		})
	}
}

func checkInlineFinishCapability(t *testing.T, committed bool) {
	t.Helper()
	parent := t.TempDir()
	directory := filepath.Join(parent, "configs")
	moved := filepath.Join(parent, "moved")
	outside := t.TempDir()
	t.Setenv("NIAC_CONFIGS_DIR", directory)
	path, finish, err := stageInlineSessionConfig("devices: []\n", defaultSessionID)
	if err != nil {
		t.Fatal(err)
	}
	wasMoved, err := moveOpenedTestDirectory(directory, moved)
	if err != nil {
		finish(false)
		t.Fatal(err)
	}
	victim := filepath.Join(outside, filepath.Base(path))
	if err = os.WriteFile(victim, []byte("external data"), 0o600); err != nil {
		finish(false)
		t.Fatal(err)
	}
	if wasMoved {
		if err = makeTestDirectoryLink(outside, directory); err != nil {
			finish(false)
			t.Fatal(err)
		}
	} else {
		moved = directory
	}
	finish(committed)
	if data, readErr := os.ReadFile(victim); readErr != nil || string(data) != "external data" {
		t.Fatalf("cleanup escaped original directory: %q %v", data, readErr)
	}
	_, statErr := os.Stat(filepath.Join(moved, filepath.Base(path)))
	if (committed && statErr != nil) || (!committed && !errors.Is(statErr, os.ErrNotExist)) {
		t.Fatalf("original staged file has wrong lifecycle after commit=%v: %v", committed, statErr)
	}
}

func TestRuntimeCleanupCannotFollowEscapedDirectory(t *testing.T) {
	directory := t.TempDir()
	outside := t.TempDir()
	const generation = "00000000000000000000000000000001"
	name := defaultSessionID + "." + generation + ".json"
	victim := filepath.Join(outside, name)
	if err := os.WriteFile(victim, []byte("external data"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := makeTestDirectoryLink(outside, filepath.Join(directory, runtimeStateDirName)); err != nil {
		t.Fatal(err)
	}
	d := recoveryTestDaemon(t, filepath.Join(directory, activeSimulationFileName))
	d.clearRuntimeState(defaultSessionID, generation)
	if data, err := os.ReadFile(victim); err != nil || string(data) != "external data" {
		t.Fatalf("runtime cleanup touched an external file: %q, %v", data, err)
	}
}
