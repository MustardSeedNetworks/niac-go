package protocols

import (
	"bytes"
	"net"
	"testing"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// DNS and TCP over IPv6 answer the way ICMPv6 and SNMP already do
// (niac-go#2411 point 2): an off-link reply leaves from the attachment
// router's MAC, toward the MAC the request came from, on the VLAN it
// arrived on. They used to send from the target device's own MAC, and a TCP
// RST looked the sender up among the simulated devices, so a tester never
// got one.

const v6ReplyClientPort = 40000

type v6Request struct {
	name    string
	payload func(t *testing.T) (gopacket.SerializableLayer, []byte)
	reply   gopacket.LayerType
}

func v6Requests() []v6Request {
	tcp := func(port layers.TCPPort, syn bool, data string) func(*testing.T) (gopacket.SerializableLayer, []byte) {
		return func(*testing.T) (gopacket.SerializableLayer, []byte) {
			segment := &layers.TCP{
				SrcPort: v6ReplyClientPort, DstPort: port, Seq: 100, Window: 1024, SYN: syn, ACK: !syn, PSH: !syn,
			}
			return segment, []byte(data)
		}
	}
	return []v6Request{
		{"DNS", func(t *testing.T) (gopacket.SerializableLayer, []byte) {
			t.Helper()
			query := &layers.DNS{
				ID:      7,
				RD:      true,
				QDCount: 1,
				Questions: []layers.DNSQuestion{
					{Name: []byte("example.test"), Type: layers.DNSTypeAAAA, Class: layers.DNSClassIN},
				},
			}
			buf := gopacket.NewSerializeBuffer()
			if err := query.SerializeTo(buf, gopacket.SerializeOptions{FixLengths: true}); err != nil {
				t.Fatalf("serialize DNS query: %v", err)
			}
			return &layers.UDP{SrcPort: v6ReplyClientPort, DstPort: layers.UDPPort(UDPPortDNS)}, buf.Bytes()
		}, layers.LayerTypeDNS},
		{"HTTP", tcp(layers.TCPPort(TCPPortHTTP), false, "GET / HTTP/1.1\r\nHost: x\r\n\r\n"), layers.LayerTypeTCP},
		{"RST", tcp(9999, true, ""), layers.LayerTypeTCP},
	}
}

func v6RequestFrame(
	t *testing.T,
	request v6Request,
	src, dst net.HardwareAddr,
	vlan uint16,
	srcIP, dstIP string,
) *Packet {
	t.Helper()
	transport, data := request.payload(t)
	ip := &layers.IPv6{Version: 6, HopLimit: 64, SrcIP: net.ParseIP(srcIP), DstIP: net.ParseIP(dstIP)}
	switch layer := transport.(type) {
	case *layers.UDP:
		ip.NextHeader = layers.IPProtocolUDP
		_ = layer.SetNetworkLayerForChecksum(ip)
	case *layers.TCP:
		ip.NextHeader = layers.IPProtocolTCP
		_ = layer.SetNetworkLayerForChecksum(ip)
	}
	eth := &layers.Ethernet{SrcMAC: src, DstMAC: dst, EthernetType: layers.EthernetTypeIPv6}
	stack := []gopacket.SerializableLayer{eth}
	if vlan != 0 {
		eth.EthernetType = layers.EthernetTypeDot1Q
		stack = append(stack, &layers.Dot1Q{VLANIdentifier: vlan, Type: layers.EthernetTypeIPv6})
	}
	stack = append(stack, ip, transport, gopacket.Payload(data))
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
	if err := gopacket.SerializeLayers(buf, opts, stack...); err != nil {
		t.Fatalf("serialize %s request: %v", request.name, err)
	}
	pkt, err := ParsePacket(buf.Bytes(), 1)
	if err != nil {
		t.Fatalf("ParsePacket: %v", err)
	}
	return pkt
}

type sentIPv6Reply struct {
	src, dst net.HardwareAddr
	srcIP    net.IP
	vlan     int
}

func queuedIPv6Replies(stack *Stack, layer gopacket.LayerType) []sentIPv6Reply {
	var got []sentIPv6Reply
	for len(stack.sendQueue) > 0 {
		pkt := <-stack.sendQueue
		frame := gopacket.NewPacket(pkt.Buffer[:pkt.Length], layers.LayerTypeEthernet, gopacket.Default)
		ip, _ := frame.Layer(layers.LayerTypeIPv6).(*layers.IPv6)
		if ip == nil || frame.Layer(layer) == nil {
			continue
		}
		got = append(got, sentIPv6Reply{
			src: net.HardwareAddr(pkt.Buffer[6:12]), dst: net.HardwareAddr(pkt.Buffer[0:6]),
			srcIP: ip.SrcIP, vlan: pkt.VLAN,
		})
	}
	return got
}

// The core's SVI address exists only as an interface address_v6, not in ips,
// so it also proves the TCP lookup asks the fabric who owns the address.
func TestFabricIPv6ServiceRepliesLeaveFromTheAttachmentRouter(t *testing.T) {
	targets := []struct{ name, target, device string }{
		{"off-link switch", v6AccessMgmt, placementAccess},
		{"attachment SVI", v6CoreData, placementCore},
	}
	for _, target := range targets {
		for _, request := range v6Requests() {
			t.Run(target.name+"/"+request.name, func(t *testing.T) {
				stack := dualStackPlacementStack(t)
				stackDevice(t, stack, target.device).DNSConfig = &config.DNSConfig{}
				routerMAC := stackDevice(t, stack, placementCore).MACAddress
				client := placementClient(1)

				stack.decodePacket(v6RequestFrame(t, request, client, routerMAC, 0, v6Client, target.target))

				got := queuedIPv6Replies(stack, request.reply)
				if len(got) != 1 {
					t.Fatalf("%s to %s answered %d times, want once", request.name, target.target, len(got))
				}
				if !bytes.Equal(got[0].src, routerMAC) || !bytes.Equal(got[0].dst, client) ||
					!got[0].srcIP.Equal(net.ParseIP(target.target)) {
					t.Fatalf("%s reply = %+v, want from router %s to %s at %s",
						request.name, got[0], routerMAC, client, target.target)
				}
			})
		}
	}
}

// A device answers from the address it was asked on, on the VLAN it was asked
// on. DNS used to answer from the device's first IPv6 address, which a
// resolver that queried the link-local discards.
func TestIPv6ServiceRepliesEchoTheRequestAddressAndVLAN(t *testing.T) {
	targets := []struct{ name, client, target string }{
		{"global", "fd00:6a::240", "fd00:6a::21"},
		{"link-local", "fe80::6a:f0", "fe80::6a:21"},
	}
	for _, target := range targets {
		for _, request := range v6Requests() {
			t.Run(target.name+"/"+request.name, func(t *testing.T) {
				stack, device := ndpSegmentStack(t)
				device.DNSConfig = &config.DNSConfig{}
				client := net.HardwareAddr{0x02, 0x00, 0x00, 0x00, 0x6a, 0xf0}

				stack.decodePacket(v6RequestFrame(
					t, request, client, device.MACAddress, ndpTestVLAN, target.client, target.target,
				))

				got := queuedIPv6Replies(stack, request.reply)
				if len(got) != 1 {
					t.Fatalf("%s to %s on VLAN %d answered %d times, want once",
						request.name, target.target, ndpTestVLAN, len(got))
				}
				if got[0].vlan != ndpTestVLAN || !bytes.Equal(got[0].src, device.MACAddress) ||
					!bytes.Equal(got[0].dst, client) || !got[0].srcIP.Equal(net.ParseIP(target.target)) {
					t.Fatalf("%s reply = %+v, want from %s at %s to %s on VLAN %d",
						request.name, got[0], device.MACAddress, target.target, client, ndpTestVLAN)
				}
			})
		}
	}
}
