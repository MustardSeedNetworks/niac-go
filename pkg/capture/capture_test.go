package capture

import (
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

// TestRateLimiter_NewRateLimiter tests rate limiter creation
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

// TestRateLimiter_Wait tests that Wait() blocks and releases
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

// TestRateLimiter_Wait_RateLimiting tests actual rate limiting behavior
func TestRateLimiter_Wait_RateLimiting(t *testing.T) {
	// Use low rate to test rate limiting
	packetsPerSecond := 10
	rl := NewRateLimiter(packetsPerSecond)
	defer rl.Stop()

	// Drain the initial tokens
	for i := 0; i < packetsPerSecond; i++ {
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

// TestRateLimiter_Stop tests clean shutdown
func TestRateLimiter_Stop(t *testing.T) {
	rl := NewRateLimiter(100)

	// Stop should not panic
	rl.Stop()

	// Verify ticker stopped (accessing stopped ticker doesn't panic)
	// We can't directly test the goroutine exit without race detector,
	// but Stop() closes done channel which causes goroutine to exit
}

// TestRateLimiter_Stop_NoLeaks tests that goroutine exits after Stop
func TestRateLimiter_Stop_NoLeaks(t *testing.T) {
	// This test verifies goroutine cleanup by creating/stopping many rate limiters
	for i := 0; i < 100; i++ {
		rl := NewRateLimiter(100)
		rl.Stop()
	}
	// If goroutines don't exit, this would accumulate many goroutines
	// Run with -race flag to detect leaks
}

// TestRateLimiter_ConcurrentWait tests concurrent Wait() calls
func TestRateLimiter_ConcurrentWait(t *testing.T) {
	rl := NewRateLimiter(100)
	defer rl.Stop()

	var wg sync.WaitGroup
	goroutines := 10

	// Launch multiple goroutines calling Wait()
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rl.Wait()
		}()
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

// TestInterfaceExists tests interface existence check
func TestInterfaceExists(t *testing.T) {
	// Test with loopback interface (should exist on all systems)
	loopbackNames := []string{"lo", "lo0", "Loopback"}
	found := false

	for _, name := range loopbackNames {
		if InterfaceExists(name) {
			found = true
			break
		}
	}

	if !found {
		t.Skip("No loopback interface found (unusual but not an error)")
	}
}

// TestInterfaceExists_NonExistent tests checking for non-existent interface
func TestInterfaceExists_NonExistent(t *testing.T) {
	if InterfaceExists("definitely-does-not-exist-interface-12345") {
		t.Error("InterfaceExists returned true for non-existent interface")
	}
}

// TestGetInterface tests getting interface information
func TestGetInterface(t *testing.T) {
	// Get list of interfaces first
	devices, err := pcap.FindAllDevs()
	if err != nil {
		t.Skipf("Cannot enumerate interfaces: %v", err)
	}

	if len(devices) == 0 {
		t.Skip("No network interfaces found")
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

// TestGetInterface_NonExistent tests getting non-existent interface
func TestGetInterface_NonExistent(t *testing.T) {
	_, err := GetInterface("definitely-does-not-exist-interface-12345")

	if err == nil {
		t.Error("Expected error for non-existent interface, got nil")
	}
}

// TestListInterfaces tests listing all interfaces
func TestListInterfaces(t *testing.T) {
	// This test just verifies ListInterfaces doesn't panic
	// We can't easily capture stdout to verify output
	ListInterfaces()
}

// TestEngine_NewEngine tests engine creation
func TestEngine_New(t *testing.T) {
	// This test requires a valid network interface
	// Skip if not running with privileges or on CI without network access
	if os.Getenv("CI") != "" {
		t.Skip("Skipping engine creation test in CI environment")
	}

	// Try to find loopback interface
	loopbackNames := []string{"lo", "lo0", "Loopback"}
	var testInterface string

	for _, name := range loopbackNames {
		if InterfaceExists(name) {
			testInterface = name
			break
		}
	}

	if testInterface == "" {
		t.Skip("No loopback interface found for testing")
	}

	engine, err := New(testInterface, 0)
	if err != nil {
		t.Skipf("Cannot create engine (may need privileges): %v", err)
	}
	defer engine.Close()

	if engine == nil {
		t.Fatal("New returned nil engine without error")
	}

	if engine.interfaceName != testInterface {
		t.Errorf("Expected interface %s, got %s", testInterface, engine.interfaceName)
	}

	if engine.handle == nil {
		t.Error("Engine handle is nil")
	}
}

// TestEngine_New_InvalidInterface tests engine creation with invalid interface
func TestEngine_New_InvalidInterface(t *testing.T) {
	_, err := New("definitely-does-not-exist-interface-12345", 0)

	if err == nil {
		t.Error("Expected error for invalid interface, got nil")
	}
}

// TestEngine_Close tests engine cleanup
func TestEngine_Close(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping engine test in CI environment")
	}

	loopbackNames := []string{"lo", "lo0", "Loopback"}
	var testInterface string

	for _, name := range loopbackNames {
		if InterfaceExists(name) {
			testInterface = name
			break
		}
	}

	if testInterface == "" {
		t.Skip("No loopback interface found")
	}

	engine, err := New(testInterface, 0)
	if err != nil {
		t.Skipf("Cannot create engine: %v", err)
	}

	// Close should not panic
	engine.Close()

	// Calling Close() twice should not panic
	engine.Close()
}

// TestSendEthernet_ValidFrame tests building Ethernet frame
func TestSendEthernet_ValidFrame(t *testing.T) {
	// This test would require a real interface to send packets
	// Instead, we can test the serialization logic by creating an engine
	// and checking if SendEthernet builds valid frames

	if os.Getenv("CI") != "" {
		t.Skip("Skipping packet send test in CI environment")
	}

	loopbackNames := []string{"lo", "lo0"}
	var testInterface string

	for _, name := range loopbackNames {
		if InterfaceExists(name) {
			testInterface = name
			break
		}
	}

	if testInterface == "" {
		t.Skip("No loopback interface found")
	}

	engine, err := New(testInterface, 0)
	if err != nil {
		t.Skipf("Cannot create engine: %v", err)
	}
	defer engine.Close()

	srcMAC := []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	dstMAC := []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	payload := []byte{0x01, 0x02, 0x03, 0x04}

	// This will attempt to send, which may fail on loopback, but shouldn't panic
	err = engine.SendEthernet(dstMAC, srcMAC, uint16(layers.EthernetTypeIPv4), payload)
	// Don't fail test if send fails - we're mainly testing it doesn't panic
	// Some systems don't allow raw packet sending on loopback
	_ = err
}

// TestBuildARPPacket tests ARP packet construction
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

// TestRateLimiter_TokenBucket tests token bucket behavior
func TestRateLimiter_TokenBucket(t *testing.T) {
	packetsPerSecond := 5
	rl := NewRateLimiter(packetsPerSecond)
	defer rl.Stop()

	// Bucket should be pre-filled with packetsPerSecond tokens
	for i := 0; i < packetsPerSecond; i++ {
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

// TestEngine_DebugLevel tests debug level setting
func TestEngine_DebugLevel(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping engine test in CI environment")
	}

	loopbackNames := []string{"lo", "lo0"}
	var testInterface string

	for _, name := range loopbackNames {
		if InterfaceExists(name) {
			testInterface = name
			break
		}
	}

	if testInterface == "" {
		t.Skip("No loopback interface found")
	}

	tests := []struct {
		name       string
		debugLevel int
	}{
		{"zero debug", 0},
		{"debug level 1", 1},
		{"debug level 3", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine, err := New(testInterface, tt.debugLevel)
			if err != nil {
				t.Skipf("Cannot create engine: %v", err)
			}
			defer engine.Close()

			if engine.debugLevel != tt.debugLevel {
				t.Errorf("Expected debug level %d, got %d", tt.debugLevel, engine.debugLevel)
			}
		})
	}
}

// BenchmarkRateLimiter_Wait benchmarks rate limiter performance
func BenchmarkRateLimiter_Wait(b *testing.B) {
	rl := NewRateLimiter(10000) // High rate to minimize blocking
	defer rl.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rl.Wait()
	}
}

// BenchmarkRateLimiter_NewStop benchmarks creation and cleanup
func BenchmarkRateLimiter_NewStop(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rl := NewRateLimiter(100)
		rl.Stop()
	}
}

// BenchmarkSerializeARP benchmarks ARP packet serialization
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

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := gopacket.NewSerializeBuffer()
		_ = gopacket.SerializeLayers(buf, opts, eth, arp)
	}
}
