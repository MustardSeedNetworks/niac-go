package protocols

import (
	"bytes"
	"net"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicecli"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

func TestFabricRoutesInternalTargetThroughAttachmentRouter(t *testing.T) {
	cfg, topology, routerMAC := forwardingFixture(t)
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	stack.ConfigureFabric(topology)
	pkt := NewPacket(64)
	pkt.PutDestMAC(routerMAC)

	targets := stack.ipHandler.getTargetDevices(
		&layers.IPv4{DstIP: net.ParseIP("10.20.0.10")}, pkt, 1, 0,
	)

	if len(targets) != 1 || targets[0].Name != "server" {
		t.Fatalf("targets = %#v, want server", targets)
	}
	if got := stack.replySourceMAC(pkt, targets[0]); !bytes.Equal(got, routerMAC) {
		t.Fatalf("reply source MAC = %s, want %s", got, routerMAC)
	}
	if !stack.deviceOwnsIPv4(targets[0], net.ParseIP("10.20.0.10")) {
		t.Fatal("resolved fabric endpoint must own its interface address")
	}
	trace := pkt.FabricTrace()
	if trace.RouteDecision != "forwarded" || trace.EgressNetwork != "internal" || trace.Hop != "edge:inside" {
		t.Fatalf("FabricTrace() = %#v", trace)
	}
	if got := stack.GetStats().FabricForwarded; got != 1 {
		t.Fatalf("FabricForwarded = %d, want 1", got)
	}
}

func TestFabricRecordsMissingRouteDrop(t *testing.T) {
	cfg, topology, routerMAC := forwardingFixture(t)
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	stack.ConfigureFabric(topology)
	pkt := NewPacket(64)
	pkt.PutDestMAC(routerMAC)

	targets := stack.ipHandler.getTargetDevices(
		&layers.IPv4{DstIP: net.ParseIP("10.99.0.10")}, pkt, 1, 0,
	)

	if targets != nil {
		t.Fatalf("targets = %#v, want nil", targets)
	}
	trace := pkt.FabricTrace()
	if trace.RouteDecision != "dropped" || trace.RejectionReason != "no_route" {
		t.Fatalf("FabricTrace() = %#v", trace)
	}
	if got := stack.GetStats().FabricDrops; got != 1 {
		t.Fatalf("FabricDrops = %d, want 1", got)
	}
}

func TestCLIShutdownChangesFabricDelivery(t *testing.T) {
	cfg, topology, routerMAC := forwardingFixture(t)
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	stack.ConfigureFabric(topology)
	server := &cfg.Devices[1]
	session := devicecli.NewSession(stack.deviceStates[server])
	for _, command := range []string{"enable", "configure terminal", "interface eth0", "shutdown"} {
		if response := session.Execute(command); strings.HasPrefix(response.Output, "%") {
			t.Fatalf("%q response = %#v", command, response)
		}
	}
	pkt := NewPacket(64)
	pkt.PutDestMAC(routerMAC)

	targets := stack.ipHandler.getTargetDevices(
		&layers.IPv4{DstIP: net.ParseIP("10.20.0.10")}, pkt, 1, 0,
	)
	if targets != nil {
		t.Fatalf("targets = %#v, want nil for shut interface", targets)
	}
}

func TestCLIShutdownStopsAttachmentARP(t *testing.T) {
	cfg, topology, _ := forwardingFixture(t)
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	stack.ConfigureFabric(topology)
	router := &cfg.Devices[0]
	session := devicecli.NewSession(stack.deviceStates[router])
	for _, command := range []string{"enable", "configure terminal", "interface outside", "shutdown"} {
		if response := session.Execute(command); strings.HasPrefix(response.Output, "%") {
			t.Fatalf("%q response = %#v", command, response)
		}
	}

	if device, ok := stack.fabric.resolveARP(net.ParseIP("10.10.200.1")); ok {
		t.Fatalf("resolveARP() = %q after shutdown", device.Name)
	}
}

func TestCLIAddressChangeMovesFabricDelivery(t *testing.T) {
	cfg, topology, routerMAC := forwardingFixture(t)
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	stack.ConfigureFabric(topology)
	server := &cfg.Devices[1]
	session := devicecli.NewSession(stack.deviceStates[server])
	for _, command := range []string{
		"enable", "configure terminal", "interface eth0", "ip address 10.20.0.20/24",
	} {
		if response := session.Execute(command); strings.HasPrefix(response.Output, "%") {
			t.Fatalf("%q response = %#v", command, response)
		}
	}
	pkt := NewPacket(64)
	pkt.PutDestMAC(routerMAC)

	oldTargets := stack.ipHandler.getTargetDevices(
		&layers.IPv4{DstIP: net.ParseIP("10.20.0.10")}, pkt, 1, 0,
	)
	newTargets := stack.ipHandler.getTargetDevices(
		&layers.IPv4{DstIP: net.ParseIP("10.20.0.20")}, pkt, 1, 0,
	)
	if oldTargets != nil || len(newTargets) != 1 || newTargets[0] != server {
		t.Fatalf("old targets = %#v new targets = %#v", oldTargets, newTargets)
	}
}

func TestCLIStaticRouteChangesFabricDelivery(t *testing.T) {
	cfg, topology, routerMAC := forwardingFixture(t)
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	stack.ConfigureFabric(topology)
	router := &cfg.Devices[0]
	routerState := stack.deviceStates[router]
	network := routerState.Snapshot().Network
	network.Routes = nil
	routerState.ReplaceNetwork(network)
	pkt := NewPacket(64)
	pkt.PutDestMAC(routerMAC)
	destination := &layers.IPv4{DstIP: net.ParseIP("10.20.0.10")}
	if targets := stack.ipHandler.getTargetDevices(destination, pkt, 1, 0); targets != nil {
		t.Fatalf("targets before route = %#v", targets)
	}

	session := devicecli.NewSession(routerState)
	for _, command := range []string{
		"enable", "configure terminal", "ip route 10.20.0.0/24 10.10.200.2 inside",
	} {
		if response := session.Execute(command); strings.HasPrefix(response.Output, "%") {
			t.Fatalf("%q response = %#v", command, response)
		}
	}
	targets := stack.ipHandler.getTargetDevices(destination, pkt, 1, 0)
	if len(targets) != 1 || targets[0] != &cfg.Devices[1] {
		t.Fatalf("targets after route = %#v", targets)
	}
}

func TestFabricRejectsInternalTargetSentToWrongMAC(t *testing.T) {
	cfg, topology, _ := forwardingFixture(t)
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	stack.ConfigureFabric(topology)
	pkt := NewPacket(64)
	pkt.PutDestMAC(mustForwardingMAC(t, "02:00:00:00:00:ff"))

	targets := stack.ipHandler.getTargetDevices(
		&layers.IPv4{DstIP: net.ParseIP("10.20.0.10")}, pkt, 1, 0,
	)

	if targets != nil {
		t.Fatalf("targets = %#v, want nil", targets)
	}
}

func TestFabricDisablesGlobalDHCPHandler(t *testing.T) {
	cfg, topology, _ := forwardingFixture(t)
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	stack.ConfigureFabric(topology)

	if stack.allowDHCP() {
		t.Fatal("routed fabric must not use the legacy global DHCP handler")
	}
}

func TestFabricEnablesOnlyAttachmentDHCP(t *testing.T) {
	cfg, _, _ := forwardingFixture(t)
	cfg.Devices = append(cfg.Devices, config.Device{
		Name: "dhcp", Type: "server", MACAddress: mustForwardingMAC(t, "02:00:00:00:00:02"),
		Interfaces: []config.Interface{
			{Name: "eth0", Network: "attachment", Address: "10.10.200.2/24"},
		},
		DHCPConfig: &config.DHCPConfig{
			ServerIdentifier: net.ParseIP("10.10.200.2"), Router: net.ParseIP("10.10.200.1"),
			PoolStart: net.ParseIP("10.10.200.200"), PoolEnd: net.ParseIP("10.10.200.220"),
		},
	})
	report := fabric.Compile(cfg, fabric.Binding{
		Attachment: "tester", Interface: "eth0", Mode: fabric.ModeAccess, AccessVLAN: 200,
		PolicyApproved: true,
	})
	if !report.Safe {
		t.Fatalf("Compile() diagnostics = %#v", report.Diagnostics)
	}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	stack.ConfigureFabric(&report.Topology)

	if !stack.allowDHCP() {
		t.Fatal("attachment DHCP scope must be enabled")
	}
	pkt := NewPacket(64)
	targets := stack.ipHandler.getTargetDevices(
		&layers.IPv4{DstIP: net.IPv4bcast}, pkt, 1, 0,
	)
	if len(targets) != 1 || targets[0].Name != "dhcp" {
		t.Fatalf("DHCP broadcast targets = %#v, want attachment dhcp", targets)
	}
}

func TestFabricRoutedDHCPDiscoverUsesConfiguredServerIdentity(t *testing.T) {
	cfg, _, _ := forwardingFixture(t)
	serverIP := net.ParseIP("10.10.200.2")
	cfg.Devices = append(cfg.Devices, config.Device{
		Name: "dhcp", Type: "server", MACAddress: mustForwardingMAC(t, "02:00:00:00:00:02"),
		Interfaces: []config.Interface{
			{Name: "eth0", Network: "attachment", Address: "10.10.200.2/24"},
		},
		DHCPConfig: &config.DHCPConfig{
			ServerIdentifier: serverIP, Router: net.ParseIP("10.10.200.1"),
			PoolStart: net.ParseIP("10.10.200.200"), PoolEnd: net.ParseIP("10.10.200.220"),
		},
	})
	report := fabric.Compile(cfg, fabric.Binding{
		Attachment: "tester", Interface: "eth0", Mode: fabric.ModeAccess, AccessVLAN: 200,
		PolicyApproved: true,
	})
	if !report.Safe {
		t.Fatalf("Compile() diagnostics = %#v", report.Diagnostics)
	}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	stack.ConfigureFabric(&report.Topology)
	stack.decodePacket(dhcpDiscoverRequest(t, mustForwardingMAC(t, "00:c0:17:57:01:7c")))

	select {
	case reply := <-stack.sendQueue:
		packet := gopacket.NewPacket(reply.Buffer, layers.LayerTypeEthernet, gopacket.Default)
		ip, ok := packet.Layer(layers.LayerTypeIPv4).(*layers.IPv4)
		if !ok || !ip.SrcIP.Equal(serverIP) {
			t.Fatalf("DHCP offer IPv4 = %#v, want source %s", ip, serverIP)
		}
		dhcp, ok := packet.Layer(layers.LayerTypeDHCPv4).(*layers.DHCPv4)
		if !ok || dhcpMsgType(dhcp) != DHCPOffer {
			t.Fatalf("DHCP response = %#v, want offer", dhcp)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for routed DHCP offer")
	}
}

func TestRoutedFabricAcceptsUntaggedAttachmentWithVirtualVLANs(t *testing.T) {
	cfg, _, routerMAC := forwardingFixture(t)
	cfg.Networks[0].VirtualVLAN = 200
	cfg.Networks[1].VirtualVLAN = 210
	cfg.Devices[0].VLAN = 200
	cfg.Devices[1].VLAN = 210

	report := fabric.Compile(cfg, fabric.Binding{
		Attachment: "tester", Interface: "eth0", Mode: fabric.ModeAccess, AccessVLAN: 200,
		PolicyApproved: true,
	})
	if !report.Safe {
		t.Fatalf("Compile() diagnostics = %#v", report.Diagnostics)
	}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	if !stack.vlanMode {
		t.Fatal("fixture must reproduce legacy VLAN metadata")
	}
	stack.ConfigureFabric(&report.Topology)
	stack.decodePacket(routedSNMPRequest(t, mustForwardingMAC(t, "02:00:00:00:00:fe"), routerMAC))

	select {
	case reply := <-stack.sendQueue:
		if reply.VLANTagged {
			t.Fatal("routed access reply carried an 802.1Q tag")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for untagged routed SNMP reply")
	}
}

func TestFabricARPAnswersOnlyAttachmentNetwork(t *testing.T) {
	cfg, topology, _ := forwardingFixture(t)
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	stack.ConfigureFabric(topology)

	attachment := stack.arpHandler.targetDevices(net.ParseIP("10.10.200.1"), 0)
	if len(attachment) != 1 || attachment[0].Name != "edge" {
		t.Fatalf("attachment ARP targets = %#v, want edge", attachment)
	}
	if internal := stack.arpHandler.targetDevices(net.ParseIP("10.20.0.10"), 0); internal != nil {
		t.Fatalf("internal ARP targets = %#v, want nil", internal)
	}
}

func TestFabricRoutesICMPEchoReplyWithGatewayMAC(t *testing.T) {
	cfg, topology, routerMAC := forwardingFixture(t)
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	stack.ConfigureFabric(topology)
	testerMAC := mustForwardingMAC(t, "02:00:00:00:00:fe")
	pkt := routedEchoRequest(t, testerMAC, routerMAC)

	stack.ipHandler.HandlePacket(pkt)

	select {
	case reply := <-stack.sendQueue:
		packet := gopacket.NewPacket(reply.Buffer, layers.LayerTypeEthernet, gopacket.Default)
		ethernet, ok := packet.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
		if !ok || !bytes.Equal(ethernet.SrcMAC, routerMAC) || !bytes.Equal(ethernet.DstMAC, testerMAC) {
			t.Fatalf("reply ethernet = %#v", ethernet)
		}
		ip, ok := packet.Layer(layers.LayerTypeIPv4).(*layers.IPv4)
		if !ok || !ip.SrcIP.Equal(net.ParseIP("10.20.0.10")) {
			t.Fatalf("reply IPv4 = %#v", ip)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for routed ICMP reply")
	}
}

func TestFabricTTLExpiryUsesAttachmentRouterIdentity(t *testing.T) {
	cfg, topology, routerMAC := forwardingFixture(t)
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	stack.ConfigureFabric(topology)
	testerMAC := mustForwardingMAC(t, "02:00:00:00:00:fe")
	pkt := routedEchoRequestWithTTL(t, testerMAC, routerMAC, 1)

	stack.ipHandler.HandlePacket(pkt)

	select {
	case reply := <-stack.sendQueue:
		packet := gopacket.NewPacket(reply.Buffer, layers.LayerTypeEthernet, gopacket.Default)
		ethernet, ok := packet.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
		if !ok || !bytes.Equal(ethernet.SrcMAC, routerMAC) || !bytes.Equal(ethernet.DstMAC, testerMAC) {
			t.Fatalf("reply ethernet = %#v", ethernet)
		}
		ip, ok := packet.Layer(layers.LayerTypeIPv4).(*layers.IPv4)
		if !ok || !ip.SrcIP.Equal(net.ParseIP("10.10.200.1")) || !ip.DstIP.Equal(net.ParseIP("10.10.200.200")) {
			t.Fatalf("reply IPv4 = %#v", ip)
		}
		icmp, ok := packet.Layer(layers.LayerTypeICMPv4).(*layers.ICMPv4)
		if !ok || icmp.TypeCode.Type() != layers.ICMPv4TypeTimeExceeded {
			t.Fatalf("reply ICMP = %#v", icmp)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for routed ICMP time-exceeded")
	}
}

func TestFabricTTLAboveFirstHopReachesEndpoint(t *testing.T) {
	cfg, topology, routerMAC := forwardingFixture(t)
	cfg.Devices = append(cfg.Devices, config.Device{
		Name: "legacy-hop", Type: "router", MACAddress: mustForwardingMAC(t, "02:00:00:00:00:02"),
		IPAddresses: []net.IP{net.ParseIP("192.0.2.2")},
		TTLConfig: &config.TTLConfig{
			TTL: 2, IP: net.ParseIP("192.0.2.2"), Mask: net.CIDRMask(24, 32),
		},
	})
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	stack.ConfigureFabric(topology)
	testerMAC := mustForwardingMAC(t, "02:00:00:00:00:fe")
	pkt := routedEchoRequestWithTTL(t, testerMAC, routerMAC, 2)

	stack.ipHandler.HandlePacket(pkt)

	select {
	case reply := <-stack.sendQueue:
		packet := gopacket.NewPacket(reply.Buffer, layers.LayerTypeEthernet, gopacket.Default)
		ip, ok := packet.Layer(layers.LayerTypeIPv4).(*layers.IPv4)
		if !ok || !ip.SrcIP.Equal(net.ParseIP("10.20.0.10")) {
			t.Fatalf("reply IPv4 = %#v", ip)
		}
		icmp, ok := packet.Layer(layers.LayerTypeICMPv4).(*layers.ICMPv4)
		if !ok || icmp.TypeCode.Type() != layers.ICMPv4TypeEchoReply {
			t.Fatalf("reply ICMP = %#v", icmp)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for routed ICMP echo reply")
	}
}

func TestFabricRoutesSNMPReplyWithGatewayMAC(t *testing.T) {
	cfg, topology, routerMAC := forwardingFixture(t)
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	stack.ConfigureFabric(topology)
	testerMAC := mustForwardingMAC(t, "02:00:00:00:00:fe")
	pkt := routedSNMPRequest(t, testerMAC, routerMAC)

	stack.ipHandler.HandlePacket(pkt)

	select {
	case reply := <-stack.sendQueue:
		packet := gopacket.NewPacket(reply.Buffer, layers.LayerTypeEthernet, gopacket.Default)
		ethernet, ok := packet.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
		if !ok || !bytes.Equal(ethernet.SrcMAC, routerMAC) {
			t.Fatalf("reply ethernet = %#v", ethernet)
		}
		udp, ok := packet.Layer(layers.LayerTypeUDP).(*layers.UDP)
		if !ok {
			t.Fatal("routed SNMP reply is missing UDP")
		}
		response, err := (&gosnmp.GoSNMP{}).SnmpDecodePacket(udp.Payload)
		if err != nil {
			t.Fatalf("SnmpDecodePacket(): %v", err)
		}
		if len(response.Variables) != 1 {
			t.Fatalf("SNMP variables = %#v", response.Variables)
		}
		value, ok := response.Variables[0].Value.([]byte)
		if !ok || string(value) != "internal-server" {
			t.Fatalf("SNMP variables = %#v", response.Variables)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for routed SNMP reply")
	}
}

func TestFabricRoutesDNSReplyWithGatewayMAC(t *testing.T) {
	cfg, topology, routerMAC := forwardingFixture(t)
	cfg.Devices[1].DNSConfig = &config.DNSConfig{ForwardRecords: []config.DNSRecord{{
		Name: "service.demo", IP: net.ParseIP("10.20.0.10"), TTL: 300,
	}}}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	stack.ConfigureFabric(topology)
	testerMAC := mustForwardingMAC(t, "02:00:00:00:00:fe")

	stack.ipHandler.HandlePacket(routedDNSRequest(t, testerMAC, routerMAC))

	assertRoutedIPv4Reply(t, stack, testerMAC, routerMAC, "10.20.0.10")
}

func TestFabricRoutesTCPRepliesWithGatewayMAC(t *testing.T) {
	tests := []struct {
		name    string
		port    layers.TCPPort
		wantSYN bool
		wantRST bool
		ssh     bool
	}{
		{name: "HTTP SYN-ACK", port: TCPPortHTTP, wantSYN: true},
		{name: "SSH SYN-ACK", port: TCPPortSSH, wantSYN: true, ssh: true},
		{name: "closed-port RST", port: 12345, wantRST: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, topology, routerMAC := forwardingFixture(t)
			if tt.ssh {
				t.Setenv("NIAC_TEST_SSH_PASSWORD", "test-password")
				cfg.Devices[1].SSHConfig = &config.SSHConfig{
					Enabled: true, Username: "admin", PasswordEnv: "NIAC_TEST_SSH_PASSWORD",
				}
			}
			stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
			if tt.ssh {
				useTemporarySSHHostKeys(t, stack)
			}
			stack.ConfigureFabric(topology)
			testerMAC := mustForwardingMAC(t, "02:00:00:00:00:fe")

			stack.ipHandler.HandlePacket(routedTCPRequest(t, testerMAC, routerMAC, tt.port))

			reply := receiveRoutedReply(t, stack)
			packet := gopacket.NewPacket(reply.Buffer, layers.LayerTypeEthernet, gopacket.Default)
			ethernet, ok := packet.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
			if !ok || !bytes.Equal(ethernet.SrcMAC, routerMAC) || !bytes.Equal(ethernet.DstMAC, testerMAC) {
				t.Fatalf("reply ethernet = %#v", ethernet)
			}
			tcp, ok := packet.Layer(layers.LayerTypeTCP).(*layers.TCP)
			if !ok || tcp.SYN != tt.wantSYN || tcp.RST != tt.wantRST || !tcp.ACK {
				t.Fatalf("reply TCP = %#v", tcp)
			}
		})
	}
}

func assertRoutedIPv4Reply(
	t *testing.T,
	stack *Stack,
	testerMAC, routerMAC net.HardwareAddr,
	wantSourceIP string,
) {
	t.Helper()
	reply := receiveRoutedReply(t, stack)
	packet := gopacket.NewPacket(reply.Buffer, layers.LayerTypeEthernet, gopacket.Default)
	ethernet, ok := packet.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
	if !ok || !bytes.Equal(ethernet.SrcMAC, routerMAC) || !bytes.Equal(ethernet.DstMAC, testerMAC) {
		t.Fatalf("reply ethernet = %#v", ethernet)
	}
	ipv4, ok := packet.Layer(layers.LayerTypeIPv4).(*layers.IPv4)
	if !ok || !ipv4.SrcIP.Equal(net.ParseIP(wantSourceIP)) {
		t.Fatalf("reply IPv4 = %#v, want source %s", ipv4, wantSourceIP)
	}
}

func receiveRoutedReply(t *testing.T, stack *Stack) *Packet {
	t.Helper()
	select {
	case reply := <-stack.sendQueue:
		return reply
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for routed reply")
		return nil
	}
}

func TestFabricBindingsAcceptOnlyUntaggedFrames(t *testing.T) {
	cfg, topology, routerMAC := forwardingFixture(t)
	testerMAC := mustForwardingMAC(t, "00:c0:17:57:01:7c")
	direct := *topology
	direct.Binding.Binding = fabric.Binding{
		Attachment: "tester", Interface: "eth0", Mode: fabric.ModeDirect, PolicyApproved: true,
	}

	bindings := []struct {
		name     string
		topology *fabric.Topology
	}{
		{name: "access", topology: topology},
		{name: "direct", topology: &direct},
	}
	for _, binding := range bindings {
		t.Run(binding.name, func(t *testing.T) {
			t.Run("untagged accepted", func(t *testing.T) {
				stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
				stack.ConfigureFabric(binding.topology)
				stack.decodePacket(routedSNMPRequest(t, testerMAC, routerMAC))

				select {
				case reply := <-stack.sendQueue:
					if reply.VLAN > 0 {
						t.Fatalf("reply VLAN metadata = %d, want untagged", reply.VLAN)
					}
				case <-time.After(time.Second):
					t.Fatal("timed out waiting for untagged routed SNMP reply")
				}
			})

			t.Run("tagged rejected", func(t *testing.T) {
				stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
				stack.ConfigureFabric(binding.topology)
				stack.decodePacket(routedTaggedSNMPRequest(t, testerMAC, routerMAC))

				select {
				case reply := <-stack.sendQueue:
					t.Fatalf("unexpected reply to tagged frame: VLAN %d", reply.VLAN)
				case <-time.After(50 * time.Millisecond):
				}
			})
		})
	}
}

func routedEchoRequest(t *testing.T, testerMAC, routerMAC net.HardwareAddr) *Packet {
	return routedEchoRequestWithTTL(t, testerMAC, routerMAC, 64)
}

func routedEchoRequestWithTTL(
	t *testing.T,
	testerMAC net.HardwareAddr,
	routerMAC net.HardwareAddr,
	ttl uint8,
) *Packet {
	t.Helper()
	buffer := gopacket.NewSerializeBuffer()
	err := gopacket.SerializeLayers(buffer, gopacket.SerializeOptions{
		FixLengths: true, ComputeChecksums: true,
	},
		&layers.Ethernet{SrcMAC: testerMAC, DstMAC: routerMAC, EthernetType: layers.EthernetTypeIPv4},
		&layers.IPv4{
			Version: 4, TTL: ttl, Protocol: layers.IPProtocolICMPv4,
			SrcIP: net.ParseIP("10.10.200.200"), DstIP: net.ParseIP("10.20.0.10"),
		},
		&layers.ICMPv4{TypeCode: layers.CreateICMPv4TypeCode(layers.ICMPv4TypeEchoRequest, 0), Id: 1},
		gopacket.Payload([]byte("routed")),
	)
	if err != nil {
		t.Fatalf("SerializeLayers(): %v", err)
	}
	pkt, err := ParsePacket(buffer.Bytes(), 1)
	if err != nil {
		t.Fatalf("ParsePacket(): %v", err)
	}
	return pkt
}

func dhcpDiscoverRequest(t *testing.T, clientMAC net.HardwareAddr) *Packet {
	t.Helper()
	ip := &layers.IPv4{
		Version: 4, TTL: 64, Protocol: layers.IPProtocolUDP,
		SrcIP: net.IPv4zero, DstIP: net.IPv4bcast,
	}
	udp := &layers.UDP{SrcPort: dhcpClientPort, DstPort: dhcpServerPort}
	if err := udp.SetNetworkLayerForChecksum(ip); err != nil {
		t.Fatalf("SetNetworkLayerForChecksum(): %v", err)
	}
	dhcp := &layers.DHCPv4{
		Operation: layers.DHCPOpRequest, HardwareType: layers.LinkTypeEthernet,
		HardwareLen: dhcpHWAddrLen, Xid: 0x12345678, ClientHWAddr: clientMAC,
		Options: []layers.DHCPOption{
			{Type: layers.DHCPOptMessageType, Length: 1, Data: []byte{DHCPDiscover}},
			{Type: layers.DHCPOptEnd},
		},
	}
	buffer := gopacket.NewSerializeBuffer()
	err := gopacket.SerializeLayers(buffer, gopacket.SerializeOptions{
		FixLengths: true, ComputeChecksums: true,
	},
		&layers.Ethernet{
			SrcMAC: clientMAC, DstMAC: net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
			EthernetType: layers.EthernetTypeIPv4,
		},
		ip, udp, dhcp,
	)
	if err != nil {
		t.Fatalf("SerializeLayers(): %v", err)
	}
	pkt, err := ParsePacket(buffer.Bytes(), 4)
	if err != nil {
		t.Fatalf("ParsePacket(): %v", err)
	}
	return pkt
}

func routedSNMPRequest(t *testing.T, testerMAC, routerMAC net.HardwareAddr) *Packet {
	t.Helper()
	payload, err := (&gosnmp.SnmpPacket{
		Version: gosnmp.Version2c, Community: "NetAllyDemo", PDUType: gosnmp.GetRequest,
		RequestID: 1, Variables: []gosnmp.SnmpPDU{{Name: ".1.3.6.1.2.1.1.5.0", Type: gosnmp.Null}},
	}).MarshalMsg()
	if err != nil {
		t.Fatalf("MarshalMsg(): %v", err)
	}
	ip := &layers.IPv4{
		Version: 4, TTL: 64, Protocol: layers.IPProtocolUDP,
		SrcIP: net.ParseIP("10.10.200.200"), DstIP: net.ParseIP("10.20.0.10"),
	}
	udp := &layers.UDP{SrcPort: 40000, DstPort: UDPPortSNMP}
	if err = udp.SetNetworkLayerForChecksum(ip); err != nil {
		t.Fatalf("SetNetworkLayerForChecksum(): %v", err)
	}
	buffer := gopacket.NewSerializeBuffer()
	err = gopacket.SerializeLayers(buffer, gopacket.SerializeOptions{
		FixLengths: true, ComputeChecksums: true,
	},
		&layers.Ethernet{SrcMAC: testerMAC, DstMAC: routerMAC, EthernetType: layers.EthernetTypeIPv4},
		ip, udp, gopacket.Payload(payload),
	)
	if err != nil {
		t.Fatalf("SerializeLayers(): %v", err)
	}
	pkt, err := ParsePacket(buffer.Bytes(), 2)
	if err != nil {
		t.Fatalf("ParsePacket(): %v", err)
	}
	return pkt
}

func routedDNSRequest(t *testing.T, testerMAC, routerMAC net.HardwareAddr) *Packet {
	t.Helper()
	ip := &layers.IPv4{
		Version: 4, TTL: 64, Protocol: layers.IPProtocolUDP,
		SrcIP: net.ParseIP("10.10.200.200"), DstIP: net.ParseIP("10.20.0.10"),
	}
	udp := &layers.UDP{SrcPort: 40001, DstPort: UDPPortDNS}
	if err := udp.SetNetworkLayerForChecksum(ip); err != nil {
		t.Fatalf("SetNetworkLayerForChecksum(): %v", err)
	}
	dns := &layers.DNS{
		ID: 1, RD: true,
		Questions: []layers.DNSQuestion{{
			Name: []byte("service.demo"), Type: layers.DNSTypeA, Class: layers.DNSClassIN,
		}},
	}
	return serializeRoutedRequest(t, testerMAC, routerMAC, ip, udp, dns)
}

func routedTCPRequest(
	t *testing.T,
	testerMAC, routerMAC net.HardwareAddr,
	port layers.TCPPort,
) *Packet {
	t.Helper()
	ip := &layers.IPv4{
		Version: 4, TTL: 64, Protocol: layers.IPProtocolTCP,
		SrcIP: net.ParseIP("10.10.200.200"), DstIP: net.ParseIP("10.20.0.10"),
	}
	tcp := &layers.TCP{SrcPort: 40002, DstPort: port, Seq: 100, SYN: true, Window: 65535}
	if err := tcp.SetNetworkLayerForChecksum(ip); err != nil {
		t.Fatalf("SetNetworkLayerForChecksum(): %v", err)
	}
	return serializeRoutedRequest(t, testerMAC, routerMAC, ip, tcp)
}

func serializeRoutedRequest(
	t *testing.T,
	testerMAC, routerMAC net.HardwareAddr,
	layersToSerialize ...gopacket.SerializableLayer,
) *Packet {
	t.Helper()
	allLayers := []gopacket.SerializableLayer{&layers.Ethernet{
		SrcMAC: testerMAC, DstMAC: routerMAC, EthernetType: layers.EthernetTypeIPv4,
	}}
	allLayers = append(allLayers, layersToSerialize...)
	buffer := gopacket.NewSerializeBuffer()
	if err := gopacket.SerializeLayers(buffer, gopacket.SerializeOptions{
		FixLengths: true, ComputeChecksums: true,
	}, allLayers...); err != nil {
		t.Fatalf("SerializeLayers(): %v", err)
	}
	pkt, err := ParsePacket(buffer.Bytes(), 5)
	if err != nil {
		t.Fatalf("ParsePacket(): %v", err)
	}
	return pkt
}

func routedTaggedSNMPRequest(t *testing.T, testerMAC, routerMAC net.HardwareAddr) *Packet {
	t.Helper()
	pkt := routedSNMPRequest(t, testerMAC, routerMAC)
	frame := insertDot1Q(pkt.Buffer[:pkt.Length], 200)
	tagged, err := ParsePacket(frame, 3)
	if err != nil {
		t.Fatalf("ParsePacket(tagged): %v", err)
	}
	return tagged
}

func forwardingFixture(t *testing.T) (*config.Config, *fabric.Topology, net.HardwareAddr) {
	t.Helper()
	routerMAC := mustForwardingMAC(t, "02:00:00:00:00:01")
	cfg := &config.Config{
		Networks: []config.Network{
			{Name: "attachment", Subnet: "10.10.200.0/24"},
			{Name: "internal", Subnet: "10.20.0.0/24"},
		},
		Attachments: []config.LogicalAttachment{{Name: "tester", Network: "attachment"}},
		Devices: []config.Device{
			{
				Name: "edge", Type: "router", MACAddress: routerMAC,
				Interfaces: []config.Interface{
					{Name: "outside", Network: "attachment", Address: "10.10.200.1/24"},
					{Name: "inside", Network: "internal", Address: "10.20.0.1/24"},
				},
			},
			{
				Name: "server", Type: "server", MACAddress: mustForwardingMAC(t, "02:00:00:00:00:10"),
				SNMPConfig: config.SNMPConfig{Community: "NetAllyDemo", SysName: "internal-server"},
				Interfaces: []config.Interface{
					{Name: "eth0", Network: "internal", Address: "10.20.0.10/24"},
				},
			},
		},
	}
	report := fabric.Compile(cfg, fabric.Binding{
		Attachment: "tester", Interface: "eth0", Mode: fabric.ModeAccess, AccessVLAN: 200,
		PolicyApproved: true,
	})
	if !report.Safe {
		t.Fatalf("Compile() diagnostics = %#v", report.Diagnostics)
	}
	return cfg, &report.Topology, routerMAC
}

func TestFabricResolutionUsesCurrentAttachmentAddress(t *testing.T) {
	cfg, topology, routerMAC := forwardingFixture(t)
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	stack.ConfigureFabric(topology)
	router := &cfg.Devices[0]
	if err := stack.deviceStates[router].UpdateInterface(
		"outside", func(iface devicestate.Interface) (devicestate.Interface, error) {
			iface.Address = netip.MustParsePrefix("10.10.200.2/24")
			return iface, nil
		},
	); err != nil {
		t.Fatal(err)
	}
	resolution, found := stack.fabric.resolveIPv4(netip.MustParseAddr("10.20.0.10"), routerMAC)
	if !found || resolution.firstHopIP != netip.MustParseAddr("10.10.200.2") {
		t.Fatalf("resolution = %#v, want current attachment address", resolution)
	}
}

func TestFabricResolutionRejectsAttachmentWithoutCurrentAddress(t *testing.T) {
	cfg, topology, routerMAC := forwardingFixture(t)
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	stack.ConfigureFabric(topology)
	router := &cfg.Devices[0]
	if err := stack.deviceStates[router].UpdateInterface(
		"outside", func(iface devicestate.Interface) (devicestate.Interface, error) {
			iface.Address = netip.Prefix{}
			return iface, nil
		},
	); err != nil {
		t.Fatal(err)
	}
	if resolution, found := stack.fabric.resolveIPv4(netip.MustParseAddr("10.20.0.10"), routerMAC); found {
		t.Fatalf("resolution = %#v, want no route without a current attachment address", resolution)
	}
}

func mustForwardingMAC(t *testing.T, value string) net.HardwareAddr {
	t.Helper()
	mac, err := net.ParseMAC(value)
	if err != nil {
		t.Fatalf("ParseMAC(%q): %v", value, err)
	}
	return mac
}
