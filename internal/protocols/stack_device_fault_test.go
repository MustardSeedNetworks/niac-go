package protocols

import (
	"errors"
	"net"
	"slices"
	"testing"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp"
)

// Every device fault type, armed and cleared through the stack against a
// device that runs the affected service.
func TestStackDeviceFaultCoversEveryType(t *testing.T) {
	for _, definition := range devicestate.DeviceFaultDefinitions() {
		t.Run(string(definition.Type), func(t *testing.T) {
			stack, device := newDeviceFaultTestStack()

			if err := stack.SetDeviceFault("edge-1", definition.Type, 1); err != nil {
				t.Fatalf("SetDeviceFault(%s) error = %v", definition.Type, err)
			}
			if !stack.deviceFaultActive(device, definition.Type) {
				t.Fatalf("fault %s is not active on the device state", definition.Type)
			}
			if got := stack.ActiveDeviceFaults()["edge-1"][definition.Type]; got != 1 {
				t.Fatalf("ActiveDeviceFaults()[%s] = %d, want 1", definition.Type, got)
			}

			if err := stack.SetDeviceFault("edge-1", definition.Type, 0); err != nil {
				t.Fatalf("SetDeviceFault(clear) error = %v", err)
			}
			if stack.deviceFaultActive(device, definition.Type) {
				t.Fatalf("fault %s survived a clear", definition.Type)
			}
		})
	}
}

// A fault the device cannot serve must be refused, not silently stored: an
// operator who arms dhcp_no_offer on a device with no DHCP server would
// otherwise see a success and no change on the wire.
func TestStackDeviceFaultRefusesAbsentService(t *testing.T) {
	device := faultTestDevice("edge-1")
	device.DNSConfig = &config.DNSConfig{ForwardRecords: []config.DNSRecord{
		{Name: "host.example.", IP: net.IP{192, 0, 2, 10}},
	}}
	cfg := &config.Config{Devices: []config.Device{device}}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))

	err := stack.SetDeviceFault("edge-1", devicestate.FaultDHCPNoOffer, 1)
	if !errors.Is(err, ErrFaultServiceAbsent) {
		t.Fatalf("error = %v, want %v", err, ErrFaultServiceAbsent)
	}
	if len(stack.ActiveDeviceFaults()) != 0 {
		t.Fatalf("refused fault was persisted: %#v", stack.ActiveDeviceFaults())
	}

	targets := stack.DeviceFaultTargets()
	if len(targets) != 1 {
		t.Fatalf("targets = %#v, want the DNS server", targets)
	}
	// The whole advertised set, not just the absence of the DHCP one: latency
	// suppresses no service and so is offered on every device, and stating
	// that here is what keeps a future universal fault from arriving unnoticed.
	want := []devicestate.DeviceFaultType{
		devicestate.FaultDNSNXDomain,
		devicestate.FaultDNSTimeout,
		devicestate.FaultLatency,
	}
	if !slices.Equal(targets[0].Faults, want) {
		t.Fatalf("DNS server faults = %v, want %v", targets[0].Faults, want)
	}
}

func TestStackDeviceFaultTargetValidation(t *testing.T) {
	stack, _ := newDeviceFaultTestStack()

	if err := stack.SetDeviceFault(
		"192.0.2.99", devicestate.FaultDHCPNoOffer, 1,
	); !errors.Is(err, ErrFaultDeviceNotFound) {
		t.Fatalf("unknown device error = %v, want %v", err, ErrFaultDeviceNotFound)
	}
	if err := stack.SetDeviceFault(
		"edge-1", devicestate.DeviceFaultType("fcs_errors"), 1,
	); !errors.Is(err, devicestate.ErrDeviceFaultTypeInvalid) {
		t.Fatalf("unknown device fault type = %v, want refusal", err)
	}
}

func TestStackClearDeviceFaultsLeavesInterfaceFaults(t *testing.T) {
	stack, device := newDeviceFaultTestStack()
	if err := stack.SetDeviceFault("edge-1", devicestate.FaultDNSTimeout, 1); err != nil {
		t.Fatalf("SetDeviceFault() error = %v", err)
	}
	if err := stack.SetInterfaceFault("edge-1", "Gi0/1", devicestate.FaultFCS, 25); err != nil {
		t.Fatalf("SetInterfaceFault() error = %v", err)
	}

	stack.ClearAllDeviceFaults()

	if len(stack.ActiveDeviceFaults()) != 0 {
		t.Fatalf("device faults = %#v, want none", stack.ActiveDeviceFaults())
	}
	if got := stack.deviceStates[device].Snapshot().Faults; len(got) != 1 {
		t.Fatalf("interface faults = %#v, want the FCS fault kept", got)
	}
}

func newDeviceFaultTestStack() (*Stack, *config.Device) {
	device := faultTestDevice("edge-1")
	device.SNMPConfig.AddMibs = resourceFaultTestMIBs()
	device.DHCPConfig = &config.DHCPConfig{
		PoolStart: net.IP{192, 0, 2, 100}, PoolEnd: net.IP{192, 0, 2, 110},
	}
	device.DNSConfig = &config.DNSConfig{ForwardRecords: []config.DNSRecord{
		{Name: "host.example.", IP: net.IP{192, 0, 2, 10}},
	}}
	cfg := &config.Config{Devices: []config.Device{device}}
	return NewStack(nil, cfg, logging.NewDebugConfig(0)), &cfg.Devices[0]
}

// F7: only the three named resource rows change under device faults. Service
// failures leave the MIB unchanged, and clearing restores every resource row.
func TestDeviceFaultMIBEffectsAreNamedSubstitutions(t *testing.T) {
	stack, device := newDeviceFaultTestStack()
	agent := stack.snmpAgents[device].baseAgent
	if agent == nil {
		t.Fatal("device has no SNMP agent to sweep")
	}

	budget := snmp.SweepBudget(0)
	baseline := sweepAgent(t, agent, budget)

	for _, definition := range devicestate.DeviceFaultDefinitions() {
		if err := stack.SetDeviceFault("edge-1", definition.Type, 1); err != nil {
			t.Fatalf("SetDeviceFault(%s) error = %v", definition.Type, err)
		}
	}
	faulted := sweepAgent(t, agent, budget)
	contract := agent.WalkContract()
	stack.ClearAllDeviceFaults()
	settled := sweepAgent(t, agent, budget)
	expected := map[string]string{
		"1.3.6.1.2.1.25.3.3.1.2.1": "1",
		"1.3.6.1.2.1.25.2.3.1.6.2": "10",
		"1.3.6.1.2.1.25.2.3.1.6.3": "100",
	}
	if err := compareDeviceFaultMIBRows(baseline, faulted, settled, expected); err != nil {
		t.Fatal(err)
	}
	for oid := range expected {
		if contract.Classify(oid) != snmp.BucketLive {
			t.Fatalf("resource substitution not classified live: %s", oid)
		}
	}
}

func sweepAgent(t *testing.T, agent *snmp.Agent, budget int) []gosnmp.SnmpPDU {
	t.Helper()

	rows, breakReason := snmp.SweepGetNext(agent, budget)
	if breakReason != "" {
		t.Fatalf("sweep stopped early: %s", breakReason)
	}
	if len(rows) == 0 {
		t.Fatal("sweep returned nothing to compare")
	}
	return rows
}
