package snmp

import (
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

const (
	phoneDrawTenthWatts  = 62  // a class-2 IP phone
	cameraDrawTenthWatts = 128 // a class-3 camera
)

// poeSwitch is a two-port PoE access switch with a phone on the first port and
// nothing on the second, which is the shape a tester walks in a wiring closet.
func poeSwitch() *config.Device {
	device := createTestDevice()
	device.Name = "closet-1"
	device.Type = "switch"
	device.Interfaces = []config.Interface{
		{Name: "Gi1/0/1", Type: "ethernet", AdminStatus: "up"},
		{Name: "Gi1/0/2", Type: "ethernet", AdminStatus: "up"},
	}
	device.TrunkPorts = []config.TrunkPort{
		{Interface: "Gi1/0/1", RemoteDevice: "phone-1"},
	}
	device.PoEConfig = &config.PoEConfig{BudgetWatts: 370}

	return device
}

func poeResolver(draw int, priority string) PeerResolver {
	return func(name, _ string) (PeerIdentity, bool) {
		if name != "phone-1" {
			return PeerIdentity{}, false
		}

		return PeerIdentity{PoEDrawTenthWatts: draw, PoEPriority: priority}, true
	}
}

func poePortOID(t *testing.T, agent *Agent, column, interfaceName string) *OIDValue {
	t.Helper()
	index, ok := agent.ifIndexForInterface(interfaceName)
	if !ok {
		t.Fatalf("no ifIndex for %s", interfaceName)
	}

	return agent.mib.Get(column + "." + pethPseGroupIndex + "." + index)
}

func wantInt(t *testing.T, value *OIDValue, want int, what string) {
	t.Helper()
	if value == nil {
		t.Fatalf("%s is absent", what)
	}
	if got := oidValueString(value); got != intString(want) {
		t.Errorf("%s = %s, want %d", what, got, want)
	}
}

func intString(value int) string {
	return oidValueString(&OIDValue{Value: value})
}

// TestPoEPortDeliveringPowerFollowsTheAttachedDevice is the whole point of the
// table: the switch reports power on the port a powered device is behind and
// keeps searching on the one that is empty. Neither is authored on the switch --
// the phone's own LLDP-MED advertisement is the single source.
func TestPoEPortDeliveringPowerFollowsTheAttachedDevice(t *testing.T) {
	agent := NewAgent(poeSwitch(), 0)
	agent.SynthesizePoEPower(poeResolver(phoneDrawTenthWatts, "high"))

	wantInt(t, poePortOID(t, agent, pethPsePortDetectionStatus, "Gi1/0/1"),
		pethPortDetectionDeliveringPower, "detection status of the phone port")
	wantInt(t, poePortOID(t, agent, pethPsePortDetectionStatus, "Gi1/0/2"),
		pethPortDetectionSearching, "detection status of the empty port")
	wantInt(t, poePortOID(t, agent, pethPsePortPowerPriority, "Gi1/0/1"),
		pethPriorityHigh, "power priority of the phone port")
	wantInt(t, poePortOID(t, agent, pethPsePortPowerClassifications, "Gi1/0/1"),
		pethPowerClass2, "power class of a 6.2 W phone")
	wantInt(t, poePortOID(t, agent, pethPsePortPowerClassifications, "Gi1/0/2"),
		pethPowerClass0, "power class of the empty port")
}

func TestPoEPowerClassFollowsTheAdvertisedDraw(t *testing.T) {
	for _, testCase := range []struct {
		name      string
		draw      int
		wantClass int
	}{
		{"class 1 sensor", 30, pethPowerClass1},
		{"class 2 phone", phoneDrawTenthWatts, pethPowerClass2},
		{"class 3 camera", cameraDrawTenthWatts, pethPowerClass3},
		{"class 4 access point", 255, pethPowerClass4},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			agent := NewAgent(poeSwitch(), 0)
			agent.SynthesizePoEPower(poeResolver(testCase.draw, "low"))

			wantInt(t, poePortOID(t, agent, pethPsePortPowerClassifications, "Gi1/0/1"),
				testCase.wantClass, "power class")
		})
	}
}

// TestPoEMainTableReportsTheAuthoredBudget is the acceptance a tester reads:
// the budget is what the author wrote and the consumption is what the attached
// devices actually draw, in the watts the MIB defines.
func TestPoEMainTableReportsTheAuthoredBudget(t *testing.T) {
	agent := NewAgent(poeSwitch(), 0)
	agent.SynthesizePoEPower(poeResolver(cameraDrawTenthWatts, "low"))

	wantInt(t, agent.mib.Get(pethMainPsePower+"."+pethPseGroupIndex), 370, "pethMainPsePower")
	wantInt(t, agent.mib.Get(pethMainPseOperStatus+"."+pethPseGroupIndex),
		pethMainPseOperStatusOn, "pethMainPseOperStatus")
	wantInt(t, agent.mib.Get(pethMainPseUsageThreshold+"."+pethPseGroupIndex),
		(&config.PoEConfig{}).UsageThreshold(), "pethMainPseUsageThreshold default")
	wantInt(t, agent.mib.Get(pethMainPseConsumptionPower+"."+pethPseGroupIndex),
		cameraDrawTenthWatts/config.PoETenthWattsPerWatt, "pethMainPseConsumptionPower")
}

func TestPoEUsageThresholdIsAuthorable(t *testing.T) {
	device := poeSwitch()
	device.PoEConfig.UsageThresholdPercent = 65
	agent := NewAgent(device, 0)

	wantInt(t, agent.mib.Get(pethMainPseUsageThreshold+"."+pethPseGroupIndex),
		65, "pethMainPseUsageThreshold")
}

// TestPoELossFaultsThePortAndDropsConsumption is the fault's observable half.
// A cut cable and a cut power budget both take the link down; only PoE reports
// the port faulted and takes its draw out of the closet total, which is how a
// tester tells them apart.
func TestPoELossFaultsThePortAndDropsConsumption(t *testing.T) {
	device := poeSwitch()
	state := devicestate.NewStore(devicestate.Identity{Hostname: device.Name})
	state.ReplaceNetwork(devicestate.Network{Interfaces: []devicestate.Interface{
		{Name: "Gi1/0/1", AdminUp: true, OperUp: true, CarrierUp: true},
		{Name: "Gi1/0/2", AdminUp: true, OperUp: true, CarrierUp: true},
	}})
	agent := NewAgentWithState(device, state, AgentOptions{})
	agent.SynthesizePoEPower(poeResolver(phoneDrawTenthWatts, "high"))

	consumed := oidValueString(agent.mib.Get(pethMainPseConsumptionPower + "." + pethPseGroupIndex))
	if consumed != intString(phoneDrawTenthWatts/config.PoETenthWattsPerWatt) {
		t.Fatalf("consumption before the fault = %s, want %d",
			consumed, phoneDrawTenthWatts/config.PoETenthWattsPerWatt)
	}

	if err := state.SetInterfaceFault("Gi1/0/1", devicestate.FaultPoELoss, 1); err != nil {
		t.Fatalf("SetInterfaceFault(poe_loss) error = %v", err)
	}

	wantInt(t, poePortOID(t, agent, pethPsePortDetectionStatus, "Gi1/0/1"),
		pethPortDetectionFault, "detection status under poe_loss")
	wantInt(t, poePortOID(t, agent, pethPsePortPowerClassifications, "Gi1/0/1"),
		pethPowerClass0, "power class under poe_loss")
	wantInt(t, agent.mib.Get(pethMainPseConsumptionPower+"."+pethPseGroupIndex),
		0, "consumption under poe_loss")
	wantInt(t, poePortOID(t, agent, pethPsePortDetectionStatus, "Gi1/0/2"),
		pethPortDetectionSearching, "detection status of the port with no fault")
}

// TestPoEAbsentWithoutAuthoredPSE keeps the MIB honest: a device the author did
// not describe as power sourcing equipment must not answer for one.
func TestPoEAbsentWithoutAuthoredPSE(t *testing.T) {
	device := poeSwitch()
	device.PoEConfig = nil
	agent := NewAgent(device, 0)
	agent.SynthesizePoEPower(poeResolver(phoneDrawTenthWatts, "high"))

	if value := agent.mib.Get(pethMainPsePower + "." + pethPseGroupIndex); value != nil {
		t.Errorf("pethMainPsePower = %v on a device with no poe block", oidValueString(value))
	}
	if value := poePortOID(t, agent, pethPsePortAdminEnable, "Gi1/0/1"); value != nil {
		t.Errorf("pethPsePortAdminEnable = %v on a device with no poe block",
			oidValueString(value))
	}
}

// TestPoENonEthernetPortsAreNotPSEPorts: a VLAN or loopback interface has no
// pair to put voltage on, so it gets no row.
func TestPoENonEthernetPortsAreNotPSEPorts(t *testing.T) {
	device := poeSwitch()
	device.Interfaces = append(device.Interfaces,
		config.Interface{Name: "Vlan10", Type: "vlan", AdminStatus: "up"})
	agent := NewAgent(device, 0)

	if value := poePortOID(t, agent, pethPsePortAdminEnable, "Vlan10"); value != nil {
		t.Errorf("pethPsePortAdminEnable on Vlan10 = %v, want absent", oidValueString(value))
	}
}

func TestWalkOwnsPoEDetectsAnyObjectInTheGroup(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		entries []WalkEntry
		want    bool
	}{
		{"no PoE at all", []WalkEntry{{OID: ".1.3.6.1.2.1.1.5.0"}}, false},
		{"full PSE port table", []WalkEntry{{OID: ".1.3.6.1.2.1.105.1.1.1.6.1.1001"}}, true},
		// The 3com capture in the corpus carries only this one table.
		{"notification control only", []WalkEntry{{OID: ".1.3.6.1.2.1.105.1.4.1.1.2.1"}}, true},
		{"a neighbouring group", []WalkEntry{{OID: ".1.3.6.1.2.1.1050.1"}}, false},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if got := walkOwnsPoE(testCase.entries); got != testCase.want {
				t.Errorf("walkOwnsPoE() = %t, want %t", got, testCase.want)
			}
		})
	}
}

// TestPoEPortsCoverTrunkOnlyPorts is the shape the shipped packs actually have:
// an access switch declares its ports as trunk_ports and authors only a
// management VLAN interface. A PoE table keyed off the authored interface list
// would be empty for every switch in every pack.
func TestPoEPortsCoverTrunkOnlyPorts(t *testing.T) {
	device := createTestDevice()
	device.Name = "HOSP-ACC-01"
	device.Type = "switch"
	device.Interfaces = []config.Interface{
		{Name: "Vlan200", Type: "vlan", Address: "10.0.200.11/24", AdminStatus: "up"},
	}
	device.TrunkPorts = []config.TrunkPort{
		{Interface: "GigabitEthernet1/0/1", RemoteDevice: "phone-1"},
		{Interface: "GigabitEthernet1/0/2", RemoteDevice: "pc-1"},
	}
	device.PoEConfig = &config.PoEConfig{BudgetWatts: 370}

	agent := NewAgent(device, 0)
	agent.SynthesizePoEPower(poeResolver(phoneDrawTenthWatts, "high"))

	wantInt(t, poePortOID(t, agent, pethPsePortDetectionStatus, "GigabitEthernet1/0/1"),
		pethPortDetectionDeliveringPower, "detection status of the phone port")
	wantInt(t, poePortOID(t, agent, pethPsePortDetectionStatus, "GigabitEthernet1/0/2"),
		pethPortDetectionSearching, "detection status of the workstation port")
	if value := poePortOID(t, agent, pethPsePortAdminEnable, "Vlan200"); value != nil {
		t.Errorf("Vlan200 has a PSE row: %v", oidValueString(value))
	}
	wantInt(t, agent.mib.Get(pethMainPseConsumptionPower+"."+pethPseGroupIndex),
		phoneDrawTenthWatts/config.PoETenthWattsPerWatt, "consumption")
}
