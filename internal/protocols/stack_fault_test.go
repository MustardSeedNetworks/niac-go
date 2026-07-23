package protocols

import (
	"errors"
	"net"
	"net/netip"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

func TestStackInterfaceFaultsUseAuthoritativeDeviceState(t *testing.T) {
	stack, device := newFaultTestStack()
	for _, fault := range []struct {
		faultType devicestate.FaultType
		value     int
	}{
		{devicestate.FaultFCS, 25},
		{devicestate.FaultDiscards, 40},
	} {
		if err := stack.SetInterfaceFault("192.0.2.1", "Gi0/1", fault.faultType, fault.value); err != nil {
			t.Fatalf("SetInterfaceFault(%s) error = %v", fault.faultType, err)
		}
	}

	faults := stack.deviceStates[device].Snapshot().Faults
	if len(faults) != 2 {
		t.Fatalf("authoritative faults = %#v, want two", faults)
	}
	active := stack.ActiveInterfaceFaults()
	if active["edge-1"]["Gi0/1"][devicestate.FaultFCS] != 25 {
		t.Fatalf("active faults = %#v", active)
	}
}

func TestStackInterfaceFaultTargetValidation(t *testing.T) {
	stack, _ := newFaultTestStack()
	if err := stack.SetInterfaceFault("192.0.2.99", "Gi0/1", devicestate.FaultFCS, 1); !errors.Is(
		err,
		ErrFaultDeviceNotFound,
	) {
		t.Fatalf("unknown device error = %v, want %v", err, ErrFaultDeviceNotFound)
	}
	if err := stack.SetInterfaceFault("192.0.2.1", "Gi0/99", devicestate.FaultFCS, 1); !errors.Is(
		err,
		devicestate.ErrInterfaceNotFound,
	) {
		t.Fatalf("unknown interface error = %v, want %v", err, devicestate.ErrInterfaceNotFound)
	}
}

func TestStackInterfaceFaultRejectsDeviceWithoutSNMPCounters(t *testing.T) {
	disabled := false
	device := faultTestDevice("edge-1")
	device.SNMPConfig.Enabled = &disabled
	cfg := &config.Config{Devices: []config.Device{device}}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))

	err := stack.SetInterfaceFault("edge-1", "Gi0/1", devicestate.FaultFCS, 1)
	if !errors.Is(err, ErrFaultUnobservable) {
		t.Fatalf("error = %v, want %v", err, ErrFaultUnobservable)
	}
	if len(stack.ActiveInterfaceFaults()) != 0 {
		t.Fatalf("unobservable fault was persisted: %#v", stack.ActiveInterfaceFaults())
	}
	targets := stack.InterfaceFaultTargets()
	if len(targets) != 1 || len(targets[0].Interfaces) != 0 {
		t.Fatalf("unobservable interfaces advertised: %#v", targets)
	}
}

func TestStackInterfaceFaultRejectsAmbiguousAddress(t *testing.T) {
	cfg := &config.Config{Devices: []config.Device{
		faultTestDevice("edge-1"),
		faultTestDevice("edge-2"),
	}}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))

	err := stack.SetInterfaceFault("192.0.2.1", "Gi0/1", devicestate.FaultFCS, 1)
	if !errors.Is(err, ErrFaultDeviceAmbiguous) {
		t.Fatalf("ambiguous device error = %v, want %v", err, ErrFaultDeviceAmbiguous)
	}
}

func TestStackInterfaceFaultDeviceNameDisambiguatesSharedAddress(t *testing.T) {
	cfg := &config.Config{Devices: []config.Device{
		faultTestDevice("edge-1"),
		faultTestDevice("edge-2"),
	}}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))

	if err := stack.SetInterfaceFault("edge-2", "Gi0/1", devicestate.FaultFCS, 7); err != nil {
		t.Fatal(err)
	}
	active := stack.ActiveInterfaceFaults()
	if len(active) != 1 || active["edge-2"]["Gi0/1"][devicestate.FaultFCS] != 7 {
		t.Fatalf("active faults = %#v", active)
	}
}

func TestStackInterfaceFaultTargetUsesCurrentDeviceState(t *testing.T) {
	stack, device := newFaultTestStack()
	if err := stack.deviceStates[device].UpdateInterface(
		"Gi0/1",
		func(iface devicestate.Interface) (devicestate.Interface, error) {
			iface.Address = netip.MustParsePrefix("198.51.100.7/24")
			return iface, nil
		},
	); err != nil {
		t.Fatal(err)
	}

	address, interfaceName, ok := stack.InterfaceFaultTarget(device)
	if !ok || address != "198.51.100.7" || interfaceName != "Gi0/1" {
		t.Fatalf("target = %q, %q, %t", address, interfaceName, ok)
	}
	targets := stack.InterfaceFaultTargets()
	if len(targets) != 1 || targets[0].Device != "edge-1" ||
		targets[0].Address != "198.51.100.7" ||
		!slices.Equal(targets[0].Interfaces, []string{"Gi0/1"}) {
		t.Fatalf("targets = %#v", targets)
	}
}

func TestStackInterfaceFaultTargetAllowsUnnumberedInterface(t *testing.T) {
	stack, device := newFaultTestStack()
	if err := stack.deviceStates[device].UpdateInterface(
		"Gi0/1",
		func(iface devicestate.Interface) (devicestate.Interface, error) {
			iface.Address = netip.Prefix{}
			return iface, nil
		},
	); err != nil {
		t.Fatal(err)
	}

	address, interfaceName, ok := stack.InterfaceFaultTarget(device)
	if !ok || address != "" || interfaceName != "Gi0/1" {
		t.Fatalf("target = %q, %q, %t", address, interfaceName, ok)
	}
}

func TestStackFaultTelemetryReachesSNMPCounters(t *testing.T) {
	stack, device := newFaultTestStack()
	group := stack.snmpAgents[device]
	start := time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC)
	stack.advanceDeviceFaultTelemetry(device, start)
	if err := stack.setInterfaceFaultAt(
		"192.0.2.1", "Gi0/1", devicestate.FaultFCS, 4, start,
	); err != nil {
		t.Fatalf("setInterfaceFaultAt() error = %v", err)
	}
	stack.advanceDeviceFaultTelemetry(device, start.Add(time.Second))
	stack.deviceStates[device].ClearAllFaults()

	index, ok := group.baseAgent.InterfaceIndex("Gi0/1")
	if !ok {
		t.Fatal("Gi0/1 has no SNMP interface index")
	}
	value, err := group.baseAgent.HandleGet("1.3.6.1.2.1.10.7.2.1.3." + strconv.Itoa(index))
	if err != nil || value.Value != uint32(4) {
		t.Fatalf("dot3StatsFCSErrors = %#v, %v; want 4", value, err)
	}
}

func newFaultTestStack() (*Stack, *config.Device) {
	cfg := &config.Config{Devices: []config.Device{faultTestDevice("edge-1")}}
	return NewStack(nil, cfg, logging.NewDebugConfig(0)), &cfg.Devices[0]
}

func faultTestDevice(name string) config.Device {
	return config.Device{
		Name: name, IPAddresses: []net.IP{{192, 0, 2, 1}},
		Interfaces: []config.Interface{{
			Name: "Gi0/1", Address: "192.0.2.1/24", Speed: 100,
		}},
		TrunkPorts: []config.TrunkPort{{Interface: "Gi0/1"}},
		SNMPConfig: config.SNMPConfig{Community: "public"},
	}
}
