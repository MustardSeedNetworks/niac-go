package protocols_test

import (
	"net"
	"testing"
	"time"

	"github.com/krisarmstrong/niac-go/pkg/config"
	"github.com/krisarmstrong/niac-go/pkg/logging"
	"github.com/krisarmstrong/niac-go/pkg/protocols"
)

// TestNewDHCPHandler tests creating a new DHCP handler.
func TestNewDHCPHandler(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))

	handler := protocols.NewDHCPHandler(stack)

	if handler == nil {
		t.Fatal("Expected DHCP handler, got nil")
	}

	if handler.DHCPHandlerStack() != stack {
		t.Error("Stack not set correctly")
	}

	if handler.DHCPHandlerLeases() == nil {
		t.Error("Leases map not initialized")
	}

	if handler.DHCPHandlerIPPool() == nil {
		t.Error("IP pool not initialized")
	}
	// Check default subnet mask
	expectedMask := net.IPv4(255, 255, 255, 0)
	if !handler.DHCPHandlerSubnetMask().Equal(expectedMask) {
		t.Errorf("Expected default subnet mask %v, got %v", expectedMask, handler.DHCPHandlerSubnetMask())
	}
}

// TestSetServerConfig tests setting server configuration.
func TestSetServerConfig(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewDHCPHandler(stack)

	serverIP := net.ParseIP("192.168.1.1")
	gateway := net.ParseIP("192.168.1.1")
	dnsServers := []net.IP{net.ParseIP("8.8.8.8"), net.ParseIP("8.8.4.4")}
	domain := "example.com"

	handler.SetServerConfig(serverIP, gateway, dnsServers, domain)

	if !handler.DHCPHandlerServerIP().Equal(serverIP) {
		t.Errorf("Expected server IP %v, got %v", serverIP, handler.DHCPHandlerServerIP())
	}

	if !handler.DHCPHandlerGateway().Equal(gateway) {
		t.Errorf("Expected gateway %v, got %v", gateway, handler.DHCPHandlerGateway())
	}

	if len(handler.DHCPHandlerDNSServers()) != 2 {
		t.Errorf("Expected 2 DNS servers, got %d", len(handler.DHCPHandlerDNSServers()))
	}

	if handler.DHCPHandlerDomainName() != domain {
		t.Errorf("Expected domain %s, got %s", domain, handler.DHCPHandlerDomainName())
	}
}

// TestSetAdvancedOptions tests setting advanced DHCP options.
func TestSetAdvancedOptions(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewDHCPHandler(stack)

	ntpServers := []net.IP{net.ParseIP("192.168.1.123")}
	domainSearch := []string{"example.com", "test.com"}
	tftpServer := "tftp.example.com"
	bootfile := "/pxelinux.0"
	vendorInfo := []byte("VendorData")

	handler.SetAdvancedOptions(ntpServers, domainSearch, tftpServer, bootfile, vendorInfo)

	if len(handler.DHCPHandlerNTPServers()) != 1 {
		t.Errorf("Expected 1 NTP server, got %d", len(handler.DHCPHandlerNTPServers()))
	}

	if len(handler.DHCPHandlerDomainSearch()) != 2 {
		t.Errorf("Expected 2 domain search entries, got %d", len(handler.DHCPHandlerDomainSearch()))
	}

	if handler.DHCPHandlerTFTPServerName() != tftpServer {
		t.Errorf("Expected TFTP server %s, got %s", tftpServer, handler.DHCPHandlerTFTPServerName())
	}

	if handler.DHCPHandlerBootfileName() != bootfile {
		t.Errorf("Expected bootfile %s, got %s", bootfile, handler.DHCPHandlerBootfileName())
	}
}

// TestSetPool tests setting the DHCP IP pool.
func TestSetPool(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewDHCPHandler(stack)

	start := net.ParseIP("192.168.1.100")
	end := net.ParseIP("192.168.1.200")

	handler.SetPool(start, end)

	if !handler.DHCPHandlerPoolStart().Equal(start) {
		t.Errorf("Expected pool start %v, got %v", start, handler.DHCPHandlerPoolStart())
	}

	if !handler.DHCPHandlerPoolEnd().Equal(end) {
		t.Errorf("Expected pool end %v, got %v", end, handler.DHCPHandlerPoolEnd())
	}

	// Pool should have 101 IPs (100-200 inclusive)
	expectedSize := 101
	if len(handler.DHCPHandlerIPPool()) != expectedSize {
		t.Errorf("Expected pool size %d, got %d", expectedSize, len(handler.DHCPHandlerIPPool()))
	}

	// Check first and last IPs in pool
	pool := handler.DHCPHandlerIPPool()
	if !pool[0].Equal(start) {
		t.Errorf("Expected first IP %v, got %v", start, pool[0])
	}

	if !pool[len(pool)-1].Equal(end) {
		t.Errorf("Expected last IP %v, got %v", end, pool[len(pool)-1])
	}
}

// TestGenerateIPPool tests IP pool generation.
func TestGenerateIPPool(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewDHCPHandler(stack)

	tests := []struct {
		name          string
		start         net.IP
		end           net.IP
		expectedCount int
	}{
		{
			name:          "Small pool",
			start:         net.ParseIP("10.0.0.1"),
			end:           net.ParseIP("10.0.0.10"),
			expectedCount: 10,
		},
		{
			name:          "Single IP",
			start:         net.ParseIP("172.16.0.1"),
			end:           net.ParseIP("172.16.0.1"),
			expectedCount: 1,
		},
		{
			name:          "Large pool",
			start:         net.ParseIP("192.168.0.1"),
			end:           net.ParseIP("192.168.0.254"),
			expectedCount: 254,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool, err := handler.GenerateIPPool(tt.start, tt.end)
			if err != nil {
				t.Fatalf("Failed to generate IP pool: %v", err)
			}

			if len(pool) != tt.expectedCount {
				t.Errorf("Expected %d IPs, got %d", tt.expectedCount, len(pool))
			}
			// Verify first and last IPs
			if len(pool) > 0 && !pool[0].Equal(tt.start) {
				t.Errorf("Expected first IP %v, got %v", tt.start, pool[0])
			}

			if len(pool) > 0 && !pool[len(pool)-1].Equal(tt.end) {
				t.Errorf("Expected last IP %v, got %v", tt.end, pool[len(pool)-1])
			}
		})
	}
}

// TestFindAvailableIP tests finding available IPs.
func TestFindAvailableIP(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewDHCPHandler(stack)

	start := net.ParseIP("192.168.1.10")
	end := net.ParseIP("192.168.1.12")
	handler.SetPool(start, end)

	// First call should return first IP
	ip1 := handler.FindAvailableIP()
	if !ip1.Equal(start) {
		t.Errorf("Expected first available IP %v, got %v", start, ip1)
	}

	// Allocate first IP
	mac1 := net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}

	_, err := handler.AllocateLease(mac1, nil, "")
	if err != nil {
		t.Fatalf("Failed to allocate lease: %v", err)
	}

	// Next call should return second IP
	ip2 := handler.FindAvailableIP()

	expectedIP2 := net.ParseIP("192.168.1.11")

	if !ip2.Equal(expectedIP2) {
		t.Errorf("Expected next available IP %v, got %v", expectedIP2, ip2)
	}
}

// TestAllocateLease tests lease allocation.
func TestAllocateLease(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewDHCPHandler(stack)

	start := net.ParseIP("192.168.1.100")
	end := net.ParseIP("192.168.1.200")
	handler.SetPool(start, end)

	mac := net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	hostname := "test-host"

	// Allocate new lease
	lease, err := handler.AllocateLease(mac, nil, hostname)
	if err != nil {
		t.Fatalf("Failed to allocate lease: %v", err)
	}

	if lease == nil {
		t.Fatal("Expected lease, got nil")
	}

	if lease.MAC.String() != mac.String() {
		t.Errorf("Expected MAC %v, got %v", mac, lease.MAC)
	}

	if lease.Hostname != hostname {
		t.Errorf("Expected hostname %s, got %s", hostname, lease.Hostname)
	}

	if lease.LeaseTime != protocols.DefaultLeaseTime {
		t.Errorf("Expected lease time %v, got %v", protocols.DefaultLeaseTime, lease.LeaseTime)
	}

	if time.Until(lease.Expiry) > protocols.DefaultLeaseTime {
		t.Error("Lease expiry is too far in the future")
	}
}

// TestAllocateLease_Renewal tests lease renewal.
func TestAllocateLease_Renewal(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewDHCPHandler(stack)

	start := net.ParseIP("192.168.1.100")
	end := net.ParseIP("192.168.1.200")
	handler.SetPool(start, end)

	mac := net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}

	// First allocation
	lease1, err := handler.AllocateLease(mac, nil, "host1")
	if err != nil {
		t.Fatalf("Failed to allocate lease: %v", err)
	}

	originalIP := lease1.IP
	originalExpiry := lease1.Expiry

	// Wait a bit to ensure time has passed
	time.Sleep(10 * time.Millisecond)

	// Renew lease (same MAC)
	lease2, err := handler.AllocateLease(mac, nil, "host2")
	if err != nil {
		t.Fatalf("Failed to renew lease: %v", err)
	}

	// Should be same lease object
	if lease1 != lease2 {
		t.Error("Expected same lease object for renewal")
	}

	// IP should remain the same
	if !lease2.IP.Equal(originalIP) {
		t.Errorf("IP changed during renewal: %v -> %v", originalIP, lease2.IP)
	}

	// Hostname should be updated
	if lease2.Hostname != "host2" {
		t.Errorf("Expected updated hostname 'host2', got '%s'", lease2.Hostname)
	}

	// Expiry should be extended
	if !lease2.Expiry.After(originalExpiry) {
		t.Error("Expiry was not extended during renewal")
	}
}

// TestAllocateLease_RequestedIP tests allocating a specific requested IP.
func TestAllocateLease_RequestedIP(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewDHCPHandler(stack)

	start := net.ParseIP("192.168.1.100")
	end := net.ParseIP("192.168.1.200")
	handler.SetPool(start, end)

	mac := net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	requestedIP := net.ParseIP("192.168.1.150")

	// Allocate with requested IP
	lease, err := handler.AllocateLease(mac, requestedIP, "")
	if err != nil {
		t.Fatalf("Failed to allocate lease: %v", err)
	}

	if !lease.IP.Equal(requestedIP) {
		t.Errorf("Expected requested IP %v, got %v", requestedIP, lease.IP)
	}
}

// TestAllocateLease_RequestedIPOutOfPool tests requesting IP outside pool.
func TestAllocateLease_RequestedIPOutOfPool(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewDHCPHandler(stack)

	start := net.ParseIP("192.168.1.100")
	end := net.ParseIP("192.168.1.200")
	handler.SetPool(start, end)

	mac := net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	requestedIP := net.ParseIP("192.168.1.50") // Outside pool

	// Should allocate from pool instead
	lease, err := handler.AllocateLease(mac, requestedIP, "")
	if err != nil {
		t.Fatalf("Failed to allocate lease: %v", err)
	}

	if lease.IP.Equal(requestedIP) {
		t.Error("Should not allocate IP outside of pool")
	}

	if !lease.IP.Equal(start) {
		t.Errorf("Expected first pool IP %v, got %v", start, lease.IP)
	}
}

// TestAllocateLease_PoolExhaustion tests behavior when pool is exhausted.
func TestAllocateLease_PoolExhaustion(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewDHCPHandler(stack)

	// Small pool with only 2 IPs
	start := net.ParseIP("192.168.1.10")
	end := net.ParseIP("192.168.1.11")
	handler.SetPool(start, end)

	// Allocate all IPs
	mac1 := net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x01}
	mac2 := net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x02}

	_, err := handler.AllocateLease(mac1, nil, "")
	if err != nil {
		t.Fatalf("Failed to allocate first lease: %v", err)
	}

	_, err = handler.AllocateLease(mac2, nil, "")
	if err != nil {
		t.Fatalf("Failed to allocate second lease: %v", err)
	}

	// Try to allocate one more - should fail
	mac3 := net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x03}

	_, err = handler.AllocateLease(mac3, nil, "")
	if err == nil {
		t.Error("Expected error when pool is exhausted, got nil")
	}
}

// TestIsIPInPool tests IP pool membership checking.
func TestIsIPInPool(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewDHCPHandler(stack)

	start := net.ParseIP("192.168.1.100")
	end := net.ParseIP("192.168.1.200")
	handler.SetPool(start, end)

	tests := []struct {
		name     string
		ip       net.IP
		expected bool
	}{
		{"First IP in pool", net.ParseIP("192.168.1.100"), true},
		{"Last IP in pool", net.ParseIP("192.168.1.200"), true},
		{"Middle IP in pool", net.ParseIP("192.168.1.150"), true},
		{"IP before pool", net.ParseIP("192.168.1.99"), false},
		{"IP after pool", net.ParseIP("192.168.1.201"), false},
		{"Different subnet", net.ParseIP("10.0.0.100"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.IsIPInPool(tt.ip)
			if result != tt.expected {
				t.Errorf("isIPInPool(%v) = %v, expected %v", tt.ip, result, tt.expected)
			}
		})
	}
}

// TestDHCPConstants tests DHCP message type constants.
func TestDHCPConstants(t *testing.T) {
	// Verify DHCP message type constants
	if protocols.DHCPDiscover != 1 {
		t.Errorf("DHCPDiscover should be 1, got %d", protocols.DHCPDiscover)
	}

	if protocols.DHCPOffer != 2 {
		t.Errorf("DHCPOffer should be 2, got %d", protocols.DHCPOffer)
	}

	if protocols.DHCPRequest != 3 {
		t.Errorf("DHCPRequest should be 3, got %d", protocols.DHCPRequest)
	}

	if protocols.DHCPDecline != 4 {
		t.Errorf("DHCPDecline should be 4, got %d", protocols.DHCPDecline)
	}

	if protocols.DHCPAck != 5 {
		t.Errorf("DHCPAck should be 5, got %d", protocols.DHCPAck)
	}

	if protocols.DHCPNak != 6 {
		t.Errorf("DHCPNak should be 6, got %d", protocols.DHCPNak)
	}

	if protocols.DHCPRelease != 7 {
		t.Errorf("DHCPRelease should be 7, got %d", protocols.DHCPRelease)
	}

	if protocols.DHCPInform != 8 {
		t.Errorf("DHCPInform should be 8, got %d", protocols.DHCPInform)
	}
}

// TestDefaultLeaseTime tests the default lease time constant.
func TestDefaultLeaseTime(t *testing.T) {
	expectedDuration := 24 * time.Hour
	if protocols.DefaultLeaseTime != expectedDuration {
		t.Errorf("DefaultLeaseTime should be %v, got %v", expectedDuration, protocols.DefaultLeaseTime)
	}
}

// BenchmarkAllocateLease benchmarks lease allocation.
func BenchmarkAllocateLease(b *testing.B) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewDHCPHandler(stack)

	start := net.ParseIP("192.168.1.1")
	end := net.ParseIP("192.168.1.254")
	handler.SetPool(start, end)

	b.ResetTimer()

	for i := range b.N {
		mac := net.HardwareAddr{0x00, 0x11, 0x22, 0x33, byte(i >> 8), byte(i & 0xFF)}
		_, _ = handler.AllocateLease(mac, nil, "")
	}
}

// BenchmarkFindAvailableIP benchmarks finding available IPs.
func BenchmarkFindAvailableIP(b *testing.B) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewDHCPHandler(stack)

	start := net.ParseIP("192.168.1.1")
	end := net.ParseIP("192.168.1.254")
	handler.SetPool(start, end)

	// Allocate half the pool
	for i := range 127 {
		mac := net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, byte(i)}
		_, _ = handler.AllocateLease(mac, nil, "")
	}

	for b.Loop() {
		handler.FindAvailableIP()
	}
}

// NOTE: HandlePacket tests for DHCP are complex due to channel infrastructure
// and asynchronous packet sending. Current test coverage focuses on:
// - Configuration methods (SetPool, SetServerConfig, SetAdvancedOptions)
// - Lease management (allocateLease, findAvailableIP, isIPInPool)
// - Pool generation and validation
// - All DHCP message type constants
// This provides good coverage of the core DHCP logic without the complexity
// of full packet flow testing.
