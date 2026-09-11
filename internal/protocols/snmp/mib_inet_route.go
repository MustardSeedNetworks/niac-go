package snmp

import (
	"net"
	"strconv"
	"strings"

	"github.com/gosnmp/gosnmp"
)

// inetCidrRouteTable (RFC 4292) alongside the deprecated ipRouteTable.
//
// A manager doing path analysis reads the current table; NIAC published only
// the deprecated one, so a consumer that asked for routes the modern way saw a
// router with none. Both are served from the same authored routes, so they
// cannot disagree — the legacy table stays because CyberScope and other older
// managers still read it.

// unspecifiedIPv4 is RFC 4292's next hop for a route that has none: a
// connected subnet is reached without one, and the index still has to carry an
// address.
const unspecifiedIPv4 = "0.0.0.0"

// inetCidrRouteIndex builds the table's INDEX, which is six objects deep:
// destination type, destination, prefix length, policy, next-hop type and next
// hop. The InetAddress values are length-prefixed and the policy OID carries
// its own sub-identifier count, so the result looks like
//
//	.1.4.10.254.200.0.27.5.0.0.0.0.0.1.4.0.0.0.0
//
// which is the shape cisco-sg500-28-28-port-01 answers with in the walk
// corpus. Deriving it from the MIB text instead would have been guesswork: an
// index a manager cannot parse is the same as no table at all.
func inetCidrRouteIndex(destination net.IP, prefixLength int, nextHop string) string {
	var index strings.Builder
	index.WriteString(".")
	index.WriteString(strconv.Itoa(inetAddressTypeIPv4))
	index.WriteString(inetAddressIndex(destination))
	index.WriteString(".")
	index.WriteString(strconv.Itoa(prefixLength))
	// inetCidrRoutePolicy. Agents that do not distinguish routes by policy
	// report zeroDotZero, which as an index is five zero sub-identifiers.
	index.WriteString(".5.0.0.0.0.0.")
	index.WriteString(strconv.Itoa(inetAddressTypeIPv4))
	index.WriteString(inetAddressIndex(net.ParseIP(nextHop)))

	return index.String()
}

// inetAddressIndex renders one InetAddress as an index: its octet count, then
// the octets. An address that does not parse as IPv4 is rendered unspecified
// rather than dropped, so the row keeps a well-formed index.
func inetAddressIndex(address net.IP) string {
	v4 := address.To4()
	if v4 == nil {
		v4 = net.IPv4zero.To4()
	}
	octets := make([]string, 0, len(v4)+1)
	octets = append(octets, strconv.Itoa(len(v4)))
	for _, octet := range v4 {
		octets = append(octets, strconv.Itoa(int(octet)))
	}

	return "." + strings.Join(octets, ".")
}

// registerInetCidrRoute publishes one route in the current table. routeType and
// proto arrive in ipRouteTable's vocabulary and are translated here, so callers
// describe a route once.
func (a *Agent) registerInetCidrRoute(
	destination net.IP,
	mask net.IPMask,
	ifIndex int,
	nextHop string,
	routeType int,
	proto int,
) {
	prefixLength, _ := mask.Size()
	if nextHop == "" {
		nextHop = unspecifiedIPv4
	}
	index := inetCidrRouteIndex(destination, prefixLength, nextHop)

	a.mib.Set(inetCidrRouteIfIndex+index, &OIDValue{Type: gosnmp.Integer, Value: ifIndex})
	a.mib.Set(inetCidrRouteType+index, &OIDValue{
		Type: gosnmp.Integer, Value: inetCidrRouteTypeFor(routeType),
	})
	a.mib.Set(inetCidrRouteProto+index, &OIDValue{Type: gosnmp.Integer, Value: proto})
	a.mib.Set(inetCidrRouteAge+index, &OIDValue{Type: gosnmp.Gauge32, Value: uint(0)})
	// The next hop's autonomous system. Zero is RFC 4292's value for "not
	// known", which is the truth for an authored route.
	a.mib.Set(inetCidrRouteNextHopAS+index, &OIDValue{Type: gosnmp.Gauge32, Value: uint(0)})
	a.mib.Set(inetCidrRouteMetric1+index, &OIDValue{Type: gosnmp.Integer, Value: 1})
	for _, column := range []string{
		inetCidrRouteMetric2, inetCidrRouteMetric3, inetCidrRouteMetric4, inetCidrRouteMetric5,
	} {
		a.mib.Set(column+index, &OIDValue{Type: gosnmp.Integer, Value: unusedRouteMetric})
	}
	a.mib.Set(inetCidrRouteStatus+index, &OIDValue{Type: gosnmp.Integer, Value: rowStatusActive})
}

// inetCidrRouteTypeFor maps ipRouteTable's route type onto RFC 4292's. The two
// vocabularies differ in name and agree in number for the cases NIAC
// publishes: direct is local, indirect is remote.
func inetCidrRouteTypeFor(legacy int) int {
	if legacy == IPRouteTypeDirect {
		return InetCidrRouteTypeLocal
	}

	return InetCidrRouteTypeRemote
}
