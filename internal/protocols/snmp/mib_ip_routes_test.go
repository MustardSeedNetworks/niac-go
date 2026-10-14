package snmp

import (
	"net"
	"testing"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

func TestConfiguredRouteUsesWalkInterfaceIndex(t *testing.T) {
	device := createTestDevice()
	device.Type = "router"
	device.Interfaces = []config.Interface{{
		Name: "GigabitEthernet1/0/48", Network: "transit", Address: "10.254.200.1/24",
	}}
	device.Routes = []config.Route{{
		Destination: "10.240.0.0/16", Via: "GigabitEthernet1/0/48", NextHop: "10.254.200.2",
	}}
	agent := NewAgent(device, 0)

	if err := agent.SetOID(ifName+".48", &OIDValue{
		Type: gosnmp.OctetString, Value: "GigabitEthernet1/0/48",
	}); err != nil {
		t.Fatalf("SetOID(): %v", err)
	}
	agent.registerConfiguredRoutes(device)

	entry := agent.mib.Get(ipRouteIfIndex + ".10.240.0.0")
	if entry == nil || entry.Value != 48 {
		t.Fatalf("ipRouteIfIndex = %#v, want 48", entry)
	}
	mask := agent.mib.Get(ipRouteMask + ".10.240.0.0")
	if mask == nil || !net.ParseIP(mask.Value.(string)).Equal(net.IPv4(255, 255, 0, 0)) {
		t.Fatalf("ipRouteMask = %#v, want 255.255.0.0", mask)
	}
	nextHop := agent.mib.Get(ipRouteNextHop + ".10.240.0.0")
	if nextHop == nil || nextHop.Value != "10.254.200.2" {
		t.Fatalf("ipRouteNextHop = %#v, want 10.254.200.2", nextHop)
	}
}

// ipRouteTable is IPv4: an IPv6 static route writes no row under its address.
func TestConfiguredIPv6RouteHasNoIPRouteTableRow(t *testing.T) {
	device := createTestDevice()
	device.Type = "router"
	device.Interfaces = []config.Interface{{
		Name: "GigabitEthernet1/0/48", Network: "transit", Address: "10.254.200.1/24",
		AddressV6: "fd00:254:200::1/64",
	}}
	device.Routes = []config.Route{{
		Destination: "fd00:240::/48", Via: "GigabitEthernet1/0/48", NextHop: "fd00:254:200::2",
	}}
	agent := NewAgent(device, 0)
	if err := agent.SetOID(ifName+".48", &OIDValue{
		Type: gosnmp.OctetString, Value: "GigabitEthernet1/0/48",
	}); err != nil {
		t.Fatalf("SetOID(): %v", err)
	}

	agent.registerConfiguredRoutes(device)

	if entry := agent.mib.Get(ipRouteDest + ".fd00:240::"); entry != nil {
		t.Fatalf("ipRouteDest.fd00:240:: = %#v, want no row", entry)
	}
}
