//go:build linux && integration

package wiretest_test

import (
	"net"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcap"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

func dhcpWireExchange(
	t *testing.T, handle *pcap.Handle, packets <-chan gopacket.Packet,
	client net.HardwareAddr, vlan int, xid uint32, selected *config.Device,
) []gopacket.Packet {
	t.Helper()
	if err := handle.WritePacketData(dhcpWireRequest(t, client, vlan, xid, selected)); err != nil {
		t.Fatal(err)
	}
	// Keep observing after the expected replies to detect competing ACK/NAK
	// and cross-segment leakage. Never retransmit a missed request.
	deadline := time.NewTimer(time.Second)
	defer deadline.Stop()
	var replies []gopacket.Packet
	for {
		select {
		case packet, open := <-packets:
			if !open {
				t.Fatal("DHCP capture closed")
			}
			layer := packet.Layer(layers.LayerTypeDHCPv4)
			if layer == nil {
				continue
			}
			dhcp := layer.(*layers.DHCPv4)
			if dhcp.Xid != xid || dhcp.Operation != layers.DHCPOpReply {
				continue
			}
			tag, ok := packet.Layer(layers.LayerTypeDot1Q).(*layers.Dot1Q)
			if !ok || int(tag.VLANIdentifier) != vlan {
				t.Fatalf("DHCP response escaped VLAN %d: %v", vlan, packet)
			}
			replies = append(replies, packet)
		case <-deadline.C:
			return replies
		}
	}
}

func dhcpWireRequest(
	t *testing.T, client net.HardwareAddr, vlan int, xid uint32, selected *config.Device,
) []byte {
	t.Helper()
	kind := layers.DHCPMsgTypeDiscover
	var options []layers.DHCPOption
	if selected != nil {
		kind = layers.DHCPMsgTypeRequest
		options = append(options,
			layers.DHCPOption{Type: layers.DHCPOptServerID, Data: selected.IPAddresses[0].To4(), Length: 4},
			layers.DHCPOption{Type: layers.DHCPOptRequestIP, Data: selected.DHCPConfig.PoolStart.To4(), Length: 4},
		)
	}
	options = append(options, layers.DHCPOption{Type: layers.DHCPOptMessageType, Data: []byte{byte(kind)}, Length: 1})
	dhcp := &layers.DHCPv4{
		Operation: layers.DHCPOpRequest, HardwareType: layers.LinkTypeEthernet, HardwareLen: 6,
		Xid: xid, ClientHWAddr: client, Flags: 0x8000, Options: options,
	}
	ip := &layers.IPv4{Version: 4, TTL: 64, Protocol: layers.IPProtocolUDP, SrcIP: net.IPv4zero, DstIP: net.IPv4bcast}
	udp := &layers.UDP{SrcPort: 68, DstPort: 67}
	if err := udp.SetNetworkLayerForChecksum(ip); err != nil {
		t.Fatal(err)
	}
	return serialize(
		t,
		&layers.Ethernet{
			SrcMAC:       client,
			DstMAC:       net.HardwareAddr{255, 255, 255, 255, 255, 255},
			EthernetType: layers.EthernetTypeDot1Q,
		},
		&layers.Dot1Q{VLANIdentifier: uint16(vlan), Type: layers.EthernetTypeIPv4},
		ip,
		udp,
		dhcp,
	)
}
