package daemon

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
)

// TestStartCapture_RejectsWhenSimulationRunning makes sure the daemon
// won't try to open the same interface twice. The simulation side of
// the lifecycle already owns the libpcap handle for the configured
// interface; spinning up a standalone capture on top would either
// fight for the handle or, worse, succeed and silently duplicate the
// sniff loop.
func TestStartCapture_RejectsWhenSimulationRunning(t *testing.T) {
	d := newTestDaemon(t)
	d.simulation = &Simulation{Interface: "lo0"}

	err := d.StartCapture(api.CaptureRequest{Interface: "lo0"})
	if err == nil {
		t.Fatal("expected error when starting capture with a sim running, got nil")
	}
	if !errors.Is(err, api.ErrCaptureConflictsWithSim) {
		t.Errorf("expected ErrCaptureConflictsWithSim, got: %v", err)
	}
}

// TestStartCapture_RejectsWhenAlreadyRunning covers the idempotency
// guard. A second POST should fail clean instead of leaking the old
// engine.
func TestStartCapture_RejectsWhenAlreadyRunning(t *testing.T) {
	d := newTestDaemon(t)
	d.capture = &standaloneCapture{
		iface:     "lo0",
		startedAt: time.Now(),
		cancel:    func() {},
	}

	err := d.StartCapture(api.CaptureRequest{Interface: "lo0"})
	if !errors.Is(err, api.ErrCaptureAlreadyRunning) {
		t.Errorf("expected ErrCaptureAlreadyRunning, got: %v", err)
	}
}

// TestStartCapture_RejectsUnknownInterface verifies the typed sentinel
// makes it through to the API handler so HTTP status mapping (404) can
// happen.
func TestStartCapture_RejectsUnknownInterface(t *testing.T) {
	d := newTestDaemon(t)

	err := d.StartCapture(api.CaptureRequest{Interface: "this-iface-does-not-exist-xyz"})
	if !errors.Is(err, api.ErrCaptureInterfaceNotFound) {
		t.Errorf("expected ErrCaptureInterfaceNotFound, got: %v", err)
	}
}

// TestStopCapture_NoActiveCaptureIsNoOp documents that the API
// handler can safely call Stop on a daemon with no live capture.
// Useful for the idempotent shutdown path.
func TestStopCapture_NoActiveCaptureIsNoOp(t *testing.T) {
	d := newTestDaemon(t)

	if err := d.StopCapture(); err != nil {
		t.Errorf("StopCapture with no active capture should return nil, got: %v", err)
	}
}

// TestGetCaptureStatus_ReportsRunningSession exercises the snapshot
// shape that the GET handler ships to the UI.
func TestGetCaptureStatus_ReportsRunningSession(t *testing.T) {
	d := newTestDaemon(t)
	started := time.Now().Add(-30 * time.Second)
	d.capture = &standaloneCapture{
		iface:     "eth0",
		filter:    "tcp port 80",
		startedAt: started,
		cancel:    func() {},
	}
	d.capture.packets.Store(42)

	status := d.GetCaptureStatus()
	if !status.Running {
		t.Error("Running = false, want true")
	}
	if status.Interface != "eth0" {
		t.Errorf("Interface = %q, want eth0", status.Interface)
	}
	if status.Filter != "tcp port 80" {
		t.Errorf("Filter = %q, want tcp port 80", status.Filter)
	}
	if status.Packets != 42 {
		t.Errorf("Packets = %d, want 42", status.Packets)
	}
	if status.StartedAt == "" {
		t.Error("StartedAt is empty; expected RFC3339 timestamp")
	}
}

// TestGetCaptureStatus_NoSession returns an empty Running=false status.
func TestGetCaptureStatus_NoSession(t *testing.T) {
	d := newTestDaemon(t)
	status := d.GetCaptureStatus()
	if status.Running {
		t.Error("Running = true with no active capture")
	}
	if status.Packets != 0 {
		t.Errorf("Packets = %d, want 0", status.Packets)
	}
}

// newTestDaemon builds a Daemon with the minimum scaffolding needed to
// exercise StartCapture / StopCapture / GetCaptureStatus. Storage is
// disabled and there's no API server attached — these tests only care
// about the daemon-internal state transitions and the typed sentinel
// errors that the API handler maps to HTTP statuses.
func newTestDaemon(t *testing.T) *Daemon {
	t.Helper()
	d, err := NewDaemon(Config{StoragePath: "disabled"})
	if err != nil {
		t.Fatalf("NewDaemon: %v", err)
	}
	return d
}

// silence the unused-import warning if a future cleanup drops context.
var _ = context.Background
