//go:build linux && integration

package wiretest_test

import (
	"net"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcap"
)

// transitGateway is the edge router on the transit network the authored
// fixtures in this package attach to. A generated pack attaches its tester
// elsewhere -- a spare access port -- and startPack reads where from the pack.
const (
	transitGateway = "10.254.200.1"
	accessVLAN     = 200
)

// openClient opens libpcap on the test end with immediate delivery, so a reply
// is readable as soon as it arrives instead of sitting in a buffer.
func openClient(t *testing.T) *pcap.Handle {
	t.Helper()
	inactive, err := pcap.NewInactiveHandle(testIface)
	if err != nil {
		t.Fatalf("pcap.NewInactiveHandle(%s): %v", testIface, err)
	}
	defer inactive.CleanUp()
	_ = inactive.SetSnapLen(65536)
	_ = inactive.SetPromisc(true)
	_ = inactive.SetTimeout(pcap.BlockForever)
	_ = inactive.SetImmediateMode(true)
	handle, err := inactive.Activate()
	if err != nil {
		t.Fatalf("activating pcap on %s: %v", testIface, err)
	}
	t.Cleanup(handle.Close)
	return handle
}

// clientMAC is the hardware address the kernel gave the test end of the veth.
func clientMAC(t *testing.T) net.HardwareAddr {
	t.Helper()
	iface, err := net.InterfaceByName(testIface)
	if err != nil {
		t.Fatalf("net.InterfaceByName(%s): %v", testIface, err)
	}
	return iface.HardwareAddr
}

func serialize(t *testing.T, ls ...gopacket.SerializableLayer) []byte {
	t.Helper()
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
	if err := gopacket.SerializeLayers(buf, opts, ls...); err != nil {
		t.Fatalf("serializing frame: %v", err)
	}
	return buf.Bytes()
}

// ARP is the cheapest end-to-end proof that the simulation is on the wire: it
// exercises capture, device lookup and frame injection, and its answer carries
// an authored field -- the gateway's MAC -- that a count-based assertion would
// miss entirely. The gateway is the first hop a tester on the pack's attachment
// port resolves, which is what a technician's tool does first.
func TestARPAnswersWithTheAttachmentGatewayMAC(t *testing.T) {
	_, attachment := startPack(t, "hospital")
	gateway := attachment.gatewayDevice

	handle := openClient(t)
	src := clientMAC(t)

	eth := &layers.Ethernet{
		SrcMAC:       src,
		DstMAC:       net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		EthernetType: layers.EthernetTypeARP,
	}
	arp := &layers.ARP{
		AddrType:          layers.LinkTypeEthernet,
		Protocol:          layers.EthernetTypeIPv4,
		HwAddressSize:     6,
		ProtAddressSize:   4,
		Operation:         layers.ARPRequest,
		SourceHwAddress:   src,
		SourceProtAddress: attachment.client.Addr().AsSlice(),
		DstHwAddress:      net.HardwareAddr{0, 0, 0, 0, 0, 0},
		DstProtAddress:    attachment.gateway.AsSlice(),
	}
	frame := serialize(t, eth, arp)

	reply := awaitARPReply(t, handle, frame, attachment.gateway.AsSlice())

	if got := net.HardwareAddr(reply.SourceHwAddress).String(); got != gateway.MACAddress.String() {
		t.Errorf("ARP reply for %s came from MAC %s, want the authored %s MAC %s",
			attachment.gateway, got, gateway.Name, gateway.MACAddress)
	}
}

// awaitARPReply retransmits the request while waiting, because the responder's
// first advertisement and the pcap handle coming up race on a fresh interface.
func awaitARPReply(t *testing.T, handle *pcap.Handle, request []byte, want net.IP) *layers.ARP {
	t.Helper()
	source := gopacket.NewPacketSource(handle, handle.LinkType())
	packets := source.Packets()
	deadline := time.After(20 * time.Second)
	retry := time.NewTicker(500 * time.Millisecond)
	defer retry.Stop()

	if err := handle.WritePacketData(request); err != nil {
		t.Fatalf("writing ARP request: %v", err)
	}
	for {
		select {
		case packet := <-packets:
			layer := packet.Layer(layers.LayerTypeARP)
			if layer == nil {
				continue
			}
			arp, ok := layer.(*layers.ARP)
			if !ok || arp.Operation != layers.ARPReply {
				continue
			}
			if net.IP(arp.SourceProtAddress).Equal(want) {
				return arp
			}
		case <-retry.C:
			if err := handle.WritePacketData(request); err != nil {
				t.Fatalf("re-writing ARP request: %v", err)
			}
		case <-deadline:
			t.Fatalf(
				"no ARP reply for %s within 20s; the simulation is not answering on %s",
				want,
				simIface,
			)
		}
	}
}
