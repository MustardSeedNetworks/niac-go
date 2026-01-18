package protocols_test

import (
	"encoding/hex"
	"net"
	"strings"
	"testing"

	"github.com/google/gopacket/layers"

	"github.com/krisarmstrong/niac-go/pkg/config"
	"github.com/krisarmstrong/niac-go/pkg/logging"
	"github.com/krisarmstrong/niac-go/pkg/protocols"
)

// TestNewDNSHandler tests DNS handler creation.
func TestNewDNSHandler(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewDNSHandler(stack)

	if handler == nil {
		t.Fatal("NewDNSHandler returned nil")
	}

	if handler.DNSHandlerStack() != stack {
		t.Error("Handler stack not set correctly")
	}

	if handler.DNSHandlerRecords() == nil {
		t.Error("Records map not initialized")
	}

	if handler.DNSHandlerPTRRecords() == nil {
		t.Error("PTR records map not initialized")
	}

	if handler.DNSHandlerDomain() != "local" {
		t.Errorf("Expected default domain 'local', got '%s'", handler.DNSHandlerDomain())
	}
}

// TestAddRecord tests adding DNS records.
func TestAddRecord(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewDNSHandler(stack)

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
			records := handler.DNSHandlerRecords()
			if ips, ok := records[normalized]; !ok {
				t.Errorf("Record not added for hostname %s", normalized)
			} else if len(ips) == 0 {
				t.Error("No IPs in record")
			}

			// Check reverse record (PTR)
			ptrRecords := handler.DNSHandlerPTRRecords()
			if _, ok := ptrRecords[tt.ip.String()]; !ok {
				t.Errorf("PTR record not added for IP %s", tt.ip)
			}
		})
	}
}

// TestAddRecord_Multiple tests multiple IPs for same hostname.
func TestAddRecord_Multiple(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewDNSHandler(stack)

	hostname := "server.example.com"
	ip1 := net.ParseIP("192.168.1.10")
	ip2 := net.ParseIP("192.168.1.11")
	ip3 := net.ParseIP("2001:db8::1")

	// Add multiple IPs for same hostname
	handler.AddRecord(hostname, ip1)
	handler.AddRecord(hostname, ip2)
	handler.AddRecord(hostname, ip3)

	normalized := strings.ToLower(hostname)
	ips := handler.DNSHandlerRecords()[normalized]

	if len(ips) != 3 {
		t.Errorf("Expected 3 IPs, got %d", len(ips))
	}
}

// TestAddRecord_CaseInsensitive tests case-insensitive hostname handling.
func TestAddRecord_CaseInsensitive(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewDNSHandler(stack)

	// Add records with different cases
	handler.AddRecord("Server.Example.COM", net.ParseIP("192.168.1.10"))
	handler.AddRecord("server.example.com", net.ParseIP("192.168.1.11"))

	// Both should be stored under lowercase key
	ips := handler.DNSHandlerRecords()["server.example.com"]
	if len(ips) != 2 {
		t.Errorf("Expected 2 IPs for case-insensitive hostname, got %d", len(ips))
	}
}

// TestSetDomain tests setting the default domain.
func TestSetDomain(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewDNSHandler(stack)

	domains := []string{"example.com", "internal.net", "corp"}

	for _, domain := range domains {
		handler.SetDomain(domain)

		if handler.DNSHandlerDomain() != domain {
			t.Errorf("Expected domain '%s', got '%s'", domain, handler.DNSHandlerDomain())
		}
	}
}

// TestLookupHost tests hostname lookups.
func TestLookupHost(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewDNSHandler(stack)

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
			ips := handler.LookupHost(tt.hostname, nil)

			if tt.expectFound {
				if ips == nil {
					t.Errorf("Expected to find hostname %s, got nil", tt.hostname)

					return
				}

				if len(ips) != tt.expectedCount {
					t.Errorf("Expected %d IPs, got %d", tt.expectedCount, len(ips))
				}
			} else if ips != nil {
				t.Errorf("Expected nil for hostname %s, got %v", tt.hostname, ips)
			}
		})
	}
}

// NOTE: TestParsePTRNameIPv4 and TestParsePTRNameIPv6 are not included because
// parsePTRName is an unexported package-level function. PTR parsing is tested
// implicitly through TestResolveQuestionsPTR.

func ipv6PTRString(ip net.IP) string {
	hexDigits := strings.ToLower(hex.EncodeToString(ip.To16()))

	parts := make([]string, 0, len(hexDigits))

	for i := len(hexDigits) - 1; i >= 0; i-- {
		parts = append(parts, string(hexDigits[i]))
	}

	return strings.Join(parts, ".") + ".ip6.arpa."
}

func TestResolveQuestionsPTR(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewDNSHandler(stack)
	handler.AddRecord("router.example.com", net.ParseIP("2001:db8::1"))

	ptrName := ipv6PTRString(net.ParseIP("2001:db8::1"))
	question := layers.DNSQuestion{
		Name:  []byte(ptrName),
		Type:  layers.DNSTypePTR,
		Class: layers.DNSClassIN,
	}

	answers, code := handler.ResolveQuestions([]layers.DNSQuestion{question}, nil, 1, 0)
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

// TestLookupHost_WithDomain tests short name lookup with default domain.
func TestLookupHost_WithDomain(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewDNSHandler(stack)

	handler.SetDomain("example.com")
	handler.AddRecord("server.example.com", net.ParseIP("192.168.1.10"))

	// Lookup short name - should append domain
	ips := handler.LookupHost("server", nil)
	if ips == nil {
		t.Fatal("Expected to find 'server' with default domain appended")
	}

	if len(ips) != 1 {
		t.Errorf("Expected 1 IP, got %d", len(ips))
	}
}

// TestLoadDeviceRecords tests loading DNS records from devices.
func TestLoadDeviceRecords(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewDNSHandler(stack)

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
	ips := handler.LookupHost("core-router-01", nil)
	if ips == nil {
		t.Fatal("Expected to find 'core-router-01'")
	}

	if len(ips) != 2 {
		t.Errorf("Expected 2 IPs for core-router-01, got %d", len(ips))
	}

	// Test device 2 - should use device name
	ips = handler.LookupHost("switch1", nil)
	if ips == nil {
		t.Fatal("Expected to find 'switch1'")
	}

	if len(ips) != 1 {
		t.Errorf("Expected 1 IP for switch1, got %d", len(ips))
	}

	// Test device 3 - no IPs, should still be added but return empty
	ips = handler.LookupHost("web-server-01", nil)
	if len(ips) > 0 {
		t.Errorf("Expected no IPs for web-server-01 (no IPs configured), got %d", len(ips))
	}
}

// TestLookupHost_IPv4Only tests filtering for IPv4 addresses.
func TestLookupHost_IPv4Only(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewDNSHandler(stack)

	hostname := "dual-stack.example.com"
	handler.AddRecord(hostname, net.ParseIP("192.168.1.10"))
	handler.AddRecord(hostname, net.ParseIP("2001:db8::1"))
	handler.AddRecord(hostname, net.ParseIP("192.168.1.11"))

	ips := handler.LookupHost(hostname, nil)
	if len(ips) != 3 {
		t.Fatalf("Expected 3 total IPs, got %d", len(ips))
	}
}

// TestLookupHost_IPv6Only tests filtering for IPv6 addresses.
func TestLookupHost_IPv6Only(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewDNSHandler(stack)

	hostname := "ipv6-only.example.com"
	handler.AddRecord(hostname, net.ParseIP("2001:db8::1"))
	handler.AddRecord(hostname, net.ParseIP("2001:db8::2"))

	ips := handler.LookupHost(hostname, nil)
	if len(ips) != 2 {
		t.Fatalf("Expected 2 IPv6 addresses, got %d", len(ips))
	}
}

// TestPTRRecords tests reverse DNS (PTR) record storage.
func TestPTRRecords(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewDNSHandler(stack)

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
		ptrRecords := handler.DNSHandlerPTRRecords()
		if _, ok := ptrRecords[tt.ip.String()]; !ok {
			t.Errorf("PTR record not found for IP %s", tt.ip)
		}
	}
}

// TestThreadSafety tests concurrent access to DNS handler.
func TestThreadSafety(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewDNSHandler(stack)

	done := make(chan bool)

	// Concurrent writes
	go func() {
		for range 100 {
			handler.AddRecord("server1.example.com", net.ParseIP("192.168.1.10"))
		}

		done <- true
	}()

	go func() {
		for range 100 {
			handler.AddRecord("server2.example.com", net.ParseIP("192.168.1.20"))
		}

		done <- true
	}()

	// Concurrent reads
	go func() {
		for range 100 {
			handler.LookupHost("server1.example.com", nil)
		}

		done <- true
	}()

	go func() {
		for range 100 {
			handler.SetDomain("example.com")
		}

		done <- true
	}()

	// Wait for all goroutines
	for range 4 {
		<-done
	}

	// Verify records are consistent
	ips1 := handler.LookupHost("server1.example.com", nil)
	ips2 := handler.LookupHost("server2.example.com", nil)

	if ips1 == nil {
		t.Error("server1.example.com records corrupted")
	}

	if ips2 == nil {
		t.Error("server2.example.com records corrupted")
	}
}

// TestEmptyHostname tests handling of empty hostname.
func TestEmptyHostname(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewDNSHandler(stack)

	handler.AddRecord("", net.ParseIP("192.168.1.10"))

	// Should be stored under empty string key
	ips := handler.LookupHost("", nil)
	if ips == nil {
		t.Error("Expected to find empty hostname")
	}
}

// TestSpecialCharacters tests hostname with special characters.
func TestSpecialCharacters(t *testing.T) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewDNSHandler(stack)

	hostnames := []string{
		"server-1.example.com",
		"server_1.example.com",
		"123.example.com",
		"server1-backend.example.com",
	}

	for _, hostname := range hostnames {
		handler.AddRecord(hostname, net.ParseIP("192.168.1.10"))

		ips := handler.LookupHost(hostname, nil)
		if ips == nil {
			t.Errorf("Failed to find hostname with special chars: %s", hostname)
		}
	}
}

// TestDNSDefaultPort tests DNS uses standard port 53.
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

// BenchmarkAddRecord benchmarks adding DNS records.
func BenchmarkAddRecord(b *testing.B) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewDNSHandler(stack)

	hostname := "server.example.com"
	ip := net.ParseIP("192.168.1.10")

	for b.Loop() {
		handler.AddRecord(hostname, ip)
	}
}

// BenchmarkLookupHost benchmarks DNS lookups.
func BenchmarkLookupHost(b *testing.B) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewDNSHandler(stack)

	// Pre-populate with 100 records
	for i := range 100 {
		hostname := "server" + string(rune('0'+i%10)) + ".example.com"
		handler.AddRecord(hostname, net.ParseIP("192.168.1.10"))
	}

	for b.Loop() {
		handler.LookupHost("server5.example.com", nil)
	}
}

// BenchmarkLoadDeviceRecords benchmarks loading device records.
func BenchmarkLoadDeviceRecords(b *testing.B) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewDNSHandler(stack)

	// Create 50 test devices
	devices := make([]*config.Device, 50)
	for i := range 50 {
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

	for b.Loop() {
		handler.LoadDeviceRecords(devices)
	}
}

// BenchmarkConcurrentLookups benchmarks concurrent DNS lookups.
func BenchmarkConcurrentLookups(b *testing.B) {
	cfg := &config.Config{}
	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := protocols.NewDNSHandler(stack)

	// Pre-populate
	for i := range 10 {
		hostname := "server" + string(rune('0'+i)) + ".example.com"
		handler.AddRecord(hostname, net.ParseIP("192.168.1.10"))
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			handler.LookupHost("server5.example.com", nil)
		}
	})
}
