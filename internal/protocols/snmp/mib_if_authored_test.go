package snmp

import (
	"strconv"
	"testing"

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
		Name: "Vlan200", Address: "10.0.0.2/24", Speed: 100_000,
		AdminStatus: "up", OperStatus: "down", Duplex: "full", Description: "core SVI",
	}}

	agent := NewAgent(device, 0)
	index, ok := agent.InterfaceIndex("Vlan200")
	if !ok {
		t.Fatal("missing Vlan200")
	}
	suffix := "." + strconv.Itoa(index)
	assertMIBValue(t, agent, ifHighSpeed+suffix, uint32(100_000))
	assertMIBValue(t, agent, ifOperStatus+suffix, interfaceStatusDown)
	assertMIBValue(t, agent, ifAlias+suffix, "core SVI")
	assertMIBValue(t, agent, dot3StatsDuplexStatus+suffix, DuplexFull)
}
