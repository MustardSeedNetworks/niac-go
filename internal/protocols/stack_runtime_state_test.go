package protocols

import (
	"errors"
	"net"
	"net/netip"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

func TestRestoreDeviceStatesRejectsAnUnknownDevice(t *testing.T) {
	stack := runtimeStateTestStack(t)
	states := stack.ExportDeviceStates()
	if len(states) != 1 {
		t.Fatalf("exported states = %#v", states)
	}
	err := stack.RestoreDeviceStates(map[string]devicestate.State{
		"a-device-the-scenario-no-longer-has": states["state-router"],
	})
	if !errors.Is(err, ErrDeviceStateNotFound) {
		t.Fatalf("RestoreDeviceStates() error = %v, want ErrDeviceStateNotFound", err)
	}
}

// A restored history is history: the dispatcher must not send it as traps to
// an NMS that already received those events before the crash.
func TestRestoreDeviceStatesWithholdsRestoredHistoryFromNotifications(t *testing.T) {
	stack := runtimeStateTestStack(t)
	device := runtimeStateTestDevice(stack)
	store := stack.deviceStates[device]
	stack.notifications.Register(device, store, func(string) (int, bool) { return 1, true }, 0)
	if err := store.SetInterfaceFault("eth0", devicestate.FaultLinkDown, 1); err != nil {
		t.Fatalf("SetInterfaceFault() error = %v", err)
	}
	state := store.ExportState()

	fresh := runtimeStateTestStack(t)
	freshDevice := runtimeStateTestDevice(fresh)
	freshDevice.SyslogConfig = &config.SyslogConfig{Enabled: true, Receivers: []string{"192.0.2.99:514"}}
	sender := &recordingDatagramSender{}
	fresh.notifications.sender = sender
	fresh.notifications.Register(
		freshDevice, fresh.deviceStates[freshDevice],
		func(string) (int, bool) { return 1, true }, 0,
	)
	if err := fresh.RestoreDeviceStates(
		map[string]devicestate.State{"state-router": state},
	); err != nil {
		t.Fatalf("RestoreDeviceStates() error = %v", err)
	}
	registration := fresh.notifications.registrations[freshDevice]
	if registration == nil {
		t.Fatal("device is not registered for notifications")
	}
	if registration.cursor != state.Version {
		t.Fatalf("notification cursor = %d, want the restored version %d",
			registration.cursor, state.Version)
	}
	pending, _ := fresh.deviceStates[freshDevice].EventsAfter(registration.cursor)
	if len(pending) != 0 {
		t.Fatalf("restored history left %d events to announce: %#v", len(pending), pending)
	}
	fresh.notifications.dispatchPending()
	if len(sender.datagrams) != 0 {
		t.Fatal("recovery resent retained notification history")
	}
	if err := fresh.deviceStates[freshDevice].SetInterfaceFault("eth0", devicestate.FaultLinkDown, 0); err != nil {
		t.Fatal(err)
	}
	fresh.notifications.dispatchPending()
	if len(sender.datagrams) != 1 ||
		!strings.Contains(string(sender.datagrams[0].payload), `kind=fault.cleared target="eth0:link_down"`) {
		t.Fatalf("expected only the new clearing notification, got %+v", sender.datagrams)
	}
}

func runtimeStateTestStack(t *testing.T) *Stack {
	t.Helper()
	cfg := &config.Config{Devices: []config.Device{{
		Name:       "state-router",
		MACAddress: net.HardwareAddr{0x02, 0, 0, 0, 0, 0x21},
		SNMPConfig: config.SNMPConfig{
			Traps: &config.TrapConfig{Enabled: true, Receivers: []string{"192.0.2.99:162"}},
		},
	}}}
	device := &cfg.Devices[0]
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	stack.registerDeviceState(device, NewDeviceTable())
	stack.deviceStates[device].ReplaceNetwork(devicestate.Network{Interfaces: []devicestate.Interface{{
		Name: "eth0", Address: netip.MustParsePrefix("192.0.2.21/24"),
		AdminUp: true, OperUp: true, CarrierUp: true,
	}}})
	return stack
}

func runtimeStateTestDevice(stack *Stack) *config.Device {
	for device := range stack.deviceStates {
		return device
	}
	return nil
}
