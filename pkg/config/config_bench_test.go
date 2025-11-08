package config

import (
	"net"
	"os"
	"path/filepath"
	"testing"
)

// BenchmarkLoadYAML_Small benchmarks loading a small YAML config
func BenchmarkLoadYAML_Small(b *testing.B) {
	// Create a minimal YAML config
	yamlContent := `
devices:
  - name: test-device
    mac: "00:11:22:33:44:55"
    ips:
      - "192.168.1.1"
`

	tmpDir := b.TempDir()
	configFile := filepath.Join(tmpDir, "small.yaml")
	if err := os.WriteFile(configFile, []byte(yamlContent), 0644); err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = LoadYAML(configFile)
	}
}

// BenchmarkLoadYAML_Medium benchmarks loading a medium YAML config
func BenchmarkLoadYAML_Medium(b *testing.B) {
	yamlContent := `
devices:
  - name: router-01
    mac: "00:1a:2b:3c:4d:01"
    ips:
      - "192.168.1.1"
      - "2001:db8::1"
    lldp:
      enabled: true
      system_description: "Test Router"
    cdp:
      enabled: true
      platform: "Test Platform"
    dhcp:
      enabled: true
      pools:
        - network: "192.168.1.0/24"
          range_start: "192.168.1.100"
          range_end: "192.168.1.200"
          gateway: "192.168.1.1"
`

	tmpDir := b.TempDir()
	configFile := filepath.Join(tmpDir, "medium.yaml")
	if err := os.WriteFile(configFile, []byte(yamlContent), 0644); err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = LoadYAML(configFile)
	}
}

// BenchmarkLoadYAML_Large benchmarks loading a large YAML config
func BenchmarkLoadYAML_Large(b *testing.B) {
	// Generate a larger config with multiple devices
	yamlContent := `
devices:
`
	for i := 0; i < 10; i++ {
		yamlContent += `
  - name: device-` + string(rune('0'+i)) + `
    mac: "00:11:22:33:44:` + string(rune('0'+i)) + string(rune('0'+i)) + `"
    ips:
      - "192.168.1.` + string(rune('1'+i)) + `"
    lldp:
      enabled: true
      system_description: "Device ` + string(rune('0'+i)) + `"
    dhcp:
      enabled: true
      pools:
        - network: "192.168.` + string(rune('1'+i)) + `.0/24"
          range_start: "192.168.` + string(rune('1'+i)) + `.100"
          range_end: "192.168.` + string(rune('1'+i)) + `.200"
`
	}

	tmpDir := b.TempDir()
	configFile := filepath.Join(tmpDir, "large.yaml")
	if err := os.WriteFile(configFile, []byte(yamlContent), 0644); err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = LoadYAML(configFile)
	}
}

// BenchmarkDeviceLookup benchmarks device iteration
func BenchmarkDeviceLookup(b *testing.B) {
	yamlContent := `
devices:
  - name: test-device
    mac: "00:11:22:33:44:55"
    ips:
      - "192.168.1.1"
`

	tmpDir := b.TempDir()
	configFile := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(configFile, []byte(yamlContent), 0644); err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}

	cfg, _ := LoadYAML(configFile)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simple device iteration
		for range cfg.Devices {
			// Simulate lookup work
		}
	}
}

// BenchmarkParseYAML benchmarks YAML parsing only (no validation)
func BenchmarkParseYAML(b *testing.B) {
	yamlContent := []byte(`
devices:
  - name: test-device
    mac: "00:11:22:33:44:55"
    ips:
      - "192.168.1.1"
`)

	tmpDir := b.TempDir()
	configFile := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(configFile, yamlContent, 0644); err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Just measure file read and YAML parse time
		data, _ := os.ReadFile(configFile)
		_ = data
	}
}

// BenchmarkConfigValidation_Simple benchmarks validating a simple configuration
func BenchmarkConfigValidation_Simple(b *testing.B) {
	yamlContent := `
devices:
  - name: test-device
    mac: "00:11:22:33:44:55"
    ips:
      - "192.168.1.1"
`

	tmpDir := b.TempDir()
	configFile := filepath.Join(tmpDir, "simple.yaml")
	if err := os.WriteFile(configFile, []byte(yamlContent), 0644); err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cfg, _ := LoadYAML(configFile)
		// Validation happens during LoadYAML
		_ = cfg
	}
}

// BenchmarkConfigValidation_Complex benchmarks validating a complex configuration
func BenchmarkConfigValidation_Complex(b *testing.B) {
	yamlContent := `
devices:
  - name: router-01
    mac: "00:1a:2b:3c:4d:01"
    ips:
      - "192.168.1.1"
      - "10.0.0.1"
      - "2001:db8::1"
    lldp:
      enabled: true
      advertise_interval: 30
      ttl: 120
      system_description: "Test Router"
      chassis_id_type: "mac"
    cdp:
      enabled: true
      advertise_interval: 60
      holdtime: 180
      version: 2
      platform: "Test Platform"
    dhcp:
      subnet_mask: "255.255.255.0"
      router: "192.168.1.1"
      domain_name_server: "8.8.8.8"
      domain_name: "example.com"
    dns:
      forward_records:
        - name: "router.example.com"
          ip: "192.168.1.1"
          ttl: 3600
    icmp:
      enabled: true
      ttl: 64
    interfaces:
      - name: "eth0"
        speed: 1000
        duplex: "full"
        admin_status: "up"
        oper_status: "up"
`

	tmpDir := b.TempDir()
	configFile := filepath.Join(tmpDir, "complex.yaml")
	if err := os.WriteFile(configFile, []byte(yamlContent), 0644); err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cfg, _ := LoadYAML(configFile)
		_ = cfg
	}
}

// BenchmarkConfigNormalization_MACAddress benchmarks MAC address normalization
func BenchmarkConfigNormalization_MACAddress(b *testing.B) {
	macStrings := []string{
		"00:11:22:33:44:55",
		"00-11-22-33-44-55",
		"0011.2233.4455",
		"001122334455",
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, macStr := range macStrings {
			_, _ = net.ParseMAC(macStr)
		}
	}
}

// BenchmarkConfigNormalization_IPAddress benchmarks IP address normalization
func BenchmarkConfigNormalization_IPAddress(b *testing.B) {
	ipStrings := []string{
		"192.168.1.1",
		"10.0.0.1",
		"2001:db8::1",
		"fe80::1",
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, ipStr := range ipStrings {
			_ = net.ParseIP(ipStr)
		}
	}
}

// BenchmarkGetDeviceByMAC benchmarks device lookup by MAC address
func BenchmarkGetDeviceByMAC(b *testing.B) {
	yamlContent := `
devices:
  - name: device-01
    mac: "00:11:22:33:44:55"
    ips:
      - "192.168.1.1"
  - name: device-02
    mac: "00:11:22:33:44:66"
    ips:
      - "192.168.1.2"
  - name: device-03
    mac: "00:11:22:33:44:77"
    ips:
      - "192.168.1.3"
`

	tmpDir := b.TempDir()
	configFile := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(configFile, []byte(yamlContent), 0644); err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}

	cfg, _ := LoadYAML(configFile)
	targetMAC, _ := net.ParseMAC("00:11:22:33:44:66")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = cfg.GetDeviceByMAC(targetMAC)
	}
}

// BenchmarkGetDeviceByIP benchmarks device lookup by IP address
func BenchmarkGetDeviceByIP(b *testing.B) {
	yamlContent := `
devices:
  - name: device-01
    mac: "00:11:22:33:44:55"
    ips:
      - "192.168.1.1"
  - name: device-02
    mac: "00:11:22:33:44:66"
    ips:
      - "192.168.1.2"
  - name: device-03
    mac: "00:11:22:33:44:77"
    ips:
      - "192.168.1.3"
`

	tmpDir := b.TempDir()
	configFile := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(configFile, []byte(yamlContent), 0644); err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}

	cfg, _ := LoadYAML(configFile)
	targetIP := net.ParseIP("192.168.1.2")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = cfg.GetDeviceByIP(targetIP)
	}
}

// BenchmarkLoadLegacyConfig benchmarks loading legacy format configuration
func BenchmarkLoadLegacyConfig(b *testing.B) {
	legacyContent := `
# Legacy format config
device router-01 {
    type = "router"
    mac = "00:11:22:33:44:55"
    ip = "192.168.1.1"
    snmp_community = "public"
    sysName = "router-01"
    sysDescr = "Test Router"
}
`

	tmpDir := b.TempDir()
	configFile := filepath.Join(tmpDir, "legacy.cfg")
	if err := os.WriteFile(configFile, []byte(legacyContent), 0644); err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = LoadLegacy(configFile)
	}
}

// BenchmarkParseSpeed benchmarks interface speed parsing
func BenchmarkParseSpeed(b *testing.B) {
	speeds := []string{"100M", "1G", "10G", "1000", "10000"}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, speed := range speeds {
			_, _ = ParseSpeed(speed)
		}
	}
}

// BenchmarkGenerateMAC benchmarks generating random MAC addresses
func BenchmarkGenerateMAC(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = GenerateMAC()
	}
}

// BenchmarkConfigWithMultipleProtocols benchmarks loading config with multiple protocol configs
func BenchmarkConfigWithMultipleProtocols(b *testing.B) {
	yamlContent := `
devices:
  - name: multi-protocol-device
    mac: "00:11:22:33:44:55"
    ips:
      - "192.168.1.1"
    lldp:
      enabled: true
      advertise_interval: 30
    cdp:
      enabled: true
      advertise_interval: 60
    edp:
      enabled: true
      advertise_interval: 30
    fdp:
      enabled: true
      advertise_interval: 60
    stp:
      enabled: true
      bridge_priority: 32768
    http:
      enabled: true
      server_name: "TestServer/1.0"
    ftp:
      enabled: true
      allow_anonymous: true
    netbios:
      enabled: true
      name: "TESTDEVICE"
    icmp:
      enabled: true
    icmpv6:
      enabled: true
`

	tmpDir := b.TempDir()
	configFile := filepath.Join(tmpDir, "multiproto.yaml")
	if err := os.WriteFile(configFile, []byte(yamlContent), 0644); err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = LoadYAML(configFile)
	}
}
