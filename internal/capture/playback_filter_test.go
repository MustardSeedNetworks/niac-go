package capture

import (
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcapgo"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// udpPacketBytes serializes an Ethernet/IPv4/UDP frame to dstPort.
func udpPacketBytes(t *testing.T, dstPort layers.UDPPort) []byte {
	t.Helper()

	eth := &layers.Ethernet{
		SrcMAC:       net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		DstMAC:       net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		EthernetType: layers.EthernetTypeIPv4,
	}
	ip := &layers.IPv4{
		Version:  4,
		IHL:      5,
		TTL:      64,
		Protocol: layers.IPProtocolUDP,
		SrcIP:    net.IP{10, 0, 0, 1},
		DstIP:    net.IP{10, 0, 0, 2},
	}
	udp := &layers.UDP{SrcPort: 40000, DstPort: dstPort}
	if err := udp.SetNetworkLayerForChecksum(ip); err != nil {
		t.Fatalf("checksum layer: %v", err)
	}

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
	if err := gopacket.SerializeLayers(buf, opts, eth, ip, udp, gopacket.Payload([]byte("x"))); err != nil {
		t.Fatalf("serialize: %v", err)
	}

	return buf.Bytes()
}

// createMixedUDPPCAP writes dns packets to UDP/53 followed by other packets to
// UDP/80 and returns the file path.
func createMixedUDPPCAP(t *testing.T, dns, other int) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "mixed.pcap")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer func() { _ = f.Close() }()

	w := pcapgo.NewWriter(f)
	if err = w.WriteFileHeader(1600, layers.LinkTypeEthernet); err != nil {
		t.Fatalf("header: %v", err)
	}

	write := func(port layers.UDPPort, n int) {
		data := udpPacketBytes(t, port)
		for range n {
			ci := gopacket.CaptureInfo{Timestamp: time.Now(), CaptureLength: len(data), Length: len(data)}
			if err = w.WritePacket(ci, data); err != nil {
				t.Fatalf("write: %v", err)
			}
		}
	}
	write(53, dns)
	write(80, other)

	return path
}

// TestMatchesFilter checks the compiled filter accepts matching packets and
// drops the rest, and that a nil VM (no filter) passes everything.
func TestMatchesFilter(t *testing.T) {
	vm, err := compileBPFVM(layers.LinkTypeEthernet, "udp port 53")
	if err != nil {
		t.Fatalf("compileBPFVM: %v", err)
	}
	filtered := &PlaybackEngine{bpfVM: vm}

	if !filtered.matchesFilter(udpPacketBytes(t, 53)) {
		t.Error("udp/53 should match \"udp port 53\"")
	}
	if filtered.matchesFilter(udpPacketBytes(t, 80)) {
		t.Error("udp/80 should not match \"udp port 53\"")
	}

	open := &PlaybackEngine{}
	if !open.matchesFilter(udpPacketBytes(t, 80)) {
		t.Error("nil filter should pass every packet")
	}
}

// TestValidateBPFExpr accepts a well-formed filter and rejects garbage.
func TestValidateBPFExpr(t *testing.T) {
	if err := ValidateBPFExpr("udp port 53"); err != nil {
		t.Errorf("valid filter rejected: %v", err)
	}
	if err := ValidateBPFExpr("this is not a bpf filter !!!"); err == nil {
		t.Error("malformed filter accepted")
	}
}

// TestReplay_BPFFilter_MalformedFailsStart proves a bad filter is reported
// synchronously by Start rather than swallowed in the playback goroutine.
func TestReplay_BPFFilter_MalformedFailsStart(t *testing.T) {
	pb := newLoopbackPlayback(t, &config.CapturePlayback{
		FileName:  createTestPCAPFile(t, 1),
		BPFFilter: "not a valid filter !!!",
	})

	if err := pb.Start(); err == nil {
		pb.Stop()
		t.Fatal("Start should fail on a malformed BPF filter")
	}
}

// TestReplay_BPFFilter_SelectsSubset replays a mixed capture with a filter and
// confirms only matching packets are sent, the rest counted as filtered.
func TestReplay_BPFFilter_SelectsSubset(t *testing.T) {
	const (
		dns   = 3
		other = 2
	)
	pb := newLoopbackPlayback(t, &config.CapturePlayback{
		FileName:  createMixedUDPPCAP(t, dns, other),
		BPFFilter: "udp port 53",
		RateMode:  config.RateTopspeed,
	})

	if err := pb.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer pb.Stop()

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		p := pb.Progress()
		if p.PacketsSent+p.PacketsFiltered == dns+other {
			break
		}
		time.Sleep(2 * time.Millisecond)
	}

	got := pb.Progress()
	if got.PacketsSent != dns {
		t.Errorf("PacketsSent = %d, want %d (udp/53 only)", got.PacketsSent, dns)
	}
	if got.PacketsFiltered != other {
		t.Errorf("PacketsFiltered = %d, want %d (udp/80 dropped)", got.PacketsFiltered, other)
	}
}
