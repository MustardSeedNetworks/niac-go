package config

import (
	"net"
	"os"
	"path/filepath"
	"testing"
)

// TestLoadYAML_Basic tests basic YAML loading functionality.
func TestLoadYAML_Basic(t *testing.T) {
	yaml := `
devices:
  - name: test-router
    mac: "00:11:22:33:44:55"
    ips:
      - "192.168.1.1"
`
	tmpfile := createTempYAML(t, yaml)

	defer func() { _ = os.Remove(tmpfile) }()

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

// TestLoadYAML_TrunkPorts verifies the YAML loader populates
// device.TrunkPorts so the topology builder can emit edges.
// Regression guard for #550 — converter.Device used to lack the
// TrunkPorts field, silently dropping every trunk_ports: block.
func TestLoadYAML_TrunkPorts(t *testing.T) {
	yaml := `
devices:
  - name: sw-a
    type: switch
    mac: "00:11:22:33:44:01"
    trunk_ports:
      - interface: "Ethernet1/1"
        vlans: [10, 20, 30]
        native_vlan: 1
        remote_device: "sw-b"
        remote_interface: "Ethernet1/1"
  - name: sw-b
    type: switch
    mac: "00:11:22:33:44:02"
    trunk_ports:
      - interface: "Ethernet1/1"
        vlans: [10, 20, 30]
        native_vlan: 1
        remote_device: "sw-a"
        remote_interface: "Ethernet1/1"
    port_channels:
      - id: 1
        members: ["Ethernet1/2", "Ethernet1/3"]
        mode: "active"
`
	tmpfile := createTempYAML(t, yaml)
	defer func() { _ = os.Remove(tmpfile) }()

	cfg, err := LoadYAML(tmpfile)
	if err != nil {
		t.Fatalf("LoadYAML failed: %v", err)
	}
	if len(cfg.Devices) != 2 {
		t.Fatalf("expected 2 devices, got %d", len(cfg.Devices))
	}

	swA := cfg.Devices[0]
	if len(swA.TrunkPorts) != 1 {
		t.Fatalf("sw-a: expected 1 trunk port, got %d", len(swA.TrunkPorts))
	}
	trunk := swA.TrunkPorts[0]
	if trunk.Interface != "Ethernet1/1" {
		t.Errorf("sw-a trunk interface = %q, want Ethernet1/1", trunk.Interface)
	}
	if trunk.RemoteDevice != "sw-b" {
		t.Errorf("sw-a trunk remote_device = %q, want sw-b", trunk.RemoteDevice)
	}
	if len(trunk.VLANs) != 3 || trunk.VLANs[0] != 10 {
		t.Errorf("sw-a trunk vlans = %v, want [10 20 30]", trunk.VLANs)
	}
	if trunk.NativeVLAN != 1 {
		t.Errorf("sw-a trunk native_vlan = %d, want 1", trunk.NativeVLAN)
	}

	swB := cfg.Devices[1]
	if len(swB.PortChannels) != 1 {
		t.Fatalf("sw-b: expected 1 port_channel, got %d", len(swB.PortChannels))
	}
	pc := swB.PortChannels[0]
	if pc.ID != 1 || pc.Mode != "active" || len(pc.Members) != 2 {
		t.Errorf("sw-b port_channel = %+v, want ID=1 mode=active 2 members", pc)
	}
}

func TestLoadYAML_Interfaces(t *testing.T) {
	yaml := `
devices:
  - name: switch-a
    type: switch
    mac: "00:11:22:33:44:55"
    interfaces:
      - name: "Ethernet1/1"
        speed: 1000
        duplex: "full"
        admin_status: "up"
        oper_status: "down"
        description: "uplink"
        vlans: [10, 20]
`
	tmpfile := createTempYAML(t, yaml)
	defer func() { _ = os.Remove(tmpfile) }()

	cfg, err := LoadYAML(tmpfile)
	if err != nil {
		t.Fatalf("LoadYAML failed: %v", err)
	}
	if len(cfg.Devices) != 1 {
		t.Fatalf("expected 1 device, got %d", len(cfg.Devices))
	}
	ifaces := cfg.Devices[0].Interfaces
	if len(ifaces) != 1 {
		t.Fatalf("expected 1 interface, got %d", len(ifaces))
	}
	iface := ifaces[0]
	if iface.Name != "Ethernet1/1" || iface.Speed != 1000 || iface.Duplex != "full" {
		t.Fatalf("interface not loaded correctly: %+v", iface)
	}
	if iface.AdminStatus != "up" || iface.OperStatus != "down" || iface.Description != "uplink" {
		t.Fatalf("interface status/description not loaded correctly: %+v", iface)
	}
	if len(iface.VLANs) != 2 || iface.VLANs[0] != 10 {
		t.Fatalf("interface VLANs = %v, want [10 20]", iface.VLANs)
	}
}

func TestMarshalConfigYAML_Interfaces(t *testing.T) {
	cfg := &Config{
		Devices: []Device{
			{
				Name:       "switch-a",
				Type:       "switch",
				MACAddress: net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
				Interfaces: []Interface{
					{
						Name:        "Ethernet1/1",
						Speed:       1000,
						Duplex:      "full",
						AdminStatus: "up",
					},
				},
			},
		},
	}

	data, err := MarshalConfigYAML(cfg)
	if err != nil {
		t.Fatalf("MarshalConfigYAML failed: %v", err)
	}
	roundTrip, err := LoadYAMLBytes(data)
	if err != nil {
		t.Fatalf("LoadYAMLBytes round trip failed: %v\n%s", err, data)
	}
	iface := roundTrip.Devices[0].Interfaces[0]
	if iface.Name != "Ethernet1/1" || iface.Speed != 1000 || iface.Duplex != "full" {
		t.Fatalf("round-trip interface = %+v", iface)
	}
}

// TestLoadYAML_MultipleIPs tests multiple IP addresses per device (v1.5.0).
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

	defer func() { _ = os.Remove(tmpfile) }()

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

func TestLoadYAML_WalkFileRelativeToConfigFile(t *testing.T) {
	dir := t.TempDir()
	walkFile := filepath.Join(dir, "device.walk")
	if err := os.WriteFile(walkFile, []byte("SNMPv2-MIB::sysName.0 = STRING: test-device\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	yaml := `
devices:
  - name: snmp-device
    type: switch
    mac: "00:11:22:33:44:55"
    ips:
      - "192.168.1.1"
    snmp_agent:
      community: "public"
      walk_file: "device.walk"
`
	configFile := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(configFile, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadYAML(configFile)
	if err != nil {
		t.Fatalf("LoadYAML failed: %v", err)
	}

	got := cfg.Devices[0].SNMPConfig.WalkFile
	if got != walkFile {
		t.Fatalf("walk file = %q, want %q", got, walkFile)
	}
}

func TestLoadYAML_IncludePathRelativeToConfigFile(t *testing.T) {
	dir := t.TempDir()
	walkDir := filepath.Join(dir, "walks")
	if err := os.Mkdir(walkDir, 0o700); err != nil {
		t.Fatal(err)
	}
	walkFile := filepath.Join(walkDir, "device.walk")
	if err := os.WriteFile(walkFile, []byte("SNMPv2-MIB::sysName.0 = STRING: test-device\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	yaml := `
include_path: "walks"
devices:
  - name: snmp-device
    type: switch
    mac: "00:11:22:33:44:55"
    ips:
      - "192.168.1.1"
    snmp_agent:
      community: "public"
      walk_file: "device.walk"
`
	configFile := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(configFile, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadYAML(configFile)
	if err != nil {
		t.Fatalf("LoadYAML failed: %v", err)
	}

	got := cfg.Devices[0].SNMPConfig.WalkFile
	if got != walkFile {
		t.Fatalf("walk file = %q, want %q", got, walkFile)
	}
}

func TestLoadYAML_InferMissingDeviceType(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{name: "edge-router-01", want: "router"},
		{name: "access-switch-01", want: "switch"},
		{name: "office-ap-01", want: "ap"},
		{name: "dns-server-01", want: "server"},
		{name: "test-client-01", want: "host"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yaml := `
devices:
  - name: ` + tt.name + `
    mac: "00:11:22:33:44:55"
    ips:
      - "192.168.1.1"
`
			tmpfile := createTempYAML(t, yaml)
			defer func() { _ = os.Remove(tmpfile) }()

			cfg, err := LoadYAML(tmpfile)
			if err != nil {
				t.Fatalf("LoadYAML failed: %v", err)
			}

			got := cfg.Devices[0].Type
			if got != tt.want {
				t.Fatalf("device type = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestLoadYAML_LLDP tests LLDP protocol configuration.
func TestLoadYAML_LLDP(t *testing.T) {
	yaml := `
devices:
  - name: lldp-device
    mac: "00:11:22:33:44:55"
    ips:
      - "192.168.1.1"
    lldp:
      enabled: true
      system_description: "Test LLDP Device"
      port_description: "GigabitEthernet0/0"
      advertise_interval: 45
      ttl: 180
      chassis_id_type: "mac"
`
	tmpfile := createTempYAML(t, yaml)

	defer func() { _ = os.Remove(tmpfile) }()

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

// TestLoadYAML_CDP tests CDP protocol configuration.
func TestLoadYAML_CDP(t *testing.T) {
	yaml := `
devices:
  - name: cisco-router
    mac: "00:11:22:33:44:55"
    ips:
      - "192.168.1.1"
    cdp:
      enabled: true
      version: 2
      platform: "Cisco 2921"
      software_version: "IOS 15.4(3)M6a"
      port_id: "GigabitEthernet0/0"
`
	tmpfile := createTempYAML(t, yaml)

	defer func() { _ = os.Remove(tmpfile) }()

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

// TestLoadYAML_SNMPTraps tests SNMP trap configuration (v1.6.0).
func TestLoadYAML_SNMPTraps(t *testing.T) {
	yaml := `
devices:
  - name: snmp-router
    mac: "00:11:22:33:44:55"
    ips:
      - "192.168.1.1"
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

	defer func() { _ = os.Remove(tmpfile) }()

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

// TestLoadYAML_TrapDefaults tests default values for trap config (v1.6.0).
func TestLoadYAML_TrapDefaults(t *testing.T) {
	yaml := `
devices:
  - name: trap-defaults
    mac: "00:11:22:33:44:55"
    ips:
      - "192.168.1.1"
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

	defer func() { _ = os.Remove(tmpfile) }()

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

// TestLoadYAML_InvalidFile tests error handling for missing file.
func TestLoadYAML_InvalidFile(t *testing.T) {
	_, err := LoadYAML("/nonexistent/file.yaml")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

// TestLoadYAML_InvalidYAML tests error handling for malformed YAML.
func TestLoadYAML_InvalidYAML(t *testing.T) {
	yaml := `
devices:
  - name: bad-device
    mac: "invalid
    ip: 192.168.1.1
`
	tmpfile := createTempYAML(t, yaml)

	defer func() { _ = os.Remove(tmpfile) }()

	_, err := LoadYAML(tmpfile)
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}
}

// TestLoadYAML_InvalidMAC tests error handling for invalid MAC address.
func TestLoadYAML_InvalidMAC(t *testing.T) {
	yaml := `
devices:
  - name: bad-mac
    mac: "not-a-mac"
    ips:
      - "192.168.1.1"
`
	tmpfile := createTempYAML(t, yaml)

	defer func() { _ = os.Remove(tmpfile) }()

	_, err := LoadYAML(tmpfile)
	if err == nil {
		t.Error("Expected error for invalid MAC, got nil")
	}
}

// TestLoadYAML_InvalidIP tests error handling for invalid IP address.
func TestLoadYAML_InvalidIP(t *testing.T) {
	yaml := `
devices:
  - name: bad-ip
    mac: "00:11:22:33:44:55"
    ips:
      - "999.999.999.999"
`
	tmpfile := createTempYAML(t, yaml)

	defer func() { _ = os.Remove(tmpfile) }()

	_, err := LoadYAML(tmpfile)
	if err == nil {
		t.Error("Expected error for invalid IP, got nil")
	}
}

// TestLoadYAML_EmptyConfig tests handling of empty configuration.
func TestLoadYAML_EmptyConfig(t *testing.T) {
	yaml := `devices: []`
	tmpfile := createTempYAML(t, yaml)

	defer func() { _ = os.Remove(tmpfile) }()

	_, err := LoadYAML(tmpfile)
	if err == nil {
		t.Error("Expected error for empty device list, got nil")
	}
}

// TestLoad_AutoDetection tests automatic format detection.
func TestLoad_AutoDetection(t *testing.T) {
	// Test YAML detection
	yaml := `
devices:
  - name: yaml-device
    mac: "00:11:22:33:44:55"
    ips:
      - "192.168.1.1"
`
	tmpfile := createTempYAML(t, yaml)

	defer func() { _ = os.Remove(tmpfile) }()

	cfg, err := Load(tmpfile)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(cfg.Devices) != 1 {
		t.Errorf("Expected 1 device, got %d", len(cfg.Devices))
	}
}

// Helper function to create temporary YAML file for testing.
func createTempYAML(t *testing.T, content string) string {
	tmpfile, err := os.CreateTemp(t.TempDir(), "test-*.yaml")
	if err != nil {
		t.Fatal(err)
	}

	_, writeErr := tmpfile.WriteString(content)
	if writeErr != nil {
		t.Fatal(writeErr)
	}

	closeErr := tmpfile.Close()
	if closeErr != nil {
		t.Fatal(closeErr)
	}

	return tmpfile.Name()
}

// BenchmarkLoadYAML benchmarks YAML loading performance.
func BenchmarkLoadYAML(b *testing.B) {
	yaml := `
devices:
  - name: bench-device
    mac: "00:11:22:33:44:55"
    ips:
      - "192.168.1.1"
    lldp:
      enabled: true
      system_description: "Benchmark Device"
`

	tmpfile := filepath.Join(b.TempDir(), "benchmark.yaml")

	err := os.WriteFile(tmpfile, []byte(yaml), 0o600)
	if err != nil {
		b.Fatal(err)
	}

	for b.Loop() {
		_, _ = LoadYAML(tmpfile)
	}
}
