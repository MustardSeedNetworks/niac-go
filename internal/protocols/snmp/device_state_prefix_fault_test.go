package snmp

import (
	"net/netip"
	"reflect"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestPrefixFaultIPMIBProjectionAndClear(t *testing.T) {
	device := createTestDevice()
	device.Interfaces = []config.Interface{
		{Name: "eth0", Address: "192.0.2.10/24"},
		{Name: "eth1", Address: "198.51.100.10/24"},
	}
	state := devicestate.NewStore(devicestate.Identity{Hostname: device.Name})
	state.ReplaceNetwork(devicestate.Network{
		Interfaces: []devicestate.Interface{
			{Name: "eth0", Address: netip.MustParsePrefix("192.0.2.10/24"), AdminUp: true, OperUp: true},
			{Name: "eth1", Address: netip.MustParsePrefix("198.51.100.10/24"), AdminUp: true, OperUp: true},
		},
		Routes: []devicestate.Route{
			{Destination: netip.MustParsePrefix("192.0.2.0/24"), Via: "eth0", Connected: true},
			{Destination: netip.MustParsePrefix("198.51.100.0/24"), Via: "eth1", Connected: true},
			{Destination: netip.MustParsePrefix("0.0.0.0/0"), Via: "eth0", NextHop: netip.MustParseAddr("192.0.2.1")},
		},
	})
	before := state.Snapshot().Network
	agent := NewAgentWithState(device, state, AgentOptions{})
	assertValue := func(oid, want string) {
		t.Helper()
		got := agent.mib.Get(oid)
		if got == nil || got.Value != want {
			t.Fatalf("%s = %#v, want %s", oid, got, want)
		}
	}
	assertValue(ipAdEntNetMask+".192.0.2.10", "255.255.255.0")
	if err := state.SetInterfacePrefixFault(devicestate.InterfacePrefixFault{
		Interface: "eth0", Type: devicestate.FaultBadMask, PrefixBits: 16,
	}); err != nil {
		t.Fatal(err)
	}
	agent.syncDeviceStateMIBs()
	assertValue(ipAdEntNetMask+".192.0.2.10", "255.255.0.0")
	assertValue(ipRouteMask+".192.0.0.0", "255.255.0.0")
	if got := agent.mib.Get(ipRouteMask + ".192.0.2.0"); got != nil {
		t.Fatalf("stale connected route remains: %#v", got)
	}
	assertValue(ipAdEntNetMask+".198.51.100.10", "255.255.255.0")
	assertValue(ipRouteMask+".198.51.100.0", "255.255.255.0")
	assertValue(ipRouteNextHop+".0.0.0.0", "192.0.2.1")
	if !reflect.DeepEqual(state.Snapshot().Network, before) {
		t.Fatal("SNMP projection changed canonical network")
	}
	if err := state.ClearInterfacePrefixFault("eth0", devicestate.FaultBadMask); err != nil {
		t.Fatal(err)
	}
	agent.syncDeviceStateMIBs()
	assertValue(ipAdEntNetMask+".192.0.2.10", "255.255.255.0")
	assertValue(ipRouteMask+".192.0.2.0", "255.255.255.0")
	if got := agent.mib.Get(ipRouteMask + ".192.0.0.0"); got != nil {
		t.Fatalf("fault route remains after clear: %#v", got)
	}
}
