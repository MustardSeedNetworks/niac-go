// Package devicestate owns mutable state shared by a simulated device's protocols.
package devicestate

import (
	"errors"
	"sync"
	"time"
)

// ErrConcurrentUpdate indicates that state changed while an update callback ran.
var ErrConcurrentUpdate = errors.New("device state changed during update")

// Identity contains administrative identity shared by management protocols.
type Identity struct {
	Hostname string
}

// Snapshot is an immutable point-in-time view of device state.
type Snapshot struct {
	Identity Identity
	Network  Network
	Faults   []InterfaceFault
	Version  uint64
}

type configuration struct {
	identity Identity
	network  Network
}

// Store serializes device-state reads and transactions.
type Store struct {
	mu             sync.RWMutex
	running        configuration
	startup        configuration
	authored       configuration
	version        uint64
	events         []Event
	checkpoints    map[string]checkpoint
	faults         map[interfaceFaultKey]InterfaceFault
	changes        chan struct{}
	changeSignal   chan<- struct{}
	changeObserver func(Snapshot)
	now            func() time.Time
}

// NewStore creates a store seeded with authored device identity.
func NewStore(identity Identity) *Store {
	initial := configuration{identity: identity}
	return &Store{
		running: initial, startup: initial, authored: initial, version: 1,
		faults: make(
			map[interfaceFaultKey]InterfaceFault,
		), changes: make(chan struct{}, 1), now: time.Now,
	}
}

// SetChangeObserver installs a synchronous observer for committed state.
// The observer must not call back into the store.
func (s *Store) SetChangeObserver(observer func(Snapshot)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.changeObserver = observer
	if observer != nil {
		observer(s.snapshot(s.running))
	}
}

// SetChangeSignal adds a coalesced notification channel for an external event
// dispatcher. The authoritative event log remains the source of truth.
func (s *Store) SetChangeSignal(signal chan<- struct{}) {
	s.mu.Lock()
	s.changeSignal = signal
	s.mu.Unlock()
}

// Snapshot returns a consistent copy of the current device state.
func (s *Store) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.snapshot(s.running)
}

// StartupSnapshot returns a consistent copy of startup configuration.
func (s *Store) StartupSnapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.snapshot(s.startup)
}

func (s *Store) snapshot(source configuration) Snapshot {
	network := cloneNetwork(source.network)
	// A link-down fault is an outcome, not a counter rate: it changes what the
	// interface *is*. Projecting it here rather than writing through to stored
	// state means clearing the fault restores the interface with no bookkeeping
	// to get wrong, and every consumer sees it because they all read a snapshot.
	applyLinkDownFaults(network.Interfaces, s.faults)
	return Snapshot{
		Identity: source.identity, Network: network,
		Faults: sortedInterfaceFaults(s.faults), Version: s.version,
	}
}

// UpdateIdentity applies one atomic identity transaction.
func (s *Store) UpdateIdentity(update func(Identity) (Identity, error)) error {
	s.mu.RLock()
	current := s.running.identity
	version := s.version
	s.mu.RUnlock()
	next, err := update(current)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.version != version {
		return ErrConcurrentUpdate
	}
	s.running.identity = next
	s.version++
	s.recordEvent(EventIdentityUpdated, next.Hostname)

	return nil
}
