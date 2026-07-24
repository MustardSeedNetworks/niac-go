package protocols

import (
	"bytes"
	"net"
	"net/netip"
	"strings"
	"testing"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicecli"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

func TestStackOwnsStateForEveryDevice(t *testing.T) {
	cfg := &config.Config{Devices: []config.Device{{Name: "edge-1"}}}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	device := &cfg.Devices[0]

	state := stack.deviceStates[device]
	if state == nil {
		t.Fatal("device state was not registered")
	}
	if got := state.Snapshot().Identity.Hostname; got != "edge-1" {
		t.Fatalf("hostname = %q, want edge-1", got)
	}
}

func TestDiscoveryAdvertisementsReadAuthoritativeHostname(t *testing.T) {
	cfg := &config.Config{Devices: []config.Device{{
		Name: "edge-1", LLDPConfig: &config.LLDPConfig{Enabled: true},
		CDPConfig: &config.CDPConfig{Enabled: true},
	}}}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	device := &cfg.Devices[0]
	state := stack.deviceStates[device]
	if err := state.UpdateIdentity(func(identity devicestate.Identity) (devicestate.Identity, error) {
		identity.Hostname = "branch-1"
		return identity, nil
	}); err != nil {
		t.Fatalf("UpdateIdentity() error = %v", err)
	}

	lldp := stack.lldpHandler.buildSystemNameTLV(device)
	cdp := stack.cdpHandler.buildDeviceIDTLV(device)
	if !bytes.Equal(lldp[2:], []byte("branch-1")) || !bytes.Equal(cdp[4:], []byte("branch-1")) {
		t.Fatalf("LLDP name = %q CDP name = %q", lldp[2:], cdp[4:])
	}
}

func TestSNMPAgentConsumesStackDeviceState(t *testing.T) {
	cfg := &config.Config{Devices: []config.Device{{
		Name:       "edge-1",
		SNMPConfig: config.SNMPConfig{Community: "public"},
	}}}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	device := &cfg.Devices[0]
	group := stack.snmpAgents[device]

	if group == nil || group.state != stack.deviceStates[device] {
		t.Fatal("SNMP agent does not share the stack-owned device state")
	}
}

func TestConfigureFabricSeedsAuthoritativeNetworkState(t *testing.T) {
	cfg := &config.Config{Devices: []config.Device{{
		Name: "edge-1",
		Interfaces: []config.Interface{{
			Name: "Gi0/1", AdminStatus: "down", Description: "WAN", VLANs: []int{200},
		}},
	}}}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	stack.ConfigureFabric(testDeviceStateTopology())

	snapshot := stack.deviceStates[&cfg.Devices[0]].Snapshot()
	if len(snapshot.Network.Interfaces) != 1 || len(snapshot.Network.Routes) != 1 {
		t.Fatalf("network state = %#v", snapshot.Network)
	}
	iface := snapshot.Network.Interfaces[0]
	if iface.AdminUp || iface.OperUp || iface.Description != "WAN" || iface.VLANs[0] != 200 {
		t.Fatalf("interface state = %#v", iface)
	}
}

func TestFlatConfigSeedsAuthoritativeNetworkState(t *testing.T) {
	cfg := &config.Config{Devices: []config.Device{{
		Name: "edge-1", IPAddresses: []net.IP{net.ParseIP("192.0.2.10")},
		Routes: []config.Route{{Destination: "198.51.100.0/24", Via: "Management", NextHop: "192.0.2.1"}},
	}}}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))

	snapshot := stack.deviceStates[&cfg.Devices[0]].Snapshot()
	if len(snapshot.Network.Interfaces) != 1 || len(snapshot.Network.Routes) != 2 {
		t.Fatalf("network state = %#v", snapshot.Network)
	}
	if got := snapshot.Network.Interfaces[0]; got.Name != "Management" || got.Address.String() != "192.0.2.10/32" {
		t.Fatalf("interface state = %#v", got)
	}
	if got := snapshot.Network.Routes[1]; got.Destination.String() != "198.51.100.0/24" ||
		got.NextHop.String() != "192.0.2.1" {
		t.Fatalf("static route = %#v", got)
	}
}

func TestFlatConfigReloadReseedsAuthoritativeNetworkState(t *testing.T) {
	cfg := &config.Config{Devices: []config.Device{{
		Name: "edge-1", IPAddresses: []net.IP{net.ParseIP("192.0.2.10")},
	}}}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	replacement := &config.Config{Devices: []config.Device{{
		Name: "edge-2", IPAddresses: []net.IP{net.ParseIP("198.51.100.20")},
	}}}

	if err := stack.ReloadConfig(replacement); err != nil {
		t.Fatalf("ReloadConfig() error = %v", err)
	}
	snapshot := stack.deviceStates[&replacement.Devices[0]].Snapshot()
	if len(snapshot.Network.Interfaces) != 1 ||
		snapshot.Network.Interfaces[0].Address.String() != "198.51.100.20/32" {
		t.Fatalf("network state = %#v", snapshot.Network)
	}
}

func TestFlatConfigReloadDetachesPreviousAddressIndex(t *testing.T) {
	initial := &config.Config{Devices: []config.Device{{
		Name: "edge-1", IPAddresses: []net.IP{net.ParseIP("192.0.2.10")},
	}}}
	stack := NewStack(nil, initial, logging.NewDebugConfig(0))
	previous := stack.deviceStates[&initial.Devices[0]]
	replacement := &config.Config{Devices: []config.Device{{
		Name: "edge-2", IPAddresses: []net.IP{net.ParseIP("198.51.100.20")},
	}}}
	if err := stack.ReloadConfig(replacement); err != nil {
		t.Fatalf("ReloadConfig() error = %v", err)
	}
	if err := previous.UpdateInterface("Management", func(iface devicestate.Interface) (devicestate.Interface, error) {
		iface.Address = netip.MustParsePrefix("203.0.113.7/32")
		return iface, nil
	}); err != nil {
		t.Fatalf("previous UpdateInterface() error = %v", err)
	}
	if devices := stack.devicesForStateIPv4(0, net.ParseIP("203.0.113.7")); len(devices) != 0 {
		t.Fatalf("stale address devices = %#v", devices)
	}
}

func TestFlatCLIAddressAndShutdownChangePacketDelivery(t *testing.T) {
	cfg := &config.Config{Devices: []config.Device{{
		Name: "edge-1", MACAddress: mustParseMAC(t, "02:00:00:00:00:01"),
		IPAddresses: []net.IP{net.ParseIP("192.0.2.10")},
		ICMPConfig:  &config.ICMPConfig{AddressMaskReply: net.IP(net.CIDRMask(24, 32))},
		DHCPConfig:  &config.DHCPConfig{},
	}}}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	device := &cfg.Devices[0]
	session := devicecli.NewSession(
		stack.deviceStates[device],
		stack.staticRouteValidator(device),
	)
	for _, command := range []string{
		"enable", "configure terminal", "interface Management", "ip address 192.0.2.20/32",
	} {
		if response := session.Execute(command); strings.HasPrefix(response.Output, "%") {
			t.Fatalf("%q response = %#v", command, response)
		}
	}

	if targets := stack.ipHandler.getTargetDevices(
		&layers.IPv4{DstIP: net.ParseIP("192.0.2.10")}, NewPacket(64), 0, 0,
	); targets != nil {
		t.Fatalf("old address targets = %#v", targets)
	}
	targets := stack.arpHandler.targetDevices(net.ParseIP("192.0.2.20"), 0)
	if len(targets) != 1 || targets[0] != device {
		t.Fatalf("new address ARP targets = %#v", targets)
	}
	newAddress := net.ParseIP("192.0.2.20")
	if advertised := stack.firstStateIPAddress(device); !advertised.Equal(newAddress) {
		t.Fatalf("advertised address = %v, want %v", advertised, newAddress)
	}
	if !stack.deviceOwnsIPv4(device, newAddress) {
		t.Fatal("new address rejected by protocol ownership check")
	}
	if got := stack.tcpHandler.findDeviceWithIP(targets, newAddress); got != device {
		t.Fatalf("new address TCP device = %#v", got)
	}
	if serverIP := stack.dhcpHandler.DHCPHandlerServerIP(); !serverIP.Equal(newAddress) {
		t.Fatalf("DHCP server address = %v, want %v", serverIP, newAddress)
	}
	assertAddressMaskReplySource(t, stack, device, newAddress)
	if response := session.Execute("shutdown"); strings.HasPrefix(response.Output, "%") {
		t.Fatalf("shutdown response = %#v", response)
	}
	if stack.deviceOwnsIPv4(device, newAddress) {
		t.Fatal("shutdown address accepted by protocol ownership check")
	}
	if shutdownTargets := stack.ipHandler.getTargetDevices(
		&layers.IPv4{DstIP: net.ParseIP("192.0.2.20")}, NewPacket(64), 0, 0,
	); shutdownTargets != nil {
		t.Fatalf("shutdown address targets = %#v", shutdownTargets)
	}
}

func TestAddressMaskBroadcastIPPreservesUnicastEthernetTarget(t *testing.T) {
	cfg := &config.Config{Devices: []config.Device{
		{
			Name: "edge-1", MACAddress: mustParseMAC(t, "02:00:00:00:00:01"),
			IPAddresses: []net.IP{net.ParseIP("192.0.2.10")},
			ICMPConfig:  &config.ICMPConfig{AddressMaskReply: net.IP(net.CIDRMask(24, 32))},
		},
		{
			Name: "edge-2", MACAddress: mustParseMAC(t, "02:00:00:00:00:02"),
			IPAddresses: []net.IP{net.ParseIP("192.0.2.20")},
			ICMPConfig:  &config.ICMPConfig{AddressMaskReply: net.IP(net.CIDRMask(24, 32))},
		},
	}}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	sourceMAC := mustParseMAC(t, "02:00:00:00:00:fe")
	ipLayer := &layers.IPv4{
		Version: 4, IHL: 5, TTL: 64, Protocol: layers.IPProtocolICMPv4,
		SrcIP: net.ParseIP("192.0.2.100"), DstIP: net.IPv4bcast,
	}
	icmp := &layers.ICMPv4{TypeCode: layers.CreateICMPv4TypeCode(layers.ICMPv4TypeAddressMaskRequest, 0)}
	buffer := gopacket.NewSerializeBuffer()
	if err := gopacket.SerializeLayers(buffer, gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true},
		&layers.Ethernet{SrcMAC: sourceMAC, DstMAC: cfg.Devices[0].MACAddress, EthernetType: layers.EthernetTypeIPv4},
		ipLayer, icmp,
	); err != nil {
		t.Fatalf("SerializeLayers() error = %v", err)
	}
	stack.icmpHandler.HandlePacket(
		&Packet{Buffer: buffer.Bytes(), Length: len(buffer.Bytes())}, ipLayer,
		[]*config.Device{&cfg.Devices[0], &cfg.Devices[1]},
	)
	if replies := len(stack.sendQueue); replies != 1 {
		t.Fatalf("address-mask replies = %d, want 1", replies)
	}
	reply := <-stack.sendQueue
	if source := reply.GetSourceMAC(); !bytes.Equal(source, cfg.Devices[0].MACAddress) {
		t.Fatalf("address-mask reply source MAC = %s, want %s", source, cfg.Devices[0].MACAddress)
	}
}

func TestStateIPv4IndexDeduplicatesDeviceAddresses(t *testing.T) {
	cfg := &config.Config{Devices: []config.Device{{
		Name: "edge-1", IPAddresses: []net.IP{net.ParseIP("192.0.2.10")},
	}}}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	device := &cfg.Devices[0]
	network := stack.deviceStates[device].Snapshot().Network
	network.Interfaces = append(network.Interfaces, devicestate.Interface{
		Name: "duplicate", Address: netip.MustParsePrefix("192.0.2.10/32"),
		AdminUp: true, OperUp: true, CarrierUp: true,
	})
	stack.deviceStates[device].ReplaceNetwork(network)

	devices := stack.devicesForStateIPv4(0, net.ParseIP("192.0.2.10"))
	if len(devices) != 1 || devices[0] != device {
		t.Fatalf("indexed devices = %#v", devices)
	}
}

func assertAddressMaskReplySource(t *testing.T, stack *Stack, device *config.Device, address net.IP) {
	t.Helper()
	sourceMAC := mustParseMAC(t, "02:00:00:00:00:fe")
	ipLayer := &layers.IPv4{
		Version: 4, IHL: 5, TTL: 64, Protocol: layers.IPProtocolICMPv4,
		SrcIP: net.ParseIP("192.0.2.100"), DstIP: address,
	}
	icmp := &layers.ICMPv4{TypeCode: layers.CreateICMPv4TypeCode(layers.ICMPv4TypeAddressMaskRequest, 0)}
	buffer := gopacket.NewSerializeBuffer()
	if err := gopacket.SerializeLayers(buffer, gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true},
		&layers.Ethernet{SrcMAC: sourceMAC, DstMAC: device.MACAddress, EthernetType: layers.EthernetTypeIPv4},
		ipLayer, icmp,
	); err != nil {
		t.Fatalf("SerializeLayers() error = %v", err)
	}
	stack.icmpHandler.HandlePacket(
		&Packet{Buffer: buffer.Bytes(), Length: len(buffer.Bytes())}, ipLayer, []*config.Device{device},
	)
	reply := <-stack.sendQueue
	packet := gopacket.NewPacket(reply.Buffer, layers.LayerTypeEthernet, gopacket.Default)
	replyIP, ok := packet.Layer(layers.LayerTypeIPv4).(*layers.IPv4)
	if !ok || !replyIP.SrcIP.Equal(address) {
		t.Fatalf("address-mask reply IPv4 = %#v, want source %s", replyIP, address)
	}
}

func testDeviceStateTopology() *fabric.Topology {
	return &fabric.Topology{
		Interfaces: []fabric.Interface{{
			Device: "edge-1", Name: "Gi0/1", Network: "wan", Address: netip.MustParsePrefix("10.0.0.1/24"),
		}},
		Routes: []fabric.Route{{
			Device: "edge-1", Destination: netip.MustParsePrefix("10.0.0.0/24"), Via: "Gi0/1", Connected: true,
		}},
	}
}
