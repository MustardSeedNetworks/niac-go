package fabric

import (
	"errors"
	"net/netip"
)

// StaticRouteSpec is an authored or runtime-mutated static route. An IPv6
// destination leaves through the egress interface's address_v6.
type StaticRouteSpec struct {
	Device      string
	Destination string
	Via         string
	NextHop     string
}

// RouteValidationContext contains the current interfaces and address owners.
type RouteValidationContext struct {
	Interfaces    map[string]Interface
	AddressOwners map[netip.Addr]string
}

// ValidateStaticRoute applies the canonical routed-scenario invariants.
func ValidateStaticRoute(spec StaticRouteSpec, context RouteValidationContext) (Route, *Diagnostic) {
	destination, err := parseRouteDestination(spec.Destination)
	if err != nil {
		return Route{}, routeDiagnostic(CodeInvalidRoute, "destination", err.Error())
	}
	iface, exists := context.Interfaces[spec.Via]
	if !exists {
		return Route{}, routeDiagnostic(
			CodeUnknownRouteInterface,
			"via",
			"route references an unknown interface",
		)
	}
	nextHop, err := netip.ParseAddr(spec.NextHop)
	if err != nil || nextHop.Is4In6() || nextHop.Is6() != destination.Addr().Is6() {
		return Route{}, routeDiagnostic(
			CodeInvalidRouteNextHop,
			"next_hop",
			"next hop must be an address of the destination's family",
		)
	}
	egress := iface.Address
	if nextHop.Is6() {
		egress = iface.AddressV6
	}
	if !egress.IsValid() || !egress.Masked().Contains(nextHop) || isReservedNextHop(egress.Masked(), nextHop) {
		return Route{}, routeDiagnostic(
			CodeRouteNextHopOffLink,
			"next_hop",
			"next hop must be a usable address on the egress network",
		)
	}
	owner, assigned := context.AddressOwners[nextHop]
	if !assigned {
		return Route{}, routeDiagnostic(
			CodeUnknownRouteNextHop,
			"next_hop",
			"next hop is not assigned to a configured peer",
		)
	}
	if owner == spec.Device {
		return Route{}, routeDiagnostic(
			CodeRouteNextHopSelf,
			"next_hop",
			"next hop cannot belong to the routed device",
		)
	}
	return Route{
		Device: spec.Device, Destination: destination, Via: spec.Via, NextHop: nextHop,
	}, nil
}

// parseRouteDestination accepts a canonical IPv4 prefix, or an IPv6 one that is
// the default route or global or unique-local unicast.
func parseRouteDestination(value string) (netip.Prefix, error) {
	prefix, err := netip.ParsePrefix(value)
	if err != nil || !prefix.Addr().Is6() || prefix.Addr().Is4In6() {
		return parseCanonicalPrefix(value)
	}
	if prefix != prefix.Masked() {
		return netip.Prefix{}, errors.New("prefix must use its network address")
	}
	if prefix.Bits() != 0 && !prefix.Addr().IsGlobalUnicast() {
		return netip.Prefix{}, errors.New("prefix must be global or unique-local unicast")
	}
	return prefix, nil
}

func isReservedNextHop(prefix netip.Prefix, address netip.Addr) bool {
	if address.Is4() {
		return isReservedEndpoint(prefix, address)
	}
	return prefix.Bits() < maxAnycastPrefixLenV6 && address == prefix.Addr()
}

func routeDiagnostic(code DiagnosticCode, field, message string) *Diagnostic {
	return &Diagnostic{Code: code, Field: field, Message: message}
}
