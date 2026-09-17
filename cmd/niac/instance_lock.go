package main

import (
	"errors"
	"fmt"
	"net"
	"strconv"

	"github.com/MustardSeedNetworks/foundation/pkg/instance"

	"github.com/MustardSeedNetworks/niac-go/internal/daemon"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

// Single-instance guard.
//
// Nothing stopped two NIAC runtimes from starting side by side. The port is no
// guard: the API listener walks +1..+9 when its port is busy (#69), so a
// second start binds a neighbour and then opens the same SQLite database, the
// same library and the same recovery record underneath the first. `--once`
// reached that state by default, because it defaults `--storage` to the very
// file the daemon writes.
//
// The lock is keyed on the data directory rather than the port or the database
// path, because the data is what must not be shared, and because a key both
// processes derive the same way cannot disagree.

// acquireInstanceLock takes the single-instance lock on NIAC's data directory.
// It returns a *instance.HeldError naming the holder's pid and port when
// another runtime already has it.
func acquireInstanceLock() (*instance.Lock, error) {
	return instance.Acquire(daemon.DefaultDataDir())
}

// onceInstanceLock is acquireInstanceLock for a one-shot run, which reports a
// running daemon as a config refusal (exit 2) rather than a runtime failure: a
// script that cannot tell "I will not do that" from "it crashed" retries the
// unretryable. The refusal names the command that does work against a live
// daemon.
func onceInstanceLock() (*instance.Lock, error) {
	lock, err := acquireInstanceLock()
	if err == nil {
		return lock, nil
	}

	if _, held := errors.AsType[*instance.HeldError](err); held {
		return nil, withExitCode(onceExitConfig,
			fmt.Errorf("%w: a daemon is running, use `niac simulation start` to run against it", err))
	}
	return nil, withExitCode(onceExitRuntime, err)
}

// publishBoundPort records the port the listener settled on in the lock file,
// so the next start names where the holder actually answers rather than where
// it was asked to. A record that cannot be written is not worth failing a
// started daemon over: the lock itself is held either way, and the port is
// only ever a courtesy to the process that loses the race.
func publishBoundPort(lock *instance.Lock, boundAddr string) {
	_, port, err := net.SplitHostPort(boundAddr)
	if err != nil {
		return
	}
	number, err := strconv.Atoi(port)
	if err != nil {
		return
	}
	if setErr := lock.SetPort(number); setErr != nil {
		logging.Warningf("could not record the listening port in the instance lock: %v", setErr)
	}
}
