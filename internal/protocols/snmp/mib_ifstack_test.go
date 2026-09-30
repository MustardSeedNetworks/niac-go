package snmp

import (
	"maps"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

func ifStackRows(agent *Agent) []string {
	prefix := ifStackStatus + "."
	agent.mib.mu.RLock()
	defer agent.mib.mu.RUnlock()
	var rows []string
	for oid := range agent.mib.entries {
		if suffix, ok := strings.CutPrefix(oid, prefix); ok {
			rows = append(rows, suffix)
		}
	}
	slices.Sort(rows)
	return rows
}

func TestPortChannelPublishesLAGAggregateOverItsMembers(t *testing.T) {
	device := createTestDevice()
	device.Interfaces = []config.Interface{
		{Name: "TenGigabitEthernet1/1/1", Speed: 25_000},
		{Name: "TenGigabitEthernet1/1/2"},
	}
	// The trunk spells the bundle the way IOS prints it; the schema says
	// port-channel<id>. Both must land on one ifTable row.
	device.TrunkPorts = []config.TrunkPort{
		{Interface: "Port-channel1"},
		{Interface: "GigabitEthernet1/0/48"},
	}
	device.PortChannels = []config.PortChannel{
		{ID: 1, Members: []string{"TenGigabitEthernet1/1/1", "TenGigabitEthernet1/1/2"}, Mode: "active"},
		{ID: 2, Members: []string{"GigabitEthernet1/0/1", "GigabitEthernet1/0/2"}, Mode: "on"},
	}

	agent := NewAgent(device, 0)
	indexOf := func(name string) string {
		t.Helper()
		index, ok := agent.InterfaceIndex(name)
		if !ok {
			t.Fatalf("no ifTable row for %s", name)
		}
		return strconv.Itoa(index)
	}
	if _, ok := agent.InterfaceIndex("port-channel1"); ok {
		t.Fatal("port-channel1 has a row of its own beside Port-channel1")
	}
	assertMIBValue(t, agent, ifNumber, 7)

	po1, po2 := indexOf("Port-channel1"), indexOf("port-channel2")
	te1, te2 := indexOf("TenGigabitEthernet1/1/1"), indexOf("TenGigabitEthernet1/1/2")
	gi1, gi2 := indexOf("GigabitEthernet1/0/1"), indexOf("GigabitEthernet1/0/2")
	uplink := indexOf("GigabitEthernet1/0/48")

	for _, aggregate := range []string{po1, po2} {
		assertMIBValue(t, agent, ifType+"."+aggregate, interfaceTypeLAG)
		assertMIBValue(t, agent, ifConnectorPresent+"."+aggregate, TruthValueFalse)
		if value := agent.mib.Get(dot3StatsDuplexStatus + "." + aggregate); value != nil {
			t.Errorf("aggregate %s has an Ethernet duplex value: %#v", aggregate, value)
		}
	}
	for _, member := range []string{te1, te2, gi1, gi2, uplink} {
		assertMIBValue(t, agent, ifType+"."+member, interfaceTypeEthernet)
		assertMIBValue(t, agent, ifConnectorPresent+"."+member, TruthValueTrue)
	}
	// An authored member speed counts toward the bundle, not the name's guess.
	assertMIBValue(t, agent, ifHighSpeed+"."+po1, uint32(35_000))
	assertMIBValue(t, agent, ifHighSpeed+"."+po2, uint32(2_000))

	want := []string{
		po1 + "." + te1, po1 + "." + te2,
		po2 + "." + gi1, po2 + "." + gi2,
		"0." + po1, "0." + po2, "0." + uplink,
		te1 + ".0", te2 + ".0", gi1 + ".0", gi2 + ".0", uplink + ".0",
	}
	slices.Sort(want)
	if got := ifStackRows(agent); !slices.Equal(got, want) {
		t.Fatalf("ifStackTable rows = %v, want %v", got, want)
	}
	for _, row := range want {
		assertMIBValue(t, agent, ifStackStatus+"."+row, rowStatusActive)
	}
	assertMIBValue(t, agent, ifStackLastChange, uint32(0))
}

func TestInterfaceWithoutBundleHasNoStackTable(t *testing.T) {
	device := createTestDevice()
	device.TrunkPorts = []config.TrunkPort{{Interface: "GigabitEthernet1/0/1"}}

	agent := NewAgent(device, 0)
	if rows := ifStackRows(agent); len(rows) != 0 {
		t.Fatalf("device without a port-channel publishes ifStackTable rows %v", rows)
	}
	if value := agent.mib.Get(ifStackLastChange); value != nil {
		t.Fatalf("device without a port-channel publishes ifStackLastChange %#v", value)
	}
}

func TestSynthesizedLogicalInterfaceHasNoConnector(t *testing.T) {
	device := createTestDevice()
	device.TrunkPorts = []config.TrunkPort{{Interface: "Vlan10"}, {Interface: "GigabitEthernet1/0/1"}}

	agent := NewAgent(device, 0)
	want := map[string]int{"Vlan10": TruthValueFalse, "GigabitEthernet1/0/1": TruthValueTrue}
	for _, name := range slices.Sorted(maps.Keys(want)) {
		index, ok := agent.InterfaceIndex(name)
		if !ok {
			t.Fatalf("no ifTable row for %s", name)
		}
		assertMIBValue(t, agent, ifConnectorPresent+"."+strconv.Itoa(index), want[name])
	}
}
