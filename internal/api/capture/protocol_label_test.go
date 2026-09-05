package capture

import (
	"testing"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
)

// buildTCPPacket assembles a real Ethernet/IPv4/TCP frame so the label is
// derived the same way it is for a capture file, not from a hand-made struct.
func buildTCPPacket(t *testing.T, srcPort, dstPort int, syn bool, payload []byte) gopacket.Packet {
	t.Helper()

	eth := &layers.Ethernet{
		SrcMAC:       []byte{0x02, 0, 0, 0, 0, 1},
		DstMAC:       []byte{0x02, 0, 0, 0, 0, 2},
		EthernetType: layers.EthernetTypeIPv4,
	}
	ip := &layers.IPv4{
		Version: 4, IHL: 5, TTL: 64, Protocol: layers.IPProtocolTCP,
		SrcIP: []byte{10, 0, 0, 1}, DstIP: []byte{10, 0, 0, 2},
	}
	tcp := &layers.TCP{
		SrcPort: layers.TCPPort(srcPort), DstPort: layers.TCPPort(dstPort),
		SYN: syn, ACK: !syn, Seq: 1, Window: 1024,
	}
	if err := tcp.SetNetworkLayerForChecksum(ip); err != nil {
		t.Fatalf("checksum layer: %v", err)
	}

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
	if err := gopacket.SerializeLayers(buf, opts, eth, ip, tcp, gopacket.Payload(payload)); err != nil {
		t.Fatalf("serialize: %v", err)
	}
	return gopacket.NewPacket(buf.Bytes(), layers.LayerTypeEthernet, gopacket.Default)
}

func labelFor(t *testing.T, packet gopacket.Packet) string {
	t.Helper()

	return parsePacket(packet, 1).Protocol
}

// TestBarePortsDoNotImplyAnApplicationProtocol guards the analyzer's PROTOCOL
// column, and now the live inspector's too: both read internal/packetdecode.
//
// The analyzer once applied its port table unconditionally, so a three-way
// handshake on port 80 — SYN, SYN-ACK, ACK, none of which carry a byte of
// application data — was labelled HTTP. tcpdump and Wireshark both show TCP for
// those and reserve the application name for segments that actually carry
// payload, so anyone cross-checking a NIAC capture against tcpdump saw a
// disagreement on the first three rows of every conversation.
func TestBarePortsDoNotImplyAnApplicationProtocol(t *testing.T) {
	tests := []struct {
		name    string
		srcPort int
		dstPort int
		syn     bool
		payload []byte
		want    string
	}{
		{"SYN to port 80", 44100, 80, true, nil, "TCP"},
		{"bare ACK on port 80", 44100, 80, false, nil, "TCP"},
		{"response ACK from port 80", 80, 44100, false, nil, "TCP"},
		{"request with payload", 44100, 80, false, []byte("GET / HTTP/1.1\r\n\r\n"), "HTTP"},
		{"response with payload", 80, 44100, false, []byte("HTTP/1.1 200 OK\r\n\r\n"), "HTTP"},
		{"payload on an unknown port", 44100, 54321, false, []byte("hello"), "TCP"},
		{"SSH with payload", 44100, 22, false, []byte("SSH-2.0-x"), "SSH"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packet := buildTCPPacket(t, tt.srcPort, tt.dstPort, tt.syn, tt.payload)
			if got := labelFor(t, packet); got != tt.want {
				t.Errorf("protocol = %q, want %q", got, tt.want)
			}
		})
	}
}
