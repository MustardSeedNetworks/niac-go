package daemon

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gopacket/gopacket"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

type lockedLog struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (l *lockedLog) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buf.Write(p)
}

func (l *lockedLog) contains(substr string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return strings.Contains(l.buf.String(), substr)
}

// captureSupervisorLog routes slog.Default, which the workers' supervisors log
// through, into a buffer. Tests using it must not run in parallel.
func captureSupervisorLog(t *testing.T) *lockedLog {
	t.Helper()
	logs := &lockedLog{}
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return logs
}

func eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(time.Millisecond)
	}
}

func TestStandaloneCapturePanicEndsTheSessionAndIsReported(t *testing.T) {
	logs := captureSupervisorLog(t)
	d := newTestDaemon(t)
	d.captureInterfaceExists = func(string) bool { return true }
	engine := &fakeCaptureEngine{}
	d.newCaptureEngine = func(string, int) (captureEngine, error) { return engine, nil }
	d.captureRunner = func(context.Context, captureEngine, func(gopacket.Packet)) error {
		panic("malformed packet")
	}

	if err := d.StartCapture(api.CaptureRequest{Interface: "test0"}); err != nil {
		t.Fatalf("StartCapture() error = %v", err)
	}
	waitForCaptureStatus(t, d, false)
	if status := d.GetCaptureStatus(); !strings.Contains(status.LastError, "panicked") {
		t.Fatalf("LastError = %q, want the panic", status.LastError)
	}
	if got := engine.closes.Load(); got != 1 {
		t.Fatalf("engine Close() calls = %d, want 1", got)
	}
	eventually(t, "the supervisor to report the panic", func() bool {
		return logs.contains("supervised worker failed, stopping") && logs.contains("worker=standalone-capture")
	})
}

func TestStopCaptureWaitsForTheLoop(t *testing.T) {
	d := newTestDaemon(t)
	d.captureInterfaceExists = func(string) bool { return true }
	d.newCaptureEngine = func(string, int) (captureEngine, error) { return &fakeCaptureEngine{}, nil }
	var exited atomic.Bool
	started := make(chan struct{})
	d.captureRunner = func(ctx context.Context, _ captureEngine, _ func(gopacket.Packet)) error {
		close(started)
		<-ctx.Done()
		time.Sleep(50 * time.Millisecond)
		exited.Store(true)
		return ctx.Err()
	}

	if err := d.StartCapture(api.CaptureRequest{Interface: "test0"}); err != nil {
		t.Fatalf("StartCapture() error = %v", err)
	}
	<-started
	if err := d.StopCapture(); err != nil {
		t.Fatalf("StopCapture() error = %v", err)
	}
	if !exited.Load() {
		t.Fatal("StopCapture returned before the capture loop exited")
	}
}

// trunkPhysical is a physical capture whose loop is supplied by the test.
type trunkPhysical struct {
	fakeTrunkPhysical

	run func(context.Context) error
}

func (p *trunkPhysical) StartCaptureContext(ctx context.Context, _ func(gopacket.Packet)) error {
	return p.run(ctx)
}

func TestTrunkCapturePanicFailsTheTrunkAndIsReported(t *testing.T) {
	logs := captureSupervisorLog(t)
	managed := startManagedTrunkCapture("trunk0", &trunkPhysical{
		run: func(context.Context) error { panic("malformed frame") },
	})
	t.Cleanup(func() { stopWorkers(managed.workers, "test trunk") })

	eventually(t, "the trunk to be marked failed", managed.capture.failed.Load)
	if message := managed.capture.failure.Load(); message == nil || !strings.Contains(*message, "panicked") {
		t.Fatalf("trunk failure = %v, want the panic", message)
	}
	if _, err := managed.capture.register(10); !errors.Is(err, ErrTrunkCaptureFailed) {
		t.Fatalf("register() on a failed trunk = %v, want ErrTrunkCaptureFailed", err)
	}
	eventually(t, "the supervisor to report the panic", func() bool {
		return logs.contains("supervised worker failed, stopping") && logs.contains("worker=trunk-capture")
	})
}

func TestCloseUnusedTrunkWaitsForTheCaptureLoop(t *testing.T) {
	var exited atomic.Bool
	started := make(chan struct{})
	managed := startManagedTrunkCapture("trunk0", &trunkPhysical{
		run: func(ctx context.Context) error {
			close(started)
			<-ctx.Done()
			time.Sleep(50 * time.Millisecond)
			exited.Store(true)
			return ctx.Err()
		},
	})
	<-started
	d := &Daemon{trunks: map[string]*managedTrunkCapture{"trunk0": managed}}

	d.closeUnusedTrunk("trunk0")

	if !exited.Load() {
		t.Fatal("closeUnusedTrunk returned before the capture loop exited")
	}
	if managed.capture.failed.Load() {
		t.Fatal("a deliberate close was recorded as a trunk failure")
	}
}

func TestRuntimeStateWriterRestartsAfterAPanicAndStopWaits(t *testing.T) {
	logs := captureSupervisorLog(t)
	t.Setenv(e2eDryRunEnv, "true")
	t.Setenv("NIAC_CONFIGS_DIR", t.TempDir())
	daemon := recoveryTestDaemon(t, t.TempDir()+"/"+activeSimulationFileName)
	if err := daemon.StartSimulation(api.SimulationRequest{
		Interface: "recovery0", ConfigData: runtimeStateConfig,
	}); err != nil {
		t.Fatalf("StartSimulation() error = %v", err)
	}
	sim := daemon.simulation
	stopRuntimeStateWriter(sim)
	stack := runtimeTestStack(t, daemon)

	var calls, completed atomic.Int32
	sim.runtimeWriter = superviseRuntimeStateWriter(sim.SessionID, stack,
		func(string, *protocols.Stack) error {
			if calls.Add(1) == 1 {
				panic("disk driver fault")
			}
			completed.Add(1)
			return nil
		}, time.Millisecond)

	if err := stack.SetDeviceFault("runtime-switch", devicestate.FaultLatency, 120); err != nil {
		t.Fatalf("SetDeviceFault() error = %v", err)
	}
	eventually(t, "the writer to recover and write", func() bool { return completed.Load() > 0 })
	if !logs.contains("supervised worker failed, restarting") || !logs.contains("worker=runtime-state-writer") {
		t.Fatal("the recovered panic was not reported")
	}

	stopRuntimeStateWriter(sim)
	settled := calls.Load()
	if err := stack.SetDeviceFault("runtime-switch", devicestate.FaultLatency, 250); err != nil {
		t.Fatalf("SetDeviceFault() error = %v", err)
	}
	time.Sleep(20 * time.Millisecond)
	if got := calls.Load(); got != settled {
		t.Fatalf("writer ran %d more times after stop", got-settled)
	}
}
