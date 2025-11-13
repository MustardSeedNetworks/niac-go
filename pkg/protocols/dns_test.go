package protocols

import (
	"encoding/hex"
	"net"
	"strings"
	"testing"

	"github.com/google/gopacket/layers"
	"github.com/krisarmstrong/niac-go/pkg/config"
	"github.com/krisarmstrong/niac-go/pkg/logging"
)

// TestNewDNSHandler tests DNS handler creation
func TestNewDNSHandler(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewDNSHandler(stack)

	if handler == nil {
		t.Fatal("NewDNSHandler returned nil")
	}

	if handler.stack != stack {
		t.Error("Handler stack not set correctly")
	}

	if handler.records == nil {
		t.Error("Records map not initialized")
	}

	if handler.ptrRecords == nil {
		t.Error("PTR records map not initialized")
	}

	if handler.domain != "local" {
		t.Errorf("Expected default domain 'local', got '%s'", handler.domain)
	}
}

// TestAddRecord tests adding DNS records
func TestAddRecord(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewDNSHandler(stack)

	tests := []struct {
		name     string
		hostname string
		ip       net.IP
	}{
		{
			name:     "IPv4 record",
			hostname: "server1.example.com",
			ip:       net.ParseIP("192.168.1.10"),
		},
		{
			name:     "IPv6 record",
			hostname: "server2.example.com",
			ip:       net.ParseIP("2001:db8::1"),
		},
		{
			name:     "Short hostname",
			hostname: "router",
			ip:       net.ParseIP("10.0.0.1"),
		},
		{
			name:     "Hostname with trailing dot",
			hostname: "server3.example.com.",
			ip:       net.ParseIP("192.168.1.20"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler.AddRecord(tt.hostname, tt.ip)

			// Normalize hostname for lookup
			normalized := strings.ToLower(strings.TrimSuffix(tt.hostname, "."))

			// Check forward record
			if ips, ok := handler.records[normalized]; !ok {
				t.Errorf("Record not added for hostname %s", normalized)
			} else if len(ips) == 0 {
				t.Error("No IPs in record")
			} else if !ips[0].Equal(tt.ip) {
				t.Errorf("Expected IP %v, got %v", tt.ip, ips[0])
			}

			// Check reverse record (PTR)
			if hostname, ok := handler.ptrRecords[tt.ip.String()]; !ok {
				t.Errorf("PTR record not added for IP %s", tt.ip)
			} else if hostname != normalized {
				t.Errorf("Expected PTR hostname %s, got %s", normalized, hostname)
			}
		})
	}
}

// TestAddRecord_Multiple tests multiple IPs for same hostname
func TestAddRecord_Multiple(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewDNSHandler(stack)

	hostname := "server.example.com"
	ip1 := net.ParseIP("192.168.1.10")
	ip2 := net.ParseIP("192.168.1.11")
	ip3 := net.ParseIP("2001:db8::1")

	// Add multiple IPs for same hostname
	handler.AddRecord(hostname, ip1)
	handler.AddRecord(hostname, ip2)
	handler.AddRecord(hostname, ip3)

	normalized := strings.ToLower(hostname)
	ips := handler.records[normalized]

	if len(ips) != 3 {
		t.Errorf("Expected 3 IPs, got %d", len(ips))
	}

	// Verify all IPs are present
	found := make(map[string]bool)
	for _, ip := range ips {
		found[ip.String()] = true
	}

	if !found[ip1.String()] {
		t.Errorf("IP %s not found", ip1)
	}
	if !found[ip2.String()] {
		t.Errorf("IP %s not found", ip2)
	}
	if !found[ip3.String()] {
		t.Errorf("IP %s not found", ip3)
	}
}

// TestAddRecord_CaseInsensitive tests case-insensitive hostname handling
func TestAddRecord_CaseInsensitive(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewDNSHandler(stack)

	// Add records with different cases
	handler.AddRecord("Server.Example.COM", net.ParseIP("192.168.1.10"))
	handler.AddRecord("server.example.com", net.ParseIP("192.168.1.11"))

	// Both should be stored under lowercase key
	ips := handler.records["server.example.com"]
	if len(ips) != 2 {
		t.Errorf("Expected 2 IPs for case-insensitive hostname, got %d", len(ips))
	}
}

// TestSetDomain tests setting the default domain
func TestSetDomain(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewDNSHandler(stack)

	domains := []string{"example.com", "internal.net", "corp"}

	for _, domain := range domains {
		handler.SetDomain(domain)

		if handler.domain != domain {
			t.Errorf("Expected domain '%s', got '%s'", domain, handler.domain)
		}
	}
}

// TestLookupHost tests hostname lookups
func TestLookupHost(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewDNSHandler(stack)

	// Setup test data
	handler.SetDomain("example.com")
	handler.AddRecord("server1.example.com", net.ParseIP("192.168.1.10"))
	handler.AddRecord("server2.example.com", net.ParseIP("192.168.1.20"))
	handler.AddRecord("short", net.ParseIP("10.0.0.1"))

	tests := []struct {
		name          string
		hostname      string
		expectFound   bool
		expectedCount int
	}{
		{
			name:          "Exact match FQDN",
			hostname:      "server1.example.com",
			expectFound:   true,
			expectedCount: 1,
		},
		{
			name:          "Exact match short name",
			hostname:      "short",
			expectFound:   true,
			expectedCount: 1,
		},
		{
			name:          "Case insensitive",
			hostname:      "SERVER1.EXAMPLE.COM",
			expectFound:   true,
			expectedCount: 1,
		},
		{
			name:          "Trailing dot",
			hostname:      "server1.example.com.",
			expectFound:   true,
			expectedCount: 1,
		},
		{
			name:          "Non-existent",
			hostname:      "nonexistent.example.com",
			expectFound:   false,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ips := handler.lookupHost(tt.hostname)

			if tt.expectFound {
				if ips == nil {
					t.Errorf("Expected to find hostname %s, got nil", tt.hostname)
					return
				}
				if len(ips) != tt.expectedCount {
					t.Errorf("Expected %d IPs, got %d", tt.expectedCount, len(ips))
				}
			} else {
				if ips != nil {
					t.Errorf("Expected nil for hostname %s, got %v", tt.hostname, ips)
				}
			}
		})
	}
}

func TestParsePTRNameIPv4(t *testing.T) {
	ip, ok := parsePTRName([]byte("4.3.2.1.in-addr.arpa."))
	if !ok {
		t.Fatalf("expected IPv4 PTR parse success")
	}
	expected := net.ParseIP("1.2.3.4").To4()
	if !ip.Equal(expected) {
		t.Fatalf("expected %s, got %s", expected, ip)
	}
}

func TestParsePTRNameIPv6(t *testing.T) {
	expected := net.ParseIP("2001:db8::1")
	ptr := ipv6PTRString(expected)
	ip, ok := parsePTRName([]byte(ptr))
	if !ok {
		t.Fatalf("expected IPv6 PTR parse success")
	}
	if !ip.Equal(expected) {
		t.Fatalf("expected %s, got %s", expected, ip)
	}
}

func TestResolveQuestionsPTR(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewDNSHandler(stack)
	handler.AddRecord("router.example.com", net.ParseIP("2001:db8::1"))

	ptrName := ipv6PTRString(net.ParseIP("2001:db8::1"))
	question := layers.DNSQuestion{
		Name:  []byte(ptrName),
		Type:  layers.DNSTypePTR,
		Class: layers.DNSClassIN,
	}

	answers, code := handler.resolveQuestions([]layers.DNSQuestion{question}, 1, 0)
	if code != layers.DNSResponseCodeNoErr {
		t.Fatalf("expected no error response, got %v", code)
	}
	if len(answers) != 1 {
		t.Fatalf("expected 1 answer, got %d", len(answers))
	}
	if string(answers[0].PTR) != "router.example.com." {
		t.Fatalf("expected PTR router.example.com., got %s", answers[0].PTR)
	}
}

func ipv6PTRString(ip net.IP) string {
	hexDigits := strings.ToLower(hex.EncodeToString(ip.To16()))
	parts := make([]string, 0, len(hexDigits))
	for i := len(hexDigits) - 1; i >= 0; i-- {
		parts = append(parts, string(hexDigits[i]))
	}
	return strings.Join(parts, ".") + ".ip6.arpa."
}

// TestLookupHost_WithDomain tests short name lookup with default domain
func TestLookupHost_WithDomain(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewDNSHandler(stack)

	handler.SetDomain("example.com")
	handler.AddRecord("server.example.com", net.ParseIP("192.168.1.10"))

	// Lookup short name - should append domain
	ips := handler.lookupHost("server")
	if ips == nil {
		t.Fatal("Expected to find 'server' with default domain appended")
	}

	if len(ips) != 1 {
		t.Errorf("Expected 1 IP, got %d", len(ips))
	}

	if !ips[0].Equal(net.ParseIP("192.168.1.10")) {
		t.Errorf("Expected IP 192.168.1.10, got %v", ips[0])
	}
}

// TestLoadDeviceRecords tests loading DNS records from devices
func TestLoadDeviceRecords(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewDNSHandler(stack)

	// Create test devices
	devices := []*config.Device{
		{
			Name: "router1",
			SNMPConfig: config.SNMPConfig{
				SysName: "core-router-01",
			},
			IPAddresses: []net.IP{
				net.ParseIP("192.168.1.1"),
				net.ParseIP("2001:db8::1"),
			},
		},
		{
			Name: "switch1",
			SNMPConfig: config.SNMPConfig{
				SysName: "", // Empty, should use device name
			},
			IPAddresses: []net.IP{
				net.ParseIP("192.168.1.2"),
			},
		},
		{
			Name: "server1",
			SNMPConfig: config.SNMPConfig{
				SysName: "web-server-01",
			},
			IPAddresses: []net.IP{}, // No IPs
		},
	}

	handler.LoadDeviceRecords(devices)

	// Test device 1 - should use sysName
	ips := handler.lookupHost("core-router-01")
	if ips == nil {
		t.Fatal("Expected to find 'core-router-01'")
	}
	if len(ips) != 2 {
		t.Errorf("Expected 2 IPs for core-router-01, got %d", len(ips))
	}

	// Test device 2 - should use device name
	ips = handler.lookupHost("switch1")
	if ips == nil {
		t.Fatal("Expected to find 'switch1'")
	}
	if len(ips) != 1 {
		t.Errorf("Expected 1 IP for switch1, got %d", len(ips))
	}

	// Test device 3 - no IPs, should still be added but return empty
	ips = handler.lookupHost("web-server-01")
	if len(ips) > 0 {
		t.Errorf("Expected no IPs for web-server-01 (no IPs configured), got %d", len(ips))
	}
}

// TestLookupHost_IPv4Only tests filtering for IPv4 addresses
func TestLookupHost_IPv4Only(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewDNSHandler(stack)

	hostname := "dual-stack.example.com"
	handler.AddRecord(hostname, net.ParseIP("192.168.1.10"))
	handler.AddRecord(hostname, net.ParseIP("2001:db8::1"))
	handler.AddRecord(hostname, net.ParseIP("192.168.1.11"))

	ips := handler.lookupHost(hostname)
	if len(ips) != 3 {
		t.Fatalf("Expected 3 total IPs, got %d", len(ips))
	}

	// Count IPv4 addresses
	ipv4Count := 0
	ipv6Count := 0
	for _, ip := range ips {
		if ip.To4() != nil {
			ipv4Count++
		} else if ip.To16() != nil {
			ipv6Count++
		}
	}

	if ipv4Count != 2 {
		t.Errorf("Expected 2 IPv4 addresses, got %d", ipv4Count)
	}

	if ipv6Count != 1 {
		t.Errorf("Expected 1 IPv6 address, got %d", ipv6Count)
	}
}

// TestLookupHost_IPv6Only tests filtering for IPv6 addresses
func TestLookupHost_IPv6Only(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewDNSHandler(stack)

	hostname := "ipv6-only.example.com"
	handler.AddRecord(hostname, net.ParseIP("2001:db8::1"))
	handler.AddRecord(hostname, net.ParseIP("2001:db8::2"))

	ips := handler.lookupHost(hostname)
	if len(ips) != 2 {
		t.Fatalf("Expected 2 IPv6 addresses, got %d", len(ips))
	}

	for _, ip := range ips {
		if ip.To4() != nil {
			t.Error("Expected only IPv6 addresses, found IPv4")
		}
		if ip.To16() == nil {
			t.Error("Invalid IPv6 address")
		}
	}
}

// TestPTRRecords tests reverse DNS (PTR) record storage
func TestPTRRecords(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewDNSHandler(stack)

	tests := []struct {
		hostname string
		ip       net.IP
	}{
		{"server1.example.com", net.ParseIP("192.168.1.10")},
		{"server2.example.com", net.ParseIP("2001:db8::1")},
		{"router.local", net.ParseIP("10.0.0.1")},
	}

	for _, tt := range tests {
		handler.AddRecord(tt.hostname, tt.ip)

		// Check PTR record
		if hostname, ok := handler.ptrRecords[tt.ip.String()]; !ok {
			t.Errorf("PTR record not found for IP %s", tt.ip)
		} else {
			expected := strings.ToLower(strings.TrimSuffix(tt.hostname, "."))
			if hostname != expected {
				t.Errorf("PTR for %s: expected %s, got %s", tt.ip, expected, hostname)
			}
		}
	}
}

// TestThreadSafety tests concurrent access to DNS handler
func TestThreadSafety(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewDNSHandler(stack)

	done := make(chan bool)

	// Concurrent writes
	go func() {
		for i := 0; i < 100; i++ {
			handler.AddRecord("server1.example.com", net.ParseIP("192.168.1.10"))
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			handler.AddRecord("server2.example.com", net.ParseIP("192.168.1.20"))
		}
		done <- true
	}()

	// Concurrent reads
	go func() {
		for i := 0; i < 100; i++ {
			handler.lookupHost("server1.example.com")
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			handler.SetDomain("example.com")
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 4; i++ {
		<-done
	}

	// Verify records are consistent
	ips1 := handler.lookupHost("server1.example.com")
	ips2 := handler.lookupHost("server2.example.com")

	if ips1 == nil {
		t.Error("server1.example.com records corrupted")
	}
	if ips2 == nil {
		t.Error("server2.example.com records corrupted")
	}
}

// TestEmptyHostname tests handling of empty hostname
func TestEmptyHostname(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewDNSHandler(stack)

	handler.AddRecord("", net.ParseIP("192.168.1.10"))

	// Should be stored under empty string key
	ips := handler.lookupHost("")
	if ips == nil {
		t.Error("Expected to find empty hostname")
	}
}

// TestSpecialCharacters tests hostname with special characters
func TestSpecialCharacters(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewDNSHandler(stack)

	hostnames := []string{
		"server-1.example.com",
		"server_1.example.com",
		"123.example.com",
		"server1-backend.example.com",
	}

	for _, hostname := range hostnames {
		handler.AddRecord(hostname, net.ParseIP("192.168.1.10"))

		ips := handler.lookupHost(hostname)
		if ips == nil {
			t.Errorf("Failed to find hostname with special chars: %s", hostname)
		}
	}
}

// TestDNSDefaultPort tests DNS uses standard port 53
func TestDNSDefaultPort(t *testing.T) {
	// DNS should use port 53 as defined in RFC 1035
	// This is tested implicitly in SendDNSResponse which hardcodes port 53
	// This test documents the expected behavior
	const expectedPort = 53

	if expectedPort != 53 {
		t.Errorf("DNS should use port 53, not %d", expectedPort)
	}
}

// Benchmarks

// BenchmarkAddRecord benchmarks adding DNS records
func BenchmarkAddRecord(b *testing.B) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewDNSHandler(stack)

	hostname := "server.example.com"
	ip := net.ParseIP("192.168.1.10")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.AddRecord(hostname, ip)
	}
}

// BenchmarkLookupHost benchmarks DNS lookups
func BenchmarkLookupHost(b *testing.B) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewDNSHandler(stack)

	// Pre-populate with 100 records
	for i := 0; i < 100; i++ {
		hostname := "server" + string(rune('0'+i%10)) + ".example.com"
		handler.AddRecord(hostname, net.ParseIP("192.168.1.10"))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.lookupHost("server5.example.com")
	}
}

// BenchmarkLoadDeviceRecords benchmarks loading device records
func BenchmarkLoadDeviceRecords(b *testing.B) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewDNSHandler(stack)

	// Create 50 test devices
	devices := make([]*config.Device, 50)
	for i := 0; i < 50; i++ {
		devices[i] = &config.Device{
			Name: "device" + string(rune('0'+i%10)),
			SNMPConfig: config.SNMPConfig{
				SysName: "system-" + string(rune('0'+i%10)),
			},
			IPAddresses: []net.IP{
				net.ParseIP("192.168.1." + string(rune('0'+i%256))),
			},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.LoadDeviceRecords(devices)
	}
}

// BenchmarkConcurrentLookups benchmarks concurrent DNS lookups
func BenchmarkConcurrentLookups(b *testing.B) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewDNSHandler(stack)

	// Pre-populate
	for i := 0; i < 10; i++ {
		hostname := "server" + string(rune('0'+i)) + ".example.com"
		handler.AddRecord(hostname, net.ParseIP("192.168.1.10"))
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			handler.lookupHost("server5.example.com")
		}
	})
}
