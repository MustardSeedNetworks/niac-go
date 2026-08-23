package protocols_test

import (
	"net"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

func mustMAC(s string) net.HardwareAddr {
	mac, err := net.ParseMAC(s)
	if err != nil {
		panic(err)
	}
	return mac
}

// TestStackConfigTracksReload guards D13.
//
// daemon.simulationStatus reported DeviceCount from Simulation.cfg, which is
// captured once at session start and never reassigned — the daemon has no
// replace-config-on-a-running-session API. Device mutations go through the API
// layer's own state, so after deleting a device and reloading,
// /api/v1/simulation reported the start-time count while /runtime, /stats,
// /sessions, /config/devices and /sessions/{id}/devices all reported the live
// one. The Dashboard rendered both numbers at once and survived a page reload.
//
// The running stack *does* track the live config (ReloadConfig ->
// initializeDevices -> s.config = cfg), so Stack.Config() is the honest source.
// This asserts that, so the accessor simulationStatus now depends on cannot
// silently go stale again.
func TestStackConfigTracksReload(t *testing.T) {
	initial := &config.Config{
		Devices: []config.Device{
			{Name: "sw-1", Type: "switch", MACAddress: mustMAC("00:00:0c:00:00:01")},
			{Name: "sw-2", Type: "switch", MACAddress: mustMAC("00:00:0c:00:00:02")},
		},
	}
	stack := protocols.NewStack(nil, initial, logging.NewDebugConfig(0))

	if got := stack.Config().DeviceCount(); got != 2 {
		t.Fatalf("initial DeviceCount = %d, want 2", got)
	}

	// Drop a device, exactly as a delete + save & reload does.
	replacement := &config.Config{
		Devices: []config.Device{
			{Name: "sw-1", Type: "switch", MACAddress: mustMAC("00:00:0c:00:00:01")},
		},
	}
	if err := stack.ReloadConfig(replacement); err != nil {
		t.Fatalf("ReloadConfig() error = %v", err)
	}

	if got := stack.Config().DeviceCount(); got != 1 {
		t.Errorf("DeviceCount after reload = %d, want 1 — the stack is serving a stale config", got)
	}
}

// TestStackConfigNilStackIsSafe keeps the accessor usable from status paths
// that may run before a stack exists.
func TestStackConfigNilStackIsSafe(t *testing.T) {
	var stack *protocols.Stack
	if cfg := stack.Config(); cfg != nil {
		t.Errorf("Config() on a nil stack = %v, want nil", cfg)
	}
}
