package protocols

import (
	"errors"
	"testing"
	"time"

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

func TestStackBehaviorTimelineReplaysAndResets(t *testing.T) {
	device := faultTestDevice("edge-1")
	cfg := &config.Config{
		Devices: []config.Device{device},
		BehaviorTimelines: []config.BehaviorTimeline{{
			Name: "link degradation", RepeatCount: 2,
			Phases: []config.BehaviorPhase{{
				Name: "congested", Duration: 5 * time.Millisecond, Reset: true,
				Traffic: []config.BehaviorTraffic{{
					Device: "edge-1", Interface: "Gi0/1", Utilization: 80,
				}},
			}},
		}},
	}
	stack := newStack(lifecycleCapture{}, cfg, logging.NewDebugConfig(0))
	if err := stack.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer stack.Stop()
	deadline := time.Now().Add(time.Second)
	for stack.BehaviorStatus().State != "completed" && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	status := stack.BehaviorStatus()
	if status.State != "completed" || status.AppliedTransitions != 3 {
		t.Fatalf("BehaviorStatus() = %+v", status)
	}
	if faults := stack.ActiveInterfaceFaults(); len(faults) != 0 {
		t.Fatalf("active faults after reset = %#v", faults)
	}
}
