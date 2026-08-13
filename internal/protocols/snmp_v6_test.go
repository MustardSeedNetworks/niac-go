package protocols

import (
	"net"
	"testing"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

// A simulator whose job is answering scanners answered none of them over IPv6:
// the dispatch logged "not yet implemented" to stdout and dropped the datagram,
// while the same query over IPv4 was served. The engine underneath never cared
// which IP version carried it.
func TestSNMPAnswersOverIPv6(t *testing.T) {
	stack, device := ipv6SNMPStack(t)
	deviceIP := net.ParseIP("2001:db8::10")

	reply := querySNMPv6(t, stack, device, net.ParseIP("2001:db8::1"), deviceIP)
	if reply == nil {
		t.Fatal("no reply was queued for an IPv6 SNMP query")
	}
	if !reply.NetworkLayer().(*layers.IPv6).SrcIP.Equal(deviceIP) {
		t.Errorf("reply source = %s, want the address that was queried (%s)",
			reply.NetworkLayer().(*layers.IPv6).SrcIP, deviceIP)
	}
}

// A manager only accepts a reply from the address it queried, which is what
// makes this matter for a link-local address rather than a stylistic choice.
func TestSNMPAnswersFromTheLinkLocalAddressItWasAsked(t *testing.T) {
	stack, device := ipv6SNMPStack(t)
	linkLocal := net.ParseIP("fe80::20")

	reply := querySNMPv6(t, stack, device, net.ParseIP("fe80::1"), linkLocal)
	if reply == nil {
		t.Fatal("no reply was queued for a link-local SNMP query")
	}
	if got := reply.NetworkLayer().(*layers.IPv6).SrcIP; !got.Equal(linkLocal) {
		t.Errorf("reply source = %s, want %s", got, linkLocal)
	}
}

// The query counter is a protocol counter, not a per-version one.
func TestIPv6QueriesAreCounted(t *testing.T) {
	stack, device := ipv6SNMPStack(t)

	querySNMPv6(t, stack, device, net.ParseIP("2001:db8::1"), net.ParseIP("2001:db8::10"))

	if got := stack.GetStats().SNMPQueries; got != 1 {
		t.Errorf("SNMPQueries = %d, want 1", got)
	}
}

// ipv6SNMPStack builds a stack with one SNMP device carrying both a global and
// a link-local address.
func ipv6SNMPStack(t *testing.T) (*Stack, *config.Device) {
	t.Helper()
	mac, err := net.ParseMAC("00:00:0c:33:06:02")
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{Devices: []config.Device{{
		Name: "MED-ACC-SW02", Type: "switch", MACAddress: mac,
		IPAddresses: []net.IP{net.ParseIP("2001:db8::10"), net.ParseIP("fe80::20")},
		SNMPConfig: config.SNMPConfig{
			Community: "NetAllyDemo", SysName: "MED-ACC-SW02",
		},
	}}}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	stack.snmpHandler = NewSNMPHandler(stack)

	return stack, &cfg.Devices[0]
}

// querySNMPv6 hands the handler a v2c GET over IPv6 and returns the reply the
// stack queued, if any.
func querySNMPv6(t *testing.T, stack *Stack, device *config.Device, from, to net.IP) gopacket.Packet {
	t.Helper()
	request := &gosnmp.SnmpPacket{
		Version: gosnmp.Version2c, Community: "NetAllyDemo",
		PDUType: gosnmp.GetRequest, RequestID: 1,
		Variables: []gosnmp.SnmpPDU{{
			Name: "1.3.6.1.2.1.1.5.0", Type: gosnmp.Null,
		}},
	}
	payload, err := request.MarshalMsg()
	if err != nil {
		t.Fatal(err)
	}
	udp := &layers.UDP{SrcPort: 40000, DstPort: 161}
	udp.Payload = payload
	ipv6 := &layers.IPv6{
		Version: snmpIPv6Version, NextHeader: layers.IPProtocolUDP,
		HopLimit: snmpIPv6HopLimit, SrcIP: from, DstIP: to,
	}
	// The handler reads the requester's MAC off the frame, so the packet needs
	// a real Ethernet header rather than a bare struct.
	pkt := &Packet{SerialNumber: 1, Buffer: ethernetHeader(
		net.HardwareAddr{0x02, 0, 0, 0, 0, 1}, net.HardwareAddr{0xaa, 0, 0, 0, 0, 1},
	)}

	stack.snmpHandler.HandlePacketV6(pkt, ipv6, udp, []*config.Device{device})

	select {
	case queued := <-stack.sendQueue:
		return gopacket.NewPacket(queued.Buffer, layers.LayerTypeEthernet, gopacket.Default)
	default:
		return nil
	}
}

// ethernetHeader builds the 14-byte header the packet helpers read MACs from.
func ethernetHeader(dst, src net.HardwareAddr) []byte {
	frame := make([]byte, 14)
	copy(frame[0:6], dst)
	copy(frame[6:12], src)
	frame[12], frame[13] = 0x86, 0xdd

	return frame
}
