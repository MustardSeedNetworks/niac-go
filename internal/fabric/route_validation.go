package fabric

import "net/netip"

// StaticRouteSpec is an authored or runtime-mutated IPv4 static route.
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
	destination, err := parseCanonicalPrefix(spec.Destination)
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
	if err != nil || !nextHop.Is4() {
		return Route{}, routeDiagnostic(
			CodeInvalidRouteNextHop,
			"next_hop",
			"next hop must be an IPv4 address",
		)
	}
	if !iface.Address.Masked().Contains(nextHop) || isReservedEndpoint(iface.Address.Masked(), nextHop) {
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

func routeDiagnostic(code DiagnosticCode, field, message string) *Diagnostic {
	return &Diagnostic{Code: code, Field: field, Message: message}
}
