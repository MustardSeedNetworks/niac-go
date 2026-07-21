package protocols

import (
	"net"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp"
)

func TestIPv4ReassemblyOutOfOrderAndTimeout(t *testing.T) {
	enabled := true
	cfg := &config.Config{Devices: []config.Device{{
		Name: "dns", MACAddress: mustParseMAC(t, "02:00:00:00:00:01"),
		IPAddresses: []net.IP{net.ParseIP("10.0.0.1")}, DNSConfig: &config.DNSConfig{},
		SNMPConfig: config.SNMPConfig{Enabled: &enabled, Community: "public"}, Properties: map[string]string{},
	}}}
	stack := newStack(nil, cfg, logging.NewDebugConfig(0))
	now := time.Unix(100, 0)
	stack.ipHandler.now = func() time.Time { return now }
	first, second := udpFragments(t, &cfg.Devices[0])
	stack.ipHandler.HandlePacket(second)
	stack.ipHandler.HandlePacket(first)
	agent := stack.getSNMPAgents(&cfg.Devices[0]).observer()
	assertStackCounter(t, agent, "1.3.6.1.2.1.4.14.0", 2)
	assertStackCounter(t, agent, "1.3.6.1.2.1.4.15.0", 1)
	assertStackCounter(t, agent, "1.3.6.1.2.1.7.1.0", 1)

	first, _ = udpFragments(t, &cfg.Devices[0])
	stack.ipHandler.HandlePacket(first)
	now = now.Add(time.Minute + time.Second)
	otherFirst, _ := udpFragmentsWithID(t, &cfg.Devices[0], 99)
	stack.ipHandler.HandlePacket(otherFirst)
	assertStackCounter(t, agent, "1.3.6.1.2.1.4.16.0", 1)
}

func udpFragments(t *testing.T, device *config.Device) (*Packet, *Packet) {
	return udpFragmentsWithID(t, device, 42)
}

func udpFragmentsWithID(t *testing.T, device *config.Device, id uint16) (*Packet, *Packet) {
	t.Helper()
	udp := &layers.UDP{SrcPort: 40000, DstPort: 53}
	ipBase := &layers.IPv4{
		Version: 4, IHL: 5, TTL: 64, Id: id, Protocol: layers.IPProtocolUDP,
		SrcIP: net.ParseIP("10.0.0.100"), DstIP: device.IPAddresses[0],
	}
	if err := udp.SetNetworkLayerForChecksum(ipBase); err != nil {
		t.Fatal(err)
	}
	l4 := gopacket.NewSerializeBuffer()
	if err := gopacket.SerializeLayers(
		l4, gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true},
		udp, gopacket.Payload(make([]byte, 24)),
	); err != nil {
		t.Fatal(err)
	}
	data := l4.Bytes()
	return fragmentPacket(t, device, ipBase, data[:16], true, 0), fragmentPacket(t, device, ipBase, data[16:], false, 2)
}

func fragmentPacket(
	t *testing.T,
	device *config.Device,
	base *layers.IPv4,
	payload []byte,
	more bool,
	offset uint16,
) *Packet {
	t.Helper()
	flags := layers.IPv4Flag(0)
	if more {
		flags = layers.IPv4MoreFragments
	}
	ip := *base
	ip.Flags, ip.FragOffset = flags, offset
	eth := &layers.Ethernet{
		SrcMAC: mustParseMAC(t, "02:00:00:00:00:ff"), DstMAC: device.MACAddress,
		EthernetType: layers.EthernetTypeIPv4,
	}
	buf := gopacket.NewSerializeBuffer()
	if err := gopacket.SerializeLayers(
		buf, gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true},
		eth, &ip, gopacket.Payload(payload),
	); err != nil {
		t.Fatal(err)
	}
	return &Packet{Buffer: buf.Bytes(), Length: len(buf.Bytes()), Device: device}
}

func assertStackCounter(t *testing.T, agent interface {
	HandleGet(string) (*snmp.OIDValue, error)
}, oid string, want uint32,
) {
	t.Helper()
	value, err := agent.HandleGet(oid)
	if err != nil {
		t.Fatal(err)
	}
	if value.Value != want {
		t.Errorf("%s=%v want %d", oid, value.Value, want)
	}
}
