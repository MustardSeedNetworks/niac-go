package protocols

import (
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

type lifecycleCapture struct{}

func (lifecycleCapture) ReadPacket([]byte) ([]byte, error) {
	return nil, errors.New("no packet")
}

func (lifecycleCapture) SendPacket([]byte) error { return nil }
func (lifecycleCapture) SetFilter(string) error  { return nil }
func (lifecycleCapture) Filter() string          { return "" }

func TestStackRejectsSendAfterStop(t *testing.T) {
	stack := newStack(lifecycleCapture{}, &config.Config{}, logging.NewDebugConfig(0))
	if err := stack.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	stack.Stop()

	if stack.Send(NewPacket(64)) {
		t.Fatal("Send() accepted a packet after Stop()")
	}
	if err := stack.SendRawPacket([]byte{0x00}); !errors.Is(err, ErrStackStopped) {
		t.Fatalf("SendRawPacket() error = %v, want %v", err, ErrStackStopped)
	}
	if got := stack.GetStats().RejectedSends; got != 2 {
		t.Fatalf("RejectedSends = %d, want 2", got)
	}
	select {
	case packet := <-stack.sendQueue:
		t.Fatalf("stopped stack queued packet %#v", packet)
	default:
	}
}

func TestStackIsSingleUseAfterStop(t *testing.T) {
	stack := newStack(lifecycleCapture{}, &config.Config{}, logging.NewDebugConfig(0))
	if err := stack.Start(); err != nil {
		t.Fatalf("first Start() error = %v", err)
	}
	stack.Stop()

	if err := stack.Start(); !errors.Is(err, ErrStackStopped) {
		t.Fatalf("second Start() error = %v, want %v", err, ErrStackStopped)
	}
}

func TestStackRejectsSecondStartWhileRunning(t *testing.T) {
	stack := newStack(lifecycleCapture{}, &config.Config{}, logging.NewDebugConfig(0))
	if err := stack.Start(); err != nil {
		t.Fatalf("first Start() error = %v", err)
	}
	defer stack.Stop()

	if err := stack.Start(); !errors.Is(err, ErrStackAlreadyRunning) {
		t.Fatalf("second Start() error = %v, want %v", err, ErrStackAlreadyRunning)
	}
}
