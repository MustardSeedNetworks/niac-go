package protocols

import (
	"bytes"
	"net"
	"testing"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

func enipTestServer() *config.EtherNetIPConfig {
	return &config.EtherNetIPConfig{
		Enabled: true, VendorID: 1, DeviceType: 14, ProductCode: 166,
		RevisionMajor: 33, RevisionMinor: 11, SerialNumber: 0xc0ffee01, ProductName: "1756-L83E/B",
	}
}

// enipRequest is an encapsulation header for command carrying data, with
// session handle 0x01020304 and sender context "context!".
func enipRequest(command uint16, data ...byte) []byte {
	header := []byte{
		byte(command), byte(command >> 8), byte(len(data)), byte(len(data) >> 8),
		0x04, 0x03, 0x02, 0x01, 0, 0, 0, 0,
		'c', 'o', 'n', 't', 'e', 'x', 't', '!', 0, 0, 0, 0,
	}

	return append(header, data...)
}

// enipTestIdentity is the List Identity reply enipTestServer sends from
// 10.20.30.44, written out field by field.
func enipTestIdentity() []byte {
	return []byte{
		0x63, 0x00, 0x33, 0x00, 0x04, 0x03, 0x02, 0x01, 0, 0, 0, 0,
		'c', 'o', 'n', 't', 'e', 'x', 't', '!', 0, 0, 0, 0,
		0x01, 0x00, // item count
		0x0c, 0x00, 0x2d, 0x00, // CIP Identity item, 45 bytes
		0x01, 0x00, // encapsulation version
		0x00, 0x02, 0xaf, 0x12, 10, 20, 30, 44, 0, 0, 0, 0, 0, 0, 0, 0, // sockaddr_in
		0x01, 0x00, // vendor ID
		0x0e, 0x00, // device type
		0xa6, 0x00, // product code
		33, 11, // revision
		0x00, 0x00, // status
		0x01, 0xee, 0xff, 0xc0, // serial number
		11, '1', '7', '5', '6', '-', 'L', '8', '3', 'E', '/', 'B',
		0x03, // state
	}
}

// enipUnsupported is the TCP reply to command: the header with status 0x0001.
func enipUnsupported(command uint16) []byte {
	reply := enipRequest(command)
	reply[8] = 0x01

	return reply
}

func TestENIPAnswer(t *testing.T) {
	t.Parallel()

	address := net.IPv4(10, 20, 30, 44)
	tests := []struct {
		name     string
		stream   []byte
		overTCP  bool
		want     []byte
		consumed int
		ended    bool
	}{
		{name: "list identity", stream: enipRequest(0x63), overTCP: true, want: enipTestIdentity(), consumed: 24},
		{name: "list identity over UDP", stream: enipRequest(0x63), want: enipTestIdentity(), consumed: 24},
		{
			name:     "register session is unsupported",
			stream:   enipRequest(0x65, 1, 0, 0, 0),
			overTCP:  true,
			want:     enipUnsupported(0x65),
			consumed: 28,
		},
		{name: "unsupported over UDP is dropped", stream: enipRequest(0x04), consumed: 24},
		{name: "NOP is never answered", stream: enipRequest(0x00, 1, 2, 3), overTCP: true, consumed: 27},
		{
			name:     "two messages in one segment",
			stream:   append(enipRequest(0x00), enipRequest(0x63)...),
			overTCP:  true,
			want:     enipTestIdentity(),
			consumed: 48,
		},
		{name: "partial header waits", stream: enipRequest(0x63)[:20], overTCP: true},
		{name: "partial data waits", stream: enipRequest(0x65, 1, 0, 0, 0)[:26], overTCP: true},
		{
			name:    "length past the largest message ends",
			stream:  append([]byte{0x63, 0x00, 0xe8, 0xff}, make([]byte, 20)...),
			overTCP: true,
			ended:   true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			answer, consumed, ended := enipAnswer(test.stream, enipTestServer(), address, test.overTCP)
			if !bytes.Equal(answer, test.want) || consumed != test.consumed || ended != test.ended {
				t.Errorf("enipAnswer(% x) = % x, %d, %v; want % x, %d, %v",
					test.stream, answer, consumed, ended, test.want, test.consumed, test.ended)
			}
		})
	}
}

func enipTestNode(enip *config.EtherNetIPConfig) *config.Device {
	return &config.Device{
		Name:             "PLT-PLC01",
		MACAddress:       net.HardwareAddr{0x00, 0x00, 0xbc, 0x00, 0x00, 0x06},
		IPAddresses:      []net.IP{net.ParseIP("10.20.30.44")},
		EtherNetIPConfig: enip,
	}
}

// A List Identity split across segments is answered once whole, and the
// connection closes with the client's FIN.
func TestTCPHandlerServesEtherNetIP(t *testing.T) {
	conversation := newServiceConversation(t, enipTestNode(enipTestServer()), PortEtherNetIP)
	if reply := conversation.reply(&layers.TCP{SYN: true, Seq: 100}); reply == nil ||
		!reply.SYN || !reply.ACK || reply.Ack != 101 {
		t.Fatalf("SYN answered with %+v, want a SYN-ACK acknowledging 101", reply)
	}
	request := enipRequest(0x63)
	if reply := conversation.reply(
		&layers.TCP{ACK: true, PSH: true, Seq: 101, Ack: 1001, Payload: request[:10]},
	); reply == nil ||
		len(reply.Payload) != 0 ||
		reply.Ack != 111 {
		t.Fatalf("partial header answered with %+v, want a bare ACK of 111", reply)
	}
	reply := conversation.reply(
		&layers.TCP{ACK: true, PSH: true, FIN: true, Seq: 111, Ack: 1001, Payload: request[10:]},
	)
	if reply == nil || !reply.FIN || reply.Seq != 1001 || reply.Ack != 126 ||
		!bytes.Equal(reply.Payload, enipTestIdentity()) {
		t.Errorf("List Identity with FIN answered with %+v, want % x and FIN", reply, enipTestIdentity())
	}
}

// A device without EtherNet/IP keeps the closed-port RST, as does one whose
// EtherNet/IP is switched off.
func TestTCPHandlerEtherNetIPClosedWithoutIt(t *testing.T) {
	for _, enip := range []*config.EtherNetIPConfig{nil, {ProductName: "1756-L83E/B"}} {
		device := enipTestNode(enip)
		if reply := serviceReply(t, device, PortEtherNetIP, &layers.TCP{SYN: true, Seq: 500}); reply == nil ||
			!reply.RST {
			t.Errorf("EtherNet/IP %+v: SYN answered with %+v, want RST", enip, reply)
		}
	}
}

func buildENIPDatagram(t *testing.T, device *config.Device, payload []byte) (*Packet, *layers.IPv4) {
	t.Helper()
	eth := &layers.Ethernet{
		SrcMAC: net.HardwareAddr{0xaa, 0, 0, 0, 0, 1}, DstMAC: device.MACAddress,
		EthernetType: layers.EthernetTypeIPv4,
	}
	ip := &layers.IPv4{
		Version: 4, IHL: 5, TTL: 64, Protocol: layers.IPProtocolUDP,
		SrcIP: net.ParseIP("192.0.2.9").To4(), DstIP: device.IPAddresses[0].To4(),
	}
	udp := &layers.UDP{SrcPort: 50000, DstPort: PortEtherNetIP}
	if err := udp.SetNetworkLayerForChecksum(ip); err != nil {
		t.Fatal(err)
	}
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
	if err := gopacket.SerializeLayers(buf, opts, eth, ip, udp, gopacket.Payload(payload)); err != nil {
		t.Fatal(err)
	}

	return &Packet{Buffer: buf.Bytes(), Length: len(buf.Bytes()), SerialNumber: 1}, ip
}

func TestUDPHandlerAnswersListIdentity(t *testing.T) {
	stack := newTestStackInternal(t)
	handler := NewUDPHandler(stack)
	plc := enipTestNode(enipTestServer())

	pkt, ip := buildENIPDatagram(t, plc, enipRequest(0x63))
	handler.HandlePacket(pkt, ip, []*config.Device{plc})

	var reply *Packet
	select {
	case reply = <-stack.sendQueue:
	default:
		t.Fatal("no List Identity reply was queued")
	}
	decoded := gopacket.NewPacket(reply.Buffer, layers.LayerTypeEthernet, gopacket.Default)
	ipOut, _ := decoded.Layer(layers.LayerTypeIPv4).(*layers.IPv4)
	udpOut, _ := decoded.Layer(layers.LayerTypeUDP).(*layers.UDP)
	if ipOut == nil || udpOut == nil {
		t.Fatalf("reply is not UDP/IPv4: %v", decoded)
	}
	if !ipOut.SrcIP.Equal(plc.IPAddresses[0]) || !ipOut.DstIP.Equal(ip.SrcIP) {
		t.Errorf("reply %s -> %s, want %s -> %s", ipOut.SrcIP, ipOut.DstIP, plc.IPAddresses[0], ip.SrcIP)
	}
	if udpOut.SrcPort != PortEtherNetIP || udpOut.DstPort != 50000 {
		t.Errorf("reply ports %d -> %d, want 44818 -> 50000", udpOut.SrcPort, udpOut.DstPort)
	}
	if !bytes.Equal(udpOut.Payload, enipTestIdentity()) {
		t.Errorf("reply % x, want % x", udpOut.Payload, enipTestIdentity())
	}
}

// A device without EtherNet/IP answers UDP 44818 with port unreachable.
func TestUDPHandlerEtherNetIPClosedWithoutIt(t *testing.T) {
	stack := newTestStackInternal(t)
	stack.icmpHandler = NewICMPHandler(stack)
	handler := NewUDPHandler(stack)
	device := enipTestNode(nil)

	pkt, ip := buildENIPDatagram(t, device, enipRequest(0x63))
	handler.HandlePacket(pkt, ip, []*config.Device{device})

	select {
	case reply := <-stack.sendQueue:
		decoded := gopacket.NewPacket(reply.Buffer, layers.LayerTypeEthernet, gopacket.Default)
		icmp, _ := decoded.Layer(layers.LayerTypeICMPv4).(*layers.ICMPv4)
		if icmp == nil || icmp.TypeCode.Code() != layers.ICMPv4CodePort {
			t.Errorf("reply is not port unreachable: %v", decoded)
		}
	default:
		t.Error("no port unreachable for a closed 44818")
	}
}
