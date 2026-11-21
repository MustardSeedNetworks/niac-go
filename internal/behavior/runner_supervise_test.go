package behavior_test

import (
	"bytes"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/behavior"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

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

// captureDefaultLog routes slog.Default, which the runner's supervisor logs
// through, into a buffer. Tests using it must not run in parallel.
func captureDefaultLog(t *testing.T) *lockedBuffer {
	t.Helper()
	logs := &lockedBuffer{}
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return logs
}

type panickingTarget struct{ recordingTarget }

func (*panickingTarget) SetInterfaceFault(string, string, devicestate.FaultType, int) error {
	panic("target fault")
}

// blockingTarget holds the transition inside the target until released.
type blockingTarget struct {
	recordingTarget

	entered chan struct{}
	release chan struct{}
}

func (b *blockingTarget) SetInterfaceFault(string, string, devicestate.FaultType, int) error {
	close(b.entered)
	<-b.release
	return nil
}

func linkDownTransition() []behavior.Transition {
	return []behavior.Transition{{
		Actions: []behavior.Action{{
			Device: "LAB-EDGE-R1", Interface: "Gi0/1", Type: devicestate.FaultLinkDown, Value: 1,
		}},
	}}
}

func TestRunnerRecoversAPanickingTarget(t *testing.T) {
	logs := captureDefaultLog(t)
	runner := behavior.New(&panickingTarget{}, linkDownTransition())
	t.Cleanup(runner.Stop)
	runner.Start()

	deadline := time.Now().Add(5 * time.Second)
	for runner.Status().State == "running" && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	status := runner.Status()
	if status.State != "failed" || !strings.Contains(status.LastError, "panicked") {
		t.Fatalf("Status() = %+v, want failed with a panic error", status)
	}
	// The supervisor logs after the runner records the failure.
	for !strings.Contains(logs.String(), "supervised worker failed, stopping") && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if got := logs.String(); !strings.Contains(got, "worker=behavior-timeline") ||
		!strings.Contains(got, "supervised worker failed, stopping") {
		t.Fatalf("panic was not reported by the supervisor: %q", got)
	}
}

func TestRunnerStopWaitsForTheWorker(t *testing.T) {
	target := &blockingTarget{entered: make(chan struct{}), release: make(chan struct{})}
	runner := behavior.New(target, linkDownTransition())
	runner.Start()
	<-target.entered

	stopped := make(chan struct{})
	go func() {
		runner.Stop()
		close(stopped)
	}()
	select {
	case <-stopped:
		t.Fatal("Stop returned while the worker was still applying a transition")
	case <-time.After(50 * time.Millisecond):
	}
	close(target.release)
	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		t.Fatal("Stop did not return after the worker finished")
	}
}
