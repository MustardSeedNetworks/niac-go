package protocols

import (
	"encoding/binary"
	"net"
	"strings"
	"testing"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// A discovery tool that finds an unknown address asks it "what are you called"
// by sending an NBSTAT node-status request for the wildcard name "*". That
// cannot be answered by matching the queried name against a device, because "*"
// matches nothing — the device is identified by the address the request was
// addressed to. Before this was handled, simulated Windows endpoints stayed
// anonymous on a discovery map even with NetBIOS enabled and port 137 open.
func TestNodeStatusRequestNamesTheDevice(t *testing.T) {
	const (
		deviceName = "MED-NURSE-B01-F01-01"
		deviceIP   = "10.51.210.21"
		clientIP   = "10.254.200.100"
	)

	stack := newTestStackInternal(t)
	handler := NewNetBIOSHandler(stack, 0)

	device := &config.Device{
		Name:        deviceName,
		MACAddress:  net.HardwareAddr{0x02, 0x00, 0x33, 0x00, 0x00, 0x21},
		IPAddresses: []net.IP{net.ParseIP(deviceIP)},
		NetBIOSConfig: &config.NetBIOSConfig{
			Enabled: true, Name: deviceName, Workgroup: "DEMO",
			NodeType: "H", Services: []string{"workstation"},
		},
	}
	table := stack.devicesFor(0)
	table.AddByIP(net.ParseIP(deviceIP), device)
	table.AddByMAC(device.MACAddress, device)

	query := nodeStatusQuery(0x4242)
	packet, udp := netbiosRequestPacket(t, clientIP, deviceIP, query)

	handler.HandleNameService(&Packet{}, packet, udp, []*config.Device{device})

	select {
	case sent := <-stack.sendQueue:
		names := decodeNodeStatusNames(t, sent.Buffer[:sent.Length])
		if len(names) == 0 {
			t.Fatal("node status reply carried no names")
		}
		// NetBIOS names are capped at 15 characters, so a longer device name is
		// reported truncated — the same thing a real Windows host does.
		const netbiosNameLimit = 15
		want := deviceName
		if len(want) > netbiosNameLimit {
			want = want[:netbiosNameLimit]
		}
		if names[0] != want {
			t.Errorf("first reported name = %q, want %q", names[0], want)
		}
	default:
		t.Fatal("no node status reply — the device stays anonymous to a discovery tool")
	}
}

// nodeStatusQuery builds an NBSTAT request for the wildcard name, the form a
// scanner sends when it knows only the address.
func nodeStatusQuery(transactionID uint16) []byte {
	payload := make([]byte, 0, 50)
	payload = binary.BigEndian.AppendUint16(payload, transactionID)
	payload = binary.BigEndian.AppendUint16(payload, 0) // query, opcode 0
	payload = binary.BigEndian.AppendUint16(payload, 1) // QDCOUNT
	payload = binary.BigEndian.AppendUint16(payload, 0) // ANCOUNT
	payload = binary.BigEndian.AppendUint16(payload, 0) // NSCOUNT
	payload = binary.BigEndian.AppendUint16(payload, 0) // ARCOUNT

	name := "*" + strings.Repeat("\x00", 15)
	payload = append(payload, 32)
	for _, ch := range []byte(name) {
		payload = append(payload, 0x41+(ch>>4), 0x41+(ch&0x0F))
	}
	payload = append(payload, 0)

	payload = binary.BigEndian.AppendUint16(payload, nbnsRecordTypeNBSTAT)
	payload = binary.BigEndian.AppendUint16(payload, nbnsClassIN)

	return payload
}

func netbiosRequestPacket(t *testing.T, srcIP, dstIP string, payload []byte) (gopacket.Packet, *layers.UDP) {
	t.Helper()
	eth := &layers.Ethernet{
		SrcMAC:       net.HardwareAddr{0xaa, 0xbb, 0xcc, 0x00, 0x00, 0x01},
		DstMAC:       net.HardwareAddr{0x02, 0x00, 0x33, 0x00, 0x00, 0x21},
		EthernetType: layers.EthernetTypeIPv4,
	}
	ip := &layers.IPv4{
		Version: 4, TTL: 64, Protocol: layers.IPProtocolUDP,
		SrcIP: net.ParseIP(srcIP).To4(), DstIP: net.ParseIP(dstIP).To4(),
	}
	udp := &layers.UDP{
		SrcPort: layers.UDPPort(NetBIOSNameServicePort),
		DstPort: layers.UDPPort(NetBIOSNameServicePort),
	}
	if err := udp.SetNetworkLayerForChecksum(ip); err != nil {
		t.Fatalf("checksum layer: %v", err)
	}

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
	if err := gopacket.SerializeLayers(buf, opts, eth, ip, udp, gopacket.Payload(payload)); err != nil {
		t.Fatalf("serialize request: %v", err)
	}

	parsed := gopacket.NewPacket(buf.Bytes(), layers.LayerTypeEthernet, gopacket.Default)
	decoded, ok := parsed.Layer(layers.LayerTypeUDP).(*layers.UDP)
	if !ok {
		t.Fatal("request has no UDP layer")
	}

	return parsed, decoded
}

// decodeNodeStatusNames walks the reply to the name list, so the test asserts
// the wire form a real client parses rather than trusting our own encoder.
func decodeNodeStatusNames(t *testing.T, frame []byte) []string {
	t.Helper()
	packet := gopacket.NewPacket(frame, layers.LayerTypeEthernet, gopacket.Default)
	udp, ok := packet.Layer(layers.LayerTypeUDP).(*layers.UDP)
	if !ok {
		t.Fatal("reply has no UDP layer")
	}
	if udp.SrcPort != layers.UDPPort(NetBIOSNameServicePort) {
		t.Errorf("reply source port = %d, want %d — a client bound to 137 drops anything else",
			udp.SrcPort, NetBIOSNameServicePort)
	}

	const (
		headerLen  = 12
		nameLen    = 34 // encoded wildcard name plus terminator
		rrPrefix   = 8  // type, class, ttl
		entryLen   = 18
		nameField  = 15
		countField = 1
	)
	offset := headerLen + nameLen + rrPrefix + 2 // + RDLENGTH
	if len(udp.Payload) <= offset {
		t.Fatalf("reply too short: %d bytes", len(udp.Payload))
	}

	count := int(udp.Payload[offset])
	offset += countField

	names := make([]string, 0, count)
	for range count {
		if offset+entryLen > len(udp.Payload) {
			break
		}
		raw := strings.TrimSpace(string(udp.Payload[offset : offset+nameField]))
		names = append(names, raw)
		offset += entryLen
	}

	return names
}
