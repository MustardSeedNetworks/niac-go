package snmp

import (
	"net"
	"testing"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

func TestIPMIBMandatoryScalars(t *testing.T) {
	device := createTestDevice()
	device.Type = "router"
	agent := NewAgent(device, 0)

	tests := []struct {
		oid  string
		want any
	}{
		{ipForwarding, ipForwardingEnabled},
		{ipDefaultTTL, defaultIPTTL},
		{ipInReceives, uint32(0)},
		{ipInHdrErrors, uint32(0)},
		{ipInAddrErrors, uint32(0)},
		{ipForwDatagrams, uint32(0)},
		{ipInUnknownProtos, uint32(0)},
		{ipInDiscards, uint32(0)},
		{ipInDelivers, uint32(0)},
		{ipOutRequests, uint32(0)},
		{ipOutDiscards, uint32(0)},
		{ipOutNoRoutes, uint32(0)},
		{ipReasmTimeout, defaultReasmTimeout},
		{ipReasmReqds, uint32(0)},
		{ipReasmOKs, uint32(0)},
		{ipReasmFails, uint32(0)},
		{ipFragOKs, uint32(0)},
		{ipFragFails, uint32(0)},
		{ipFragCreates, uint32(0)},
	}
	for _, test := range tests {
		t.Run(test.oid, func(t *testing.T) { assertMIBValue(t, agent, test.oid, test.want) })
	}
}

func TestIPTablesUseAuthoredInterfaceIdentity(t *testing.T) {
	device := createTestDevice()
	device.Type = "router"
	device.SNMPConfig.WalkFile = "router.walk"
	device.Interfaces = []config.Interface{{
		Name: "GigabitEthernet1/0/48", Address: "10.254.200.1/27",
	}}
	device.Routes = []config.Route{{
		Destination: "0.0.0.0/0", Via: "GigabitEthernet1/0/48", NextHop: "10.254.200.30",
	}}
	agent := NewAgent(device, 0)
	if err := agent.SetOID(ifName+".10048", &OIDValue{
		Type: gosnmp.OctetString, Value: "GigabitEthernet1/0/48",
	}); err != nil {
		t.Fatal(err)
	}
	agent.registerConfiguredRoutes(device)

	addressRows := []struct {
		oid  string
		want any
	}{
		{ipAdEntAddr + ".10.254.200.1", "10.254.200.1"},
		{ipAdEntIfIndex + ".10.254.200.1", 10048},
		{ipAdEntNetMask + ".10.254.200.1", "255.255.255.224"},
		{ipAdEntBcastAddr + ".10.254.200.1", 1},
		{ipAdEntReasmMaxSize + ".10.254.200.1", MaxIPPort},
	}
	for _, row := range addressRows {
		t.Run(row.oid, func(t *testing.T) { assertMIBValue(t, agent, row.oid, row.want) })
	}

	columns := []string{
		ipRouteDest, ipRouteIfIndex, ipRouteMetric1, ipRouteMetric2, ipRouteMetric3,
		ipRouteMetric4, ipRouteNextHop, ipRouteType, ipRouteProto, ipRouteAge,
		ipRouteMask, ipRouteInfo, ipRouteMetric5,
	}
	for _, column := range columns {
		if entry := agent.mib.Get(column + ".0.0.0.0"); entry == nil {
			t.Errorf("route column %s is absent", column)
		}
	}
	assertMIBValue(t, agent, ipRouteIfIndex+".0.0.0.0", 10048)
	assertMIBValue(t, agent, ipRouteNextHop+".0.0.0.0", "10.254.200.30")
	assertMIBValue(t, agent, ipRouteMetric5+".0.0.0.0", unusedRouteMetric)
}

func TestARPEntryPopulatesCurrentAndLegacyTables(t *testing.T) {
	agent := NewAgent(createTestDevice(), 0)
	mac, _ := net.ParseMAC("02:00:00:00:00:2a")
	agent.registerARPEntry(10048, net.ParseIP("10.254.200.30"), mac)

	assertMIBValue(t, agent, ipNetToMediaIfIndex+".10048.10.254.200.30", 10048)
	assertMIBValue(t, agent, ipNetToMediaNetAddress+".10048.10.254.200.30", "10.254.200.30")
	assertMIBValue(t, agent, atIfIndex+".10048.1.10.254.200.30", 10048)
	assertMIBValue(t, agent, atNetAddress+".10048.1.10.254.200.30", "10.254.200.30")
}

func TestARPTableDoesNotInventTopologyPeers(t *testing.T) {
	device := createTestDevice()
	device.TrunkPorts = []config.TrunkPort{{Interface: "eth0", RemoteDevice: "peer"}}
	agent := NewAgent(device, 0)
	for _, oid := range agent.mib.AllOIDs() {
		if oidHasPrefix(oid, ipNetToMediaTable) || oidHasPrefix(oid, atEntry) {
			t.Fatalf("invented ARP row %s", oid)
		}
	}
}

func oidHasPrefix(oid, prefix string) bool {
	return oid == prefix || len(oid) > len(prefix) && oid[:len(prefix)+1] == prefix+"."
}
