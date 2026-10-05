package fabric_test

import (
	"net"
	"net/netip"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

// relayedDHCPConfig serves evt-data from LAB-DHCP01 through LAB-EDGE-R1's
// relay on that network.
func relayedDHCPConfig() *config.Config {
	cfg := referenceConfig()
	cfg.Devices[0].Interfaces[1].DHCPRelay = "10.10.200.2"
	cfg.Devices[1].DHCPConfig.Scopes = []config.DHCPScope{{
		PoolStart: net.ParseIP("10.20.210.100"),
		PoolEnd:   net.ParseIP("10.20.210.150"),
		Router:    net.ParseIP("10.20.210.1"),
	}}
	return cfg
}

func TestCompileBindsRelayedDHCPScopeToItsRelay(t *testing.T) {
	report := fabric.Compile(relayedDHCPConfig(), accessBinding())
	if !report.Safe {
		t.Fatalf("Compile() diagnostics = %#v", report.Diagnostics)
	}
	want := fabric.DHCPScope{
		Device: "LAB-DHCP01", Network: "evt-data",
		Start: netip.MustParseAddr("10.20.210.100"), End: netip.MustParseAddr("10.20.210.150"),
		Router: netip.MustParseAddr("10.20.210.1"),
		Relay: fabric.DHCPRelay{
			Device: "LAB-EDGE-R1", Interface: "evt",
			Address: netip.MustParseAddr("10.20.210.1"), Server: netip.MustParseAddr("10.10.200.2"),
		},
	}
	for _, scope := range report.Topology.DHCPScopes {
		if scope.Network == "evt-data" {
			if scope != want {
				t.Fatalf("relayed scope = %#v, want %#v", scope, want)
			}
			return
		}
	}
	t.Fatalf("DHCPScopes = %#v, want the relayed evt-data scope", report.Topology.DHCPScopes)
}

func TestCompileRejectsInvalidRelayedDHCP(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*config.Config)
		code   fabric.DiagnosticCode
	}{
		{
			name:   "scope with no relay",
			mutate: func(cfg *config.Config) { cfg.Devices[0].Interfaces[1].DHCPRelay = "" },
			code:   fabric.CodeInvalidDHCPRelay,
		},
		{
			name: "relay to an address that serves no scope",
			mutate: func(cfg *config.Config) {
				cfg.Devices[0].Interfaces[1].DHCPRelay = "10.10.200.99"
				cfg.Devices[1].DHCPConfig.Scopes = nil
			},
			code: fabric.CodeInvalidDHCPRelay,
		},
		{
			name:   "relay target is not IPv4",
			mutate: func(cfg *config.Config) { cfg.Devices[0].Interfaces[1].DHCPRelay = "2001:db8::2" },
			code:   fabric.CodeInvalidDHCPRelay,
		},
		{
			name: "relay on a device that does not route",
			mutate: func(cfg *config.Config) {
				cfg.Devices = append(cfg.Devices, config.Device{
					Name: "EVT-SW01", Type: "switch",
					Interfaces: []config.Interface{{
						Name: "Vlan210", Network: "evt-data", Address: "10.20.210.9/24", DHCPRelay: "10.10.200.2",
					}},
				})
			},
			code: fabric.CodeInvalidDHCPRelay,
		},
		{
			name: "scope on the server's own network",
			mutate: func(cfg *config.Config) {
				cfg.Devices[0].Interfaces[0].DHCPRelay = "10.10.200.2"
				cfg.Devices[0].Interfaces[1].DHCPRelay = ""
				cfg.Devices[1].DHCPConfig.Scopes[0] = config.DHCPScope{
					PoolStart: net.ParseIP("10.10.200.100"), PoolEnd: net.ParseIP("10.10.200.150"),
					Router: net.ParseIP("10.10.200.1"),
				}
			},
			code: fabric.CodeInvalidDHCPRelay,
		},
		{
			name: "scope outside every network",
			mutate: func(cfg *config.Config) {
				cfg.Devices[1].DHCPConfig.Scopes[0].PoolStart = net.ParseIP("192.0.2.10")
			},
			code: fabric.CodeDHCPPoolOutsideNetwork,
		},
		{
			name: "scope across two networks",
			mutate: func(cfg *config.Config) {
				cfg.Devices[1].DHCPConfig.Scopes[0].PoolEnd = net.ParseIP("10.10.200.150")
			},
			code: fabric.CodeDHCPPoolOutsideNetwork,
		},
		{
			name: "scope start follows end",
			mutate: func(cfg *config.Config) {
				cfg.Devices[1].DHCPConfig.Scopes[0].PoolStart = net.ParseIP("10.20.210.151")
			},
			code: fabric.CodeInvalidDHCPRange,
		},
		{
			name: "scope includes the broadcast address",
			mutate: func(cfg *config.Config) {
				cfg.Devices[1].DHCPConfig.Scopes[0].PoolEnd = net.ParseIP("10.20.210.255")
			},
			code: fabric.CodeReservedDHCPAddress,
		},
		{
			name: "scope router outside its network",
			mutate: func(cfg *config.Config) {
				cfg.Devices[1].DHCPConfig.Scopes[0].Router = net.ParseIP("10.10.200.1")
			},
			code: fabric.CodeInvalidDHCPRouter,
		},
		{
			name: "scope contains an interface address",
			mutate: func(cfg *config.Config) {
				cfg.Devices[1].DHCPConfig.Scopes[0].PoolStart = net.ParseIP("10.20.210.2")
			},
			code: fabric.CodeDHCPAddressCollision,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := relayedDHCPConfig()
			tc.mutate(cfg)
			report := fabric.Compile(cfg, accessBinding())
			if report.Safe {
				t.Fatal("Compile() accepted an invalid relayed DHCP scope")
			}
			assertDiagnostic(t, report, tc.code)
		})
	}
}
