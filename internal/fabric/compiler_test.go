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
		Attachment: "tester",
		Interface:  "eth0",
		Mode:       fabric.ModeAccess,
		AccessVLAN: 200,
	}

	report := fabric.Compile(cfg, binding)

	if !report.Safe {
		t.Fatalf("Compile() safe = false, diagnostics = %#v", report.Diagnostics)
	}
	if got := len(report.Topology.Networks); got != 2 {
		t.Fatalf("networks = %d, want 2", got)
	}
	if got := len(report.Topology.Routes); got != 3 {
		t.Fatalf("routes = %d, want 3 (two connected and one static)", got)
	}
	if got := len(report.Topology.DHCPScopes); got != 1 {
		t.Fatalf("DHCP scopes = %d, want 1", got)
	}
	if report.Topology.Binding.WireTagged {
		t.Fatal("access binding must be untagged inside NIAC")
	}
}

func TestCompileAccessVLANIsDeploymentSpecific(t *testing.T) {
	cfg := referenceConfig()

	for _, vlan := range []uint16{2, 200, 300, 4094} {
		t.Run(fmt.Sprintf("vlan-%d", vlan), func(t *testing.T) {
			report := fabric.Compile(cfg, fabric.Binding{
				Attachment: "tester",
				Interface:  "eth0",
				Mode:       fabric.ModeAccess,
				AccessVLAN: vlan,
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
			name: "direct interface not affirmed dedicated",
			binding: fabric.Binding{
				Attachment: "tester",
				Interface:  "eth0",
				Mode:       fabric.ModeDirect,
			},
			code: fabric.CodeDedicatedRequired,
		},
		{
			name: "access VLAN missing",
			binding: fabric.Binding{
				Attachment: "tester",
				Interface:  "eth0",
				Mode:       fabric.ModeAccess,
			},
			code: fabric.CodeInvalidAccessVLAN,
		},
		{
			name: "unknown logical attachment",
			binding: fabric.Binding{
				Attachment: "missing",
				Interface:  "eth0",
				Mode:       fabric.ModeAccess,
				AccessVLAN: 200,
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
