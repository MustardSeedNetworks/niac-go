package protocols

import (
	"errors"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

func TestStackInterfaceFaultsUseAuthoritativeDeviceState(t *testing.T) {
	stack, device := newFaultTestStack()
	for faultType, value := range map[devicestate.FaultType]int{
		devicestate.FaultFCS: 25, devicestate.FaultDiscards: 40,
	} {
		if err := stack.SetInterfaceFault("192.0.2.1", "Gi0/1", faultType, value); err != nil {
			t.Fatalf("SetInterfaceFault(%s) error = %v", faultType, err)
		}
	}

	faults := stack.deviceStates[device].Snapshot().Faults
	if len(faults) != 2 {
		t.Fatalf("authoritative faults = %#v, want two", faults)
	}
	active := stack.ActiveInterfaceFaults()
	if active["192.0.2.1"]["Gi0/1"][devicestate.FaultFCS] != 25 {
		t.Fatalf("active faults = %#v", active)
	}
}

func TestStackInterfaceFaultTargetValidation(t *testing.T) {
	stack, _ := newFaultTestStack()
	if err := stack.SetInterfaceFault("192.0.2.99", "Gi0/1", devicestate.FaultFCS, 1); !errors.Is(err, ErrFaultDeviceNotFound) {
		t.Fatalf("unknown device error = %v, want %v", err, ErrFaultDeviceNotFound)
	}
	if err := stack.SetInterfaceFault("192.0.2.1", "Gi0/99", devicestate.FaultFCS, 1); !errors.Is(err, devicestate.ErrInterfaceNotFound) {
		t.Fatalf("unknown interface error = %v, want %v", err, devicestate.ErrInterfaceNotFound)
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
	cfg := &config.Config{Devices: []config.Device{{
		Name: "edge-1", IPAddresses: []net.IP{{192, 0, 2, 1}},
		Interfaces: []config.Interface{{
			Name: "Gi0/1", Address: "192.0.2.1/24", Speed: 100,
		}},
		TrunkPorts: []config.TrunkPort{{Interface: "Gi0/1"}},
		SNMPConfig: config.SNMPConfig{Community: "public"},
	}}}
	return NewStack(nil, cfg, logging.NewDebugConfig(0)), &cfg.Devices[0]
}
