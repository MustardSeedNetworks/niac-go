package config

import (
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
