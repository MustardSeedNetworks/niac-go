package protocols

import (
	"net"
	"net/netip"
	"strings"
	"testing"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

const sipTestUserAgent = "Cisco-CP8841/12.8.1"

func sipTestPhone() *config.Device {
	return &config.Device{
		Name:        "MED-PHONE-01",
		MACAddress:  net.HardwareAddr{0x00, 0x1e, 0x7a, 0x00, 0x00, 0x02},
		IPAddresses: []net.IP{net.ParseIP("10.20.30.40")},
		SIPConfig:   &config.SIPConfig{Enabled: true, UserAgent: sipTestUserAgent},
	}
}

// sipOptions is the request a scanner such as svmap sends.
func sipOptions(via, to string) string {
	return "OPTIONS sip:100@10.20.30.40 SIP/2.0\r\n" +
		"Via: " + via + "\r\n" +
		"Max-Forwards: 70\r\n" +
		"From: \"scanner\" <sip:100@192.0.2.9>;tag=6162\r\n" +
		"To: " + to + "\r\n" +
		"Call-ID: 3f2b@192.0.2.9\r\n" +
		"CSeq: 1 OPTIONS\r\n" +
		"Content-Length: 0\r\n\r\n"
}

func TestSIPOptionsResponse(t *testing.T) {
	source := netip.MustParseAddrPort("192.0.2.9:5062")
	tag := sipTag("MED-PHONE-01", "3f2b@192.0.2.9")
	tests := []struct {
		name     string
		request  string
		wantPort uint16
		want     []string // lines the response must carry
	}{
		{
			// Behind NAT the sent-by port is not the one the request left from.
			name:     "rport answers the source port and records it",
			request:  sipOptions("SIP/2.0/UDP 192.0.2.9:5070;branch=z9hG4bK-1;rport", "<sip:100@10.20.30.40>"),
			wantPort: 5062,
			want: []string{
				"SIP/2.0 200 OK",
				"Via: SIP/2.0/UDP 192.0.2.9:5070;branch=z9hG4bK-1;rport=5062;received=192.0.2.9",
				"From: \"scanner\" <sip:100@192.0.2.9>;tag=6162",
				"To: <sip:100@10.20.30.40>;tag=" + tag,
				"Call-ID: 3f2b@192.0.2.9",
				"CSeq: 1 OPTIONS",
				"Allow: " + sipAllow,
				"Server: " + sipTestUserAgent,
				"User-Agent: " + sipTestUserAgent,
				"Content-Length: 0",
			},
		},
		{
			name:     "sent-by matching the source needs no received",
			request:  sipOptions("SIP/2.0/UDP 192.0.2.9:5070;branch=z9hG4bK-2", "<sip:100@10.20.30.40>"),
			wantPort: 5070,
			want:     []string{"Via: SIP/2.0/UDP 192.0.2.9:5070;branch=z9hG4bK-2"},
		},
		{
			name:     "a named sent-by gets received and the default port",
			request:  sipOptions("SIP/2.0/UDP scanner.example;branch=z9hG4bK-3", "<sip:100@10.20.30.40>"),
			wantPort: 5060,
			want:     []string{"Via: SIP/2.0/UDP scanner.example;branch=z9hG4bK-3;received=192.0.2.9"},
		},
		{
			name: "every Via is echoed in order",
			request: sipOptions(
				"SIP/2.0/UDP 192.0.2.9:5062;branch=z9hG4bK-4, SIP/2.0/UDP 198.51.100.1;branch=z9hG4bK-0",
				"<sip:100@10.20.30.40>",
			),
			wantPort: 5062,
			want: []string{
				"Via: SIP/2.0/UDP 192.0.2.9:5062;branch=z9hG4bK-4\r\nVia: SIP/2.0/UDP 198.51.100.1;branch=z9hG4bK-0",
			},
		},
		{
			name:     "an existing To tag is kept",
			request:  sipOptions("SIP/2.0/UDP 192.0.2.9:5062;branch=z9hG4bK-5", "<sip:100@10.20.30.40>;tag=abc"),
			wantPort: 5062,
			want:     []string{"To: <sip:100@10.20.30.40>;tag=abc\r\n"},
		},
		{
			name: "compact headers and bare line feeds",
			request: "OPTIONS sip:10.20.30.40 SIP/2.0\n" +
				"v: SIP/2.0/UDP 192.0.2.9:5062;branch=z9hG4bK-6\n" +
				"f: <sip:a@192.0.2.9>;tag=1\nt: <sip:10.20.30.40>\ni: compact-1\nCSeq: 7 OPTIONS\n\n",
			wantPort: 5062,
			want:     []string{"Call-ID: compact-1", "CSeq: 7 OPTIONS", "From: <sip:a@192.0.2.9>;tag=1"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response, port, ok := sipOptionsResponse([]byte(test.request), source, sipTestPhone())
			if !ok {
				t.Fatal("no response to a valid OPTIONS")
			}
			if port != test.wantPort {
				t.Errorf("response port = %d, want %d", port, test.wantPort)
			}
			text := string(response)
			if !strings.HasSuffix(text, "\r\n\r\n") {
				t.Errorf("response does not end its header block:\n%s", text)
			}
			for _, want := range test.want {
				if !strings.Contains(text, want) {
					t.Errorf("response lacks %q:\n%s", want, text)
				}
			}
		})
	}
}

func TestSIPOptionsResponseIgnores(t *testing.T) {
	source := netip.MustParseAddrPort("192.0.2.9:5062")
	valid := sipOptions("SIP/2.0/UDP 192.0.2.9:5062;branch=z9hG4bK-1", "<sip:100@10.20.30.40>")
	tests := []struct {
		name    string
		request string
	}{
		{"INVITE is not answered", strings.ReplaceAll(valid, "OPTIONS", "INVITE")},
		{"ACK is never answered", strings.ReplaceAll(valid, "OPTIONS", "ACK")},
		{"a response is not a request", "SIP/2.0 200 OK\r\n" + valid[strings.Index(valid, "\r\n")+2:]},
		{"CSeq names another method", strings.Replace(valid, "CSeq: 1 OPTIONS", "CSeq: 1 INVITE", 1)},
		{"no Call-ID", strings.Replace(valid, "Call-ID: 3f2b@192.0.2.9\r\n", "", 1)},
		{"no Via", strings.Replace(valid, "Via: SIP/2.0/UDP 192.0.2.9:5062;branch=z9hG4bK-1\r\n", "", 1)},
		{"Via over TCP", strings.Replace(valid, "SIP/2.0/UDP", "SIP/2.0/TCP", 1)},
		{"wrong version", strings.Replace(valid, "SIP/2.0\r\n", "SIP/3.0\r\n", 1)},
		{"not SIP at all", "GET / HTTP/1.1\r\nHost: x\r\n\r\n"},
		{"empty", ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if response, _, ok := sipOptionsResponse([]byte(test.request), source, sipTestPhone()); ok {
				t.Errorf("answered:\n%s", response)
			}
		})
	}
}

func buildSIPDatagram(t *testing.T, device *config.Device, payload string) (*Packet, *layers.IPv4) {
	t.Helper()
	eth := &layers.Ethernet{
		SrcMAC: net.HardwareAddr{0xaa, 0, 0, 0, 0, 1}, DstMAC: device.MACAddress,
		EthernetType: layers.EthernetTypeIPv4,
	}
	ip := &layers.IPv4{
		Version: 4, IHL: 5, TTL: 64, Protocol: layers.IPProtocolUDP,
		SrcIP: net.ParseIP("192.0.2.9").To4(), DstIP: device.IPAddresses[0].To4(),
	}
	udp := &layers.UDP{SrcPort: 5062, DstPort: UDPPortSIP}
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

func TestUDPHandlerAnswersSIPOptions(t *testing.T) {
	stack := newTestStackInternal(t)
	handler := NewUDPHandler(stack)
	phone := sipTestPhone()
	request := sipOptions("SIP/2.0/UDP 192.0.2.9:5062;branch=z9hG4bK-1;rport", "<sip:100@10.20.30.40>")

	pkt, ip := buildSIPDatagram(t, phone, request)
	handler.HandlePacket(pkt, ip, []*config.Device{phone})

	var reply *Packet
	select {
	case reply = <-stack.sendQueue:
	default:
		t.Fatal("no SIP response was queued")
	}
	decoded := gopacket.NewPacket(reply.Buffer, layers.LayerTypeEthernet, gopacket.Default)
	ipOut, _ := decoded.Layer(layers.LayerTypeIPv4).(*layers.IPv4)
	udpOut, _ := decoded.Layer(layers.LayerTypeUDP).(*layers.UDP)
	if ipOut == nil || udpOut == nil {
		t.Fatalf("response is not UDP/IPv4: %v", decoded)
	}
	if !ipOut.SrcIP.Equal(phone.IPAddresses[0]) || !ipOut.DstIP.Equal(ip.SrcIP) {
		t.Errorf("response %s -> %s, want %s -> %s", ipOut.SrcIP, ipOut.DstIP, phone.IPAddresses[0], ip.SrcIP)
	}
	if udpOut.SrcPort != UDPPortSIP || udpOut.DstPort != 5062 {
		t.Errorf("response ports %d -> %d, want 5060 -> 5062", udpOut.SrcPort, udpOut.DstPort)
	}
	if !strings.Contains(string(udpOut.Payload), "User-Agent: "+sipTestUserAgent+"\r\n") {
		t.Errorf("response does not name the phone:\n%s", udpOut.Payload)
	}
}

// A device without SIP keeps the closed-port answer it gave before SIP had a
// case of its own, and so does a phone whose listener is switched off.
func TestUDPHandlerSIPPortClosedWithoutSIP(t *testing.T) {
	for _, sip := range []*config.SIPConfig{nil, {Enabled: false, UserAgent: sipTestUserAgent}} {
		stack := newTestStackInternal(t)
		stack.icmpHandler = NewICMPHandler(stack)
		handler := NewUDPHandler(stack)
		device := sipTestPhone()
		device.SIPConfig = sip

		pkt, ip := buildSIPDatagram(t, device, sipOptions("SIP/2.0/UDP 192.0.2.9:5062;branch=z", "<sip:x@y>"))
		handler.HandlePacket(pkt, ip, []*config.Device{device})

		select {
		case reply := <-stack.sendQueue:
			decoded := gopacket.NewPacket(reply.Buffer, layers.LayerTypeEthernet, gopacket.Default)
			icmp, _ := decoded.Layer(layers.LayerTypeICMPv4).(*layers.ICMPv4)
			if icmp == nil || icmp.TypeCode.Code() != layers.ICMPv4CodePort {
				t.Errorf("SIP %+v: reply is not port unreachable: %v", sip, decoded)
			}
		default:
			t.Errorf("SIP %+v: no port unreachable for a closed 5060", sip)
		}
	}
}
