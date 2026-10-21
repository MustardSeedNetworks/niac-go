package protocols

import (
	"bytes"
	"log/slog"
	"net"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

// panickingTransport idles until armed, then panics on its next panics reads,
// delivers frame once, and idles again. Arming after Start returns keeps the
// panics out of the supervisor's launch, which foundation v0.7.4 does not
// survive (foundation#164).
type panickingTransport struct {
	armed     atomic.Bool
	panics    atomic.Int64
	frame     []byte
	delivered atomic.Bool
}

func (p *panickingTransport) ReadPacket([]byte) ([]byte, error) {
	if !p.armed.Load() {
		time.Sleep(time.Millisecond)
		return nil, errIdleTransport
	}
	if p.panics.Add(-1) >= 0 {
		panic("malformed frame")
	}
	if p.frame != nil && p.delivered.CompareAndSwap(false, true) {
		return bytes.Clone(p.frame), nil
	}
	time.Sleep(time.Millisecond)
	return nil, errIdleTransport
}
func (*panickingTransport) SendPacket([]byte) error { return nil }
func (*panickingTransport) SetFilter(string) error  { return nil }
func (*panickingTransport) Filter() string          { return "" }

type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// captureDefaultLog routes slog.Default, which the stack's supervisor logs
// through, into a buffer for the rest of the test. Tests using it must not run
// in parallel.
func captureDefaultLog(t *testing.T) *lockedBuffer {
	t.Helper()
	logs := &lockedBuffer{}
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return logs
}

func supervisedStack(t *testing.T, transport PacketTransport) *Stack {
	t.Helper()
	cfg := &config.Config{Devices: []config.Device{{
		Name: "sw1", Type: "switch", MACAddress: deviceMAC(),
		IPAddresses: []net.IP{net.ParseIP(testGatewayI)},
	}}}
	stack := NewStackWithTransport(transport, cfg, logging.NewDebugConfig(0))
	if err := stack.Start(); err != nil {
		t.Fatalf("Start() = %v", err)
	}
	return stack
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestStackRestartsAPanickingWorkerAndKeepsServing(t *testing.T) {
	logs := captureDefaultLog(t)
	transport := &panickingTransport{frame: arpRequestFrame(clientMAC()).Buffer}
	transport.panics.Store(1)
	stack := supervisedStack(t, transport)
	defer stack.Stop()
	transport.armed.Store(true)

	waitFor(t, "the frame read after the panic to be decoded", func() bool {
		return len(stack.GetObservedClients()) == 1
	})
	if failure := stack.WorkerFailure(); failure != "" {
		t.Errorf("WorkerFailure() = %q after one recovered panic, want none", failure)
	}
	if got := logs.String(); !strings.Contains(got, "supervised worker failed, restarting") ||
		!strings.Contains(got, "worker=stack-receive") {
		t.Errorf("log does not report the restarted worker:\n%s", got)
	}
}

func TestStackReportsAWorkerThatExhaustsItsRestarts(t *testing.T) {
	logs := captureDefaultLog(t)
	transport := &panickingTransport{}
	transport.panics.Store(1 << 40)
	stack := supervisedStack(t, transport)
	transport.armed.Store(true)

	waitFor(t, "the worker group to fail", func() bool { return stack.WorkerFailure() != "" })
	if failure := stack.WorkerFailure(); !strings.Contains(failure, `"stack-receive" panicked`) {
		t.Errorf("WorkerFailure() = %q, want the receive worker's panic", failure)
	}
	if got := logs.String(); !strings.Contains(got, "supervised worker failed, stopping") {
		t.Errorf("log does not report the stopped group:\n%s", got)
	}

	stopped := make(chan struct{})
	go func() {
		stack.Stop()
		close(stopped)
	}()
	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		t.Fatal("Stop() did not return after the worker group failed")
	}
}

func TestStackStopLeavesNoGoroutines(t *testing.T) {
	before := runtime.NumGoroutine()
	stack := supervisedStack(t, idleTransport{})
	stack.Stop()

	// The discovery handlers' advertisement loops exit asynchronously once
	// Stop closes their channels, so allow them a moment.
	waitFor(t, "every stack goroutine to exit", func() bool { return runtime.NumGoroutine() <= before })
	if failure := stack.WorkerFailure(); failure != "" {
		t.Errorf("WorkerFailure() = %q after a clean stop, want none", failure)
	}
}
