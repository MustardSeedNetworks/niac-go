package snmp

import (
	"testing"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// inetCidrRouteTable (RFC 4292), which is what a modern manager reads for path
// analysis. NIAC published only the deprecated ipRouteTable, so a consumer
// that asks for the current table saw a router with no routes at all.
//
// The index encoding is copied from a real agent rather than derived from the
// MIB text, because an index a manager cannot parse is the same as no table.
// cisco-sg500-28-28-port-01 in the walk corpus answers:
//
//	.1.3.6.1.2.1.4.24.7.1.8.1.4.0.0.0.0.0.5.0.0.0.0.0.1.4.192.168.201.1
//	                        │ └ dest 0.0.0.0    │ └ policy   │ └ next hop
//	                        └ destType ipv4     └ pfxLen 0   └ nextHopType
//
// so each InetAddress is length-prefixed, the policy OID carries its own
// sub-identifier count, and the prefix length sits between them.
func routeTestAgent(t *testing.T) *Agent {
	t.Helper()
	device := createTestDevice()
	device.Type = "router"
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

	return agent
}

func TestInetCidrRouteTableIndexesLikeARealAgent(t *testing.T) {
	agent := routeTestAgent(t)

	// The authored static default route, via the authored next hop.
	defaultIndex := ".1.4.0.0.0.0.0.5.0.0.0.0.0.1.4.10.254.200.30"
	// The connected route the interface address implies. A connected route has
	// no next hop, which RFC 4292 spells as the unspecified address.
	connectedIndex := ".1.4.10.254.200.0.27.5.0.0.0.0.0.1.4.0.0.0.0"

	rows := []struct {
		oid  string
		want any
	}{
		{inetCidrRouteIfIndex + defaultIndex, 10048},
		{inetCidrRouteType + defaultIndex, InetCidrRouteTypeRemote},
		{inetCidrRouteProto + defaultIndex, IPRouteProtoNetMgmt},
		{inetCidrRouteNextHopAS + defaultIndex, uint(0)},
		{inetCidrRouteMetric1 + defaultIndex, 1},
		{inetCidrRouteStatus + defaultIndex, rowStatusActive},

		{inetCidrRouteIfIndex + connectedIndex, 10048},
		{inetCidrRouteType + connectedIndex, InetCidrRouteTypeLocal},
		{inetCidrRouteProto + connectedIndex, IPRouteProtoLocal},
	}
	for _, row := range rows {
		t.Run(row.oid, func(t *testing.T) { assertMIBValue(t, agent, row.oid, row.want) })
	}
}

// RFC 4292 marks the six index columns not-accessible, and a real agent
// returns only 7..17. Publishing the index columns as rows would make a sweep
// report objects the MIB says cannot exist.
func TestInetCidrRouteTableOmitsItsIndexColumns(t *testing.T) {
	agent := routeTestAgent(t)
	defaultIndex := ".1.4.0.0.0.0.0.5.0.0.0.0.0.1.4.10.254.200.30"

	for _, column := range []string{
		inetCidrRouteEntry + ".1", inetCidrRouteEntry + ".2", inetCidrRouteEntry + ".3",
		inetCidrRouteEntry + ".4", inetCidrRouteEntry + ".5", inetCidrRouteEntry + ".6",
	} {
		if entry := agent.mib.Get(column + defaultIndex); entry != nil {
			t.Errorf("%s is published; RFC 4292 marks it not-accessible", column)
		}
	}
}

// The deprecated table stays: CyberScope and other older managers read it, and
// removing it to add the modern one would trade one blind consumer for another.
func TestLegacyRouteTableSurvivesTheModernOne(t *testing.T) {
	agent := routeTestAgent(t)

	assertMIBValue(t, agent, ipRouteIfIndex+".0.0.0.0", 10048)
	assertMIBValue(t, agent, ipRouteNextHop+".0.0.0.0", "10.254.200.30")
}
