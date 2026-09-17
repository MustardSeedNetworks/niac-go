package main

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/foundation/pkg/instance"

	"github.com/MustardSeedNetworks/niac-go/internal/daemon"
)

// isolateDataDir points NIAC's data directory at a temporary one for the test,
// so nothing here touches the operator's real ~/.niac lock file.
func isolateDataDir(t *testing.T) {
	t.Helper()

	root := t.TempDir()
	t.Setenv("NIAC_LIBRARY_ROOT", filepath.Join(root, "library"))

	if got := daemon.DefaultDataDir(); got != root {
		t.Fatalf("DefaultDataDir() = %q, want %q", got, root)
	}
}

// The whole point of the lock: a one-shot run beside a live daemon must be
// refused, and refused by name, rather than opening the same database and
// library underneath it.
func TestOnceIsRefusedWhileAnotherInstanceHoldsTheDataDirectory(t *testing.T) {
	isolateDataDir(t)

	held, err := instance.Acquire(daemon.DefaultDataDir())
	if err != nil {
		t.Fatalf("acquiring the lock for the test holder: %v", err)
	}
	t.Cleanup(func() { _ = held.Release() })
	if portErr := held.SetPort(8445); portErr != nil {
		t.Fatalf("recording the holder's port: %v", portErr)
	}

	lock, onceErr := onceInstanceLock()
	if onceErr == nil {
		_ = lock.Release()
		t.Fatal("a one-shot run took the lock while another instance held it")
	}

	var coded codedError
	if !errors.As(onceErr, &coded) || coded.code != onceExitConfig {
		t.Fatalf("refusal did not carry exit code %d: %#v", onceExitConfig, onceErr)
	}

	message := onceErr.Error()
	for _, want := range []string{
		"another instance is already running",
		strconv.Itoa(os.Getpid()),
		"8445",
		"niac simulation start",
	} {
		if !strings.Contains(message, want) {
			t.Errorf("refusal %q does not name %q", message, want)
		}
	}
}

// Alone, a one-shot run behaves as it always did: it takes the lock, and hands
// it back so the next run -- or the daemon -- can start.
func TestOnceTakesAndReleasesTheLockWhenNothingHoldsIt(t *testing.T) {
	isolateDataDir(t)

	lock, err := onceInstanceLock()
	if err != nil {
		t.Fatalf("onceInstanceLock() on a free data directory: %v", err)
	}
	if releaseErr := lock.Release(); releaseErr != nil {
		t.Fatalf("releasing: %v", releaseErr)
	}

	next, err := onceInstanceLock()
	if err != nil {
		t.Fatalf("the data directory was not free after Release: %v", err)
	}
	if releaseErr := next.Release(); releaseErr != nil {
		t.Fatalf("releasing the second lock: %v", releaseErr)
	}
}

// The daemon takes the same lock on the same key, which is what makes the
// one-shot refusal above true rather than theoretical.
func TestTheDaemonIsRefusedByALiveInstanceToo(t *testing.T) {
	isolateDataDir(t)

	held, err := instance.Acquire(daemon.DefaultDataDir())
	if err != nil {
		t.Fatalf("acquiring the lock for the test holder: %v", err)
	}
	t.Cleanup(func() { _ = held.Release() })

	lock, daemonErr := acquireInstanceLock()
	if daemonErr == nil {
		_ = lock.Release()
		t.Fatal("a second daemon took the lock while another instance held it")
	}

	var heldErr *instance.HeldError
	if !errors.As(daemonErr, &heldErr) {
		t.Fatalf("the daemon's refusal is not a *instance.HeldError: %#v", daemonErr)
	}
	if heldErr.PID != os.Getpid() {
		t.Errorf("refusal names pid %d, want %d", heldErr.PID, os.Getpid())
	}
}
