package replay_test

import (
	"errors"
	"testing"

	"github.com/krisarmstrong/niac-go/internal/api"
	"github.com/krisarmstrong/niac-go/internal/replay"
)

// TestStart_NoEngine verifies a controller without a capture engine refuses
// to Start with the public sentinel rather than panicking. This is the
// failure mode the daemon hits when StartSimulation is called before the
// engine attaches.
func TestStart_NoEngine(t *testing.T) {
	c := replay.New(nil, 0)

	state, err := c.Start(api.ReplayRequest{File: "/tmp/example.pcap"})
	if !errors.Is(err, replay.ErrEngineUnavailable) {
		t.Fatalf("got %v, want ErrEngineUnavailable", err)
	}
	if state.Running {
		t.Errorf("state should not be Running after a failed Start, got %+v", state)
	}
}

// TestStop_FreshController is the path the daemon's shutdown sequence relies
// on — Stop on a never-started controller is a no-op and must never error.
func TestStop_FreshController(t *testing.T) {
	c := replay.New(nil, 0)

	state, err := c.Stop()
	if err != nil {
		t.Errorf("Stop on fresh controller should not error, got %v", err)
	}
	if state.Running {
		t.Errorf("state.Running should be false after Stop on fresh controller, got %+v", state)
	}
}

// TestStatus_Default returns the zero ReplayState before Start.
func TestStatus_Default(t *testing.T) {
	c := replay.New(nil, 0)
	if c.Status().Running {
		t.Errorf("fresh controller should report Running=false")
	}
}
