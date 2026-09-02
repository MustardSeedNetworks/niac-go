//go:build linux && integration

package wiretest_test

import (
	"bytes"
	"net"
	"slices"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcap"
	"github.com/gosnmp/gosnmp"
)

// DHCP is the one exchange where the simulation has to carry state across four
// packets. Asserting only that an OFFER arrived would miss the case that
// matters operationally: an address outside the authored pool, or an ACK that
// contradicts the OFFER it followed.
func TestDHCPLeaseComesFromTheAuthoredPool(t *testing.T) {
	authored := startHospital(t)
	edge := edgeRouter(t, authored)
	if edge.DHCPConfig == nil {
		t.Fatalf(
			"%s has no authored DHCP server; the pack no longer serves leases on the transit network",
			edgeRouterName,
		)
	}
	poolStart, poolEnd := edge.DHCPConfig.PoolStart, edge.DHCPConfig.PoolEnd

	handle := openClient(t)
	src := clientMAC(t)
	xid := uint32(0x9e57c0de)

	offer := dhcpExchange(t, handle, src, xid, layers.DHCPMsgTypeDiscover, nil)
	offered := append(net.IP(nil), offer.YourClientIP...)
	if !addrWithin(offered, poolStart, poolEnd) {
		t.Fatalf(
			"DHCPOFFER yiaddr = %s, outside the authored pool %s-%s",
			offered,
			poolStart,
			poolEnd,
		)
	}

	ack := dhcpExchange(t, handle, src, xid, layers.DHCPMsgTypeRequest, offered)
	if !ack.YourClientIP.Equal(offered) {
		t.Errorf(
			"DHCPACK yiaddr = %s, want the offered %s — the server did not honour its own offer",
			ack.YourClientIP,
			offered,
		)
	}
	if router := dhcpOptionIP(ack, layers.DHCPOptRouter); !router.Equal(edge.DHCPConfig.Router) {
		t.Errorf("DHCPACK router option = %s, want the authored %s", router, edge.DHCPConfig.Router)
	}
}

// addrWithin reports whether addr falls inside the inclusive authored range.
func addrWithin(addr, start, end net.IP) bool {
	a, s, e := addr.To4(), start.To4(), end.To4()
	if a == nil || s == nil || e == nil {
		return false
	}
	return bytes.Compare(a, s) >= 0 && bytes.Compare(a, e) <= 0
}

func dhcpOptionIP(pkt *layers.DHCPv4, want layers.DHCPOpt) net.IP {
	for _, opt := range pkt.Options {
		if opt.Type == want && len(opt.Data) >= 4 {
			return net.IP(opt.Data[:4])
		}
	}
	return nil
}

func dhcpMessageType(pkt *layers.DHCPv4) layers.DHCPMsgType {
	for _, opt := range pkt.Options {
		if opt.Type == layers.DHCPOptMessageType && len(opt.Data) == 1 {
			return layers.DHCPMsgType(opt.Data[0])
		}
	}
	return layers.DHCPMsgTypeUnspecified
}

// dhcpExchange sends one DHCP message and returns the simulation's reply. The
// request is retransmitted on a tick because a real client does the same and
// because the responder and the pcap handle race on a freshly created veth.
func dhcpExchange(
	t *testing.T,
	handle *pcap.Handle,
	src net.HardwareAddr,
	xid uint32,
	kind layers.DHCPMsgType,
	requested net.IP,
) *layers.DHCPv4 {
	t.Helper()

	opts := []layers.DHCPOption{
		{Type: layers.DHCPOptMessageType, Data: []byte{byte(kind)}, Length: 1},
	}
	if requested != nil {
		opts = append(
			opts,
			layers.DHCPOption{Type: layers.DHCPOptRequestIP, Data: requested.To4(), Length: 4},
		)
	}
	dhcp := &layers.DHCPv4{
		Operation:    layers.DHCPOpRequest,
		HardwareType: layers.LinkTypeEthernet,
		HardwareLen:  6,
		Xid:          xid,
		ClientHWAddr: src,
		Flags:        0x8000, // broadcast: the client has no lease to be unicast at
		Options:      opts,
	}
	eth := &layers.Ethernet{
		SrcMAC:       src,
		DstMAC:       net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		EthernetType: layers.EthernetTypeIPv4,
	}
	ip := &layers.IPv4{
		Version:  4,
		TTL:      64,
		Protocol: layers.IPProtocolUDP,
		SrcIP:    net.IPv4zero,
		DstIP:    net.IPv4bcast,
	}
	udp := &layers.UDP{SrcPort: 68, DstPort: 67}
	if err := udp.SetNetworkLayerForChecksum(ip); err != nil {
		t.Fatalf("dhcp checksum setup: %v", err)
	}
	frame := serialize(t, eth, ip, udp, dhcp)

	want := layers.DHCPMsgTypeOffer
	if kind == layers.DHCPMsgTypeRequest {
		want = layers.DHCPMsgTypeAck
	}

	packets := gopacket.NewPacketSource(handle, handle.LinkType()).Packets()
	deadline := time.After(25 * time.Second)
	retry := time.NewTicker(time.Second)
	defer retry.Stop()
	if err := handle.WritePacketData(frame); err != nil {
		t.Fatalf("writing DHCP %v: %v", kind, err)
	}
	for {
		select {
		case packet := <-packets:
			layer := packet.Layer(layers.LayerTypeDHCPv4)
			if layer == nil {
				continue
			}
			reply, ok := layer.(*layers.DHCPv4)
			if !ok || reply.Xid != xid || reply.Operation != layers.DHCPOpReply {
				continue
			}
			if dhcpMessageType(reply) == want {
				return reply
			}
		case <-retry.C:
			if err := handle.WritePacketData(frame); err != nil {
				t.Fatalf("re-writing DHCP %v: %v", kind, err)
			}
		case <-deadline:
			t.Fatalf("no %v within 25s in response to %v", want, kind)
		}
	}
}

// LLDP is unsolicited: nothing the test sends causes it. This asserts the
// simulation advertises itself on its own timer, and that the system name it
// puts on the wire is the authored device name rather than a placeholder.
//
// Only the edge router's advertisement can arrive here — validateDiscoveryEgress
// drops discovery frames from devices that are not on the attachment network,
// so a hospital-site switch's LLDP never reaches this wire.
func TestLLDPAdvertisesTheAuthoredSystemName(t *testing.T) {
	authored := startHospital(t)
	edge := edgeRouter(t, authored)

	want := edge.Name
	if edge.SNMPConfig.SysName != "" {
		want = edge.SNMPConfig.SysName
	}

	handle := openClient(t)
	packets := gopacket.NewPacketSource(handle, handle.LinkType()).Packets()
	// The advertisement interval is 15s, so one full interval plus slack.
	deadline := time.After(40 * time.Second)
	for {
		select {
		case packet := <-packets:
			info := packet.Layer(layers.LayerTypeLinkLayerDiscoveryInfo)
			if info == nil {
				continue
			}
			lldp, ok := info.(*layers.LinkLayerDiscoveryInfo)
			if !ok || lldp.SysName == "" {
				continue
			}
			if lldp.SysName != want {
				t.Fatalf(
					"LLDP system name on the wire = %q, want the authored %q",
					lldp.SysName,
					want,
				)
			}
			return
		case <-deadline:
			t.Fatal("no LLDP advertisement carrying a system name within 40s")
		}
	}
}

// The IF-MIB walk is the assertion that would most easily pass while being
// wrong: a responder that returns the right number of ifTable rows with
// synthesized names looks identical to a correct one under a count assertion.
// Compare the ifDescr values against the authored interface list instead.
func TestSNMPWalkReturnsAuthoredInterfaceNames(t *testing.T) {
	authored := startHospital(t)
	edge := edgeRouter(t, authored)

	community := edge.SNMPConfig.Community
	if community == "" {
		t.Fatal("the edge router has no authored SNMP community")
	}

	client := &gosnmp.GoSNMP{
		Target:    transitGateway,
		Port:      161,
		Community: community,
		Version:   gosnmp.Version2c,
		Timeout:   3 * time.Second,
		Retries:   4,
	}
	if err := client.Connect(); err != nil {
		t.Fatalf("connecting to %s: %v", transitGateway, err)
	}
	t.Cleanup(func() { _ = client.Conn.Close() })

	const ifDescr = ".1.3.6.1.2.1.2.2.1.2"
	results, err := client.WalkAll(ifDescr)
	if err != nil {
		t.Fatalf("walking ifDescr on %s: %v", transitGateway, err)
	}
	if len(results) == 0 {
		t.Fatalf("ifDescr walk returned no rows; the SNMP agent is not answering on the wire")
	}

	got := make([]string, 0, len(results))
	for _, row := range results {
		if value, ok := row.Value.([]byte); ok {
			got = append(got, string(value))
		}
	}

	// TrunkPorts are synthesized ahead of Interfaces, so an authored interface
	// missing from the walk is the failure worth naming precisely.
	for _, iface := range edge.Interfaces {
		if !slices.Contains(got, iface.Name) {
			t.Errorf(
				"authored interface %q is absent from the ifDescr walk; got %v",
				iface.Name,
				got,
			)
		}
	}
}
