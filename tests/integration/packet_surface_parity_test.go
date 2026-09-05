package integration

import (
	"bytes"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/api/capture"
	"github.com/MustardSeedNetworks/niac-go/internal/api/sse"
	"github.com/MustardSeedNetworks/niac-go/internal/capturering"
)

// The two packet surfaces must agree. A capture exported from the live
// inspector (P1c-10) and uploaded to the analyzer used to come back described
// differently, because each surface had its own decoder: the live one knew
// 802.1Q, STP, CDP and LLDP; the analyzer knew DNS and well-known ports. A
// tester comparing "what NIAC sent" with "what NIAC recorded" saw two answers.
//
// This drives the real path end to end — build frames, decode them live, write
// the pcapng the export writes, analyze those bytes — so the two cannot drift
// apart without failing here.
func TestExportedCaptureReadsBackAsTheLiveViewShowedIt(t *testing.T) {
	frames := []struct {
		name  string
		bytes []byte
		want  string
	}{
		{"ARP", arpFrame(t), "ARP"},
		{"DHCP", udpFrame(t, 68, 67, []byte{0x01, 0x01, 0x06, 0x00}), "DHCP"},
		{"SNMP", udpFrame(t, 44100, 161, []byte{0x30, 0x26, 0x02, 0x01, 0x01}), "SNMP"},
		{"LLDP", lldpFrame(t), "LLDP"},
	}

	live := make([]string, len(frames))
	liveSummary := make([]string, len(frames))
	ring := make([]capturering.Frame, 0, len(frames))
	for i, f := range frames {
		wire := sse.GopacketToWire(
			gopacket.NewPacket(f.bytes, layers.LayerTypeEthernet, gopacket.Default))
		live[i], _ = wire["protocol"].(string)
		liveSummary[i], _ = wire["summary"].(string)

		if live[i] != f.want {
			t.Errorf("live view called the %s frame %q, want %q", f.name, live[i], f.want)
		}
		ring = append(ring, capturering.Frame{
			Data:      f.bytes,
			Timestamp: time.Unix(int64(1700000000+i), 0).UTC(),
		})
	}

	var exported bytes.Buffer
	if err := capturering.WritePcapng(&exported, "niac0", ring); err != nil {
		t.Fatalf("WritePcapng: %v", err)
	}

	result, err := capture.Analyze(exported.Bytes(), "export.pcapng")
	if err != nil {
		t.Fatalf("Analyze the exported capture: %v", err)
	}
	if len(result.Packets) != len(frames) {
		t.Fatalf("analyzer read %d packets from the export, want %d",
			len(result.Packets), len(frames))
	}

	for i, f := range frames {
		got := result.Packets[i]
		if got.Protocol != live[i] {
			t.Errorf("%s: analyzer says %q, live view said %q", f.name, got.Protocol, live[i])
		}
		if liveSummary[i] != "" && got.Info != liveSummary[i] {
			t.Errorf("%s: analyzer summary %q, live view %q", f.name, got.Info, liveSummary[i])
		}
	}
}

func serialize(t *testing.T, ls ...gopacket.SerializableLayer) []byte {
	t.Helper()

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
	if err := gopacket.SerializeLayers(buf, opts, ls...); err != nil {
		t.Fatalf("serialize: %v", err)
	}

	return buf.Bytes()
}

func arpFrame(t *testing.T) []byte {
	t.Helper()

	return serialize(t,
		&layers.Ethernet{
			SrcMAC: []byte{0x02, 0, 0, 0, 0, 1}, DstMAC: []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
			EthernetType: layers.EthernetTypeARP,
		},
		&layers.ARP{
			AddrType: layers.LinkTypeEthernet, Protocol: layers.EthernetTypeIPv4,
			HwAddressSize: 6, ProtAddressSize: 4, Operation: 1,
			SourceHwAddress: []byte{0x02, 0, 0, 0, 0, 1}, SourceProtAddress: []byte{10, 0, 0, 1},
			DstHwAddress: []byte{0, 0, 0, 0, 0, 0}, DstProtAddress: []byte{10, 0, 0, 2},
		})
}

func udpFrame(t *testing.T, src, dst int, payload []byte) []byte {
	t.Helper()

	ip := &layers.IPv4{
		Version: 4, IHL: 5, TTL: 64, Protocol: layers.IPProtocolUDP,
		SrcIP: []byte{10, 0, 0, 1}, DstIP: []byte{10, 0, 0, 2},
	}
	udp := &layers.UDP{SrcPort: layers.UDPPort(src), DstPort: layers.UDPPort(dst)}
	if err := udp.SetNetworkLayerForChecksum(ip); err != nil {
		t.Fatalf("checksum layer: %v", err)
	}

	return serialize(t,
		&layers.Ethernet{
			SrcMAC: []byte{0x02, 0, 0, 0, 0, 1}, DstMAC: []byte{0x02, 0, 0, 0, 0, 2},
			EthernetType: layers.EthernetTypeIPv4,
		}, ip, udp, gopacket.Payload(payload))
}

func lldpFrame(t *testing.T) []byte {
	t.Helper()

	return serialize(t,
		&layers.Ethernet{
			SrcMAC: []byte{0x02, 0, 0, 0, 0, 1}, DstMAC: []byte{0x01, 0x80, 0xc2, 0, 0, 0x0e},
			EthernetType: layers.EthernetTypeLinkLayerDiscovery,
		},
		&layers.LinkLayerDiscovery{
			ChassisID: layers.LLDPChassisID{
				Subtype: layers.LLDPChassisIDSubTypeMACAddr,
				ID:      []byte{0x02, 0, 0, 0, 0, 1},
			},
			PortID: layers.LLDPPortID{
				Subtype: layers.LLDPPortIDSubtypeIfaceName,
				ID:      []byte("eth0"),
			},
			TTL: 120,
		})
}
