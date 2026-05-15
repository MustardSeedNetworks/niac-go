package daemon

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/gopacket/gopacket"

	"github.com/krisarmstrong/niac-go/internal/api"
	"github.com/krisarmstrong/niac-go/internal/capture"
	"github.com/krisarmstrong/niac-go/internal/logging"
)

// standaloneCapture is the runtime state for a /api/v1/capture session.
// Distinct from the Simulation type because no protocols.Stack is
// involved — the engine runs in promiscuous-read mode and ships each
// packet straight to the SSE hub. The atomic packet counter lets the
// status endpoint report progress without contending on a mutex on the
// hot path.
type standaloneCapture struct {
	iface     string
	filter    string
	startedAt time.Time
	engine    *capture.Engine
	cancel    context.CancelFunc
	packets   atomic.Uint64
}

// StartCapture spins up a capture session on req.Interface, optionally
// applies a BPF filter, and broadcasts every observed packet on the
// SSE packets stream. Returns one of the api.ErrCapture* sentinels for
// the typed-rejection paths (already running, sim active, no such
// interface) so the API handler can map to a precise HTTP status.
func (d *Daemon) StartCapture(req api.CaptureRequest) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.simulation != nil {
		return api.ErrCaptureConflictsWithSim
	}
	if d.capture != nil {
		return api.ErrCaptureAlreadyRunning
	}

	if !capture.InterfaceExists(req.Interface) {
		return fmt.Errorf("%w: %s", api.ErrCaptureInterfaceNotFound, req.Interface)
	}

	engine, err := capture.New(req.Interface, DefaultDebugLevel)
	if err != nil {
		return fmt.Errorf("create capture engine: %w", err)
	}

	if req.Filter != "" {
		if filterErr := engine.SetFilter(req.Filter); filterErr != nil {
			engine.Close()
			return fmt.Errorf("apply BPF filter %q: %w", req.Filter, filterErr)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	session := &standaloneCapture{
		iface:     req.Interface,
		filter:    req.Filter,
		startedAt: time.Now(),
		engine:    engine,
		cancel:    cancel,
	}

	// Drive the capture in its own goroutine so the API call returns
	// promptly. StartCaptureContext blocks until ctx is cancelled or
	// the engine errors out, and invokes our handler synchronously
	// for every packet — which is fine because the handler is just a
	// counter bump + non-blocking SSE broadcast.
	go d.runStandaloneCapture(ctx, session)

	d.capture = session
	logging.Successf("✓ Standalone capture started on %s", req.Interface)
	return nil
}

// runStandaloneCapture is the engine.StartCaptureContext driver loop.
// Lives on its own goroutine; exits when ctx is cancelled (StopCapture
// or daemon shutdown) or when the engine surfaces an unrecoverable
// error.
func (d *Daemon) runStandaloneCapture(ctx context.Context, c *standaloneCapture) {
	handler := func(pkt gopacket.Packet) {
		c.packets.Add(1)
		// Broadcast non-blocking: the SSE hub drops on overflow.
		if d.apiServer != nil {
			d.apiServer.BroadcastCapturePacket(pkt)
		}
	}

	if err := c.engine.StartCaptureContext(ctx, handler); err != nil && ctx.Err() == nil {
		logging.Errorf("standalone capture loop exited with error: %v", err)
	}
}

// StopCapture cancels the active standalone capture (if any) and closes
// its engine. Idempotent — calling with no active capture returns nil.
func (d *Daemon) StopCapture() error {
	d.mu.Lock()
	session := d.capture
	d.capture = nil
	d.mu.Unlock()

	if session == nil {
		return nil
	}

	session.cancel()
	session.engine.Close()
	logging.Successf("✓ Standalone capture stopped on %s", session.iface)
	return nil
}

// GetCaptureStatus returns a snapshot of the capture session. Safe to
// call concurrently with the capture loop — the packet counter is
// atomic and the snapshot fields are read under the daemon mutex.
func (d *Daemon) GetCaptureStatus() api.CaptureStatus {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.capture == nil {
		return api.CaptureStatus{Running: false}
	}
	return api.CaptureStatus{
		Running:   true,
		Interface: d.capture.iface,
		Filter:    d.capture.filter,
		StartedAt: d.capture.startedAt.Format(time.RFC3339),
		Packets:   d.capture.packets.Load(),
	}
}
