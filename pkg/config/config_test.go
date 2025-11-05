package config

import (
	"net"
	"testing"
)

func TestParseSimpleConfig(t *testing.T) {
	lines := []string{
		"Router1 router 192.168.1.1 00:11:22:33:44:55",
		"Switch1 switch 192.168.1.10 00:11:22:33:44:66 /path/to/walk.walk",
		"# Comment line",
		"",
		"AP1 ap 192.168.1.20 00:11:22:33:44:77",
	}

	cfg, err := ParseSimpleConfig(lines)
	if err != nil {
		t.Fatalf("ParseSimpleConfig failed: %v", err)
	}

	if len(cfg.Devices) != 3 {
		t.Errorf("Expected 3 devices, got %d", len(cfg.Devices))
	}

	// Test first device
	router := cfg.Devices[0]
	if router.Name != "Router1" {
		t.Errorf("Expected name 'Router1', got '%s'", router.Name)
	}
	if router.Type != "router" {
		t.Errorf("Expected type 'router', got '%s'", router.Type)
	}
	expectedIP := net.ParseIP("192.168.1.1")
	if !router.IPAddresses[0].Equal(expectedIP) {
		t.Errorf("Expected IP %s, got %s", expectedIP, router.IPAddresses[0])
	}

	// Test walk file parsing
	sw := cfg.Devices[1]
	if sw.SNMPConfig.WalkFile != "/path/to/walk.walk" {
		t.Errorf("Expected walk file path, got '%s'", sw.SNMPConfig.WalkFile)
	}
}

func TestGetDeviceByMAC(t *testing.T) {
	cfg := &Config{
		Devices: []Device{
			{
				Name:       "Router1",
				MACAddress: net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
			},
			{
				Name:       "Switch1",
				MACAddress: net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x66},
			},
		},
	}

	mac := net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	device := cfg.GetDeviceByMAC(mac)

	if device == nil {
		t.Fatal("GetDeviceByMAC returned nil")
	}
	if device.Name != "Router1" {
		t.Errorf("Expected Router1, got %s", device.Name)
	}

	// Test non-existent MAC
	mac = net.HardwareAddr{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
	device = cfg.GetDeviceByMAC(mac)
	if device != nil {
		t.Error("Expected nil for non-existent MAC")
	}
}

func TestGetDeviceByIP(t *testing.T) {
	cfg := &Config{
		Devices: []Device{
			{
				Name:        "Router1",
				IPAddresses: []net.IP{net.ParseIP("192.168.1.1")},
			},
			{
				Name:        "Switch1",
				IPAddresses: []net.IP{net.ParseIP("192.168.1.10")},
			},
		},
	}

	ip := net.ParseIP("192.168.1.1")
	device := cfg.GetDeviceByIP(ip)

	if device == nil {
		t.Fatal("GetDeviceByIP returned nil")
	}
	if device.Name != "Router1" {
		t.Errorf("Expected Router1, got %s", device.Name)
	}
}

func TestParseSpeed(t *testing.T) {
	tests := []struct {
		input    string
		expected int
		wantErr  bool
	}{
		{"100M", 100, false},
		{"1G", 1000, false},
		{"10G", 10000, false},
		{"100", 100, false},
		{"invalid", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ParseSpeed(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestGenerateMAC(t *testing.T) {
	mac := GenerateMAC()
	if len(mac) != 6 {
		t.Errorf("Expected MAC length 6, got %d", len(mac))
	}
	// Check locally administered bit is set
	if (mac[0] & 0x02) == 0 {
		t.Error("Locally administered bit not set")
	}
}

func BenchmarkParseSimpleConfig(b *testing.B) {
	lines := []string{
		"Router1 router 192.168.1.1 00:11:22:33:44:55",
		"Switch1 switch 192.168.1.10 00:11:22:33:44:66",
		"AP1 ap 192.168.1.20 00:11:22:33:44:77",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseSimpleConfig(lines)
	}
}
