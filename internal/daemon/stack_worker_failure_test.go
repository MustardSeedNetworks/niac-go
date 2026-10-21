package daemon

import (
	"errors"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

// faultingTransport idles until armed, then panics on every read.
type faultingTransport struct{ armed atomic.Bool }

func (f *faultingTransport) ReadPacket([]byte) ([]byte, error) {
	if f.armed.Load() {
		panic("malformed frame")
	}
	time.Sleep(time.Millisecond)
	return nil, errors.New("idle")
}
func (*faultingTransport) SendPacket([]byte) error { return nil }
func (*faultingTransport) SetFilter(string) error  { return nil }
func (*faultingTransport) Filter() string          { return "" }

func TestSessionWithFailedStackWorkersIsDegraded(t *testing.T) {
	cfg := &config.Config{Devices: []config.Device{{
		Name: "sw1", Type: "switch",
		MACAddress:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		IPAddresses: []net.IP{net.ParseIP("10.0.0.1")},
	}}}
	transport := &faultingTransport{}
	stack := protocols.NewStackWithTransport(transport, cfg, logging.NewDebugConfig(0))
	if err := stack.Start(); err != nil {
		t.Fatalf("Start() = %v", err)
	}
	defer stack.Stop()
	d := &Daemon{}
	sim := &Simulation{SessionID: "s1", stack: stack, cfg: cfg, StartedAt: time.Now()}

	if status := d.simulationStatusLocked(sim, true); status.Degraded {
		t.Fatalf("healthy session reported degraded: %q", status.DegradedReason)
	}

	// Armed after Start returns: foundation v0.7.4 does not survive a worker
	// failing while the supervisor is still launching (foundation#164).
	transport.armed.Store(true)
	deadline := time.Now().Add(5 * time.Second)
	for stack.WorkerFailure() == "" {
		if time.Now().After(deadline) {
			t.Fatal("stack workers never failed")
		}
		time.Sleep(5 * time.Millisecond)
	}

	status := d.simulationStatusLocked(sim, true)
	if !status.Degraded || !strings.Contains(status.DegradedReason, "stack-receive") {
		t.Errorf("status = degraded %v reason %q, want degraded naming the receive worker",
			status.Degraded, status.DegradedReason)
	}
	if !status.Running {
		t.Error("status.Running = false, want the session still listed as running")
	}
}
