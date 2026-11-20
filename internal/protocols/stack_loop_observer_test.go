package protocols

import (
	"bytes"
	"context"
	"errors"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

type countingTransport struct{ sent atomic.Int64 }

func (*countingTransport) ReadPacket([]byte) ([]byte, error) { return nil, errors.New("no packet") }
func (c *countingTransport) SendPacket([]byte) error         { c.sent.Add(1); return nil }
func (*countingTransport) SetFilter(string) error            { return nil }
func (*countingTransport) Filter() string                    { return "" }

func loopedPacket(every time.Duration) *Packet {
	return &Packet{Buffer: make([]byte, 64), Length: 64, VLAN: -1, LoopTime: every}
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

func TestLoopedPacketResendsAreBoundToStop(t *testing.T) {
	transport := &countingTransport{}
	stack := newStack(transport, &config.Config{}, logging.NewDebugConfig(0))
	// Taken before Start so the stack's own workers count as leaks too if
	// Stop fails to end them, not as slack the resends could hide in.
	baseline := runtime.NumGoroutine()
	if err := stack.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// One packet resends quickly so a resend is in flight when Stop runs; the
	// rest wait an hour, which is how a leaked sleeper stays visible.
	if !stack.Send(loopedPacket(2 * time.Millisecond)) {
		t.Fatal("Send() rejected the fast looped packet")
	}
	for range 20 {
		if !stack.Send(loopedPacket(time.Hour)) {
			t.Fatal("Send() rejected a slow looped packet")
		}
	}
	eventually(t, "the fast packet to resend", func() bool { return transport.sent.Load() >= 5 })

	stack.Stop()
	afterStop := transport.sent.Load()

	// A resend scheduled after Stop is refused rather than parked.
	stack.scheduleResend(loopedPacket(time.Millisecond))
	time.Sleep(30 * time.Millisecond)
	if got := transport.sent.Load(); got != afterStop {
		t.Fatalf("%d packets transmitted after Stop returned", got-afterStop)
	}

	// Stop has returned, so anything still alive is a leak. The count is
	// polled, not read once, only because goroutines that have already
	// finished can take a scheduler tick to disappear from NumGoroutine.
	eventually(t, "goroutines to return to baseline", func() bool {
		return runtime.NumGoroutine() <= baseline
	})
}

func TestSleepOrDoneReturnsWhenContextEnds(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan bool, 1)
	go func() { done <- sleepOrDone(ctx, time.Hour) }()
	cancel()
	select {
	case elapsed := <-done:
		if elapsed {
			t.Fatal("sleepOrDone reported the full delay elapsed after cancellation")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("sleepOrDone ignored cancellation")
	}
	if !sleepOrDone(context.Background(), time.Millisecond) {
		t.Fatal("sleepOrDone did not report an elapsed delay")
	}
}

type countingObserver struct{ got atomic.Int64 }

func (r *countingObserver) OnPacket(string, *Packet) { r.got.Add(1) }

type panickingObserver struct{ calls atomic.Int64 }

func (p *panickingObserver) OnPacket(string, *Packet) {
	p.calls.Add(1)
	panic("observer exploded")
}

// uncomparableObserver cannot be compared with ==; removal must not try.
type uncomparableObserver struct {
	_    []string
	hits *atomic.Int64
}

func (u uncomparableObserver) OnPacket(string, *Packet) {
	u.hits.Add(1)
	panic("slice-backed observer exploded")
}

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func TestPanickingObserverIsLoggedOnceAndRemoved(t *testing.T) {
	var logs syncBuffer
	logging.SetOutput(&logs)
	t.Cleanup(func() { logging.SetOutput(nil) })

	stack := newStack(&countingTransport{}, &config.Config{}, logging.NewDebugConfig(0))
	before, after := &countingObserver{}, &countingObserver{}
	bad := &panickingObserver{}
	var uncomparableHits atomic.Int64
	stack.AddPacketObserver(before)
	stack.AddPacketObserver(bad)
	stack.AddPacketObserver(uncomparableObserver{hits: &uncomparableHits})
	stack.AddPacketObserver(after)

	// Concurrent deliveries can all panic the same observer; it is still
	// logged and removed exactly once.
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() { stack.notifyObservers("rx", &Packet{}) })
	}
	wg.Wait()
	for range 3 {
		stack.notifyObservers("rx", &Packet{})
	}

	if got := before.got.Load(); got != 11 {
		t.Errorf("observer before the panicker received %d events, want 11", got)
	}
	if got := after.got.Load(); got != 11 {
		t.Errorf("observer after the panicker received %d events, want 11", got)
	}
	if bad.calls.Load() < 1 || uncomparableHits.Load() < 1 {
		t.Fatal("panicking observers were never called")
	}
	callsAfterRemoval := bad.calls.Load()
	stack.notifyObservers("tx", &Packet{})
	if bad.calls.Load() != callsAfterRemoval {
		t.Error("panicking observer was called again after removal")
	}

	out := logs.String()
	if got := strings.Count(out, "*protocols.panickingObserver panicked and was removed: observer exploded"); got != 1 {
		t.Errorf("panicking observer logged %d times, want 1\n%s", got, out)
	}
	if got := strings.Count(out, "protocols.uncomparableObserver panicked and was removed"); got != 1 {
		t.Errorf("uncomparable observer logged %d times, want 1\n%s", got, out)
	}
	if !strings.Contains(out, "deliverToObserver") {
		t.Errorf("log carries no stack trace\n%s", out)
	}
}
