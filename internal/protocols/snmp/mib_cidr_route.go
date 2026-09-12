package snmp

import (
	"net"
	"strconv"
	"strings"

	"github.com/gosnmp/gosnmp"
)

// ipCidrRouteTable (RFC 2096), the routing table a real consumer reads.
//
// NIAC served the deprecated ipRouteTable and RFC 4292's inetCidrRouteTable,
// and seed's routing collector walks neither: it walks this one, and says so
// in its own package doc — "V1.0 uses ipCidrRouteTable (RFC 2096) which is
// IPv4-only but universally implemented. The newer inetCidrRouteTable
// (RFC 4292, dual-stack) lands when an IPv6 customer asks for it."
//
// So a replayed router appeared to have no routes, which is what blocked G4's
// cross-product acceptance. All three tables are written from one walk over
// the authored routes, so no consumer can be told a different story depending
// on which it happens to read.

// ipCidrRouteTos is the type-of-service every authored route carries. RFC 2096
// makes it part of the INDEX, and a scenario does not author per-TOS routes.
const ipCidrRouteDefaultTos = 0

// registerIPCidrRoute publishes one route in the RFC 2096 table. routeType and
// proto arrive in ipRouteTable's vocabulary, which RFC 2096 shares.
func (a *Agent) registerIPCidrRoute(
	destination net.IP,
	mask net.IPMask,
	ifIndex int,
	nextHop string,
	routeType int,
	proto int,
) {
	if nextHop == "" {
		nextHop = unspecifiedIPv4
	}
	maskAddress := net.IP(mask).String()
	index := cidrRouteIndex(destination, mask, nextHop)

	// The four index columns are accessible here, unlike RFC 4292's, and a
	// real device returns them, so a sweep that reads values rather than
	// parsing the index still resolves the route.
	a.mib.Set(ipCidrRouteDest+index, &OIDValue{
		Type: gosnmp.IPAddress, Value: destination.String(),
	})
	a.mib.Set(ipCidrRouteMask+index, &OIDValue{Type: gosnmp.IPAddress, Value: maskAddress})
	a.mib.Set(ipCidrRouteTos+index, &OIDValue{
		Type: gosnmp.Integer, Value: ipCidrRouteDefaultTos,
	})
	a.mib.Set(ipCidrRouteNextHop+index, &OIDValue{Type: gosnmp.IPAddress, Value: nextHop})

	a.mib.Set(ipCidrRouteIfIndex+index, &OIDValue{Type: gosnmp.Integer, Value: ifIndex})
	a.mib.Set(ipCidrRouteType+index, &OIDValue{Type: gosnmp.Integer, Value: routeType})
	a.mib.Set(ipCidrRouteProto+index, &OIDValue{Type: gosnmp.Integer, Value: proto})
	a.mib.Set(ipCidrRouteAge+index, &OIDValue{Type: gosnmp.Integer, Value: 0})
	a.mib.Set(ipCidrRouteInfo+index, &OIDValue{
		Type: gosnmp.ObjectIdentifier, Value: "0.0",
	})
	// Zero is RFC 2096's "not known", which is the truth for an authored route.
	a.mib.Set(ipCidrRouteNextHopAS+index, &OIDValue{Type: gosnmp.Integer, Value: 0})
	a.mib.Set(ipCidrRouteMetric1+index, &OIDValue{Type: gosnmp.Integer, Value: 1})
	for _, column := range []string{
		ipCidrRouteMetric2, ipCidrRouteMetric3, ipCidrRouteMetric4, ipCidrRouteMetric5,
	} {
		a.mib.Set(column+index, &OIDValue{Type: gosnmp.Integer, Value: unusedRouteMetric})
	}
	a.mib.Set(ipCidrRouteStatus+index, &OIDValue{Type: gosnmp.Integer, Value: rowStatusActive})
}

// cidrRouteIndex builds RFC 2096's INDEX: destination, mask, type-of-service
// and next hop, each IPv4 address as four bare sub-identifiers. Thirteen
// fields, which is exactly what seed's parseRouteOID expects.
func cidrRouteIndex(destination net.IP, mask net.IPMask, nextHop string) string {
	var index strings.Builder
	index.WriteString(ipv4IndexOctets(destination))
	index.WriteString(ipv4IndexOctets(net.IP(mask)))
	index.WriteString(".")
	index.WriteString(strconv.Itoa(ipCidrRouteDefaultTos))
	index.WriteString(ipv4IndexOctets(net.ParseIP(nextHop)))

	return index.String()
}

// ipv4IndexOctets renders one IPv4 address as four leading-dot
// sub-identifiers. An address that is not IPv4 is rendered unspecified rather
// than dropped, so the row keeps a well-formed index.
func ipv4IndexOctets(address net.IP) string {
	v4 := address.To4()
	if v4 == nil {
		v4 = net.IPv4zero.To4()
	}
	var octets strings.Builder
	for _, octet := range v4 {
		octets.WriteString(".")
		octets.WriteString(strconv.Itoa(int(octet)))
	}

	return octets.String()
}
