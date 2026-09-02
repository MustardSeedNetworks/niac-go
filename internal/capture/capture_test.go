package capture

import (
	"bytes"
	"context"
	"errors"
	"io"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcap"
)

// TestRateLimiter_NewRateLimiter tests rate limiter creation.
func TestRateLimiter_NewRateLimiter(t *testing.T) {
	rl := NewRateLimiter(100)
	defer rl.Stop()

	if rl == nil {
		t.Fatal("NewRateLimiter returned nil")
	}

	if rl.packetsPerSecond != 100 {
		t.Errorf("Expected 100 packets/sec, got %d", rl.packetsPerSecond)
	}

	if rl.tokens == nil {
		t.Error("tokens channel not initialized")
	}

	if rl.done == nil {
		t.Error("done channel not initialized")
	}

	if rl.ticker == nil {
		t.Error("ticker not initialized")
	}
}

// TestRateLimiter_Wait tests that Wait() blocks and releases.
func TestRateLimiter_Wait(t *testing.T) {
	// Use high rate to make test fast
	rl := NewRateLimiter(1000)
	defer rl.Stop()

	start := time.Now()

	rl.Wait() // Should not block initially (bucket pre-filled)

	elapsed := time.Since(start)

	// Should return almost immediately since bucket is pre-filled
	if elapsed > 10*time.Millisecond {
		t.Errorf("Wait() took too long: %v", elapsed)
	}
}

// TestRateLimiter_Wait_RateLimiting tests actual rate limiting behavior.
func TestRateLimiter_Wait_RateLimiting(t *testing.T) {
	// Use low rate to test rate limiting
	packetsPerSecond := 10

	rl := NewRateLimiter(packetsPerSecond)
	defer rl.Stop()

	// Drain the initial tokens
	for range packetsPerSecond {
		rl.Wait()
	}

	// Next wait should block until token refill
	start := time.Now()

	rl.Wait()

	elapsed := time.Since(start)

	// Should wait approximately 1/packetsPerSecond seconds (100ms for 10 pps)
	expectedWait := time.Second / time.Duration(packetsPerSecond)
	tolerance := 50 * time.Millisecond

	if elapsed < expectedWait-tolerance || elapsed > expectedWait+tolerance {
		t.Errorf("Wait() timing off: expected ~%v, got %v", expectedWait, elapsed)
	}
}

// TestRateLimiter_Stop asserts Stop closes the done channel and that refilling
// actually ceases: a drained bucket stays empty across several tick periods.
func TestRateLimiter_Stop(t *testing.T) {
	const pps = 100

	rl := NewRateLimiter(pps)

	// Drain the initial bucket so any post-Stop refill is observable.
	for range pps {
		<-rl.tokens
	}

	rl.Stop()

	select {
	case <-rl.done:
	default:
		t.Error("Stop() left the done channel open")
	}

	if got := rl.stopped.Load(); got != 1 {
		t.Errorf("stopped flag = %d, want 1", got)
	}

	// Three tick periods: a live refill goroutine would have added tokens.
	time.Sleep(3 * time.Second / pps)

	if got := len(rl.tokens); got != 0 {
		t.Errorf("bucket refilled after Stop(): %d tokens, want 0", got)
	}
}

// TestRateLimiter_Stop_NoLeaks asserts the refill goroutine is gone once Stop
// returns. Stop waits on the WaitGroup, so the count is exact, not approximate.
func TestRateLimiter_Stop_NoLeaks(t *testing.T) {
	before := runtime.NumGoroutine()

	for range 100 {
		rl := NewRateLimiter(100)
		rl.Stop()
	}

	if after := runtime.NumGoroutine(); after > before {
		t.Errorf("goroutines leaked: %d before, %d after 100 create/stop cycles", before, after)
	}
}

// TestRateLimiter_ConcurrentWait tests concurrent Wait() calls.
func TestRateLimiter_ConcurrentWait(t *testing.T) {
	rl := NewRateLimiter(100)
	defer rl.Stop()

	var wg sync.WaitGroup

	goroutines := 10

	// Launch multiple goroutines calling Wait()
	for range goroutines {
		wg.Go(func() {
			rl.Wait()
		})
	}

	// Wait for all goroutines to complete
	done := make(chan struct{})

	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Error("Concurrent Wait() calls deadlocked or took too long")
	}
}

// TestInterfaceExists tests interface existence check.
func TestInterfaceExists(t *testing.T) {
	// Test with loopback interface (should exist on all systems)
	loopbackNames := []string{"lo", "lo0", "Loopback"}

	if !slices.ContainsFunc(loopbackNames, InterfaceExists) {
		t.Fatal("no loopback interface found; every host this suite runs on has one")
	}
}

// TestInterfaceExists_NonExistent tests checking for non-existent interface.
func TestInterfaceExists_NonExistent(t *testing.T) {
	if InterfaceExists("definitely-does-not-exist-interface-12345") {
		t.Error("InterfaceExists returned true for non-existent interface")
	}
}

// TestGetInterface tests getting interface information.
func TestGetInterface(t *testing.T) {
	// Get list of interfaces first
	devices, err := pcap.FindAllDevs()
	if err != nil {
		t.Fatalf("cannot enumerate interfaces: %v", err)
	}

	if len(devices) == 0 {
		t.Fatal("no network interfaces found; enumeration needs no privileges and must find at least loopback")
	}

	// Test with first available interface
	testInterface := devices[0].Name

	iface, err := GetInterface(testInterface)
	if err != nil {
		t.Fatalf("GetInterface failed: %v", err)
	}

	if iface == nil {
		t.Fatal("GetInterface returned nil without error")
	}

	if iface.Name != testInterface {
		t.Errorf("Expected interface name %s, got %s", testInterface, iface.Name)
	}
}

// TestGetInterface_NonExistent tests getting non-existent interface.
func TestGetInterface_NonExistent(t *testing.T) {
	_, err := GetInterface("definitely-does-not-exist-interface-12345")
	if err == nil {
		t.Error("Expected error for non-existent interface, got nil")
	}
}

// TestListInterfaces asserts the listing names every interface libpcap reports.
func TestListInterfaces(t *testing.T) {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		t.Fatalf("cannot enumerate interfaces: %v", err)
	}

	if len(devices) == 0 {
		t.Fatal("no network interfaces found; enumeration needs no privileges and must find at least loopback")
	}

	var buf bytes.Buffer

	ListInterfaces(&buf)

	out := buf.String()
	for _, device := range devices {
		if !strings.Contains(out, device.Name) {
			t.Errorf("listing omitted interface %q; got:\n%s", device.Name, out)
		}
	}
}

// TestEngine_New_InvalidInterface tests engine creation with invalid interface.
func TestEngine_New_InvalidInterface(t *testing.T) {
	_, err := New("definitely-does-not-exist-interface-12345", 0)
	if err == nil {
		t.Error("Expected error for invalid interface, got nil")
	}
}

// TestSendEthernet_ValidFrame asserts the frame SendEthernet hands to the
// handle decodes back to the addresses, ethertype and payload it was given.
func TestSendEthernet_ValidFrame(t *testing.T) {
	engine, handle := newFakeEngine(0)

	srcMAC := []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	dstMAC := []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	payload := []byte{0x01, 0x02, 0x03, 0x04}

	err := engine.SendEthernet(dstMAC, srcMAC, uint16(layers.EthernetTypeIPv4), payload)
	if err != nil {
		t.Fatalf("SendEthernet: %v", err)
	}

	frames := handle.writtenFrames()
	if len(frames) != 1 {
		t.Fatalf("handle received %d frames, want exactly 1", len(frames))
	}

	packet := gopacket.NewPacket(frames[0], layers.LayerTypeEthernet, gopacket.Default)

	eth, ok := packet.Layer(layers.LayerTypeEthernet).(*layers.Ethernet)
	if !ok {
		t.Fatalf("frame has no Ethernet layer: %v", packet)
	}

	if !slices.Equal(eth.SrcMAC, srcMAC) || !slices.Equal(eth.DstMAC, dstMAC) {
		t.Errorf("MACs = %s -> %s, want %x -> %x", eth.SrcMAC, eth.DstMAC, srcMAC, dstMAC)
	}
	if eth.EthernetType != layers.EthernetTypeIPv4 {
		t.Errorf("ethertype = %v, want IPv4", eth.EthernetType)
	}
	// SendEthernet serializes with FixLengths, so a 4-byte payload is zero-padded
	// to the 46-byte Ethernet minimum — a 60-byte frame on the wire.
	const (
		minPayload = 46
		minFrame   = 60
	)

	if len(frames[0]) != minFrame {
		t.Errorf("frame length = %d, want %d (padded to the Ethernet minimum)", len(frames[0]), minFrame)
	}
	if !slices.Equal(eth.Payload[:len(payload)], payload) {
		t.Errorf("payload = %x, want it to start with %x", eth.Payload, payload)
	}
	padding := eth.Payload[len(payload):]
	wantPadding := make([]byte, minPayload-len(payload))

	if !slices.Equal(padding, wantPadding) {
		t.Errorf("payload padding = %x, want %d zero bytes", padding, len(wantPadding))
	}
}

// TestSendEthernet_WriteFails propagates a handle that refuses the frame,
// rather than reporting a send that never happened as a success.
func TestSendEthernet_WriteFails(t *testing.T) {
	engine, handle := newFakeEngine(0)
	handle.writeErr = errFakeHandle

	err := engine.SendEthernet(
		[]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		[]byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		uint16(layers.EthernetTypeIPv4),
		[]byte{0x01},
	)
	if !errors.Is(err, errFakeHandle) {
		t.Fatalf("SendEthernet error = %v, want errFakeHandle", err)
	}
}

// TestEngine_ReadPacket covers the three outcomes the read path distinguishes:
// a frame, a timeout (which must not read as an error, or shutdown stops being
// responsive), and a real failure.
func TestEngine_ReadPacket(t *testing.T) {
	frame := []byte{0xaa, 0xbb, 0xcc, 0xdd}

	t.Run("returns the frame", func(t *testing.T) {
		engine, handle := newFakeEngine(0)
		handle.reads = []readResult{{data: frame}}

		got, err := engine.ReadPacket(make([]byte, snapshotLength))
		if err != nil {
			t.Fatalf("ReadPacket: %v", err)
		}
		if !slices.Equal(got, frame) {
			t.Errorf("ReadPacket = %x, want %x", got, frame)
		}
	})

	t.Run("timeout is not an error", func(t *testing.T) {
		engine, handle := newFakeEngine(0)
		handle.reads = []readResult{{err: pcap.NextErrorTimeoutExpired}}

		got, err := engine.ReadPacket(make([]byte, snapshotLength))
		if err != nil {
			t.Fatalf("ReadPacket on timeout returned %v, want nil", err)
		}
		if got != nil {
			t.Errorf("ReadPacket on timeout returned %x, want nil", got)
		}
	})

	t.Run("read failure propagates", func(t *testing.T) {
		engine, handle := newFakeEngine(0)
		handle.reads = []readResult{{err: errFakeHandle}}

		if _, err := engine.ReadPacket(make([]byte, snapshotLength)); !errors.Is(err, errFakeHandle) {
			t.Fatalf("ReadPacket error = %v, want errFakeHandle", err)
		}
	})

	t.Run("frame larger than the buffer is returned whole", func(t *testing.T) {
		engine, handle := newFakeEngine(0)
		handle.reads = []readResult{{data: frame}}

		got, err := engine.ReadPacket(make([]byte, 2))
		if err != nil {
			t.Fatalf("ReadPacket: %v", err)
		}
		if !slices.Equal(got, frame) {
			t.Errorf("ReadPacket = %x, want the whole %x rather than a truncated copy", got, frame)
		}
	})
}

// TestEngine_StartCaptureContext_DeliversPackets drives the capture loop over a
// queued handle and asserts every frame reaches the handler in order.
func TestEngine_StartCaptureContext_DeliversPackets(t *testing.T) {
	engine, handle := newFakeEngine(0)

	first := ethernetFrame(t, 0x01)
	second := ethernetFrame(t, 0x02)
	handle.reads = []readResult{
		{data: first, ci: gopacket.CaptureInfo{CaptureLength: len(first), Length: len(first)}},
		{data: second, ci: gopacket.CaptureInfo{CaptureLength: len(second), Length: len(second)}},
	}
	handle.readErr = io.EOF

	var seen [][]byte

	err := engine.StartCaptureContext(t.Context(), func(p gopacket.Packet) {
		seen = append(seen, p.Data())
	})
	if err != nil {
		t.Fatalf("StartCaptureContext: %v", err)
	}

	if len(seen) != 2 {
		t.Fatalf("handler saw %d packets, want 2", len(seen))
	}
	if !slices.Equal(seen[0], first) || !slices.Equal(seen[1], second) {
		t.Errorf("handler saw %x, want %x then %x", seen, first, second)
	}
}

// TestEngine_StartCaptureContext_Cancellation proves a cancelled context stops
// the loop and reports why, which is how the daemon shuts capture down.
func TestEngine_StartCaptureContext_Cancellation(t *testing.T) {
	engine, handle := newFakeEngine(0)
	handle.readErr = pcap.NextErrorTimeoutExpired

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := engine.StartCaptureContext(ctx, func(gopacket.Packet) {
		t.Error("handler must not run after the context is cancelled")
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("StartCaptureContext error = %v, want context.Canceled", err)
	}
}

// TestEngine_Close_ClosesHandle asserts Close reaches the handle. Close is the
// only thing that releases the kernel ring buffer.
func TestEngine_Close_ClosesHandle(t *testing.T) {
	engine, handle := newFakeEngine(0)

	engine.Close()

	if got := handle.closeCount(); got != 1 {
		t.Errorf("handle closed %d times, want 1", got)
	}
}

// ethernetFrame serializes a minimal Ethernet frame carrying tag as its
// payload, so two frames are distinguishable.
func ethernetFrame(t *testing.T, tag byte) []byte {
	t.Helper()

	eth := &layers.Ethernet{
		SrcMAC:       []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		DstMAC:       []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		EthernetType: layers.EthernetTypeIPv4,
	}

	buf := gopacket.NewSerializeBuffer()
	if err := gopacket.SerializeLayers(buf, gopacket.SerializeOptions{FixLengths: true},
		eth, gopacket.Payload([]byte{tag, 0x00, 0x00, 0x00})); err != nil {
		t.Fatalf("serialize frame: %v", err)
	}

	return slices.Clone(buf.Bytes())
}

// TestBuildARPPacket tests ARP packet construction.
func TestBuildARPPacket(t *testing.T) {
	// Test ARP packet serialization without actually sending
	srcMAC := []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	dstMAC := []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	srcIP := "192.168.1.1"
	dstIP := "192.168.1.2"

	// Build ARP request
	arp := &layers.ARP{
		AddrType:          layers.LinkTypeEthernet,
		Protocol:          layers.EthernetTypeIPv4,
		HwAddressSize:     6,
		ProtAddressSize:   4,
		Operation:         uint16(layers.ARPRequest),
		SourceHwAddress:   srcMAC,
		SourceProtAddress: []byte(srcIP),
		DstHwAddress:      dstMAC,
		DstProtAddress:    []byte(dstIP),
	}

	eth := &layers.Ethernet{
		SrcMAC:       srcMAC,
		DstMAC:       dstMAC,
		EthernetType: layers.EthernetTypeARP,
	}

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	err := gopacket.SerializeLayers(buf, opts, eth, arp)
	if err != nil {
		t.Fatalf("Failed to serialize ARP packet: %v", err)
	}

	data := buf.Bytes()
	if len(data) == 0 {
		t.Error("Serialized ARP packet is empty")
	}

	// Verify packet can be parsed back
	packet := gopacket.NewPacket(data, layers.LayerTypeEthernet, gopacket.Default)

	arpLayer := packet.Layer(layers.LayerTypeARP)
	if arpLayer == nil {
		t.Error("Cannot parse ARP layer from serialized packet")
	}
}

// TestRateLimiter_TokenBucket tests token bucket behavior.
func TestRateLimiter_TokenBucket(t *testing.T) {
	packetsPerSecond := 5

	rl := NewRateLimiter(packetsPerSecond)
	defer rl.Stop()

	// Bucket should be pre-filled with packetsPerSecond tokens
	for i := range packetsPerSecond {
		select {
		case <-rl.tokens:
			// Success - got token
		case <-time.After(100 * time.Millisecond):
			t.Errorf("Expected token %d to be immediately available", i+1)
		}
	}

	// Next token should not be immediately available
	select {
	case <-rl.tokens:
		t.Error("Got token when bucket should be empty")
	case <-time.After(50 * time.Millisecond):
		// Expected - no token available immediately
	}
}

// BenchmarkRateLimiter_Wait benchmarks rate limiter performance.
func BenchmarkRateLimiter_Wait(b *testing.B) {
	rl := NewRateLimiter(10000) // High rate to minimize blocking
	defer rl.Stop()

	for b.Loop() {
		rl.Wait()
	}
}

// BenchmarkRateLimiter_NewStop benchmarks creation and cleanup.
func BenchmarkRateLimiter_NewStop(b *testing.B) {
	for b.Loop() {
		rl := NewRateLimiter(100)
		rl.Stop()
	}
}

// BenchmarkSerializeARP benchmarks ARP packet serialization.
func BenchmarkSerializeARP(b *testing.B) {
	srcMAC := []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	dstMAC := []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	srcIP := "192.168.1.1"
	dstIP := "192.168.1.2"

	arp := &layers.ARP{
		AddrType:          layers.LinkTypeEthernet,
		Protocol:          layers.EthernetTypeIPv4,
		HwAddressSize:     6,
		ProtAddressSize:   4,
		Operation:         uint16(layers.ARPRequest),
		SourceHwAddress:   srcMAC,
		SourceProtAddress: []byte(srcIP),
		DstHwAddress:      dstMAC,
		DstProtAddress:    []byte(dstIP),
	}

	eth := &layers.Ethernet{
		SrcMAC:       srcMAC,
		DstMAC:       dstMAC,
		EthernetType: layers.EthernetTypeARP,
	}

	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	for b.Loop() {
		buf := gopacket.NewSerializeBuffer()
		_ = gopacket.SerializeLayers(buf, opts, eth, arp)
	}
}
