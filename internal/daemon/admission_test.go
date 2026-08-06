package daemon

import (
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

func configWithDevices(count int) *config.Config {
	cfg := &config.Config{Devices: make([]config.Device, count)}
	for index := range cfg.Devices {
		cfg.Devices[index].Name = "device"
	}
	return cfg
}

func daemonWithSessions(t *testing.T, deviceCounts map[string]int) *Daemon {
	t.Helper()
	d := &Daemon{sessions: newSessionRegistry()}
	for id, count := range deviceCounts {
		d.sessions.sessions[id] = &Simulation{SessionID: id, cfg: configWithDevices(count)}
	}
	return d
}

func TestAdmissionSumsDevicesAcrossSessions(t *testing.T) {
	// The absolute ceiling is a daemon budget. Checking it per config would let
	// each session carry its own full allowance.
	d := daemonWithSessions(t, map[string]int{"hospital": api.MaxDeviceCount - 10})

	if err := d.admitSessionLocked("warehouse", configWithDevices(10)); err != nil {
		t.Errorf("start that exactly fills the budget was refused: %v", err)
	}
	err := d.admitSessionLocked("warehouse", configWithDevices(11))
	if !errors.Is(err, api.ErrSimulationDeviceCapacity) {
		t.Errorf("error = %v, want ErrSimulationDeviceCapacity for a start that overruns the total", err)
	}
}

func TestAdmissionExcludesTheSessionBeingReplaced(t *testing.T) {
	// Restarting a session swaps it rather than adding one, so its own devices
	// must not be counted against it.
	d := daemonWithSessions(t, map[string]int{"hospital": api.MaxDeviceCount})

	if err := d.admitSessionLocked("hospital", configWithDevices(api.MaxDeviceCount)); err != nil {
		t.Errorf("replacing a session with the same size was refused: %v", err)
	}
	if err := d.admitSessionLocked("warehouse", configWithDevices(1)); !errors.Is(
		err, api.ErrSimulationDeviceCapacity,
	) {
		t.Errorf("error = %v, want the budget to be full for a different session", err)
	}
}

func TestAdmissionBoundsConcurrentSessionCount(t *testing.T) {
	counts := make(map[string]int, maxActiveSessions)
	for index := range maxActiveSessions {
		counts[string(rune('a'+index))] = 1
	}
	d := daemonWithSessions(t, counts)

	err := d.admitSessionLocked("one-too-many", configWithDevices(1))
	if !errors.Is(err, api.ErrSimulationSessionCapacity) {
		t.Errorf("error = %v, want ErrSimulationSessionCapacity", err)
	}
	// Replacing one of the existing sessions is still allowed at the cap.
	if err = d.admitSessionLocked("a", configWithDevices(1)); err != nil {
		t.Errorf("replacing an existing session at the cap was refused: %v", err)
	}
}

func TestAggregateUsageReportsBudgets(t *testing.T) {
	d := daemonWithSessions(t, map[string]int{"hospital": 75, "warehouse": 57})
	usage := d.aggregateUsageLocked()

	if usage.Sessions != 2 {
		t.Errorf("sessions = %d, want 2", usage.Sessions)
	}
	if usage.Devices != 132 {
		t.Errorf("devices = %d, want 132", usage.Devices)
	}
	if usage.MaxSessions != maxActiveSessions || usage.MaxDevices != api.MaxDeviceCount {
		t.Errorf("budgets = %d/%d, want %d/%d",
			usage.MaxSessions, usage.MaxDevices, maxActiveSessions, api.MaxDeviceCount)
	}
}
