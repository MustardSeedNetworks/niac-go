package protocols

import (
	"context"

	"github.com/MustardSeedNetworks/foundation/pkg/supervise"

	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

// stackWorkerRestarts is how many times a faulted stack worker is restarted
// before the stack gives up. Every worker is a loop that carries no state
// between iterations, so one bad frame or one bad device must not cost the
// session its receive path for the rest of its life; a worker that faults
// every time stops after this many tries instead of spinning, and the session
// reports itself degraded.
const stackWorkerRestarts = 3

// Start begins a single-use protocol stack. A stopped stack cannot be restarted.
func (s *Stack) Start() error {
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()

	if s.started {
		if s.stopped {
			return ErrStackStopped
		}
		return ErrStackAlreadyRunning
	}
	if err := s.ValidateBehaviorTargets(); err != nil {
		return err
	}
	s.started = true
	s.running.Store(true)

	// Registration order is dependency order, so Stop unwinds it: the
	// periodic loops stop first and the receive path that feeds decode last.
	workers := s.workers
	restart := supervise.RestartN(stackWorkerRestarts)
	workers.Add("stack-receive", restart, s.receiveLoop)
	workers.Add("stack-decode", restart, s.decodeLoop)
	workers.Add("stack-send", restart, s.sendLoop)
	workers.Add("stack-babble", restart, s.babbleLoop)
	workers.Add("stack-notifications", restart, s.notifications.Run)
	if s.neighbors != nil {
		workers.Add("stack-neighbor-cleanup", restart, s.neighborCleanupLoop)
	}
	if s.tcpHandler != nil && s.tcpHandler.ssh != nil {
		workers.Add("stack-ssh-session-cleanup", restart, s.sessionCleanupLoop)
	}
	workers.Add("stack-stp", restart, s.stpHandler.originate)
	workers.Add("stack-icmpv6-ra", restart, s.icmpv6Handler.originateRouterAdvertisements)
	workers.Add("stack-lldp", restart, s.lldpHandler.advertise)
	workers.Add("stack-cdp", restart, s.cdpHandler.advertise)
	workers.Add("stack-edp", restart, s.edpHandler.advertise)
	workers.Add("stack-fdp", restart, s.fdpHandler.advertise)
	workers.Start(context.Background())
	go s.recordWorkerFailure()

	s.startBehaviorTimelines()

	if s.debugConfig.GetGlobal() >= DebugLevelBasic {
		logging.Debugf("Protocol stack started")
	}

	return nil
}

// recordWorkerFailure keeps the error that stopped the worker group, so the
// session can report it instead of claiming to run with a dead receive path.
// It returns when the group does: on Stop, or when a worker exhausts its
// restarts and the supervisor stops the rest.
func (s *Stack) recordWorkerFailure() {
	if err := s.workers.Wait(); err != nil {
		message := err.Error()
		s.workerFailure.Store(&message)
	}
}

// WorkerFailure reports why the stack's workers stopped while the session was
// still running, or "" while they are healthy. Like Config it accepts a nil
// stack, which a session that never built one has.
func (s *Stack) WorkerFailure() string {
	if s == nil {
		return ""
	}
	if message := s.workerFailure.Load(); message != nil {
		return *message
	}
	return ""
}
