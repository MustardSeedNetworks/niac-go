//go:build linux && integration

package wiretest_test

import (
	"net"
	"net/netip"
	"slices"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcap"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestDuplicateIPOnWire(t *testing.T) {
	requireWire(t)
	handle, err := pcap.OpenLive(testIface, 65536, true, 20*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(handle.Close)
	if err = handle.SetBPFFilter("arp or (vlan and arp)"); err != nil {
		t.Fatal(err)
	}
	probe := conflictARPProbe{
		handle:  handle,
		packets: gopacket.NewPacketSource(handle, handle.LinkType()).Packets(),
		client:  clientMAC(t),
	}
	cfg := duplicateOfferWireConfig()
	stack := startActionWireStack(t, cfg, nil)
	server, peer := cfg.Segments[0].Devices[0], cfg.Segments[0].Devices[2]
	iface := stack.ExportDeviceStates()[server.Name].Running.Network.Interfaces[0].Name
	fault := devicestate.InterfaceAddressFault{
		Interface: iface,
		Type:      devicestate.FaultDuplicateIP,
		Address:   netip.MustParseAddr("10.254.200.101"),
	}
	probe.assertReplies(t, conflictARPWant{vlan: 200, macs: []string{peer.MACAddress.String()}})
	if err = stack.SetInterfaceAddressFault(server.Name, fault); err != nil {
		t.Fatal(err)
	}
	probe.assertReplies(
		t,
		conflictARPWant{vlan: 200, macs: []string{peer.MACAddress.String(), server.MACAddress.String()}},
	)
	probe.assertReplies(t, conflictARPWant{vlan: 300, macs: []string{cfg.Segments[1].Devices[2].MACAddress.String()}})
	if err = stack.ClearInterfaceAddressFault(server.Name, iface, devicestate.FaultDuplicateIP); err != nil {
		t.Fatal(err)
	}
	probe.assertReplies(t, conflictARPWant{vlan: 200, macs: []string{peer.MACAddress.String()}})
	t.Log("actual ARP wire: healthy owner, two exact conflict MACs, other VLAN unchanged, clear restores sole owner")
}

type conflictARPProbe struct {
	handle  *pcap.Handle
	packets <-chan gopacket.Packet
	client  net.HardwareAddr
}

type conflictARPWant struct {
	vlan int
	macs []string
}

func (p conflictARPProbe) assertReplies(t *testing.T, want conflictARPWant) {
	t.Helper()
	frame := serialize(
		t,
		&layers.Ethernet{
			SrcMAC:       p.client,
			DstMAC:       net.HardwareAddr{255, 255, 255, 255, 255, 255},
			EthernetType: layers.EthernetTypeDot1Q,
		},
		&layers.Dot1Q{VLANIdentifier: uint16(want.vlan), Type: layers.EthernetTypeARP},
		&layers.ARP{
			AddrType:          layers.LinkTypeEthernet,
			Protocol:          layers.EthernetTypeIPv4,
			HwAddressSize:     6,
			ProtAddressSize:   4,
			Operation:         layers.ARPRequest,
			SourceHwAddress:   p.client,
			SourceProtAddress: net.IPv4(10, 254, 200, 99).To4(),
			DstHwAddress:      make([]byte, 6),
			DstProtAddress:    net.IPv4(10, 254, 200, 101).To4(),
		},
	)
	if err := p.handle.WritePacketData(frame); err != nil {
		t.Fatal(err)
	}
	deadline := time.NewTimer(time.Second)
	defer deadline.Stop()
	seen := make(map[string]bool)
	for {
		select {
		case packet, ok := <-p.packets:
			if !ok {
				t.Fatal("capture ended before observation window")
			}
			p.checkReply(t, packet, want, seen)
		case <-deadline.C:
			if len(seen) != len(want.macs) {
				t.Fatalf("VLAN %d reply identities %v, want %v", want.vlan, seen, want.macs)
			}
			return
		}
	}
}

func (p conflictARPProbe) checkReply(t *testing.T, packet gopacket.Packet, want conflictARPWant, seen map[string]bool) {
	t.Helper()
	arp, ok := packet.Layer(layers.LayerTypeARP).(*layers.ARP)
	if !ok || arp.Operation != layers.ARPReply {
		return
	}
	vlan, ok := packet.Layer(layers.LayerTypeDot1Q).(*layers.Dot1Q)
	if !ok || int(vlan.VLANIdentifier) != want.vlan {
		t.Fatalf("ARP reply escaped requested VLAN %d: %v", want.vlan, packet)
	}
	mac := net.HardwareAddr(arp.SourceHwAddress).String()
	ethernet, ok := packet.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
	if !ok || ethernet.SrcMAC.String() != mac || ethernet.DstMAC.String() != p.client.String() ||
		!net.IP(arp.DstProtAddress).Equal(net.IPv4(10, 254, 200, 99)) || seen[mac] ||
		!slices.Contains(want.macs, mac) || !net.IP(arp.SourceProtAddress).Equal(net.IPv4(10, 254, 200, 101)) ||
		net.HardwareAddr(arp.DstHwAddress).String() != p.client.String() {
		t.Fatalf("unexpected conflict ARP identity: %v", packet)
	}
	seen[mac] = true
}
