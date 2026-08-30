package capture

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcap"
	"github.com/gopacket/gopacket/pcapgo"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// --- Engine tests that exercise Filter, Stats, SendARP, GetInterfaceMAC ---

// TestEngine_Filter tests Filter getter and SetFilter.
func TestEngine_Filter(t *testing.T) {
	engine, handle := newFakeEngine(0)

	if f := engine.Filter(); f != "" {
		t.Errorf("Expected empty filter, got %q", f)
	}

	if err := engine.SetFilter("ip"); err != nil {
		t.Fatalf("SetFilter(ip): %v", err)
	}

	if f := engine.Filter(); f != "ip" {
		t.Errorf("Expected filter 'ip', got %q", f)
	}

	if err := engine.SetFilter("tcp"); err != nil {
		t.Fatalf("SetFilter(tcp): %v", err)
	}

	if f := engine.Filter(); f != "tcp" {
		t.Errorf("Expected filter 'tcp', got %q", f)
	}

	// Filter() reporting the right string is not enough: the expression has to
	// reach the handle, in order, or capture is unfiltered on the wire.
	want := []string{"ip", "tcp"}
	if got := handle.installedFilters(); !slices.Equal(got, want) {
		t.Errorf("handle received filters %q, want %q", got, want)
	}
}

// TestEngine_SetFilter_Rejected proves a filter the handle refuses is reported
// and does not become the engine's active filter — otherwise Filter() would
// claim a filter that was never installed.
func TestEngine_SetFilter_Rejected(t *testing.T) {
	engine, handle := newFakeEngine(0)

	if err := engine.SetFilter("ip"); err != nil {
		t.Fatalf("SetFilter(ip): %v", err)
	}

	handle.filterErr = errFakeHandle

	if err := engine.SetFilter("not a valid bpf expression !!!"); !errors.Is(err, errFakeHandle) {
		t.Fatalf("SetFilter error = %v, want errFakeHandle", err)
	}

	if f := engine.Filter(); f != "ip" {
		t.Errorf("active filter = %q, want the previous 'ip' to survive a rejected set", f)
	}
}

// TestEngine_Stats tests Stats retrieval.
func TestEngine_Stats(t *testing.T) {
	engine, handle := newFakeEngine(0)
	handle.stats = pcap.Stats{PacketsReceived: 42, PacketsDropped: 7, PacketsIfDropped: 1}

	stats, err := engine.Stats()
	if err != nil {
		t.Fatalf("Stats() failed: %v", err)
	}

	if stats.PacketsReceived != 42 {
		t.Errorf("PacketsReceived = %d, want 42", stats.PacketsReceived)
	}
	if stats.PacketsDropped != 7 {
		t.Errorf("PacketsDropped = %d, want 7", stats.PacketsDropped)
	}
	if stats.PacketsIfDropped != 1 {
		t.Errorf("PacketsIfDropped = %d, want 1", stats.PacketsIfDropped)
	}
}

// TestEngine_Stats_Error surfaces a handle that cannot report statistics.
func TestEngine_Stats_Error(t *testing.T) {
	engine, handle := newFakeEngine(0)
	handle.statsErr = errFakeHandle

	if _, err := engine.Stats(); !errors.Is(err, errFakeHandle) {
		t.Fatalf("Stats error = %v, want errFakeHandle", err)
	}
}

// decodeARP decodes the single frame the engine wrote and returns its Ethernet
// and ARP layers.
func decodeARP(t *testing.T, handle *fakeHandle) (*layers.Ethernet, *layers.ARP) {
	t.Helper()

	frames := handle.writtenFrames()
	if len(frames) != 1 {
		t.Fatalf("handle received %d frames, want exactly 1", len(frames))
	}

	packet := gopacket.NewPacket(frames[0], layers.LayerTypeEthernet, gopacket.Default)

	eth, ok := packet.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
	if !ok {
		t.Fatalf("frame has no Ethernet layer: %v", packet)
	}

	arp, ok := packet.Layer(layers.LayerTypeARP).(*layers.ARP)
	if !ok {
		t.Fatalf("frame has no ARP layer: %v", packet)
	}

	return eth, arp
}

// TestEngine_SendARP_Request asserts the bytes handed to the handle really are
// an ARP request carrying the addresses the caller asked for.
func TestEngine_SendARP_Request(t *testing.T) {
	engine, handle := newFakeEngine(0)

	srcMAC := []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	dstMAC := []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}

	if err := engine.SendARP(srcMAC, dstMAC, "192.168.1.1", "192.168.1.2", true); err != nil {
		t.Fatalf("SendARP: %v", err)
	}

	eth, arp := decodeARP(t, handle)

	if !slices.Equal(eth.SrcMAC, srcMAC) || !slices.Equal(eth.DstMAC, dstMAC) {
		t.Errorf("ethernet MACs = %s -> %s, want %x -> %x", eth.SrcMAC, eth.DstMAC, srcMAC, dstMAC)
	}
	if eth.EthernetType != layers.EthernetTypeARP {
		t.Errorf("ethertype = %v, want ARP", eth.EthernetType)
	}
	if arp.Operation != uint16(layers.ARPRequest) {
		t.Errorf("ARP operation = %d, want request (%d)", arp.Operation, layers.ARPRequest)
	}
	if got := net.IP(arp.SourceProtAddress).String(); got != "192.168.1.1" {
		t.Errorf("ARP sender IP = %s, want 192.168.1.1", got)
	}
	if got := net.IP(arp.DstProtAddress).String(); got != "192.168.1.2" {
		t.Errorf("ARP target IP = %s, want 192.168.1.2", got)
	}
	if !slices.Equal(arp.SourceHwAddress, srcMAC) {
		t.Errorf("ARP sender MAC = %x, want %x", arp.SourceHwAddress, srcMAC)
	}
}

// TestEngine_SendARP_Reply asserts the reply opcode, which is the only thing
// distinguishing a reply from a request on the wire.
func TestEngine_SendARP_Reply(t *testing.T) {
	engine, handle := newFakeEngine(0)

	srcMAC := []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	dstMAC := []byte{0x00, 0x66, 0x77, 0x88, 0x99, 0xaa}

	if err := engine.SendARP(srcMAC, dstMAC, "10.0.0.1", "10.0.0.2", false); err != nil {
		t.Fatalf("SendARP: %v", err)
	}

	_, arp := decodeARP(t, handle)

	if arp.Operation != uint16(layers.ARPReply) {
		t.Errorf("ARP operation = %d, want reply (%d)", arp.Operation, layers.ARPReply)
	}
	if !slices.Equal(arp.DstHwAddress, dstMAC) {
		t.Errorf("ARP target MAC = %x, want %x", arp.DstHwAddress, dstMAC)
	}
}

// TestEngine_SendARP_InvalidIP rejects unparseable addresses before anything
// reaches the wire.
func TestEngine_SendARP_InvalidIP(t *testing.T) {
	tests := []struct {
		name         string
		srcIP, dstIP string
	}{
		{"invalid source", "not-an-ip", "192.168.1.2"},
		{"invalid destination", "192.168.1.1", "invalid-ip"},
		{"IPv6 source is not an ARP address", "2001:db8::1", "192.168.1.2"},
	}

	srcMAC := []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	dstMAC := []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine, handle := newFakeEngine(0)

			if err := engine.SendARP(srcMAC, dstMAC, tt.srcIP, tt.dstIP, true); err == nil {
				t.Fatal("SendARP returned nil, want an error")
			}

			if got := len(handle.writtenFrames()); got != 0 {
				t.Errorf("handle received %d frames, want none for a rejected address", got)
			}
		})
	}
}

// TestEngine_GetInterfaceMAC tests MAC address retrieval.
func TestEngine_GetInterfaceMAC(t *testing.T) {
	want := net.HardwareAddr{0x02, 0x00, 0x00, 0x00, 0x00, 0x01}

	engine, _ := newFakeEngine(0)
	engine.lookupInterface = func(name string) (*net.Interface, error) {
		if name != "fake0" {
			t.Errorf("looked up %q, want the engine's own interface", name)
		}

		return &net.Interface{Name: name, HardwareAddr: want}, nil
	}

	got, err := engine.GetInterfaceMAC()
	if err != nil {
		t.Fatalf("GetInterfaceMAC: %v", err)
	}

	if !slices.Equal(got, want) {
		t.Errorf("MAC = %x, want %x", got, want)
	}
}

// TestEngine_GetInterfaceMAC_NoAddress is the loopback case: an interface with
// no hardware address must report ErrNoMACAddressFound rather than a short MAC.
func TestEngine_GetInterfaceMAC_NoAddress(t *testing.T) {
	engine, _ := newFakeEngine(0)
	engine.lookupInterface = func(name string) (*net.Interface, error) {
		return &net.Interface{Name: name}, nil
	}

	if _, err := engine.GetInterfaceMAC(); !errors.Is(err, ErrNoMACAddressFound) {
		t.Fatalf("error = %v, want ErrNoMACAddressFound", err)
	}
}

// TestEngine_GetInterfaceMAC_LookupFails propagates a failed lookup.
func TestEngine_GetInterfaceMAC_LookupFails(t *testing.T) {
	engine, _ := newFakeEngine(0)
	engine.lookupInterface = func(string) (*net.Interface, error) {
		return nil, errFakeHandle
	}

	if _, err := engine.GetInterfaceMAC(); !errors.Is(err, errFakeHandle) {
		t.Fatalf("error = %v, want errFakeHandle", err)
	}
}

// TestEngine_SendPacket_VerboseDebug tests SendPacket at verbose debug level.
func TestEngine_SendPacket_VerboseDebug(t *testing.T) {
	engine, handle := newFakeEngine(debugLevelVerbose)

	frame := []byte{0xde, 0xad, 0xbe, 0xef}
	if err := engine.SendPacket(frame); err != nil {
		t.Fatalf("SendPacket: %v", err)
	}

	written := handle.writtenFrames()
	if len(written) != 1 || !slices.Equal(written[0], frame) {
		t.Fatalf("handle received %x, want exactly [%x]", written, frame)
	}
}

// TestEngine_SendPacket_Error surfaces a driver refusing the frame.
func TestEngine_SendPacket_Error(t *testing.T) {
	engine, handle := newFakeEngine(0)
	handle.writeErr = errFakeHandle

	if err := engine.SendPacket([]byte{0x01}); !errors.Is(err, errFakeHandle) {
		t.Fatalf("SendPacket error = %v, want errFakeHandle", err)
	}
}

// TestEngine_Close_NilHandle tests Close with nil handle.
func TestEngine_Close_NilHandle(_ *testing.T) {
	engine := &Engine{
		interfaceName: "test",
		handle:        nil,
	}

	// Should not panic
	engine.Close()
}

// --- Playback tests for Start with debug logging paths ---

// createTestPCAPFile creates a valid PCAP file for playback tests.
func createTestPCAPFile(t *testing.T, count int) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "test.pcap")

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Failed to create PCAP file: %v", err)
	}
	defer func() { _ = f.Close() }()

	w := pcapgo.NewWriter(f)

	if err = w.WriteFileHeader(1600, layers.LinkTypeEthernet); err != nil {
		t.Fatalf("Failed to write PCAP header: %v", err)
	}

	srcMAC := []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	dstMAC := []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	baseTime := time.Now()

	for i := range count {
		eth := &layers.Ethernet{
			SrcMAC:       srcMAC,
			DstMAC:       dstMAC,
			EthernetType: layers.EthernetTypeIPv4,
		}

		buf := gopacket.NewSerializeBuffer()
		_ = gopacket.SerializeLayers(buf, gopacket.SerializeOptions{}, eth, gopacket.Payload([]byte{byte(i)}))

		ts := baseTime.Add(time.Duration(i*10) * time.Millisecond)
		info := gopacket.CaptureInfo{
			Timestamp:     ts,
			CaptureLength: len(buf.Bytes()),
			Length:        len(buf.Bytes()),
		}

		_ = w.WritePacket(info, buf.Bytes())
	}

	return path
}

// TestPlaybackEngine_Start_WithDebugLogging tests Start with debug levels that trigger logging.
func TestPlaybackEngine_Start_WithDebugLogging(t *testing.T) {
	const packetCount = 3

	pcapFile := createTestPCAPFile(t, packetCount)
	sender := &recordingSender{}

	// ScaleTime and LoopTime set so the debug logging branches fire.
	pb := NewPlaybackEngine(sender, &config.CapturePlayback{
		FileName:  pcapFile,
		ScaleTime: 2.0,
		LoopTime:  500,
		RateMode:  config.RateTopspeed,
	}, 1) // debug >= 1 triggers logging

	if err := pb.Start(); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	// Stop on the first completed pass rather than after a fixed sleep.
	waitPasses(t, pb, 1, 2*time.Second)
	pb.Stop()

	if pb.IsRunning() {
		t.Error("Expected playback to be stopped")
	}
	if got := sender.Count(); got != packetCount {
		t.Errorf("sent %d frames in the first pass, want %d", got, packetCount)
	}
}

// TestPlaybackEngine_Start_PlayOnce tests Start without looping (plays once and exits).
func TestPlaybackEngine_Start_PlayOnce(t *testing.T) {
	const packetCount = 2

	pcapFile := createTestPCAPFile(t, packetCount)
	sender := &recordingSender{}

	pb := NewPlaybackEngine(sender, &config.CapturePlayback{
		FileName:  pcapFile,
		ScaleTime: 1.0,
		LoopTime:  0, // no loop, play once
		RateMode:  config.RateTopspeed,
	}, debugLevelVerbose) // verbose to cover sendPacketWithLogging

	if err := pb.Start(); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer pb.Stop()

	// The single-shot loop returns on its own; wait for that rather than for a
	// duration guessed from the capture timing.
	waitLoopExit(t, pb, 2*time.Second)

	if got := pb.Progress().Passes; got != 1 {
		t.Errorf("passes = %d, want exactly 1", got)
	}

	assertFramesMatchPCAP(t, sender, pcapFile)
}

// TestPlaybackEngine_LoadPCAP_EmptyFile tests loading a PCAP with zero packets.
func TestPlaybackEngine_LoadPCAP_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.pcap")

	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}

	w := pcapgo.NewWriter(f)

	err = w.WriteFileHeader(1600, layers.LinkTypeEthernet)
	if err != nil {
		t.Fatal(err)
	}

	_ = f.Close()

	pb := &PlaybackEngine{
		config: &config.CapturePlayback{
			FileName: path,
		},
		stopChan: make(chan struct{}),
	}

	packets, err := collectPlayback(pb)
	if err != nil {
		t.Fatalf("collectPlayback on empty file failed: %v", err)
	}

	if len(packets) != 0 {
		t.Errorf("Expected 0 packets from empty PCAP, got %d", len(packets))
	}
}

// TestPlaybackEngine_LoadPCAP_InvalidFile tests loadPCAP with a non-PCAP file.
func TestPlaybackEngine_LoadPCAP_InvalidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.pcap")

	err := os.WriteFile(path, []byte("not a pcap file"), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	pb := &PlaybackEngine{
		config: &config.CapturePlayback{
			FileName: path,
		},
		stopChan: make(chan struct{}),
	}

	_, err = collectPlayback(pb)
	if err == nil {
		t.Error("Expected error loading invalid PCAP file")
	}
}

// TestRateLimiter_LargeRate tests RateLimiter with a large packets-per-second value.
func TestRateLimiter_LargeRate(t *testing.T) {
	rl := NewRateLimiter(10000)
	defer rl.Stop()

	if rl.packetsPerSecond != 10000 {
		t.Errorf("Expected 10000, got %d", rl.packetsPerSecond)
	}

	// Should be able to drain many tokens quickly
	for range 100 {
		rl.Wait()
	}
}

// TestRateLimiter_OneRate tests RateLimiter with exactly 1 pps.
func TestRateLimiter_OneRate(t *testing.T) {
	rl := NewRateLimiter(1)
	defer rl.Stop()

	if rl.packetsPerSecond != 1 {
		t.Errorf("Expected 1, got %d", rl.packetsPerSecond)
	}

	// First wait should succeed immediately (pre-filled)
	start := time.Now()
	rl.Wait()

	if time.Since(start) > 50*time.Millisecond {
		t.Error("First Wait on 1-pps limiter should be immediate")
	}
}
