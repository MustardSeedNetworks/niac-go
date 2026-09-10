package protocols

import (
	"bytes"
	"net"
	"testing"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

func TestNotificationMaskRetainsAttachmentHostFirstHop(t *testing.T) {
	for _, test := range []struct {
		bits        int
		destination string
	}{
		{8, "10.20.0.10"},
		{32, "10.10.200.100"},
	} {
		cfg, _, routerMAC := forwardingFixture(t)
		cfg.Devices = append(cfg.Devices, config.Device{
			Name: "local-host", Type: "server", MACAddress: mustParseMAC(t, "02:00:00:00:00:30"),
			Interfaces: []config.Interface{{Name: "eth0", Address: "10.10.200.30/24", Network: "attachment"}},
			Routes:     []config.Route{{Destination: "0.0.0.0/0", Via: "eth0", NextHop: "10.10.200.1"}},
		})
		report := fabric.Compile(cfg, fabric.Binding{
			Attachment: "tester", Interface: "eth0", Mode: fabric.ModeTrunk, AccessVLAN: 200,
			PolicyApproved: true,
		})
		if !report.Safe {
			t.Fatal(report.Diagnostics)
		}
		stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
		stack.ConfigureFabric(&report.Topology)
		stack.running.Store(true)
		t.Cleanup(stack.Stop)
		host := &cfg.Devices[2]
		if err := stack.deviceStates[host].SetInterfacePrefixFault(devicestate.InterfacePrefixFault{
			Interface: "eth0", Type: devicestate.FaultBadMask, PrefixBits: test.bits,
		}); err != nil {
			t.Fatal(err)
		}
		if err := stack.notifications.sender.Send(
			host,
			200,
			net.JoinHostPort(test.destination, "514"),
			514,
			[]byte("event"),
		); err != nil {
			t.Fatal(err)
		}
		packet := receiveRoutedReply(t, stack)
		if test.bits == 8 {
			assertNotificationProbe(t, packet, "10.10.200.30", test.destination)
			continue
		}
		decoded := gopacket.NewPacket(packet.Buffer, layers.LayerTypeEthernet, gopacket.Default)
		ethernet, _ := decoded.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
		ip, _ := decoded.Layer(layers.LayerTypeIPv4).(*layers.IPv4)
		if ethernet == nil || ip == nil || !bytes.Equal(ethernet.SrcMAC, host.MACAddress) ||
			!bytes.Equal(ethernet.DstMAC, routerMAC) || ip.TTL != ipIPv4TTL || packet.fabricTrace.RouteDecision != "" {
			t.Fatalf(
				"faulted host was pre-forwarded: Ethernet=%+v IPv4=%+v trace=%+v",
				ethernet,
				ip,
				packet.fabricTrace,
			)
		}
	}
}

func TestNotificationMaskExcludesRoutedDeviceRoles(t *testing.T) {
	for _, role := range []string{"router", "layer3-switch", "firewall"} {
		t.Run(role, func(t *testing.T) {
			stack, packet := hostEgressFixture(t, "192.0.2.20")
			packet.generatedHost.Type = role
			setNotificationMask(t, stack, packet, 30)
			if err := stack.notifications.sender.Send(
				packet.generatedHost,
				200,
				"192.0.2.20:514",
				514,
				[]byte("event"),
			); err != nil {
				t.Fatal(err)
			}
			assertNotificationProbe(t, receiveRoutedReply(t, stack), "192.0.2.10", "192.0.2.20")
		})
	}
}
