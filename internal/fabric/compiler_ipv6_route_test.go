package fabric_test

import (
	"net/netip"
	"slices"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

// dualStackRoutedConfig gives the reference router an IPv6 static route
// beside its IPv4 one, through the evt peer's address_v6.
func dualStackRoutedConfig() *config.Config {
	cfg := dualStackConfig()
	cfg.Devices[2].Interfaces[0].AddressV6 = "fd00:20:210::2/64"
	cfg.Devices[0].Routes = append(cfg.Devices[0].Routes, config.Route{
		Destination: "fd00:20::/48", Via: "evt", NextHop: "fd00:20:210::2",
	})
	return cfg
}

func TestCompileAcceptsIPv6StaticRoutes(t *testing.T) {
	for _, tt := range []struct {
		name        string
		destination string
	}{
		{"site prefix", "fd00:20::/48"},
		{"default route", "::/0"},
		{"host route", "2001:db8:99::1/128"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := dualStackRoutedConfig()
			cfg.Devices[0].Routes[1].Destination = tt.destination

			report := fabric.Compile(cfg, accessBinding())
			if !report.Safe {
				t.Fatalf("Compile() diagnostics = %#v", report.Diagnostics)
			}
			want := fabric.Route{
				Device: "LAB-EDGE-R1", Destination: netip.MustParsePrefix(tt.destination),
				Via: "evt", NextHop: netip.MustParseAddr("fd00:20:210::2"),
			}
			if !slices.Contains(report.Topology.Routes, want) {
				t.Fatalf("Topology.Routes = %#v, want %#v", report.Topology.Routes, want)
			}
		})
	}
}

func TestCompileRejectsInvalidIPv6RouteSemantics(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*config.Route)
		code   fabric.DiagnosticCode
		field  string
	}{
		{
			name:   "destination not at its network address",
			mutate: func(route *config.Route) { route.Destination = "fd00:20::1/48" },
			code:   fabric.CodeInvalidRoute,
			field:  "destination",
		},
		{
			name:   "destination is link-local",
			mutate: func(route *config.Route) { route.Destination = "fe80::/64" },
			code:   fabric.CodeInvalidRoute,
			field:  "destination",
		},
		{
			name:   "destination is multicast",
			mutate: func(route *config.Route) { route.Destination = "ff05::/16" },
			code:   fabric.CodeInvalidRoute,
			field:  "destination",
		},
		{
			name:   "IPv4 next hop for an IPv6 destination",
			mutate: func(route *config.Route) { route.NextHop = "10.20.210.2" },
			code:   fabric.CodeInvalidRouteNextHop,
			field:  "next_hop",
		},
		{
			name:   "IPv4-mapped next hop",
			mutate: func(route *config.Route) { route.NextHop = "::ffff:10.20.210.2" },
			code:   fabric.CodeInvalidRouteNextHop,
			field:  "next_hop",
		},
		{
			name: "IPv6 next hop for an IPv4 destination",
			mutate: func(route *config.Route) {
				route.Destination = "10.30.0.0/16"
			},
			code:  fabric.CodeInvalidRouteNextHop,
			field: "next_hop",
		},
		{
			name:   "next hop outside the egress IPv6 network",
			mutate: func(route *config.Route) { route.Via = "outside" },
			code:   fabric.CodeRouteNextHopOffLink,
			field:  "next_hop",
		},
		{
			name:   "next hop is the subnet-router anycast address",
			mutate: func(route *config.Route) { route.NextHop = "fd00:20:210::" },
			code:   fabric.CodeRouteNextHopOffLink,
			field:  "next_hop",
		},
		{
			name:   "next hop is not configured",
			mutate: func(route *config.Route) { route.NextHop = "fd00:20:210::99" },
			code:   fabric.CodeUnknownRouteNextHop,
			field:  "next_hop",
		},
		{
			name:   "next hop belongs to the routed device",
			mutate: func(route *config.Route) { route.NextHop = "fd00:20:210::1" },
			code:   fabric.CodeRouteNextHopSelf,
			field:  "next_hop",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := dualStackRoutedConfig()
			tt.mutate(&cfg.Devices[0].Routes[1])

			assertIPv6RouteDiagnostic(t, fabric.Compile(cfg, accessBinding()), tt.code, tt.field)
		})
	}
}

func TestCompileRejectsIPv6RouteThroughAnIPv4OnlyInterface(t *testing.T) {
	cfg := dualStackRoutedConfig()
	cfg.Devices[0].Interfaces[1].AddressV6 = ""

	assertIPv6RouteDiagnostic(t, fabric.Compile(cfg, accessBinding()), fabric.CodeRouteNextHopOffLink, "next_hop")
}

func assertIPv6RouteDiagnostic(t *testing.T, report fabric.Report, code fabric.DiagnosticCode, field string) {
	t.Helper()
	want := "devices[LAB-EDGE-R1].routes[1]." + field
	for _, diagnostic := range report.Diagnostics {
		if diagnostic.Code == code && diagnostic.Field == want {
			return
		}
	}
	t.Fatalf("Compile() diagnostics = %#v, want %s at %s", report.Diagnostics, code, want)
}
