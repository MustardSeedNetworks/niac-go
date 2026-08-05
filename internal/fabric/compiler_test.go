package fabric_test

import (
	"fmt"
	"net"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

func TestCompileReferenceLab(t *testing.T) {
	cfg := referenceConfig()
	binding := fabric.Binding{
		Attachment:     "tester",
		Interface:      "eth0",
		Mode:           fabric.ModeAccess,
		AccessVLAN:     200,
		PolicyApproved: true,
	}

	report := fabric.Compile(cfg, binding)

	if !report.Safe {
		t.Fatalf("Compile() safe = false, diagnostics = %#v", report.Diagnostics)
	}
	if got := len(report.Topology.Networks); got != 2 {
		t.Fatalf("networks = %d, want 2", got)
	}
	if got := len(report.Topology.Routes); got != 4 {
		t.Fatalf("routes = %d, want 4 (three connected and one static)", got)
	}
	if got := len(report.Topology.DHCPScopes); got != 1 {
		t.Fatalf("DHCP scopes = %d, want 1", got)
	}
	if report.Topology.Binding.WireTagged {
		t.Fatal("access binding must be untagged inside NIAC")
	}
}

func TestCompileLayer3SwitchCreatesConnectedRoutes(t *testing.T) {
	cfg := referenceConfig()
	cfg.Devices[0].Type = "layer3-switch"
	report := fabric.Compile(cfg, accessBinding())
	if !report.Safe {
		t.Fatalf("Compile() diagnostics = %#v", report.Diagnostics)
	}
	if got := len(report.Topology.Routes); got != 4 {
		t.Fatalf("routes = %d, want 4", got)
	}
}

func TestCompileDHCPInfersNetworkOnMultiInterfaceRouter(t *testing.T) {
	cfg := referenceConfig()
	cfg.Devices[0].DHCPConfig = &config.DHCPConfig{
		PoolStart: net.ParseIP("10.10.200.100"),
		PoolEnd:   net.ParseIP("10.10.200.199"),
		Router:    net.ParseIP("10.10.200.1"),
	}
	cfg.Devices[1].DHCPConfig = nil

	report := fabric.Compile(cfg, accessBinding())

	if !report.Safe {
		t.Fatalf("Compile() diagnostics = %#v", report.Diagnostics)
	}
	if len(report.Topology.DHCPScopes) != 1 ||
		report.Topology.DHCPScopes[0].Network != "lab-access" {
		t.Fatalf("DHCP scopes = %#v, want one lab-access scope", report.Topology.DHCPScopes)
	}
}

func TestCompileRejectsDHCPPoolAcrossRoutedNetworks(t *testing.T) {
	cfg := referenceConfig()
	cfg.Devices[0].DHCPConfig = &config.DHCPConfig{
		PoolStart: net.ParseIP("10.10.200.100"),
		PoolEnd:   net.ParseIP("10.20.210.199"),
		Router:    net.ParseIP("10.10.200.1"),
	}
	cfg.Devices[1].DHCPConfig = nil

	report := fabric.Compile(cfg, accessBinding())

	assertDiagnostic(t, report, fabric.CodeDHCPPoolOutsideNetwork)
}

func TestCompileAccessVLANIsDeploymentSpecific(t *testing.T) {
	cfg := referenceConfig()

	for _, vlan := range []uint16{2, 200, 300, 4094} {
		t.Run(fmt.Sprintf("vlan-%d", vlan), func(t *testing.T) {
			report := fabric.Compile(cfg, fabric.Binding{
				Attachment:     "tester",
				Interface:      "eth0",
				Mode:           fabric.ModeAccess,
				AccessVLAN:     vlan,
				PolicyApproved: true,
			})
			if !report.Safe {
				t.Fatalf("VLAN %d rejected: %#v", vlan, report.Diagnostics)
			}
			if report.Topology.Binding.AccessVLAN != vlan {
				t.Fatalf("access VLAN = %d, want %d", report.Topology.Binding.AccessVLAN, vlan)
			}
		})
	}
}

func TestCompileRejectsUnsafeBindings(t *testing.T) {
	tests := []struct {
		name    string
		binding fabric.Binding
		code    fabric.DiagnosticCode
	}{
		{
			name: "direct interface denied by operator policy",
			binding: fabric.Binding{
				Attachment: "tester",
				Interface:  "eth0",
				Mode:       fabric.ModeDirect,
			},
			code: fabric.CodeAttachmentPolicyDenied,
		},
		{
			name: "access interface denied by operator policy",
			binding: fabric.Binding{
				Attachment: "tester", Interface: "eth0", Mode: fabric.ModeAccess, AccessVLAN: 200,
			},
			code: fabric.CodeAttachmentPolicyDenied,
		},
		{
			name: "access VLAN missing",
			binding: fabric.Binding{
				Attachment:     "tester",
				Interface:      "eth0",
				Mode:           fabric.ModeAccess,
				PolicyApproved: true,
			},
			code: fabric.CodeInvalidAccessVLAN,
		},
		{
			name: "unknown logical attachment",
			binding: fabric.Binding{
				Attachment:     "missing",
				Interface:      "eth0",
				Mode:           fabric.ModeAccess,
				AccessVLAN:     200,
				PolicyApproved: true,
			},
			code: fabric.CodeUnknownAttachment,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := fabric.Compile(referenceConfig(), tt.binding)
			if report.Safe {
				t.Fatal("Compile() safe = true, want false")
			}
			assertDiagnostic(t, report, tt.code)
		})
	}
}

func TestCompileAcceptsApprovedDirectPolicy(t *testing.T) {
	report := fabric.Compile(referenceConfig(), fabric.Binding{
		Attachment: "tester", Interface: "eth1", Mode: fabric.ModeDirect, PolicyApproved: true,
	})
	if !report.Safe {
		t.Fatalf("Compile() diagnostics = %#v", report.Diagnostics)
	}
}

func TestCompileAcceptsApprovedTrunkBinding(t *testing.T) {
	report := fabric.Compile(referenceConfig(), fabric.Binding{
		Attachment: "tester", Interface: "eth0", Mode: fabric.ModeTrunk,
		AccessVLAN: 200, PolicyApproved: true,
	})
	if !report.Safe {
		t.Fatalf("Compile() diagnostics = %#v", report.Diagnostics)
	}
	if !report.Topology.Binding.WireTagged {
		t.Fatal("Compile() WireTagged = false, want true")
	}
}

func TestCompileRejectsInvalidNetworkSemantics(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*config.Config)
		code   fabric.DiagnosticCode
	}{
		{
			name: "overlapping networks",
			mutate: func(cfg *config.Config) {
				cfg.Networks[1].Subnet = "10.10.200.128/25"
			},
			code: fabric.CodeOverlappingNetworks,
		},
		{
			name: "interface outside network",
			mutate: func(cfg *config.Config) {
				cfg.Devices[0].Interfaces[1].Address = "10.30.210.1/24"
			},
			code: fabric.CodeAddressOutsideNetwork,
		},
		{
			name: "DHCP pool outside network",
			mutate: func(cfg *config.Config) {
				cfg.Devices[1].DHCPConfig.PoolEnd = net.ParseIP("10.20.210.220")
			},
			code: fabric.CodeDHCPPoolOutsideNetwork,
		},
		{
			name: "duplicate device name",
			mutate: func(cfg *config.Config) {
				cfg.Devices[1].Name = cfg.Devices[0].Name
			},
			code: fabric.CodeDuplicateDevice,
		},
		{
			name: "duplicate interface address",
			mutate: func(cfg *config.Config) {
				cfg.Devices[1].Interfaces[0].Address = cfg.Devices[0].Interfaces[0].Address
				cfg.Devices[1].Interfaces[0].Network = cfg.Devices[0].Interfaces[0].Network
			},
			code: fabric.CodeDuplicateInterfaceAddr,
		},
		{
			name: "duplicate interface name",
			mutate: func(cfg *config.Config) {
				cfg.Devices[0].Interfaces[1].Name = cfg.Devices[0].Interfaces[0].Name
			},
			code: fabric.CodeDuplicateInterface,
		},
		{
			name: "interface prefix differs from network",
			mutate: func(cfg *config.Config) {
				cfg.Devices[0].Interfaces[0].Address = "10.10.200.1/25"
			},
			code: fabric.CodeInterfacePrefixMismatch,
		},
		{
			name: "interface uses network address",
			mutate: func(cfg *config.Config) {
				cfg.Devices[0].Interfaces[0].Address = "10.10.200.0/24"
			},
			code: fabric.CodeReservedInterfaceAddr,
		},
		{
			name: "interface uses broadcast address",
			mutate: func(cfg *config.Config) {
				cfg.Devices[0].Interfaces[0].Address = "10.10.200.255/24"
			},
			code: fabric.CodeReservedInterfaceAddr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := referenceConfig()
			tt.mutate(cfg)
			report := fabric.Compile(cfg, accessBinding())
			if report.Safe {
				t.Fatal("Compile() safe = true, want false")
			}
			assertDiagnostic(t, report, tt.code)
		})
	}
}

func TestCompileRejectsInvalidRouteSemantics(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*config.Config)
		code   fabric.DiagnosticCode
	}{
		{
			name: "invalid next hop",
			mutate: func(cfg *config.Config) {
				cfg.Devices[0].Routes[0].NextHop = "not-an-address"
			},
			code: fabric.CodeInvalidRouteNextHop,
		},
		{
			name: "next hop outside egress network",
			mutate: func(cfg *config.Config) {
				cfg.Devices[0].Routes[0].NextHop = "10.10.200.2"
			},
			code: fabric.CodeRouteNextHopOffLink,
		},
		{
			name: "next hop is network address",
			mutate: func(cfg *config.Config) {
				cfg.Devices[0].Routes[0].NextHop = "10.20.210.0"
			},
			code: fabric.CodeRouteNextHopOffLink,
		},
		{
			name: "next hop is not configured",
			mutate: func(cfg *config.Config) {
				cfg.Devices[0].Routes[0].NextHop = "10.20.210.99"
			},
			code: fabric.CodeUnknownRouteNextHop,
		},
		{
			name: "next hop belongs to routed device",
			mutate: func(cfg *config.Config) {
				cfg.Devices[0].Routes[0].NextHop = "10.20.210.1"
			},
			code: fabric.CodeRouteNextHopSelf,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := referenceConfig()
			tt.mutate(cfg)
			report := fabric.Compile(cfg, accessBinding())
			if report.Safe {
				t.Fatal("Compile() safe = true, want false")
			}
			assertDiagnostic(t, report, tt.code)
		})
	}
}

func TestCompileRejectsInvalidDHCPSemantics(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*config.Config)
		code   fabric.DiagnosticCode
	}{
		{
			name: "pool start follows end",
			mutate: func(cfg *config.Config) {
				cfg.Devices[1].DHCPConfig.PoolStart = net.ParseIP("10.10.200.221")
			},
			code: fabric.CodeInvalidDHCPRange,
		},
		{
			name: "router outside network",
			mutate: func(cfg *config.Config) {
				cfg.Devices[1].DHCPConfig.Router = net.ParseIP("10.20.210.1")
			},
			code: fabric.CodeInvalidDHCPRouter,
		},
		{
			name: "pool includes network address",
			mutate: func(cfg *config.Config) {
				cfg.Devices[1].DHCPConfig.PoolStart = net.ParseIP("10.10.200.0")
			},
			code: fabric.CodeReservedDHCPAddress,
		},
		{
			name: "pool includes broadcast address",
			mutate: func(cfg *config.Config) {
				cfg.Devices[1].DHCPConfig.PoolEnd = net.ParseIP("10.10.200.255")
			},
			code: fabric.CodeReservedDHCPAddress,
		},
		{
			name: "pool collides with interface",
			mutate: func(cfg *config.Config) {
				cfg.Devices[1].DHCPConfig.PoolStart = net.ParseIP("10.10.200.1")
			},
			code: fabric.CodeDHCPAddressCollision,
		},
		{
			name: "lease outside network",
			mutate: func(cfg *config.Config) {
				cfg.Devices[1].DHCPConfig.ClientLeases = []config.DHCPLease{{
					ClientIP:   net.ParseIP("192.0.2.10"),
					MACAddress: net.HardwareAddr{0x02, 0, 0, 0, 0, 0x10},
				}}
			},
			code: fabric.CodeInvalidDHCPLease,
		},
		{
			name: "lease uses reserved address",
			mutate: func(cfg *config.Config) {
				cfg.Devices[1].DHCPConfig.ClientLeases = []config.DHCPLease{{
					ClientIP:   net.ParseIP("10.10.200.255"),
					MACAddress: net.HardwareAddr{0x02, 0, 0, 0, 0, 0x10},
				}}
			},
			code: fabric.CodeInvalidDHCPLease,
		},
		{
			name: "lease has invalid MAC",
			mutate: func(cfg *config.Config) {
				cfg.Devices[1].DHCPConfig.ClientLeases = []config.DHCPLease{{
					ClientIP: net.ParseIP("10.10.200.50"), MACAddress: net.HardwareAddr{0x02},
				}}
			},
			code: fabric.CodeInvalidDHCPLease,
		},
		{
			name: "lease has invalid MAC mask",
			mutate: func(cfg *config.Config) {
				cfg.Devices[1].DHCPConfig.ClientLeases = []config.DHCPLease{{
					ClientIP:   net.ParseIP("10.10.200.50"),
					MACAddress: net.HardwareAddr{0x02, 0, 0, 0, 0, 0x10},
					MACMask:    net.HardwareAddr{0xff},
				}}
			},
			code: fabric.CodeInvalidDHCPLease,
		},
		{
			name: "lease overlaps pool",
			mutate: func(cfg *config.Config) {
				cfg.Devices[1].DHCPConfig.ClientLeases = []config.DHCPLease{{
					ClientIP:   net.ParseIP("10.10.200.210"),
					MACAddress: net.HardwareAddr{0x02, 0, 0, 0, 0, 0x10},
				}}
			},
			code: fabric.CodeDHCPAddressCollision,
		},
		{
			name: "lease collides with interface",
			mutate: func(cfg *config.Config) {
				cfg.Devices[1].DHCPConfig.ClientLeases = []config.DHCPLease{{
					ClientIP:   net.ParseIP("10.10.200.1"),
					MACAddress: net.HardwareAddr{0x02, 0, 0, 0, 0, 0x10},
				}}
			},
			code: fabric.CodeDHCPAddressCollision,
		},
		{
			name: "duplicate lease address",
			mutate: func(cfg *config.Config) {
				cfg.Devices[1].DHCPConfig.ClientLeases = []config.DHCPLease{
					{
						ClientIP:   net.ParseIP("10.10.200.50"),
						MACAddress: net.HardwareAddr{0x02, 0, 0, 0, 0, 0x10},
					},
					{
						ClientIP:   net.ParseIP("10.10.200.50"),
						MACAddress: net.HardwareAddr{0x02, 0, 0, 0, 0, 0x11},
					},
				}
			},
			code: fabric.CodeDHCPAddressCollision,
		},
		{
			name: "duplicate lease MAC",
			mutate: func(cfg *config.Config) {
				cfg.Devices[1].DHCPConfig.ClientLeases = []config.DHCPLease{
					{
						ClientIP:   net.ParseIP("10.10.200.50"),
						MACAddress: net.HardwareAddr{0x02, 0, 0, 0, 0, 0x10},
					},
					{
						ClientIP:   net.ParseIP("10.10.200.51"),
						MACAddress: net.HardwareAddr{0x02, 0, 0, 0, 0, 0x10},
					},
				}
			},
			code: fabric.CodeDHCPAddressCollision,
		},
		{
			name: "overlapping masked lease MAC",
			mutate: func(cfg *config.Config) {
				cfg.Devices[1].DHCPConfig.ClientLeases = []config.DHCPLease{
					{
						ClientIP:   net.ParseIP("10.10.200.50"),
						MACAddress: net.HardwareAddr{0x02, 0, 0, 0, 0, 0},
						MACMask:    net.HardwareAddr{0xff, 0xff, 0xff, 0, 0, 0},
					},
					{
						ClientIP:   net.ParseIP("10.10.200.51"),
						MACAddress: net.HardwareAddr{0x02, 0, 0, 0, 0, 1},
						MACMask:    net.HardwareAddr{0xff, 0xff, 0xff, 0, 0, 0},
					},
				}
			},
			code: fabric.CodeDHCPAddressCollision,
		},
		{
			name: "invalid DNS server address",
			mutate: func(cfg *config.Config) {
				cfg.Devices[1].DHCPConfig.DomainNameServer = []net.IP{nil}
			},
			code: fabric.CodeInvalidDHCPOption,
		},
		{
			name: "IPv6 DNS server address",
			mutate: func(cfg *config.Config) {
				cfg.Devices[1].DHCPConfig.DomainNameServer = []net.IP{
					net.ParseIP("2001:4860:4860::8888"),
				}
			},
			code: fabric.CodeInvalidDHCPOption,
		},
		{
			name: "server identifier outside network",
			mutate: func(cfg *config.Config) {
				cfg.Devices[1].DHCPConfig.ServerIdentifier = net.ParseIP("192.0.2.10")
			},
			code: fabric.CodeInvalidDHCPOption,
		},
		{
			name: "next server uses reserved address",
			mutate: func(cfg *config.Config) {
				cfg.Devices[1].DHCPConfig.NextServerIP = net.ParseIP("10.10.200.0")
			},
			code: fabric.CodeInvalidDHCPOption,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := referenceConfig()
			tt.mutate(cfg)
			report := fabric.Compile(cfg, accessBinding())
			if report.Safe {
				t.Fatal("Compile() safe = true, want false")
			}
			assertDiagnostic(t, report, tt.code)
		})
	}
}

func TestCompileAcceptsValidDHCPLeasesAndOptions(t *testing.T) {
	cfg := referenceConfig()
	cfg.Devices[1].DHCPConfig.ClientLeases = []config.DHCPLease{{
		ClientIP:   net.ParseIP("10.10.200.50"),
		MACAddress: net.HardwareAddr{0x02, 0, 0, 0, 0, 0x50},
	}}
	cfg.Devices[1].DHCPConfig.DomainNameServer = []net.IP{net.ParseIP("8.8.8.8")}
	cfg.Devices[1].DHCPConfig.ServerIdentifier = net.ParseIP("10.10.200.2")
	cfg.Devices[1].DHCPConfig.NextServerIP = net.ParseIP("10.10.200.1")

	report := fabric.Compile(cfg, accessBinding())

	if !report.Safe {
		t.Fatalf("Compile() diagnostics = %#v, want safe DHCP configuration", report.Diagnostics)
	}
}

func TestCompileCopiesSourceConfiguration(t *testing.T) {
	cfg := referenceConfig()
	report := fabric.Compile(cfg, accessBinding())
	if !report.Safe {
		t.Fatalf("Compile() failed: %#v", report.Diagnostics)
	}

	cfg.Networks[0].Name = "changed"
	cfg.Devices[0].Interfaces[0].Address = "192.0.2.1/24"

	if report.Topology.Networks[0].Name != "lab-access" {
		t.Fatal("compiled topology changed after source mutation")
	}
	if got := report.Topology.Interfaces[0].Address.String(); got != "10.10.200.1/24" {
		t.Fatalf("compiled interface changed to %q", got)
	}
}
