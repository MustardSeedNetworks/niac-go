//go:build linux && integration

package wiretest_test

import (
	"fmt"
	"net"
	"net/netip"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcap"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestDuplicateDHCPOfferOnWire(t *testing.T) {
	requireWire(t)
	handle, err := pcap.OpenLive(testIface, 65536, true, 20*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(handle.Close)
	if err = handle.SetBPFFilter("udp src port 67 or (vlan and udp src port 67)"); err != nil {
		t.Fatal(err)
	}
	packets := gopacket.NewPacketSource(handle, handle.LinkType()).Packets()
	client := clientMAC(t)
	cfg := duplicateOfferWireConfig()
	stack := startActionWireStack(t, cfg, nil)
	server := cfg.Segments[0].Devices[0]
	conflict := netip.MustParseAddr("10.254.200.101")
	if err = stack.SetDeviceAddressFault(server.Name, devicestate.FaultDuplicateDHCPOffer, conflict); err != nil {
		t.Fatal(err)
	}
	faulted := server
	faultedDHCP := *server.DHCPConfig
	faultedDHCP.PoolStart = net.IP(conflict.AsSlice())
	faulted.DHCPConfig = &faultedDHCP
	assertDHCPWireReplies(t, dhcpWireExchange(t, handle, packets, client, 200, 0xad001, nil),
		[]config.Device{faulted, cfg.Segments[0].Devices[1]}, layers.DHCPMsgTypeOffer, client)
	assertDHCPWireReplies(t, dhcpWireExchange(t, handle, packets, client, 300, 0xad002, nil),
		cfg.Segments[1].Devices[:2], layers.DHCPMsgTypeOffer, client)
	assertConflictNAK(t, dhcpWireExchange(t, handle, packets, client, 200, 0xad003, &faulted), server)
	if err = stack.ClearDeviceFault(server.Name, devicestate.FaultDuplicateDHCPOffer); err != nil {
		t.Fatal(err)
	}
	assertDHCPWireReplies(t, dhcpWireExchange(t, handle, packets, client, 200, 0xad004, nil),
		cfg.Segments[0].Devices[:2], layers.DHCPMsgTypeOffer, client)
	t.Log(
		"actual wire: peer-owned in-pool conflict OFFER, exact identity/options, other VLAN healthy, conflict NAK and unchanged normal lease after clear",
	)
}

func duplicateOfferWireConfig() *config.Config {
	cfg := &config.Config{}
	for index, vlan := range []int{200, 300} {
		first := dhcpWireServer(vlan, "a", byte(index*2+2), 2, 100)
		first.DHCPConfig.PoolEnd = net.IPv4(10, 254, 200, 101)
		cfg.Segments = append(cfg.Segments, config.Segment{Tag: vlan, Devices: []config.Device{
			first, dhcpWireServer(vlan, "b", byte(index*2+3), 3, 110),
			{
				Name: fmt.Sprintf("peer-%d", vlan), MACAddress: net.HardwareAddr{2, 0, 0, 0, 0xdb, byte(index + 1)},
				IPAddresses: []net.IP{net.IPv4(10, 254, 200, 101)},
			},
		}})
	}
	return cfg
}

func assertConflictNAK(t *testing.T, replies []gopacket.Packet, server config.Device) {
	t.Helper()
	if len(replies) != 1 {
		t.Fatalf("conflict REQUEST received %d replies, want one NAK", len(replies))
	}
	packet := replies[0]
	eth := packet.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
	ip := packet.Layer(layers.LayerTypeIPv4).(*layers.IPv4)
	dhcp := packet.Layer(layers.LayerTypeDHCPv4).(*layers.DHCPv4)
	if dhcpMessageType(dhcp) != layers.DHCPMsgTypeNak || !dhcp.YourClientIP.IsUnspecified() ||
		eth.SrcMAC.String() != server.MACAddress.String() || !ip.SrcIP.Equal(server.IPAddresses[0]) ||
		!dhcpOptionIP(dhcp, layers.DHCPOptServerID).Equal(server.IPAddresses[0]) {
		t.Fatalf("wrong conflict NAK identity or payload: %v", packet)
	}
}
