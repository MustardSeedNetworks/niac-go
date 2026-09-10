//go:build linux && integration

package wiretest_test

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcap"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestDHCPServersIsolatedOnWire(t *testing.T) {
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
	cfg := &config.Config{}
	for index, vlan := range []int{200, 300} {
		cfg.Segments = append(cfg.Segments, config.Segment{Tag: vlan, Devices: []config.Device{
			dhcpWireServer(vlan, "a", byte(index*2+2), 2, 100),
			dhcpWireServer(vlan, "b", byte(index*2+3), 3, 110),
		}})
	}
	stack := startActionWireStack(t, cfg, nil)
	xid := uint32(0xa0040000)
	for _, segment := range cfg.Segments {
		xid++
		assertDHCPWireReplies(t, dhcpWireExchange(t, handle, packets, client, segment.Tag, xid, nil),
			segment.Devices, layers.DHCPMsgTypeOffer, client)
		for _, server := range segment.Devices {
			xid++
			assertDHCPWireReplies(t, dhcpWireExchange(t, handle, packets, client, segment.Tag, xid, &server),
				[]config.Device{server}, layers.DHCPMsgTypeAck, client)
		}
	}
	target := cfg.Segments[0].Devices[0]
	if err = stack.SetDeviceFault(target.Name, devicestate.FaultDHCPNoOffer, 100); err != nil {
		t.Fatal(err)
	}
	xid++
	assertDHCPWireReplies(t, dhcpWireExchange(t, handle, packets, client, 200, xid, nil),
		cfg.Segments[0].Devices[1:], layers.DHCPMsgTypeOffer, client)
	xid++
	assertDHCPWireReplies(t, dhcpWireExchange(t, handle, packets, client, 300, xid, nil),
		cfg.Segments[1].Devices, layers.DHCPMsgTypeOffer, client)
	if err = stack.SetDeviceFault(target.Name, devicestate.FaultDHCPNoOffer, 0); err != nil {
		t.Fatal(err)
	}
	xid++
	assertDHCPWireReplies(t, dhcpWireExchange(t, handle, packets, client, 200, xid, nil),
		cfg.Segments[0].Devices, layers.DHCPMsgTypeOffer, client)
	t.Log(
		"actual tagged DHCP: isolated overlapping pools and client identity, exact options, selected ACK only, independent fault/clear",
	)
}

func dhcpWireServer(vlan int, label string, macSuffix, ipSuffix, poolSuffix byte) config.Device {
	return config.Device{
		Name:        fmt.Sprintf("dhcp-%d-%s", vlan, label),
		MACAddress:  net.HardwareAddr{2, 0, 0, 0, 0xda, macSuffix},
		IPAddresses: []net.IP{net.IPv4(10, 254, 200, ipSuffix)},
		DHCPConfig: &config.DHCPConfig{
			PoolStart: net.IPv4(10, 254, 200, poolSuffix), PoolEnd: net.IPv4(10, 254, 200, poolSuffix),
			SubnetMask: net.CIDRMask(23, 32), Router: net.IPv4(10, 254, 200, 1),
			DomainNameServer: []net.IP{net.IPv4(10, 254, 200, 53)},
			DomainName:       fmt.Sprintf("%s.vlan%d.example", label, vlan),
		},
	}
}

func assertDHCPWireReplies(
	t *testing.T, packets []gopacket.Packet, servers []config.Device,
	kind layers.DHCPMsgType, client net.HardwareAddr,
) {
	t.Helper()
	if len(packets) != len(servers) {
		t.Fatalf("received %d DHCP replies, want exactly %d", len(packets), len(servers))
	}
	seen := make(map[string]bool)
	for _, packet := range packets {
		eth := packet.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
		ip := packet.Layer(layers.LayerTypeIPv4).(*layers.IPv4)
		dhcp := packet.Layer(layers.LayerTypeDHCPv4).(*layers.DHCPv4)
		var expected *config.Device
		for index := range servers {
			if servers[index].MACAddress.String() == eth.SrcMAC.String() {
				expected = &servers[index]
			}
		}
		if expected == nil || seen[eth.SrcMAC.String()] {
			t.Fatalf("unexpected or duplicate DHCP source: %s", eth.SrcMAC)
		}
		seen[eth.SrcMAC.String()] = true
		if dhcpMessageType(dhcp) != kind || !ip.SrcIP.Equal(expected.IPAddresses[0]) ||
			!dhcpOptionIP(dhcp, layers.DHCPOptServerID).Equal(expected.IPAddresses[0]) ||
			!dhcp.YourClientIP.Equal(expected.DHCPConfig.PoolStart) || dhcp.ClientHWAddr.String() != client.String() {
			t.Fatalf("wrong DHCP identity/type/address: source=%s packet=%+v", ip.SrcIP, dhcp)
		}
		assertDHCPWireOptions(t, dhcp, expected)
	}
}

func assertDHCPWireOptions(t *testing.T, dhcp *layers.DHCPv4, expected *config.Device) {
	t.Helper()
	for _, option := range []struct {
		kind layers.DHCPOpt
		want net.IP
	}{
		{layers.DHCPOptSubnetMask, net.IP(expected.DHCPConfig.SubnetMask)},
		{layers.DHCPOptRouter, expected.DHCPConfig.Router},
		{layers.DHCPOptDNS, expected.DHCPConfig.DomainNameServer[0]},
	} {
		if got := dhcpOptionIP(dhcp, option.kind); !got.Equal(option.want) {
			t.Fatalf("%s option %s=%s, want %s", expected.Name, option.kind, got, option.want)
		}
	}
	for _, option := range dhcp.Options {
		if option.Type == layers.DHCPOptDomainName && string(option.Data) == expected.DHCPConfig.DomainName {
			return
		}
	}
	t.Fatalf("missing authored domain %s", expected.DHCPConfig.DomainName)
}
