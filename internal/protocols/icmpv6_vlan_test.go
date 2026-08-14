package protocols

import (
	"net"
	"testing"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

// NDP is how a host finds a device before it can send it anything, so a
// simulation that does not answer Neighbor Solicitation is unreachable over
// IPv6 no matter which protocols it serves above. These tests cover the two
// ways that went wrong on a tagged segment: the lookup consulted the flat
// device table rather than the segment's, and the answer went out untagged.

const ndpTestVLAN = 200

// TestNeighborSolicitationOnATaggedSegmentIsAnswered: with segments configured,
// devices live in per-tag tables and the flat table is empty, so a lookup that
// ignores the frame's VLAN finds nothing and the device never answers.
func TestNeighborSolicitationOnATaggedSegmentIsAnswered(t *testing.T) {
	stack, device := ndpSegmentStack(t)

	solicit(t, stack, device, net.ParseIP("fd00:6a::21"))

	select {
	case <-stack.sendQueue:
	default:
		t.Fatal("no Neighbor Advertisement queued: the device is unreachable over IPv6 on its own segment")
	}
}

// TestNeighborAdvertisementEchoesVLAN: a tester on an 802.1Q trunk drops an
// untagged reply before it reaches the sender -- the same defect fixed for ARP
// in #876 and for DNS later.
func TestNeighborAdvertisementEchoesVLAN(t *testing.T) {
	stack, device := ndpSegmentStack(t)

	solicit(t, stack, device, net.ParseIP("fd00:6a::21"))

	select {
	case pkt := <-stack.sendQueue:
		if pkt.VLAN != ndpTestVLAN {
			t.Errorf("Neighbor Advertisement VLAN = %d, want %d (must echo the request VLAN)",
				pkt.VLAN, ndpTestVLAN)
		}
	default:
		t.Fatal("no Neighbor Advertisement queued")
	}
}

// TestNeighborAdvertisementAnswersForTheAddressSolicited: a device holds both a
// global and a link-local address. A host that solicited the link-local rejects
// an advertisement naming any other address, so answering from whichever
// address happens to be first leaves link-local unusable.
func TestNeighborAdvertisementAnswersForTheAddressSolicited(t *testing.T) {
	stack, device := ndpSegmentStack(t)
	linkLocal := net.ParseIP("fe80::6a:21")

	solicit(t, stack, device, linkLocal)

	select {
	case pkt := <-stack.sendQueue:
		reply := gopacket.NewPacket(pkt.Buffer[:pkt.Length], layers.LayerTypeEthernet, gopacket.Default)
		ipv6, ok := reply.Layer(layers.LayerTypeIPv6).(*layers.IPv6)
		if !ok {
			t.Fatal("advertisement carried no IPv6 layer")
		}
		if !ipv6.SrcIP.Equal(linkLocal) {
			t.Errorf("advertisement source = %s, want %s (the address solicited)", ipv6.SrcIP, linkLocal)
		}
	default:
		t.Fatal("no Neighbor Advertisement queued")
	}
}

// TestNeighborAdvertisementCarriesItsBody: the advertisement's target address
// and link-layer option live in the message body. A reply carrying only the
// 4-byte ICMPv6 header parses as an advertisement and tells the host nothing,
// so the neighbour entry stays unresolved and the address is unusable.
func TestNeighborAdvertisementCarriesItsBody(t *testing.T) {
	stack, device := ndpSegmentStack(t)
	target := net.ParseIP("fd00:6a::21")

	solicit(t, stack, device, target)

	select {
	case pkt := <-stack.sendQueue:
		reply := gopacket.NewPacket(pkt.Buffer[:pkt.Length], layers.LayerTypeEthernet, gopacket.Default)
		na, ok := reply.Layer(layers.LayerTypeICMPv6NeighborAdvertisement).(*layers.ICMPv6NeighborAdvertisement)
		if !ok {
			t.Fatal("advertisement body missing: the reply carried only an ICMPv6 header")
		}
		if !na.TargetAddress.Equal(target) {
			t.Errorf("advertisement target = %s, want %s", na.TargetAddress, target)
		}
		if len(na.Options) == 0 {
			t.Error("advertisement carried no target link-layer address option")
		}
	default:
		t.Fatal("no Neighbor Advertisement queued")
	}
}

// ndpSegmentStack builds a stack in segment mode -- the shape every scenario
// pack runs in -- holding one device with a global and a link-local address.
func ndpSegmentStack(t *testing.T) (*Stack, *config.Device) {
	t.Helper()

	device := config.Device{
		Name:       "v6-probe-sw01",
		Type:       "switch",
		MACAddress: net.HardwareAddr{0x02, 0x00, 0x00, 0x00, 0x6a, 0x01},
		IPAddresses: []net.IP{
			net.ParseIP("10.66.200.21"),
			net.ParseIP("fd00:6a::21"),
			net.ParseIP("fe80::6a:21"),
		},
	}
	cfg := &config.Config{
		Segments: []config.Segment{{Tag: ndpTestVLAN, Devices: []config.Device{device}}},
	}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))

	found := stack.devicesFor(ndpTestVLAN).GetByIP(net.ParseIP("fd00:6a::21"))
	if len(found) != 1 {
		t.Fatalf("segment table holds %d devices for fd00:6a::21, want 1", len(found))
	}

	return stack, found[0]
}

// solicit hands the handler a Neighbor Solicitation for target, tagged onto the
// segment, exactly as a host on the trunk sends it: to the solicited-node
// multicast address, not to the device.
func solicit(t *testing.T, stack *Stack, device *config.Device, target net.IP) {
	t.Helper()

	from := net.ParseIP("fd00:6a::240")
	senderMAC := net.HardwareAddr{0x02, 0x00, 0x00, 0x00, 0x6a, 0xf0}

	payload := make([]byte, 24)
	payload[0] = ICMPv6TypeNeighborSolicitation
	copy(payload[8:24], target.To16())

	ipv6 := &layers.IPv6{
		Version: 6, NextHeader: layers.IPProtocolICMPv6, HopLimit: 255,
		SrcIP: from, DstIP: solicitedNodeAddress(target), Length: uint16(len(payload)),
	}

	frame := append(ethernetHeader(multicastMACFor(target), senderMAC), encodeIPv6Header(ipv6)...)
	frame = append(frame, payload...)

	packet := gopacket.NewPacket(frame, layers.LayerTypeEthernet, gopacket.Default)
	pkt := &Packet{Buffer: frame, Length: len(frame), VLAN: ndpTestVLAN, VLANTagged: true}

	handler := NewICMPv6Handler(stack, 0)
	handler.handleNeighborSolicitation(pkt, packet, ipv6)

	_ = device
}

// solicitedNodeAddress is ff02::1:ff00:0/104 with the low 24 bits of target,
// per RFC 4291.
func solicitedNodeAddress(target net.IP) net.IP {
	addr := net.ParseIP("ff02::1:ff00:0").To16()
	copy(addr[13:16], target.To16()[13:16])

	return addr
}

// multicastMACFor is 33:33 followed by the low 32 bits of the multicast
// address, per RFC 2464.
func multicastMACFor(target net.IP) net.HardwareAddr {
	solicited := solicitedNodeAddress(target).To16()

	return net.HardwareAddr{0x33, 0x33, solicited[12], solicited[13], solicited[14], solicited[15]}
}

// encodeIPv6Header serializes just the fixed header; the callers above supply
// their own payload rather than a full gopacket stack.
func encodeIPv6Header(ipv6 *layers.IPv6) []byte {
	header := make([]byte, IPv6HeaderSize)
	header[0] = 0x60
	header[4] = byte(ipv6.Length >> 8)
	header[5] = byte(ipv6.Length)
	header[6] = byte(ipv6.NextHeader)
	header[7] = ipv6.HopLimit
	copy(header[8:24], ipv6.SrcIP.To16())
	copy(header[24:40], ipv6.DstIP.To16())

	return header
}
