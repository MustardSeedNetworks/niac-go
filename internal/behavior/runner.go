package behavior

import (
	"context"
	"sync"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

// Target applies one timeline action to authoritative device state.
type Target interface {
	SetInterfaceFault(string, string, devicestate.FaultType, int) error
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
	now         func() time.Time
}

// New creates an idle runner.
func New(target Target, transitions []Transition) *Runner {
	return &Runner{
		target: target, transitions: append([]Transition(nil), transitions...),
		status: Status{State: "idle", TotalTransitions: len(transitions)}, now: time.Now,
	}
}

// Start begins timeline replay. It is safe to call only once.
func (r *Runner) Start() {
	r.mu.Lock()
	if r.status.State != "idle" {
		r.mu.Unlock()
		return
	}
	if len(r.transitions) == 0 {
		r.status.State = "completed"
		r.status.CompletedAt = r.now().UTC()
		r.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel
	r.done = make(chan struct{})
	r.status.State = "running"
	started := r.now()
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
	for _, transition := range r.transitions {
		if !waitUntil(ctx, started.Add(transition.Offset)) {
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
		r.mu.Lock()
		r.status.AppliedTransitions++
		r.status.ActivePhases = append([]string(nil), transition.Phases...)
		r.mu.Unlock()
	}
	r.finish("completed", "")
}

func (r *Runner) finish(state, lastError string) {
	r.mu.Lock()
	r.status.State = state
	r.status.LastError = lastError
	r.status.ActivePhases = nil
	r.status.CompletedAt = r.now().UTC()
	r.mu.Unlock()
}

func waitUntil(ctx context.Context, deadline time.Time) bool {
	delay := time.Until(deadline)
	if delay <= 0 {
		return true
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}
