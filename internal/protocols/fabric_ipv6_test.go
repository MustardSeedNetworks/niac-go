package protocols

import (
	"bytes"
	"net"
	"testing"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

// IPv6 on the fabric answers like IPv4 does (niac-go#2411 point 2): Neighbor
// Discovery only for attachment-network addresses, and an off-link
// destination only through the attachment router, whose MAC the reply then
// carries. The access switch also lists its management address in ips, which
// is what the flat per-VLAN table answered on any link before.

const (
	v6CoreData   = "fd00:51:d2::2"
	v6CoreMgmt   = "fd00:51:c8::2"
	v6AccessMgmt = "fd00:51:c8::21"
	v6Client     = "fd00:51:d2::100"
)

func dualStackPlacementStack(t *testing.T) *Stack {
	t.Helper()
	cfg := placementConfig()
	cfg.Networks[0].SubnetV6 = "fd00:51:c8::/64"
	cfg.Networks[1].SubnetV6 = "fd00:51:d2::/64"
	for index := range cfg.Devices {
		device := &cfg.Devices[index]
		switch device.Name {
		case placementCore:
			setAddressV6(device, "Vlan200", v6CoreMgmt+"/64")
			setAddressV6(device, "Vlan210", v6CoreData+"/64")
		case placementAccess:
			setAddressV6(device, "Vlan200", v6AccessMgmt+"/64")
			device.IPAddresses = append(device.IPAddresses, net.ParseIP(v6AccessMgmt))
		}
	}
	stack := NewStackWithTransport(idleTransport{}, cfg, logging.NewDebugConfig(0))
	stack.ConfigureFabric(compilePlacement(t, cfg))
	return stack
}

func setAddressV6(device *config.Device, name, address string) {
	for index := range device.Interfaces {
		if device.Interfaces[index].Name == name {
			device.Interfaces[index].AddressV6 = address
		}
	}
}

func TestFabricNeighborSolicitationAnswersOnlyOnTheAttachmentLink(t *testing.T) {
	cases := []struct {
		name   string
		target string
		want   string
	}{
		{"attachment SVI", v6CoreData, placementCore},
		{"attachment router link-local", "fe80::251:ff:fe00:2", placementCore},
		{"core address on another network", v6CoreMgmt, ""},
		{"off-link switch in ips", v6AccessMgmt, ""},
		{"off-link switch link-local", "fe80::251:ff:fe00:21", ""},
		{"unknown address", "fd00:51:d2::99", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stack := dualStackPlacementStack(t)

			stack.decodePacket(neighborSolicitationFrame(t, placementClient(1), v6Client, tc.target))

			got := queuedICMPv6(t, stack, layers.LayerTypeICMPv6NeighborAdvertisement)
			if tc.want == "" {
				if len(got) != 0 {
					t.Fatalf("NS for %s answered %d times, want silence", tc.target, len(got))
				}
				return
			}
			want := stackDevice(t, stack, tc.want).MACAddress
			if len(got) != 1 || !bytes.Equal(got[0].src, want) {
				t.Fatalf("NS for %s answered from %v, want one NA from %s", tc.target, got, want)
			}
		})
	}
}

func TestFabricIPv6EchoResolvesThroughTheAttachmentRouter(t *testing.T) {
	cases := []struct {
		name          string
		source        string
		target        string
		viaMAC        string
		wantReplyFrom string
		wantReason    string
		wantForward   uint64
	}{
		{"on-link SVI", v6Client, v6CoreData, placementCore, placementCore, "", 0},
		{"off-link via attachment router", v6Client, v6AccessMgmt, placementCore, placementCore, "", 1},
		{"router's other network", v6Client, v6CoreMgmt, placementCore, placementCore, "", 1},
		{
			"off-link addressed to the switch itself",
			v6Client,
			v6AccessMgmt,
			placementAccess,
			"",
			fabricRejectionNoRoute,
			0,
		},
		{"router link-local", "fe80::ff:fe00:201", "fe80::251:ff:fe00:2", placementCore, placementCore, "", 0},
		{
			"source outside the attachment prefix",
			"fd00:99::1",
			v6CoreData,
			placementCore,
			"",
			fabricRejectionAttachmentSource,
			0,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stack := dualStackPlacementStack(t)
			pkt := icmpv6EchoFrame(
				t,
				placementClient(1),
				stackDevice(t, stack, tc.viaMAC).MACAddress,
				tc.source,
				tc.target,
			)

			stack.decodePacket(pkt)

			got := queuedICMPv6(t, stack, layers.LayerTypeICMPv6Echo)
			if tc.wantReplyFrom == "" {
				assertIPv6EchoDropped(t, stack, pkt, got, tc.wantReason)
				return
			}
			want := stackDevice(t, stack, tc.wantReplyFrom).MACAddress
			if len(got) != 1 || !bytes.Equal(got[0].src, want) || !got[0].srcIP.Equal(net.ParseIP(tc.target)) {
				t.Fatalf("echo to %s answered %v, want one reply from %s at %s", tc.target, got, tc.target, want)
			}
			if forwarded := stack.GetStats().FabricForwarded; forwarded != tc.wantForward {
				t.Fatalf("FabricForwarded = %d, want %d", forwarded, tc.wantForward)
			}
		})
	}
}

func assertIPv6EchoDropped(t *testing.T, stack *Stack, pkt *Packet, replies []sentICMPv6, reason string) {
	t.Helper()
	if len(replies) != 0 {
		t.Fatalf("echo answered %v, want drop", replies)
	}
	if trace := pkt.FabricTrace(); trace.RejectionReason != reason {
		t.Fatalf("FabricTrace() = %#v, want rejection %q", trace, reason)
	}
	if drops := stack.GetStats().FabricDrops; drops != 1 {
		t.Fatalf("FabricDrops = %d, want 1", drops)
	}
}

func TestFabricIPv6StopsAnsweringWhenEitherEndOfTheRouteIsShut(t *testing.T) {
	for _, shut := range []string{placementAccess, placementCore} {
		t.Run(shut, func(t *testing.T) {
			stack := dualStackPlacementStack(t)
			state := stack.deviceStates[stackDevice(t, stack, shut)]
			err := state.UpdateInterface("Vlan200", func(iface devicestate.Interface) (devicestate.Interface, error) {
				iface.AdminUp, iface.OperUp = false, false
				return iface, nil
			})
			if err != nil {
				t.Fatalf("shut %s Vlan200: %v", shut, err)
			}

			stack.decodePacket(icmpv6EchoFrame(
				t, placementClient(1), stackDevice(t, stack, placementCore).MACAddress, v6Client, v6AccessMgmt,
			))

			if got := queuedICMPv6(t, stack, layers.LayerTypeICMPv6Echo); len(got) != 0 {
				t.Fatalf("echo across a shut %s Vlan200 answered %v", shut, got)
			}
		})
	}
}

type sentICMPv6 struct {
	src   net.HardwareAddr
	srcIP net.IP
}

func queuedICMPv6(t *testing.T, stack *Stack, layer gopacket.LayerType) []sentICMPv6 {
	t.Helper()
	var got []sentICMPv6
	for len(stack.sendQueue) > 0 {
		pkt := <-stack.sendQueue
		frame := gopacket.NewPacket(pkt.Buffer[:pkt.Length], layers.LayerTypeEthernet, gopacket.Default)
		ip, _ := frame.Layer(layers.LayerTypeIPv6).(*layers.IPv6)
		if ip == nil || frame.Layer(layer) == nil {
			continue
		}
		got = append(got, sentICMPv6{src: net.HardwareAddr(pkt.Buffer[6:12]), srcIP: ip.SrcIP})
	}
	return got
}

func neighborSolicitationFrame(t *testing.T, src net.HardwareAddr, srcIP, target string) *Packet {
	t.Helper()
	targetIP := net.ParseIP(target)
	group := net.ParseIP("ff02::1:ff00:0")
	copy(group[13:], targetIP[13:])
	return icmpv6Frame(t, src, IPv6MulticastToMAC(group), srcIP, group.String(),
		layers.ICMPv6TypeNeighborSolicitation, &layers.ICMPv6NeighborSolicitation{
			TargetAddress: targetIP,
			Options: layers.ICMPv6Options{{
				Type: layers.ICMPv6OptSourceAddress, Data: src,
			}},
		})
}

func icmpv6EchoFrame(t *testing.T, src, dst net.HardwareAddr, srcIP, dstIP string) *Packet {
	t.Helper()
	return icmpv6Frame(t, src, dst, srcIP, dstIP, layers.ICMPv6TypeEchoRequest,
		&layers.ICMPv6Echo{Identifier: 1, SeqNumber: 1})
}

func icmpv6Frame(
	t *testing.T,
	src, dst net.HardwareAddr,
	srcIP, dstIP string,
	icmpType uint8,
	body gopacket.SerializableLayer,
) *Packet {
	t.Helper()
	eth := &layers.Ethernet{SrcMAC: src, DstMAC: dst, EthernetType: layers.EthernetTypeIPv6}
	ip := &layers.IPv6{
		Version: 6, NextHeader: layers.IPProtocolICMPv6, HopLimit: icmpv6NDPHopLimit,
		SrcIP: net.ParseIP(srcIP), DstIP: net.ParseIP(dstIP),
	}
	icmp := &layers.ICMPv6{TypeCode: layers.CreateICMPv6TypeCode(icmpType, 0)}
	if err := icmp.SetNetworkLayerForChecksum(ip); err != nil {
		t.Fatalf("checksum layer: %v", err)
	}
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
	if err := gopacket.SerializeLayers(buf, opts, eth, ip, icmp, body); err != nil {
		t.Fatalf("serialize ICMPv6 type %d: %v", icmpType, err)
	}
	pkt, err := ParsePacket(buf.Bytes(), 1)
	if err != nil {
		t.Fatalf("ParsePacket: %v", err)
	}
	return pkt
}
