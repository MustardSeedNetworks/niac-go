package snmp

import (
	"strconv"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

func TestSynthesizedIFMIBMergesAuthoredAndTopologyInterfaces(t *testing.T) {
	device := createTestDevice()
	device.Interfaces = []config.Interface{
		{Name: "Vlan200", Address: "10.0.0.2/24"},
		{Name: "HundredGigabitEthernet0/0/1", Address: "192.0.2.2/29"},
	}
	device.Routes = []config.Route{{
		Destination: "0.0.0.0/0", Via: "HundredGigabitEthernet0/0/1", NextHop: "192.0.2.1",
	}}
	device.TrunkPorts = []config.TrunkPort{{Interface: "HundredGigabitEthernet0/0/2"}}
	agent := NewAgent(device, 0)
	for _, name := range []string{"Vlan200", "HundredGigabitEthernet0/0/1", "HundredGigabitEthernet0/0/2"} {
		if _, ok := agent.InterfaceIndex(name); !ok {
			t.Errorf("missing synthesized interface %s", name)
		}
	}
	assertMIBValue(t, agent, ipAdEntAddr+".10.0.0.2", "10.0.0.2")
	assertMIBValue(t, agent, ipRouteNextHop+".0.0.0.0", "192.0.2.1")
}

func TestSynthesizedIFMIBAppliesAuthoredAttributes(t *testing.T) {
	device := createTestDevice()
	device.Interfaces = []config.Interface{{
		Name: "Vlan200", Type: "l2vlan", Address: "10.0.0.2/24", MTU: 9000, Speed: 100_000,
		AdminStatus: "up", OperStatus: "down", Description: "core SVI",
		InUtilization: 12.5, OutUtilization: 7.25,
	}}

	agent := NewAgent(device, 0)
	agent.startTime = time.Now().Add(-time.Minute)
	index, ok := agent.InterfaceIndex("Vlan200")
	if !ok {
		t.Fatal("missing Vlan200")
	}
	suffix := "." + strconv.Itoa(index)
	assertMIBValue(t, agent, ifHighSpeed+suffix, uint32(100_000))
	assertMIBValue(t, agent, ifType+suffix, interfaceTypeL2VLAN)
	assertMIBValue(t, agent, ifMtu+suffix, 9000)
	assertMIBValue(t, agent, ifOperStatus+suffix, interfaceStatusDown)
	assertMIBValue(t, agent, ifAlias+suffix, "core SVI")
	assertMIBValue(t, agent, ifConnectorPresent+suffix, TruthValueFalse)
	if value := agent.mib.Get(dot3StatsDuplexStatus + suffix); value != nil {
		t.Fatalf("logical interface has Ethernet duplex value: %#v", value)
	}
	assertPositiveCounter(t, agent, ifHCInOctets+suffix)
	assertPositiveCounter(t, agent, ifHCOutOctets+suffix)
	assertMIBValue(t, agent, ifInErrors+suffix, uint32(0))
	assertMIBValue(t, agent, ifOutErrors+suffix, uint32(0))
}

func TestSynthesizedIFMIBPublishesIEEE80211InterfaceType(t *testing.T) {
	device := createTestDevice()
	device.Interfaces = []config.Interface{{
		Name: "Dot11Radio0", Type: "ieee80211", MTU: 1500, Speed: 5_800,
		AdminStatus: "up", OperStatus: "up",
	}}

	agent := NewAgent(device, 0)
	index, ok := agent.InterfaceIndex("Dot11Radio0")
	if !ok {
		t.Fatal("missing Dot11Radio0")
	}
	assertMIBValue(t, agent, ifType+"."+strconv.Itoa(index), 71)
}

func TestSynthesizedTopologyInterfaceDefaultsToFullDuplex(t *testing.T) {
	device := createTestDevice()
	device.TrunkPorts = []config.TrunkPort{{Interface: "GigabitEthernet1/0/1"}}

	agent := NewAgent(device, 0)
	index, ok := agent.InterfaceIndex("GigabitEthernet1/0/1")
	if !ok {
		t.Fatal("missing synthesized topology interface")
	}
	suffix := "." + strconv.Itoa(index)
	assertMIBValue(t, agent, dot3StatsDuplexStatus+suffix, DuplexFull)
	assertMIBValue(t, agent, ifMtu+suffix, DefaultMTU)
}

func assertPositiveCounter(t *testing.T, agent *Agent, oid string) {
	t.Helper()
	value, err := agent.HandleGet(oid)
	if err != nil {
		t.Fatalf("GET %s: %v", oid, err)
	}
	counter, ok := value.Value.(uint64)
	if !ok || counter == 0 {
		t.Fatalf("%s = %#v, want positive Counter64", oid, value.Value)
	}
}
