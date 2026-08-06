package behavior

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

// Target applies one timeline action to authoritative device state.
type Target interface {
	SetInterfaceFault(string, string, devicestate.FaultType, int) error
}

// Clock is the single time seam for timeline replay: both the timestamps a run
// reports and the waiting between transitions. One seam covering both is what
// lets a replay be reproduced deterministically instead of depending on how
// long the wall clock happened to take.
type Clock interface {
	Now() time.Time
	// Wait blocks for d, or until ctx is done. It reports false when the wait
	// was cut short so a cancelled run stops, rather than racing through every
	// remaining transition whose offset has already passed.
	Wait(ctx context.Context, d time.Duration) bool
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now() }

func (systemClock) Wait(ctx context.Context, d time.Duration) bool {
	if ctx.Err() != nil {
		return false
	}
	if d <= 0 {
		return true
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}

// Status is an immutable timeline-run snapshot.
type Status struct {
	State              string    `json:"state"`
	StartedAt          time.Time `json:"startedAt,omitzero"`
	CompletedAt        time.Time `json:"completedAt,omitzero"`
	AppliedTransitions int       `json:"appliedTransitions"`
	TotalTransitions   int       `json:"totalTransitions"`
	ActivePhases       []string  `json:"activePhases,omitempty"`
	LastError          string    `json:"lastError,omitempty"`
}

// Runner replays one compiled transition list once per simulation start.
type Runner struct {
	mu          sync.RWMutex
	target      Target
	transitions []Transition
	status      Status
	cancel      context.CancelFunc
	done        chan struct{}
	clock       Clock
}

// New creates an idle runner driven by the wall clock.
func New(target Target, transitions []Transition) *Runner {
	return NewWithClock(target, transitions, systemClock{})
}

// NewWithClock creates an idle runner whose replay is driven by clock, so a
// caller that needs the same timeline to produce the same result every time
// can supply one it controls.
func NewWithClock(target Target, transitions []Transition, clock Clock) *Runner {
	if clock == nil {
		clock = systemClock{}
	}
	return &Runner{
		target: target, transitions: append([]Transition(nil), transitions...),
		status: Status{State: "idle", TotalTransitions: len(transitions)}, clock: clock,
	}
}

// Start begins timeline replay. A runner replays once; call Reset to run the
// same timeline again.
func (r *Runner) Start() {
	r.mu.Lock()
	if r.status.State != "idle" {
		r.mu.Unlock()
		return
	}
	if len(r.transitions) == 0 {
		r.status.State = "completed"
		r.status.CompletedAt = r.clock.Now().UTC()
		r.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel
	r.done = make(chan struct{})
	r.status.State = "running"
	started := r.clock.Now()
	r.status.StartedAt = started.UTC()
	done := r.done
	r.mu.Unlock()
	go r.run(ctx, started, done)
}

// Stop cancels an active replay and waits for its worker to exit.
func (r *Runner) Stop() {
	r.mu.RLock()
	cancel, done := r.cancel, r.done
	r.mu.RUnlock()
	if cancel == nil || done == nil {
		return
	}
	cancel()
	<-done
}

// Status returns a concurrency-safe snapshot.
func (r *Runner) Status() Status {
	r.mu.RLock()
	defer r.mu.RUnlock()
	status := r.status
	status.ActivePhases = append([]string(nil), status.ActivePhases...)
	return status
}

func (r *Runner) run(ctx context.Context, started time.Time, done chan struct{}) {
	defer close(done)
	activePhases := make(map[string]string)
	for _, transition := range r.transitions {
		if !r.clock.Wait(ctx, transition.Offset-r.clock.Now().Sub(started)) {
			r.finish("stopped", "")
			return
		}
		for _, action := range transition.Actions {
			if err := r.target.SetInterfaceFault(
				action.Device, action.Interface, action.Type, action.Value,
			); err != nil {
				r.finish("failed", err.Error())
				return
			}
		}
		for _, phase := range transition.EndPhases {
			delete(activePhases, phase.ID)
		}
		for _, phase := range transition.StartPhases {
			activePhases[phase.ID] = phase.Label
		}
		phases := make([]string, 0, len(activePhases))
		for _, label := range activePhases {
			phases = append(phases, label)
		}
		slices.Sort(phases)
		r.mu.Lock()
		r.status.AppliedTransitions++
		r.status.ActivePhases = phases
		r.mu.Unlock()
	}
	r.finish("completed", "")
}

func (r *Runner) finish(state, lastError string) {
	r.mu.Lock()
	r.status.State = state
	r.status.LastError = lastError
	r.status.ActivePhases = nil
	r.status.CompletedAt = r.clock.Now().UTC()
	r.mu.Unlock()
}

// Reset stops any active replay and returns the runner to idle so the same
// timeline can be run again. Without it a runner was single-use, which made
// "run this scenario again from the top" mean rebuilding it.
func (r *Runner) Reset() {
	r.Stop()
	r.mu.Lock()
	defer r.mu.Unlock()
	r.status = Status{State: "idle", TotalTransitions: len(r.transitions)}
	r.cancel, r.done = nil, nil
}
