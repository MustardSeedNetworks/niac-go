package protocols

import (
	"errors"
	"net"
	"reflect"
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
	for _, faultType := range targets[0].Faults {
		if faultType == devicestate.FaultDHCPNoOffer {
			t.Fatalf("device with no DHCP server advertises %s", faultType)
		}
	}
	if len(targets[0].Faults) != 2 {
		t.Fatalf("DNS server faults = %v, want the two DNS types", targets[0].Faults)
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
		"edge-1", devicestate.FaultFCS, 1,
	); !errors.Is(err, devicestate.ErrDeviceFaultTypeInvalid) {
		t.Fatalf("interface fault on the device axis = %v, want refusal", err)
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
	device.DHCPConfig = &config.DHCPConfig{
		PoolStart: net.IP{192, 0, 2, 100}, PoolEnd: net.IP{192, 0, 2, 110},
	}
	device.DNSConfig = &config.DNSConfig{ForwardRecords: []config.DNSRecord{
		{Name: "host.example.", IP: net.IP{192, 0, 2, 10}},
	}}
	cfg := &config.Config{Devices: []config.Device{device}}
	return NewStack(nil, cfg, logging.NewDebugConfig(0)), &cfg.Devices[0]
}

// F7: every fault type must have its MIB effect stated. These three have
// none — a DHCP server that stops offering and a DNS server that answers
// NXDOMAIN both keep their inventory, interfaces and counters — so the
// assertion is that nothing leaks: an armed device fault leaves the walk the
// agent serves byte-identical. A future device fault with a real MIB effect
// (P2-4's hrProcessorLoad, sysUpTime reset) fails here and must instead be
// classified as a named substitution.
func TestDeviceFaultsDoNotPerturbTheServedMIB(t *testing.T) {
	stack, device := newDeviceFaultTestStack()
	agent := stack.snmpAgents[device].baseAgent
	if agent == nil {
		t.Fatal("device has no SNMP agent to sweep")
	}

	budget := snmp.SweepBudget(0)
	// Two healthy sweeps first: the agent's own SNMP-group counters
	// (snmpInTotalReqVars and friends) move because a sweep happened, so a
	// bare before/after comparison would fail on the measurement itself.
	// Calibrating against a second healthy sweep names those rows instead of
	// hard-coding a subtree to ignore.
	first := sweepAgent(t, agent, budget)
	baseline := sweepAgent(t, agent, budget)
	volatile := changedOIDs(first, baseline)

	for _, definition := range devicestate.DeviceFaultDefinitions() {
		if err := stack.SetDeviceFault("edge-1", definition.Type, 1); err != nil {
			t.Fatalf("SetDeviceFault(%s) error = %v", definition.Type, err)
		}
	}

	faulted := sweepAgent(t, agent, budget)
	if len(faulted) != len(baseline) {
		t.Fatalf("row count changed under device faults: %d -> %d",
			len(baseline), len(faulted))
	}

	compared := 0
	for index := range baseline {
		if baseline[index].Name != faulted[index].Name {
			t.Fatalf("row %d OID changed: %s -> %s",
				index, baseline[index].Name, faulted[index].Name)
		}
		if _, moves := volatile[baseline[index].Name]; moves {
			continue
		}
		compared++
		if !reflect.DeepEqual(baseline[index].Value, faulted[index].Value) {
			t.Fatalf("row %s value changed under device faults: %v -> %v",
				baseline[index].Name, baseline[index].Value, faulted[index].Value)
		}
	}
	// The volatile set must not swallow the walk: a calibration that excluded
	// everything would make this test vacuous.
	if compared < len(baseline)/2 {
		t.Fatalf("only %d of %d rows were stable enough to compare",
			compared, len(baseline))
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

// changedOIDs names the rows that move between two identical healthy sweeps.
func changedOIDs(first, second []gosnmp.SnmpPDU) map[string]struct{} {
	changed := make(map[string]struct{})
	values := make(map[string]any, len(first))
	for _, row := range first {
		values[row.Name] = row.Value
	}
	for _, row := range second {
		if previous, seen := values[row.Name]; !seen ||
			!reflect.DeepEqual(previous, row.Value) {
			changed[row.Name] = struct{}{}
		}
	}
	return changed
}
