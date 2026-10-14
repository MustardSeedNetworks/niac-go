//go:build linux && integration

package wiretest_test

import (
	"bytes"
	"net"
	"net/netip"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcap"

	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

// D-NIAC-42: packs are dual-stack. A tester on the pool solicits a router and
// hears the core SVI advertise the tester VLAN's /64, which the client kernel
// autoconfigures from; an off-link IPv6 address, the firewall's side of the
// core transit, answers an echo through the core.
func TestPackAdvertisesTheAttachmentPrefixAndRoutesIPv6(t *testing.T) {
	_, attachment := startPack(t, "hospital")
	t.Cleanup(func() {
		_ = exec.Command("ip", "-6", "route", "flush", "dev", testIface, "proto", "ra").Run()
		_ = exec.Command("ip", "-6", "addr", "flush", "dev", testIface, "scope", "global").Run()
	})
	core := attachment.gatewayDevice
	prefix := attachmentPrefixV6(t, attachment)
	firewall := defaultNextHopV6(t, attachment.topology, core.Name)

	handle := openClient(t)
	src := clientMAC(t)
	clientLinkLocal := eui64LinkLocal(src)
	advertised := awaitSolicitedPrefix(t, handle, src, clientLinkLocal, core.MACAddress)
	if advertised != prefix {
		t.Fatalf("%s advertised %s on %s, want the VLAN's %s", core.Name, advertised, attachment.network, prefix)
	}

	address := awaitAutoconfiguredAddress(t, prefix)

	reply := awaitEchoReplyV6(t, handle, src, core.MACAddress, address, firewall)
	if !bytes.Equal(reply.SrcMAC, core.MACAddress) {
		t.Errorf("echo reply from %s came from MAC %s, want the routing core %s (%s)",
			firewall, reply.SrcMAC, core.Name, core.MACAddress)
	}
	t.Logf("%s advertised %s; kernel autoconfigured %s; %s answered through %s",
		core.Name, advertised, address, firewall, core.Name)
}

func attachmentPrefixV6(t *testing.T, attachment packAttachment) netip.Prefix {
	t.Helper()
	for _, network := range attachment.topology.Networks {
		if network.Name == attachment.network && network.PrefixV6.IsValid() {
			return network.PrefixV6
		}
	}
	t.Fatalf("attachment network %q has no IPv6 prefix", attachment.network)
	return netip.Prefix{}
}

func defaultNextHopV6(t *testing.T, topology fabric.Topology, device string) netip.Addr {
	t.Helper()
	for _, route := range topology.Routes {
		if route.Device == device && route.Destination == netip.MustParsePrefix("::/0") {
			return route.NextHop
		}
	}
	t.Fatalf("%s has no IPv6 default route", device)
	return netip.Addr{}
}

func eui64LinkLocal(mac net.HardwareAddr) net.IP {
	return net.IP{
		0xfe, 0x80, 0, 0, 0, 0, 0, 0,
		mac[0] ^ 0x02, mac[1], mac[2], 0xff, 0xfe, mac[3], mac[4], mac[5],
	}
}

// awaitSolicitedPrefix solicits until router answers and returns the prefix
// its advertisement carries.
func awaitSolicitedPrefix(
	t *testing.T, handle *pcap.Handle, src net.HardwareAddr, srcIP net.IP, router net.HardwareAddr,
) netip.Prefix {
	t.Helper()
	ip6 := &layers.IPv6{
		Version: 6, NextHeader: layers.IPProtocolICMPv6, HopLimit: 255,
		SrcIP: srcIP, DstIP: net.ParseIP("ff02::2"),
	}
	icmp := &layers.ICMPv6{TypeCode: layers.CreateICMPv6TypeCode(layers.ICMPv6TypeRouterSolicitation, 0)}
	if err := icmp.SetNetworkLayerForChecksum(ip6); err != nil {
		t.Fatalf("checksum layer: %v", err)
	}
	solicitation := serialize(t,
		&layers.Ethernet{
			SrcMAC: src, DstMAC: net.HardwareAddr{0x33, 0x33, 0, 0, 0, 0x02}, EthernetType: layers.EthernetTypeIPv6,
		},
		ip6, icmp, &layers.ICMPv6RouterSolicitation{},
	)

	packets := gopacket.NewPacketSource(handle, handle.LinkType()).Packets()
	deadline := time.After(20 * time.Second)
	retry := time.NewTicker(time.Second)
	defer retry.Stop()
	write := func() {
		if err := handle.WritePacketData(solicitation); err != nil {
			t.Fatalf("writing router solicitation: %v", err)
		}
	}
	write()
	for {
		select {
		case packet := <-packets:
			ethernet, _ := packet.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
			ra, _ := packet.Layer(layers.LayerTypeICMPv6RouterAdvertisement).(*layers.ICMPv6RouterAdvertisement)
			if ethernet == nil || ra == nil || !bytes.Equal(ethernet.SrcMAC, router) {
				continue
			}
			for _, option := range ra.Options {
				const prefixInfoLen = 30
				if option.Type != layers.ICMPv6OptPrefixInfo || len(option.Data) != prefixInfoLen {
					continue
				}
				address, _ := netip.AddrFromSlice(option.Data[14:30])
				return netip.PrefixFrom(address, int(option.Data[0]))
			}
			t.Fatalf("advertisement from %s carries no prefix", router)
		case <-retry.C:
			write()
		case <-deadline:
			t.Fatalf("no router advertisement from %s within 20s", router)
		}
	}
}

// awaitAutoconfiguredAddress waits for the client kernel to accept the
// advertisement and form an address in prefix, which it does only for a valid
// one (link-local source, hop limit 255, correct checksum, autonomous flag).
func awaitAutoconfiguredAddress(t *testing.T, prefix netip.Prefix) netip.Addr {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for {
		for field := range strings.FieldsSeq(ip(t, "-6", "-o", "addr", "show", "dev", testIface, "scope", "global")) {
			if address, err := netip.ParsePrefix(field); err == nil && prefix.Contains(address.Addr()) {
				return address.Addr()
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("the client kernel formed no address in %s:\n%s",
				prefix, ip(t, "-6", "addr", "show", "dev", testIface))
		}
		time.Sleep(250 * time.Millisecond)
	}
}

func awaitEchoReplyV6(
	t *testing.T, handle *pcap.Handle, src, router net.HardwareAddr, from, to netip.Addr,
) *layers.Ethernet {
	t.Helper()
	ip6 := &layers.IPv6{
		Version: 6, NextHeader: layers.IPProtocolICMPv6, HopLimit: 64,
		SrcIP: from.AsSlice(), DstIP: to.AsSlice(),
	}
	icmp := &layers.ICMPv6{TypeCode: layers.CreateICMPv6TypeCode(layers.ICMPv6TypeEchoRequest, 0)}
	if err := icmp.SetNetworkLayerForChecksum(ip6); err != nil {
		t.Fatalf("checksum layer: %v", err)
	}
	request := serialize(t,
		&layers.Ethernet{SrcMAC: src, DstMAC: router, EthernetType: layers.EthernetTypeIPv6},
		ip6, icmp, &layers.ICMPv6Echo{Identifier: 0x2411, SeqNumber: 1},
	)

	packets := gopacket.NewPacketSource(handle, handle.LinkType()).Packets()
	deadline := time.After(10 * time.Second)
	retry := time.NewTicker(500 * time.Millisecond)
	defer retry.Stop()
	write := func() {
		if err := handle.WritePacketData(request); err != nil {
			t.Fatalf("writing echo request: %v", err)
		}
	}
	write()
	for {
		select {
		case packet := <-packets:
			ethernet, _ := packet.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
			reply, _ := packet.Layer(layers.LayerTypeIPv6).(*layers.IPv6)
			icmpReply, _ := packet.Layer(layers.LayerTypeICMPv6).(*layers.ICMPv6)
			if ethernet == nil || reply == nil || icmpReply == nil ||
				icmpReply.TypeCode.Type() != layers.ICMPv6TypeEchoReply || !reply.SrcIP.Equal(to.AsSlice()) {
				continue
			}
			return ethernet
		case <-retry.C:
			write()
		case <-deadline:
			t.Fatalf("no echo reply from %s through %s within 10s", to, router)
		}
	}
}
