package protocols

import (
	"net"
	"testing"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

func TestStackProtocolTelemetryIsPerDevice(t *testing.T) {
	enabled := true
	cfg := &config.Config{Devices: []config.Device{
		{
			Name: "one", MACAddress: mustParseMAC(t, "02:00:00:00:00:01"),
			IPAddresses: []net.IP{net.ParseIP("10.0.0.1")},
			SNMPConfig:  config.SNMPConfig{Enabled: &enabled, Community: "public"}, Properties: map[string]string{},
		},
		{
			Name: "two", MACAddress: mustParseMAC(t, "02:00:00:00:00:02"),
			IPAddresses: []net.IP{net.ParseIP("10.0.0.2")},
			SNMPConfig:  config.SNMPConfig{Enabled: &enabled, Community: "public"}, Properties: map[string]string{},
		},
	}}
	stack := newStack(nil, cfg, logging.NewDebugConfig(0))
	frame, ip := telemetryUDPFrame(t, cfg.Devices[0].MACAddress, cfg.Devices[0].IPAddresses[0])
	pkt := &Packet{Buffer: frame, Length: len(frame), Device: &cfg.Devices[0]}

	stack.recordInboundProtocol(pkt, ip, []*config.Device{&cfg.Devices[0]})
	stack.recordOutboundProtocol(pkt)

	for i, want := range []uint32{1, 0} {
		agent := stack.getSNMPAgents(&cfg.Devices[i]).observer()
		for _, oid := range []string{"1.3.6.1.2.1.7.1.0", "1.3.6.1.2.1.7.4.0"} {
			value, err := agent.HandleGet(oid)
			if err != nil {
				t.Fatalf("device %d GET %s: %v", i, oid, err)
			}
			if value.Value != want {
				t.Errorf("device %d %s = %v, want %d", i, oid, value.Value, want)
			}
		}
	}
}

func TestStackInterfaceTelemetryUsesAuthoredDestinationInterface(t *testing.T) {
	enabled := true
	device := config.Device{
		Name: "switch", MACAddress: mustParseMAC(t, "02:00:00:00:00:01"),
		IPAddresses: []net.IP{net.ParseIP("10.0.0.1")},
		Interfaces:  []config.Interface{{Name: "GigabitEthernet1/0/5", Address: "10.0.0.1/24"}},
		TrunkPorts:  []config.TrunkPort{{Interface: "GigabitEthernet1/0/5"}},
		SNMPConfig:  config.SNMPConfig{Enabled: &enabled, Community: "public"}, Properties: map[string]string{},
	}
	cfg := &config.Config{Devices: []config.Device{device}}
	stack := newStack(nil, cfg, logging.NewDebugConfig(0))
	frame, ip := telemetryUDPFrame(t, cfg.Devices[0].MACAddress, cfg.Devices[0].IPAddresses[0])
	pkt := &Packet{Buffer: frame, Length: len(frame)}

	stack.recordInboundProtocol(pkt, ip, []*config.Device{&cfg.Devices[0]})

	agent := stack.getSNMPAgents(&cfg.Devices[0]).observer()
	value, err := agent.HandleGet("1.3.6.1.2.1.2.2.1.10.1")
	if err != nil {
		t.Fatal(err)
	}
	if value.Value != uint32(len(frame)) {
		t.Fatalf("ifInOctets = %v, want %d", value.Value, len(frame))
	}
}

func telemetryUDPFrame(t *testing.T, dstMAC net.HardwareAddr, dstIP net.IP) ([]byte, *layers.IPv4) {
	t.Helper()
	eth := &layers.Ethernet{
		SrcMAC: mustParseMAC(t, "02:00:00:00:00:ff"), DstMAC: dstMAC,
		EthernetType: layers.EthernetTypeIPv4,
	}
	ip := &layers.IPv4{
		Version: 4, IHL: 5, TTL: 64, Protocol: layers.IPProtocolUDP,
		SrcIP: net.ParseIP("10.0.0.100"), DstIP: dstIP,
	}
	udp := &layers.UDP{SrcPort: 40000, DstPort: 53}
	if err := udp.SetNetworkLayerForChecksum(ip); err != nil {
		t.Fatalf("UDP checksum layer: %v", err)
	}
	buffer := gopacket.NewSerializeBuffer()
	options := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
	if err := gopacket.SerializeLayers(buffer, options, eth, ip, udp); err != nil {
		t.Fatalf("serialize UDP: %v", err)
	}

	return buffer.Bytes(), ip
}

func mustParseMAC(t *testing.T, value string) net.HardwareAddr {
	t.Helper()
	mac, err := net.ParseMAC(value)
	if err != nil {
		t.Fatalf("parse MAC: %v", err)
	}

	return mac
}
