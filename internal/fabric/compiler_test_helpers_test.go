package fabric_test

import (
	"net"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

func referenceConfig() *config.Config {
	return &config.Config{
		Networks: []config.Network{
			{Name: "lab-access", Subnet: "10.10.200.0/24", VirtualVLAN: 200},
			{Name: "evt-data", Subnet: "10.20.210.0/24", VirtualVLAN: 210},
		},
		Attachments: []config.LogicalAttachment{{Name: "tester", Network: "lab-access"}},
		Devices: []config.Device{
			referenceRouter(),
			referenceDHCPServer(),
		},
	}
}

func referenceRouter() config.Device {
	return config.Device{
		Name: "LAB-EDGE-R1",
		Type: "router",
		Interfaces: []config.Interface{
			{Name: "outside", Network: "lab-access", Address: "10.10.200.1/24"},
			{Name: "evt", Network: "evt-data", Address: "10.20.210.1/24"},
		},
		Routes: []config.Route{{Destination: "10.20.0.0/16", Via: "evt"}},
	}
}

func referenceDHCPServer() config.Device {
	return config.Device{
		Name: "LAB-DHCP01",
		Type: "server",
		Interfaces: []config.Interface{
			{Name: "eth0", Network: "lab-access", Address: "10.10.200.2/24"},
		},
		DHCPConfig: &config.DHCPConfig{
			PoolStart: net.ParseIP("10.10.200.200"),
			PoolEnd:   net.ParseIP("10.10.200.220"),
			Router:    net.ParseIP("10.10.200.1"),
		},
	}
}

func accessBinding() fabric.Binding {
	return fabric.Binding{
		Attachment: "tester",
		Interface:  "eth0",
		Mode:       fabric.ModeAccess,
		AccessVLAN: 200,
	}
}

func assertDiagnostic(t *testing.T, report fabric.Report, code fabric.DiagnosticCode) {
	t.Helper()
	for _, diagnostic := range report.Diagnostics {
		if diagnostic.Code == code {
			return
		}
	}
	t.Fatalf("diagnostics = %#v, want code %q", report.Diagnostics, code)
}
