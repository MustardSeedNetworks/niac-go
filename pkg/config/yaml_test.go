package config

import (
	"net"
	"os"
	"path/filepath"
	"testing"
)

// TestLoadYAML_Basic tests basic YAML loading functionality
func TestLoadYAML_Basic(t *testing.T) {
	yaml := `
devices:
  - name: test-router
    mac: "00:11:22:33:44:55"
    ip: "192.168.1.1"
`
	tmpfile := createTempYAML(t, yaml)
	defer os.Remove(tmpfile)

	cfg, err := LoadYAML(tmpfile)
	if err != nil {
		t.Fatalf("LoadYAML failed: %v", err)
	}

	if len(cfg.Devices) != 1 {
		t.Errorf("Expected 1 device, got %d", len(cfg.Devices))
	}

	device := cfg.Devices[0]
	if device.Name != "test-router" {
		t.Errorf("Expected name 'test-router', got '%s'", device.Name)
	}

	expectedMAC, _ := net.ParseMAC("00:11:22:33:44:55")
	if device.MACAddress.String() != expectedMAC.String() {
		t.Errorf("Expected MAC %s, got %s", expectedMAC, device.MACAddress)
	}

	expectedIP := net.ParseIP("192.168.1.1")
	if !device.IPAddresses[0].Equal(expectedIP) {
		t.Errorf("Expected IP %s, got %s", expectedIP, device.IPAddresses[0])
	}
}

// TestLoadYAML_MultipleIPs tests multiple IP addresses per device (v1.5.0)
func TestLoadYAML_MultipleIPs(t *testing.T) {
	yaml := `
devices:
  - name: dual-stack-router
    mac: "00:11:22:33:44:55"
    ips:
      - "192.168.1.1"
      - "2001:db8::1"
      - "10.0.0.1"
`
	tmpfile := createTempYAML(t, yaml)
	defer os.Remove(tmpfile)

	cfg, err := LoadYAML(tmpfile)
	if err != nil {
		t.Fatalf("LoadYAML failed: %v", err)
	}

	device := cfg.Devices[0]
	if len(device.IPAddresses) != 3 {
		t.Errorf("Expected 3 IP addresses, got %d", len(device.IPAddresses))
	}

	// Verify each IP
	expectedIPs := []string{"192.168.1.1", "2001:db8::1", "10.0.0.1"}
	for i, expected := range expectedIPs {
		if !device.IPAddresses[i].Equal(net.ParseIP(expected)) {
			t.Errorf("IP %d: expected %s, got %s", i, expected, device.IPAddresses[i])
		}
	}
}

// TestLoadYAML_LLDP tests LLDP protocol configuration
func TestLoadYAML_LLDP(t *testing.T) {
	yaml := `
devices:
  - name: lldp-device
    mac: "00:11:22:33:44:55"
    ip: "192.168.1.1"
    lldp:
      enabled: true
      system_description: "Test LLDP Device"
      port_description: "GigabitEthernet0/0"
      advertise_interval: 45
      ttl: 180
      chassis_id_type: "mac"
`
	tmpfile := createTempYAML(t, yaml)
	defer os.Remove(tmpfile)

	cfg, err := LoadYAML(tmpfile)
	if err != nil {
		t.Fatalf("LoadYAML failed: %v", err)
	}

	device := cfg.Devices[0]
	if device.LLDPConfig == nil {
		t.Fatal("Expected LLDP config, got nil")
	}

	lldp := device.LLDPConfig
	if !lldp.Enabled {
		t.Error("Expected LLDP enabled")
	}
	if lldp.SystemDescription != "Test LLDP Device" {
		t.Errorf("Expected system description 'Test LLDP Device', got '%s'", lldp.SystemDescription)
	}
	if lldp.PortDescription != "GigabitEthernet0/0" {
		t.Errorf("Expected port description 'GigabitEthernet0/0', got '%s'", lldp.PortDescription)
	}
	if lldp.AdvertiseInterval != 45 {
		t.Errorf("Expected advertise interval 45, got %d", lldp.AdvertiseInterval)
	}
	if lldp.TTL != 180 {
		t.Errorf("Expected TTL 180, got %d", lldp.TTL)
	}
	if lldp.ChassisIDType != "mac" {
		t.Errorf("Expected chassis ID type 'mac', got '%s'", lldp.ChassisIDType)
	}
}

// TestLoadYAML_CDP tests CDP protocol configuration
func TestLoadYAML_CDP(t *testing.T) {
	yaml := `
devices:
  - name: cisco-router
    mac: "00:11:22:33:44:55"
    ip: "192.168.1.1"
    cdp:
      enabled: true
      version: 2
      platform: "Cisco 2921"
      software_version: "IOS 15.4(3)M6a"
      port_id: "GigabitEthernet0/0"
`
	tmpfile := createTempYAML(t, yaml)
	defer os.Remove(tmpfile)

	cfg, err := LoadYAML(tmpfile)
	if err != nil {
		t.Fatalf("LoadYAML failed: %v", err)
	}

	device := cfg.Devices[0]
	if device.CDPConfig == nil {
		t.Fatal("Expected CDP config, got nil")
	}

	cdp := device.CDPConfig
	if !cdp.Enabled {
		t.Error("Expected CDP enabled")
	}
	if cdp.Version != 2 {
		t.Errorf("Expected version 2, got %d", cdp.Version)
	}
	if cdp.Platform != "Cisco 2921" {
		t.Errorf("Expected platform 'Cisco 2921', got '%s'", cdp.Platform)
	}
}

// TestLoadYAML_TrafficConfig tests traffic pattern configuration (v1.6.0)
func TestLoadYAML_TrafficConfig(t *testing.T) {
	yaml := `
devices:
  - name: traffic-router
    mac: "00:11:22:33:44:55"
    ip: "192.168.1.1"
    traffic:
      enabled: true
      arp_announcements:
        enabled: true
        interval: 30
      periodic_pings:
        enabled: true
        interval: 60
        payload_size: 128
      random_traffic:
        enabled: true
        interval: 90
        packet_count: 20
        patterns:
          - broadcast_arp
          - multicast
          - udp
`
	tmpfile := createTempYAML(t, yaml)
	defer os.Remove(tmpfile)

	cfg, err := LoadYAML(tmpfile)
	if err != nil {
		t.Fatalf("LoadYAML failed: %v", err)
	}

	device := cfg.Devices[0]
	if device.TrafficConfig == nil {
		t.Fatal("Expected traffic config, got nil")
	}

	traffic := device.TrafficConfig
	if !traffic.Enabled {
		t.Error("Expected traffic enabled")
	}

	// Test ARP announcements
	if traffic.ARPAnnouncements == nil {
		t.Fatal("Expected ARP announcements config, got nil")
	}
	if !traffic.ARPAnnouncements.Enabled {
		t.Error("Expected ARP announcements enabled")
	}
	if traffic.ARPAnnouncements.Interval != 30 {
		t.Errorf("Expected interval 30, got %d", traffic.ARPAnnouncements.Interval)
	}

	// Test periodic pings
	if traffic.PeriodicPings == nil {
		t.Fatal("Expected periodic pings config, got nil")
	}
	if traffic.PeriodicPings.Interval != 60 {
		t.Errorf("Expected interval 60, got %d", traffic.PeriodicPings.Interval)
	}
	if traffic.PeriodicPings.PayloadSize != 128 {
		t.Errorf("Expected payload size 128, got %d", traffic.PeriodicPings.PayloadSize)
	}

	// Test random traffic
	if traffic.RandomTraffic == nil {
		t.Fatal("Expected random traffic config, got nil")
	}
	if traffic.RandomTraffic.PacketCount != 20 {
		t.Errorf("Expected packet count 20, got %d", traffic.RandomTraffic.PacketCount)
	}
	if len(traffic.RandomTraffic.Patterns) != 3 {
		t.Errorf("Expected 3 patterns, got %d", len(traffic.RandomTraffic.Patterns))
	}
}

// TestLoadYAML_SNMPTraps tests SNMP trap configuration (v1.6.0)
func TestLoadYAML_SNMPTraps(t *testing.T) {
	yaml := `
devices:
  - name: snmp-router
    mac: "00:11:22:33:44:55"
    ip: "192.168.1.1"
    snmp_agent:
      community: "public"
      traps:
        enabled: true
        receivers:
          - "192.168.1.100:162"
          - "192.168.1.101:162"
        cold_start:
          enabled: true
          on_startup: true
        link_state:
          enabled: true
          link_down: true
          link_up: true
        high_cpu:
          enabled: true
          threshold: 80
          interval: 300
        high_memory:
          enabled: true
          threshold: 90
          interval: 300
        interface_errors:
          enabled: true
          threshold: 100
          interval: 60
`
	tmpfile := createTempYAML(t, yaml)
	defer os.Remove(tmpfile)

	cfg, err := LoadYAML(tmpfile)
	if err != nil {
		t.Fatalf("LoadYAML failed: %v", err)
	}

	device := cfg.Devices[0]
	if device.SNMPConfig.Traps == nil {
		t.Fatal("Expected SNMP traps config, got nil")
	}

	traps := device.SNMPConfig.Traps
	if !traps.Enabled {
		t.Error("Expected traps enabled")
	}

	// Test receivers
	if len(traps.Receivers) != 2 {
		t.Errorf("Expected 2 receivers, got %d", len(traps.Receivers))
	}
	if traps.Receivers[0] != "192.168.1.100:162" {
		t.Errorf("Expected receiver '192.168.1.100:162', got '%s'", traps.Receivers[0])
	}

	// Test cold start trap
	if traps.ColdStart == nil {
		t.Fatal("Expected cold start config, got nil")
	}
	if !traps.ColdStart.Enabled || !traps.ColdStart.OnStartup {
		t.Error("Expected cold start enabled with on_startup")
	}

	// Test link state trap
	if traps.LinkState == nil {
		t.Fatal("Expected link state config, got nil")
	}
	if !traps.LinkState.LinkDown || !traps.LinkState.LinkUp {
		t.Error("Expected link down and link up enabled")
	}

	// Test high CPU trap
	if traps.HighCPU == nil {
		t.Fatal("Expected high CPU config, got nil")
	}
	if traps.HighCPU.Threshold != 80 {
		t.Errorf("Expected threshold 80, got %d", traps.HighCPU.Threshold)
	}
	if traps.HighCPU.Interval != 300 {
		t.Errorf("Expected interval 300, got %d", traps.HighCPU.Interval)
	}
}

// TestLoadYAML_TrafficDefaults tests default values for traffic config (v1.6.0)
func TestLoadYAML_TrafficDefaults(t *testing.T) {
	yaml := `
devices:
  - name: traffic-defaults
    mac: "00:11:22:33:44:55"
    ip: "192.168.1.1"
    traffic:
      enabled: true
      arp_announcements:
        enabled: true
      periodic_pings:
        enabled: true
      random_traffic:
        enabled: true
`
	tmpfile := createTempYAML(t, yaml)
	defer os.Remove(tmpfile)

	cfg, err := LoadYAML(tmpfile)
	if err != nil {
		t.Fatalf("LoadYAML failed: %v", err)
	}

	device := cfg.Devices[0]
	traffic := device.TrafficConfig

	// Check defaults are applied
	if traffic.ARPAnnouncements.Interval != 60 {
		t.Errorf("Expected default ARP interval 60, got %d", traffic.ARPAnnouncements.Interval)
	}
	if traffic.PeriodicPings.Interval != 120 {
		t.Errorf("Expected default ping interval 120, got %d", traffic.PeriodicPings.Interval)
	}
	if traffic.PeriodicPings.PayloadSize != 32 {
		t.Errorf("Expected default payload size 32, got %d", traffic.PeriodicPings.PayloadSize)
	}
	if traffic.RandomTraffic.Interval != 180 {
		t.Errorf("Expected default random interval 180, got %d", traffic.RandomTraffic.Interval)
	}
	if traffic.RandomTraffic.PacketCount != 5 {
		t.Errorf("Expected default packet count 5, got %d", traffic.RandomTraffic.PacketCount)
	}
}

// TestLoadYAML_TrapDefaults tests default values for trap config (v1.6.0)
func TestLoadYAML_TrapDefaults(t *testing.T) {
	yaml := `
devices:
  - name: trap-defaults
    mac: "00:11:22:33:44:55"
    ip: "192.168.1.1"
    snmp_agent:
      traps:
        enabled: true
        receivers:
          - "192.168.1.100:162"
        high_cpu:
          enabled: true
        high_memory:
          enabled: true
        interface_errors:
          enabled: true
`
	tmpfile := createTempYAML(t, yaml)
	defer os.Remove(tmpfile)

	cfg, err := LoadYAML(tmpfile)
	if err != nil {
		t.Fatalf("LoadYAML failed: %v", err)
	}

	device := cfg.Devices[0]
	traps := device.SNMPConfig.Traps

	// Check defaults are applied
	if traps.HighCPU.Threshold != 80 {
		t.Errorf("Expected default CPU threshold 80, got %d", traps.HighCPU.Threshold)
	}
	if traps.HighCPU.Interval != 300 {
		t.Errorf("Expected default CPU interval 300, got %d", traps.HighCPU.Interval)
	}
	if traps.HighMemory.Threshold != 90 {
		t.Errorf("Expected default memory threshold 90, got %d", traps.HighMemory.Threshold)
	}
	if traps.InterfaceErrors.Threshold != 100 {
		t.Errorf("Expected default error threshold 100, got %d", traps.InterfaceErrors.Threshold)
	}
	if traps.InterfaceErrors.Interval != 60 {
		t.Errorf("Expected default error interval 60, got %d", traps.InterfaceErrors.Interval)
	}
}

// TestLoadYAML_InvalidFile tests error handling for missing file
func TestLoadYAML_InvalidFile(t *testing.T) {
	_, err := LoadYAML("/nonexistent/file.yaml")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

// TestLoadYAML_InvalidYAML tests error handling for malformed YAML
func TestLoadYAML_InvalidYAML(t *testing.T) {
	yaml := `
devices:
  - name: bad-device
    mac: "invalid
    ip: 192.168.1.1
`
	tmpfile := createTempYAML(t, yaml)
	defer os.Remove(tmpfile)

	_, err := LoadYAML(tmpfile)
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}
}

// TestLoadYAML_InvalidMAC tests error handling for invalid MAC address
func TestLoadYAML_InvalidMAC(t *testing.T) {
	yaml := `
devices:
  - name: bad-mac
    mac: "not-a-mac"
    ip: "192.168.1.1"
`
	tmpfile := createTempYAML(t, yaml)
	defer os.Remove(tmpfile)

	_, err := LoadYAML(tmpfile)
	if err == nil {
		t.Error("Expected error for invalid MAC, got nil")
	}
}

// TestLoadYAML_InvalidIP tests error handling for invalid IP address
func TestLoadYAML_InvalidIP(t *testing.T) {
	yaml := `
devices:
  - name: bad-ip
    mac: "00:11:22:33:44:55"
    ip: "999.999.999.999"
`
	tmpfile := createTempYAML(t, yaml)
	defer os.Remove(tmpfile)

	_, err := LoadYAML(tmpfile)
	if err == nil {
		t.Error("Expected error for invalid IP, got nil")
	}
}

// TestLoadYAML_EmptyConfig tests handling of empty configuration
func TestLoadYAML_EmptyConfig(t *testing.T) {
	yaml := `devices: []`
	tmpfile := createTempYAML(t, yaml)
	defer os.Remove(tmpfile)

	_, err := LoadYAML(tmpfile)
	if err == nil {
		t.Error("Expected error for empty device list, got nil")
	}
}

// TestLoad_AutoDetection tests automatic format detection
func TestLoad_AutoDetection(t *testing.T) {
	// Test YAML detection
	yaml := `
devices:
  - name: yaml-device
    mac: "00:11:22:33:44:55"
    ip: "192.168.1.1"
`
	tmpfile := createTempYAML(t, yaml)
	defer os.Remove(tmpfile)

	cfg, err := Load(tmpfile)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(cfg.Devices) != 1 {
		t.Errorf("Expected 1 device, got %d", len(cfg.Devices))
	}
}

// Helper function to create temporary YAML file for testing
func createTempYAML(t *testing.T, content string) string {
	tmpfile, err := os.CreateTemp("", "test-*.yaml")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}

	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	return tmpfile.Name()
}

// BenchmarkLoadYAML benchmarks YAML loading performance
func BenchmarkLoadYAML(b *testing.B) {
	yaml := `
devices:
  - name: bench-device
    mac: "00:11:22:33:44:55"
    ip: "192.168.1.1"
    lldp:
      enabled: true
      system_description: "Benchmark Device"
`
	tmpfile := filepath.Join(os.TempDir(), "benchmark.yaml")
	if err := os.WriteFile(tmpfile, []byte(yaml), 0644); err != nil {
		b.Fatal(err)
	}
	defer os.Remove(tmpfile)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = LoadYAML(tmpfile)
	}
}
