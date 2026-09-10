package protocols

import (
	"errors"
	"fmt"
	"net"
	"slices"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

func checkpointTestDevice(name string, host byte) config.Device {
	device := faultTestDevice(name)
	device.IPAddresses = []net.IP{{192, 0, 2, host}}
	device.Interfaces[0].Address = fmt.Sprintf("192.0.2.%d/24", host)
	return device
}

func newCheckpointTestStack(t *testing.T) *Stack {
	t.Helper()
	cfg := &config.Config{Devices: []config.Device{
		checkpointTestDevice("edge-1", 1),
		checkpointTestDevice("edge-2", 2),
	}}
	return NewStack(nil, cfg, logging.NewDebugConfig(0))
}

// A scenario checkpoint has to cover the whole stack: an acceptance run that
// saved one device and mutated another would reset to a state that never
// existed.
func TestStackCheckpointCoversEveryDevice(t *testing.T) {
	stack := newCheckpointTestStack(t)

	saved := stack.SaveCheckpoint("healthy")
	if saved != 2 {
		t.Fatalf("SaveCheckpoint() = %d devices, want 2", saved)
	}

	for _, device := range []string{"edge-1", "edge-2"} {
		if err := stack.SetDeviceFault(device, devicestate.FaultLatency, 1); err != nil {
			t.Fatalf("SetDeviceFault(%s) error = %v", device, err)
		}
	}
	if len(stack.ActiveDeviceFaults()) != 2 {
		t.Fatalf("expected both devices faulted, got %#v", stack.ActiveDeviceFaults())
	}

	if err := stack.RestoreCheckpoint("healthy"); err != nil {
		t.Fatalf("RestoreCheckpoint() error = %v", err)
	}
	if active := stack.ActiveDeviceFaults(); len(active) != 0 {
		t.Fatalf("restore left faults active: %#v", active)
	}
}

// Restoring a name nothing saved must fail rather than silently do nothing:
// a harness that mistypes a checkpoint would otherwise assert against a
// scenario it never reset.
func TestStackRestoreUnknownCheckpointFails(t *testing.T) {
	stack := newCheckpointTestStack(t)

	err := stack.RestoreCheckpoint("never-saved")
	if !errors.Is(err, devicestate.ErrCheckpointNotFound) {
		t.Fatalf("error = %v, want %v", err, devicestate.ErrCheckpointNotFound)
	}
}

// The names a caller can restore have to be discoverable, or a harness has to
// remember state the daemon already holds.
func TestStackCheckpointNamesAreListed(t *testing.T) {
	stack := newCheckpointTestStack(t)
	stack.SaveCheckpoint("healthy")
	stack.SaveCheckpoint("degraded")

	names := stack.CheckpointNames()
	slices.Sort(names)
	if !slices.Equal(names, []string{"degraded", "healthy"}) {
		t.Fatalf("CheckpointNames() = %v, want [degraded healthy]", names)
	}
}
