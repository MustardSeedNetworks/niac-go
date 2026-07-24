package daemon

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gopacket/gopacket"

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

func TestStandaloneCaptureUnexpectedExitClearsSessionAndAllowsRestart(t *testing.T) {
	d := newTestDaemon(t)
	d.captureInterfaceExists = func(string) bool { return true }
	var engines []*fakeCaptureEngine
	d.newCaptureEngine = func(string, int) (captureEngine, error) {
		engine := &fakeCaptureEngine{}
		engines = append(engines, engine)
		return engine, nil
	}
	var runs atomic.Int32
	d.captureRunner = func(
		ctx context.Context,
		_ captureEngine,
		_ func(gopacket.Packet),
	) error {
		if runs.Add(1) == 1 {
			return errors.New("packet source closed")
		}
		<-ctx.Done()
		return ctx.Err()
	}

	if err := d.StartCapture(api.CaptureRequest{Interface: "test0"}); err != nil {
		t.Fatalf("first StartCapture() error = %v", err)
	}
	waitForCaptureStatus(t, d, false)

	status := d.GetCaptureStatus()
	if !strings.Contains(status.LastError, "packet source closed") {
		t.Fatalf("LastError = %q, want terminal runner error", status.LastError)
	}
	if got := engines[0].closes.Load(); got != 1 {
		t.Fatalf("failed engine Close() calls = %d, want 1", got)
	}

	if err := d.StartCapture(api.CaptureRequest{Interface: "test0"}); err != nil {
		t.Fatalf("restart StartCapture() error = %v", err)
	}
	waitForCaptureStatus(t, d, true)
	if status = d.GetCaptureStatus(); status.LastError != "" {
		t.Fatalf("LastError after restart = %q, want cleared", status.LastError)
	}
	if err := d.StopCapture(); err != nil {
		t.Fatalf("StopCapture() error = %v", err)
	}
	if got := engines[1].closes.Load(); got != 1 {
		t.Fatalf("replacement engine Close() calls = %d, want 1", got)
	}
}

func TestStandaloneCaptureOldRunnerCannotClearReplacement(t *testing.T) {
	d := newTestDaemon(t)
	d.captureInterfaceExists = func(string) bool { return true }
	d.newCaptureEngine = func(string, int) (captureEngine, error) {
		return &fakeCaptureEngine{}, nil
	}
	oldRelease := make(chan struct{})
	var runs atomic.Int32
	d.captureRunner = func(
		ctx context.Context,
		_ captureEngine,
		_ func(gopacket.Packet),
	) error {
		if runs.Add(1) == 1 {
			<-oldRelease
			return errors.New("old capture failed")
		}
		<-ctx.Done()
		return ctx.Err()
	}

	if err := d.StartCapture(api.CaptureRequest{Interface: "old0"}); err != nil {
		t.Fatalf("first StartCapture() error = %v", err)
	}
	if err := d.StopCapture(); err != nil {
		t.Fatalf("StopCapture() error = %v", err)
	}
	if err := d.StartCapture(api.CaptureRequest{Interface: "new0"}); err != nil {
		t.Fatalf("replacement StartCapture() error = %v", err)
	}
	close(oldRelease)
	waitForCaptureStatus(t, d, true)

	status := d.GetCaptureStatus()
	if status.Interface != "new0" || status.LastError != "" {
		t.Fatalf("replacement status = %#v", status)
	}
	if err := d.StopCapture(); err != nil {
		t.Fatalf("replacement StopCapture() error = %v", err)
	}
}

type fakeCaptureEngine struct {
	closes atomic.Int32
}

func (f *fakeCaptureEngine) SetFilter(string) error { return nil }

func (f *fakeCaptureEngine) StartCaptureContext(context.Context, func(gopacket.Packet)) error {
	return nil
}

func (f *fakeCaptureEngine) Close() {
	f.closes.Add(1)
}

func waitForCaptureStatus(t *testing.T, d *Daemon, running bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if d.GetCaptureStatus().Running == running {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("capture Running did not become %v", running)
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
