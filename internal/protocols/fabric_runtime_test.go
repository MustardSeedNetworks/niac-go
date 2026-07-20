package protocols_test

import (
	"bytes"
	"net"
	"net/netip"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

func TestFabricRuntimeResolveIPv4(t *testing.T) {
	cfg, topology := routedRuntimeFixture(t)
	runtime := protocols.NewFabricRuntimeForTest(topology, cfg)
	routerMAC := cfg.Devices[0].MACAddress

	tests := []struct {
		name       string
		dst        string
		ingressMAC net.HardwareAddr
		wantDevice string
		wantMAC    net.HardwareAddr
		wantRouted bool
		wantOK     bool
	}{
		{
			name: "attachment endpoint is directly reachable", dst: "10.10.200.1",
			ingressMAC: routerMAC, wantDevice: "edge", wantMAC: routerMAC, wantOK: true,
		},
		{
			name: "internal endpoint uses attachment router return identity", dst: "10.20.0.10",
			ingressMAC: routerMAC, wantDevice: "server", wantMAC: routerMAC, wantRouted: true, wantOK: true,
		},
		{
			name: "internal endpoint rejects unrelated ingress MAC", dst: "10.20.0.10",
			ingressMAC: mustRuntimeMAC(t, "02:00:00:00:00:ff"), wantOK: false,
		},
		{
			name: "unassigned destination is not resolved", dst: "10.20.0.99",
			ingressMAC: routerMAC, wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			device, replyMAC, routed, ok := runtime.ResolveIPv4(netip.MustParseAddr(tt.dst), tt.ingressMAC)
			if ok != tt.wantOK {
				t.Fatalf("ResolveIPv4() ok = %t, want %t", ok, tt.wantOK)
			}
			if !tt.wantOK {
				return
			}
			if device.Name != tt.wantDevice || !bytes.Equal(replyMAC, tt.wantMAC) || routed != tt.wantRouted {
				t.Fatalf(
					"ResolveIPv4() = (%s, %s, %t), want (%s, %s, %t)",
					device.Name, replyMAC, routed, tt.wantDevice, tt.wantMAC, tt.wantRouted,
				)
			}
		})
	}
}

func TestFabricRuntimeCopiesTopologyAndMACIdentity(t *testing.T) {
	cfg, topology := routedRuntimeFixture(t)
	runtime := protocols.NewFabricRuntimeForTest(topology, cfg)
	routerMAC := append(net.HardwareAddr(nil), cfg.Devices[0].MACAddress...)

	topology.Routes = nil
	cfg.Devices[0].MACAddress[0] = 0xff

	device, replyMAC, routed, ok := runtime.ResolveIPv4(netip.MustParseAddr("10.20.0.10"), routerMAC)
	if !ok || device.Name != "server" || !routed || !bytes.Equal(replyMAC, routerMAC) {
		t.Fatalf(
			"resolver changed after source mutation: device=%v mac=%s routed=%t ok=%t",
			device,
			replyMAC,
			routed,
			ok,
		)
	}
}

func routedRuntimeFixture(t *testing.T) (*config.Config, *fabric.Topology) {
	t.Helper()
	cfg := &config.Config{
		Networks: []config.Network{
			{Name: "attachment", Subnet: "10.10.200.0/24"},
			{Name: "internal", Subnet: "10.20.0.0/24"},
		},
		Attachments: []config.LogicalAttachment{{Name: "tester", Network: "attachment"}},
		Devices: []config.Device{
			{
				Name: "edge", Type: "router", MACAddress: mustRuntimeMAC(t, "02:00:00:00:00:01"),
				Interfaces: []config.Interface{
					{Name: "outside", Network: "attachment", Address: "10.10.200.1/24"},
					{Name: "inside", Network: "internal", Address: "10.20.0.1/24"},
				},
			},
			{
				Name: "server", Type: "server", MACAddress: mustRuntimeMAC(t, "02:00:00:00:00:10"),
				Interfaces: []config.Interface{
					{Name: "eth0", Network: "internal", Address: "10.20.0.10/24"},
				},
			},
		},
	}
	report := fabric.Compile(cfg, fabric.Binding{
		Attachment: "tester", Interface: "eth0", Mode: fabric.ModeAccess, AccessVLAN: 200,
	})
	if !report.Safe {
		t.Fatalf("Compile() diagnostics = %#v", report.Diagnostics)
	}
	return cfg, &report.Topology
}

func mustRuntimeMAC(t *testing.T, value string) net.HardwareAddr {
	t.Helper()
	mac, err := net.ParseMAC(value)
	if err != nil {
		t.Fatalf("ParseMAC(%q): %v", value, err)
	}
	return mac
}
