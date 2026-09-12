package snmp

import (
	"strings"
	"testing"
)

// ipCidrRouteTable (RFC 2096) is the table seed's routing collector actually
// walks — `internal/polling/snmp/collectors/routing` has tablePrefix
// "1.3.6.1.2.1.4.24.4.1" and expects thirteen index fields,
// `<4 dest>.<4 mask>.<tos>.<4 nextHop>`. NIAC served .21 and .24.7 and nothing
// at .24.4, so a replayed router looked routeless to the one consumer that
// matters.
//
// The index is asserted literally rather than through a helper, because a
// helper that built it the same way the production code does could not detect
// the two disagreeing.
func TestIPCidrRouteTableMatchesTheConsumersIndex(t *testing.T) {
	agent := routeTestAgent(t)

	// 0.0.0.0/0 via 10.254.200.30: dest, mask, tos, next hop.
	defaultIndex := ".0.0.0.0.0.0.0.0.0.10.254.200.30"
	// The connected 10.254.200.0/27, which has no next hop.
	connectedIndex := ".10.254.200.0.255.255.255.224.0.0.0.0.0"

	rows := []struct {
		oid  string
		want any
	}{
		{ipCidrRouteDest + defaultIndex, "0.0.0.0"},
		{ipCidrRouteMask + defaultIndex, "0.0.0.0"},
		{ipCidrRouteTos + defaultIndex, ipCidrRouteDefaultTos},
		{ipCidrRouteNextHop + defaultIndex, "10.254.200.30"},
		{ipCidrRouteIfIndex + defaultIndex, 10048},
		{ipCidrRouteType + defaultIndex, IPRouteTypeIndirect},
		{ipCidrRouteProto + defaultIndex, IPRouteProtoNetMgmt},
		{ipCidrRouteMetric1 + defaultIndex, 1},
		{ipCidrRouteStatus + defaultIndex, rowStatusActive},

		{ipCidrRouteDest + connectedIndex, "10.254.200.0"},
		{ipCidrRouteMask + connectedIndex, "255.255.255.224"},
		{ipCidrRouteNextHop + connectedIndex, "0.0.0.0"},
		{ipCidrRouteType + connectedIndex, IPRouteTypeDirect},
		{ipCidrRouteProto + connectedIndex, IPRouteProtoLocal},
	}
	for _, row := range rows {
		t.Run(row.oid, func(t *testing.T) { assertMIBValue(t, agent, row.oid, row.want) })
	}
}

// Thirteen index fields is what seed's parseRouteOID requires; twelve or
// fourteen and it discards the row without saying so.
func TestIPCidrRouteIndexIsThirteenFields(t *testing.T) {
	agent := routeTestAgent(t)

	found := 0
	agent.mib.mu.RLock()
	oids := make([]string, 0, len(agent.mib.entries))
	for oid := range agent.mib.entries {
		oids = append(oids, oid)
	}
	agent.mib.mu.RUnlock()

	for _, oid := range oids {
		if !strings.HasPrefix(oid, ipCidrRouteIfIndex+".") {
			continue
		}
		found++
		index := strings.TrimPrefix(oid, ipCidrRouteIfIndex+".")
		if fields := len(strings.Split(index, ".")); fields != 13 {
			t.Errorf("%s has %d index fields, want 13", oid, fields)
		}
	}
	if found == 0 {
		t.Fatal("no ipCidrRouteTable rows at all; the consumer sees no routes")
	}
}
