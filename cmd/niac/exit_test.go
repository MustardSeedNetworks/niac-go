package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// errExitSentinel unwinds the stack from a substituted exitProcess.
var errExitSentinel = errors.New("exitProcess called")

// withExitCapture runs fn with exitProcess substituted and reports the code fn
// exited with, if it exited at all.
//
// The substitute panics rather than returning. os.Exit never returns, so code
// written after it is unreachable — loadConfigOrExit, for example, returns its
// config variable on the line after exiting, which is nil on the failure path.
// A substitute that returned normally would let that nil escape and the test
// would panic somewhere unrelated, or worse, assert against a state the real
// binary can never reach.
func withExitCapture(t *testing.T, fn func()) (int, bool) {
	t.Helper()

	original := exitProcess
	t.Cleanup(func() { exitProcess = original })

	var (
		code   int
		exited bool
	)

	exitProcess = func(c int) {
		code = c
		exited = true

		panic(errExitSentinel)
	}

	func() {
		defer func() {
			recovered := recover()
			if recovered == nil {
				return
			}

			if err, ok := recovered.(error); ok && errors.Is(err, errExitSentinel) {
				return
			}

			panic(recovered)
		}()

		fn()
	}()

	return code, exited
}

// The helper is load-bearing for every test below, so it gets its own check:
// if it stopped halting execution, those tests would report passes for code
// paths that had actually run past their own exit.
func TestWithExitCaptureHaltsExecution(t *testing.T) {
	reachedAfterExit := false

	code, exited := withExitCapture(t, func() {
		exitProcess(3)

		reachedAfterExit = true
	})

	if !exited {
		t.Fatal("exited = false, want true")
	}

	if code != 3 {
		t.Errorf("code = %d, want 3", code)
	}

	if reachedAfterExit {
		t.Error("execution continued past exitProcess; the substitute must not return")
	}
}

func TestWithExitCaptureReportsNoExitForASuccessfulCall(t *testing.T) {
	if _, exited := withExitCapture(t, func() {}); exited {
		t.Error("exited = true for a function that never exits")
	}
}

// writeFile creates a file with the given contents and returns its path.
func writeFile(t *testing.T, dir, name, contents string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}

	return path
}

// minimalConfig is the smallest document config.Load accepts, used where a test
// needs a valid input and cares only about what happens to the output file.
const minimalConfig = `devices:
  - name: SW1
    type: switch
`
