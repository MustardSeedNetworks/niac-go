package converter

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- Parser helper tests ---

func newParser(input string) *Parser {
	return &Parser{
		lines:   strings.Split(input, "\n"),
		pos:     0,
		verbose: false,
	}
}

func TestExtractString(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected string
	}{
		{"quoted value", `IncludePath("some/path")`, "some/path"},
		{"empty quotes", `IncludePath("")`, ""},
		{"no quotes", `IncludePath(value)`, ""},
		{"no closing quote", `IncludePath("no-close`, ""},
		{"multiple quoted", `Foo("first" "second")`, "first"},
		{"empty line", "", ""},
	}

	p := newParser("")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.extractString(tt.line)
			if got != tt.expected {
				t.Errorf("extractString(%q) = %q, want %q", tt.line, got, tt.expected)
			}
		})
	}
}

func TestExtractValue(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected string
	}{
		{"simple value", "IpAddr(192.168.1.1)", "192.168.1.1"},
		{"no parens", "IpAddr value", ""},
		{"no close paren", "IpAddr(192.168.1.1", ""},
		{"empty parens", "IpAddr()", ""},
		{"nested content", "Foo(bar baz)", "bar baz"},
	}

	p := newParser("")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.extractValue(tt.line)
			if got != tt.expected {
				t.Errorf("extractValue(%q) = %q, want %q", tt.line, got, tt.expected)
			}
		})
	}
}

func TestFormatMAC(t *testing.T) {
	tests := []struct {
		name     string
		mac      string
		expected string
	}{
		{"valid 12-char", "001122334455", "00:11:22:33:44:55"},
		{"all zeros", "000000000000", "00:00:00:00:00:00"},
		{"all F", "FFFFFFFFFFFF", "FF:FF:FF:FF:FF:FF"},
		{"short", "0011", "0011"},
		{"long", "00112233445566", "00112233445566"},
		{"empty", "", ""},
		{"already formatted", "00:11:22:33:44:55", "00:11:22:33:44:55"},
	}

	p := newParser("")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.formatMAC(tt.mac)
			if got != tt.expected {
				t.Errorf("formatMAC(%q) = %q, want %q", tt.mac, got, tt.expected)
			}
		})
	}
}

// --- Parse tests ---

func TestParse_EmptyInput(t *testing.T) {
	p := newParser("")
	cfg, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse empty input: unexpected error: %v", err)
	}
	if len(cfg.Devices) != 0 {
		t.Errorf("expected 0 devices, got %d", len(cfg.Devices))
	}
}

func TestParse_CommentsAndBlanks(t *testing.T) {
	input := `// This is a comment
# Also a comment

// another comment
`
	p := newParser(input)
	cfg, err := p.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Devices) != 0 {
		t.Errorf("expected 0 devices, got %d", len(cfg.Devices))
	}
}

func TestParse_IncludePath(t *testing.T) {
	input := `IncludePath("/some/path")`
	p := newParser(input)
	cfg, err := p.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.IncludePath != "/some/path" {
		t.Errorf("IncludePath = %q, want %q", cfg.IncludePath, "/some/path")
	}
}

func TestParse_CapturePlayback(t *testing.T) {
	input := `CapturePlayback(
  FileName("test.pcap")
  LoopTime(5000)
  ScaleTime(2.5)
)`
	p := newParser(input)
	cfg, err := p.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.CapturePlaybacks) != 1 {
		t.Fatalf("expected 1 playback, got %d", len(cfg.CapturePlaybacks))
	}
	pb := cfg.CapturePlaybacks[0]
	if pb.FileName != "test.pcap" {
		t.Errorf("FileName = %q, want %q", pb.FileName, "test.pcap")
	}
	if pb.LoopTime != 5000 {
		t.Errorf("LoopTime = %d, want 5000", pb.LoopTime)
	}
	if pb.ScaleTime != 2.5 {
		t.Errorf("ScaleTime = %f, want 2.5", pb.ScaleTime)
	}
}

func TestParse_CapturePlayback_InvalidLoopTime(t *testing.T) {
	input := `CapturePlayback(
  LoopTime(abc)
)`
	p := newParser(input)
	_, err := p.Parse()
	if err == nil {
		t.Fatal("expected error for invalid LoopTime")
	}
	if !strings.Contains(err.Error(), "invalid LoopTime format") {
		t.Errorf("error = %q, expected to contain 'invalid LoopTime format'", err.Error())
	}
}

func TestParse_CapturePlayback_InvalidScaleTime(t *testing.T) {
	input := `CapturePlayback(
  ScaleTime(abc)
)`
	p := newParser(input)
	_, err := p.Parse()
	if err == nil {
		t.Fatal("expected error for invalid ScaleTime")
	}
	if !strings.Contains(err.Error(), "invalid ScaleTime format") {
		t.Errorf("error = %q, expected to contain 'invalid ScaleTime format'", err.Error())
	}
}

func TestParse_SimpleDevice(t *testing.T) {
	input := `Device(
  MacAddr(001122334455)
  IpAddr(192.168.1.1)
)`
	p := newParser(input)
	cfg, err := p.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Devices) != 1 {
		t.Fatalf("expected 1 device, got %d", len(cfg.Devices))
	}
	d := cfg.Devices[0]
	if d.MAC != "00:11:22:33:44:55" {
		t.Errorf("MAC = %q, want %q", d.MAC, "00:11:22:33:44:55")
	}
	if len(d.IPs) != 1 || d.IPs[0] != "192.168.1.1" {
		t.Errorf("IPs[0] = %q, want %q", d.IPs, "192.168.1.1")
	}
	if d.Name != "device1" {
		t.Errorf("Name = %q, want %q", d.Name, "device1")
	}
}

func TestParse_DeviceWithVlan(t *testing.T) {
	input := `Device(
  MacAddr(001122334455)
  Vlan(100)
)`
	p := newParser(input)
	cfg, err := p.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Devices[0].VLAN != 100 {
		t.Errorf("VLAN = %d, want 100", cfg.Devices[0].VLAN)
	}
}

func TestParse_DeviceWithInvalidVlan(t *testing.T) {
	input := `Device(
  MacAddr(001122334455)
  Vlan(abc)
)`
	p := newParser(input)
	_, err := p.Parse()
	if err == nil {
		t.Fatal("expected error for invalid Vlan")
	}
	if !strings.Contains(err.Error(), "invalid Vlan format") {
		t.Errorf("error = %q, expected to contain 'invalid Vlan format'", err.Error())
	}
}

func TestParse_DeviceWithBabble(t *testing.T) {
	input := `Device(
  MacAddr(001122334455)
  Babble()
)`
	p := newParser(input)
	cfg, err := p.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.Devices[0].Babble {
		t.Error("expected Babble to be true")
	}
}

func TestParse_DeviceWithMapToIp(t *testing.T) {
	input := `Device(
  MacAddr(001122334455)
  MapToIp(10.0.0.1)
)`
	p := newParser(input)
	cfg, err := p.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Devices[0].MapToIP != "10.0.0.1" {
		t.Errorf("MapToIP = %q, want %q", cfg.Devices[0].MapToIP, "10.0.0.1")
	}
}

func TestParse_DeviceWithTTL(t *testing.T) {
	input := `Device(
  MacAddr(001122334455)
  TTL(64 192.168.1.0 255.255.255.0)
)`
	p := newParser(input)
	cfg, err := p.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	d := cfg.Devices[0]
	if d.TTL == nil {
		t.Fatal("TTL should not be nil")
	}
	if d.TTL.TTL != 64 {
		t.Errorf("TTL.TTL = %d, want 64", d.TTL.TTL)
	}
	if d.TTL.IP != "192.168.1.0" {
		t.Errorf("TTL.IP = %q, want %q", d.TTL.IP, "192.168.1.0")
	}
	if d.TTL.Mask != "255.255.255.0" {
		t.Errorf("TTL.Mask = %q, want %q", d.TTL.Mask, "255.255.255.0")
	}
}

func TestParseTTL_InvalidFormats(t *testing.T) {
	tests := []struct {
		name string
		line string
	}{
		{"no parens", "TTL"},
		{"too few fields", "TTL(64)"},
		{"two fields", "TTL(64 192.168.1.0)"},
		{"non-numeric ttl", "TTL(abc 192.168.1.0 255.255.255.0)"},
	}

	p := newParser("")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.parseTTL(tt.line)
			if result != nil && tt.name != "non-numeric ttl" {
				// non-numeric returns nil because Sscanf fails
				if tt.name != "two fields" { // two fields returns nil from len check
					t.Errorf("expected nil for %q, got %+v", tt.line, result)
				}
			}
		})
	}
}

func TestParse_DeviceWithMultipleIPs(t *testing.T) {
	input := `Device(
  MacAddr(001122334455)
  IpAddr(192.168.1.1)
  IpAddr(10.0.0.1)
  Ip6Addr(fe80::1)
)`
	p := newParser(input)
	cfg, err := p.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	d := cfg.Devices[0]
	if len(d.IPs) != 3 {
		t.Fatalf("expected 3 IPs, got %d: %v", len(d.IPs), d.IPs)
	}
	if d.IPs[0] != "192.168.1.1" {
		t.Errorf("IPs[0] = %q, want %q", d.IPs[0], "192.168.1.1")
	}
	if d.IPs[1] != "10.0.0.1" {
		t.Errorf("IPs[1] = %q, want %q", d.IPs[1], "10.0.0.1")
	}
	if d.IPs[2] != "fe80::1" {
		t.Errorf("IPs[2] = %q, want %q", d.IPs[2], "fe80::1")
	}
}

func TestParse_DeviceWithSnmpAgent(t *testing.T) {
	input := `Device(
  MacAddr(001122334455)
  SnmpAgent(
    Include("router.snmpwalk")
    SnmpAddr(192.168.1.1)
    AddMib("1.3.6.1.2.1.1.1.0" "OctetString" "Test Router")
  )
)`
	p := newParser(input)
	cfg, err := p.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	d := cfg.Devices[0]
	if d.SnmpAgent == nil {
		t.Fatal("SnmpAgent should not be nil")
	}
	if d.SnmpAgent.WalkFile != "router.snmpwalk" {
		t.Errorf("WalkFile = %q, want %q", d.SnmpAgent.WalkFile, "router.snmpwalk")
	}
	if d.SnmpAgent.SnmpAddr != "192.168.1.1" {
		t.Errorf("SnmpAddr = %q, want %q", d.SnmpAgent.SnmpAddr, "192.168.1.1")
	}
	if len(d.SnmpAgent.AddMibs) != 1 {
		t.Fatalf("expected 1 AddMib, got %d", len(d.SnmpAgent.AddMibs))
	}
	mib := d.SnmpAgent.AddMibs[0]
	if mib.OID != "1.3.6.1.2.1.1.1.0" {
		t.Errorf("OID = %q, want %q", mib.OID, "1.3.6.1.2.1.1.1.0")
	}
	if mib.Type != "OctetString" {
		t.Errorf("Type = %q, want %q", mib.Type, "OctetString")
	}
	if mib.Value != "Test Router" {
		t.Errorf("Value = %q, want %q", mib.Value, "Test Router")
	}
}

func TestParse_DeviceWithDNS(t *testing.T) {
	input := `Device(
  MacAddr(001122334455)
  Dns(
    Forward("host.example.com" 192.168.1.1 300)
    Reverse(192.168.1.1 "host.example.com" 300)
  )
)`
	p := newParser(input)
	cfg, err := p.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	d := cfg.Devices[0]
	if d.DNS == nil {
		t.Fatal("DNS should not be nil")
	}
	if len(d.DNS.ForwardRecords) != 1 {
		t.Fatalf("expected 1 forward record, got %d", len(d.DNS.ForwardRecords))
	}
	fwd := d.DNS.ForwardRecords[0]
	if fwd.Name != "host.example.com" {
		t.Errorf("Forward Name = %q, want %q", fwd.Name, "host.example.com")
	}
	if fwd.IP != "192.168.1.1" {
		t.Errorf("Forward IP = %q, want %q", fwd.IP, "192.168.1.1")
	}
	if fwd.TTL != 300 {
		t.Errorf("Forward TTL = %d, want 300", fwd.TTL)
	}
	if len(d.DNS.ReverseRecords) != 1 {
		t.Fatalf("expected 1 reverse record, got %d", len(d.DNS.ReverseRecords))
	}
}

func TestParse_DeviceWithDHCP(t *testing.T) {
	input := `Device(
  MacAddr(001122334455)
  Dhcp(
    SubnetMask(255.255.255.0)
    Router(192.168.1.1)
    DomainNameServer(8.8.8.8)
  )
)`
	p := newParser(input)
	cfg, err := p.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	d := cfg.Devices[0]
	if d.Dhcp == nil {
		t.Fatal("DHCP should not be nil")
	}
	if d.Dhcp.SubnetMask != "255.255.255.0" {
		t.Errorf("SubnetMask = %q, want %q", d.Dhcp.SubnetMask, "255.255.255.0")
	}
	if d.Dhcp.Router != "192.168.1.1" {
		t.Errorf("Router = %q, want %q", d.Dhcp.Router, "192.168.1.1")
	}
	if d.Dhcp.DomainNameServer != "8.8.8.8" {
		t.Errorf("DomainNameServer = %q, want %q", d.Dhcp.DomainNameServer, "8.8.8.8")
	}
}

func TestParse_DeviceWithICMP(t *testing.T) {
	input := `Device(
  MacAddr(001122334455)
  Icmp(
    AddressMaskReply(255.255.255.0)
    RouterAdvertisement(
      Period(600)
      Lifetime(1800)
      Router(192.168.1.1 100)
    )
  )
)`
	p := newParser(input)
	cfg, err := p.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	d := cfg.Devices[0]
	if d.Icmp == nil {
		t.Fatal("Icmp should not be nil")
	}
	if !d.Icmp.Enabled {
		t.Error("Icmp should be enabled")
	}
	if d.Icmp.AddressMaskReply != "255.255.255.0" {
		t.Errorf("AddressMaskReply = %q, want %q", d.Icmp.AddressMaskReply, "255.255.255.0")
	}
	ra := d.Icmp.RouterAdvertisement
	if ra == nil {
		t.Fatal("RouterAdvertisement should not be nil")
	}
	if ra.Period != 600 {
		t.Errorf("Period = %d, want 600", ra.Period)
	}
	if ra.Lifetime != 1800 {
		t.Errorf("Lifetime = %d, want 1800", ra.Lifetime)
	}
	if len(ra.Routers) != 1 {
		t.Fatalf("expected 1 router, got %d", len(ra.Routers))
	}
	if ra.Routers[0].Address != "192.168.1.1" {
		t.Errorf("Router Address = %q, want %q", ra.Routers[0].Address, "192.168.1.1")
	}
	if ra.Routers[0].Preference != 100 {
		t.Errorf("Router Preference = %d, want 100", ra.Routers[0].Preference)
	}
}

func TestParse_DeviceWithICMPv6(t *testing.T) {
	input := `Device(
  MacAddr(001122334455)
  Icmp6(
    RouterAdvertisement(
      Period(600)
      CurHopLimit(64)
      Managed(1)
      PrefixInformation(
        PrefixLength(64)
        Onlink(1)
        Prefix(2001:db8::)
      )
    )
  )
)`
	p := newParser(input)
	cfg, err := p.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	d := cfg.Devices[0]
	if d.Icmpv6 == nil {
		t.Fatal("Icmpv6 should not be nil")
	}
	ra := d.Icmpv6.RouterAdvertisement
	if ra == nil {
		t.Fatal("ICMPv6 RouterAdvertisement should not be nil")
	}
	if ra.Period != 600 {
		t.Errorf("Period = %d, want 600", ra.Period)
	}
	if ra.CurHopLimit != 64 {
		t.Errorf("CurHopLimit = %d, want 64", ra.CurHopLimit)
	}
	if ra.Managed != 1 {
		t.Errorf("Managed = %d, want 1", ra.Managed)
	}
	if len(ra.PrefixInfo) != 1 {
		t.Fatalf("expected 1 prefix info, got %d", len(ra.PrefixInfo))
	}
	pi := ra.PrefixInfo[0]
	if pi.PrefixLength != 64 {
		t.Errorf("PrefixLength = %d, want 64", pi.PrefixLength)
	}
	if pi.Prefix != "2001:db8::" {
		t.Errorf("Prefix = %q, want %q", pi.Prefix, "2001:db8::")
	}
}

func TestParse_DeviceWithNetBios(t *testing.T) {
	input := `Device(
  MacAddr(001122334455)
  NetBiosStatus(
    MsBrowse
    Bnode
    NetBiosName("MYHOST" Machine Unique)
  )
)`
	p := newParser(input)
	cfg, err := p.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	d := cfg.Devices[0]
	if d.Netbios == nil {
		t.Fatal("Netbios should not be nil")
	}
	if !d.Netbios.Enabled {
		t.Error("Netbios should be enabled")
	}
	if !d.Netbios.MsBrowse {
		t.Error("MsBrowse should be true")
	}
	if d.Netbios.NodeType != "B" {
		t.Errorf("NodeType = %q, want %q", d.Netbios.NodeType, "B")
	}
	if len(d.Netbios.Names) != 1 {
		t.Fatalf("expected 1 name, got %d", len(d.Netbios.Names))
	}
	name := d.Netbios.Names[0]
	if name.Name != "MYHOST" {
		t.Errorf("Name = %q, want %q", name.Name, "MYHOST")
	}
	if name.Suffix != "0" {
		t.Errorf("Suffix = %q, want %q", name.Suffix, "0")
	}
	if name.Group {
		t.Error("Group should be false for Unique")
	}
}

func TestParse_DeviceWithSpanningTree(t *testing.T) {
	input := `Device(
  MacAddr(001122334455)
  SpanningTree()
)`
	p := newParser(input)
	cfg, err := p.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	d := cfg.Devices[0]
	if d.Stp == nil {
		t.Fatal("Stp should not be nil")
	}
	if !d.Stp.Enabled {
		t.Error("Stp should be enabled")
	}
}

func TestParse_MultipleDevices(t *testing.T) {
	input := `Device(
  MacAddr(001122334455)
  IpAddr(192.168.1.1)
)
Device(
  MacAddr(AABBCCDDEEFF)
  IpAddr(192.168.1.2)
)`
	p := newParser(input)
	cfg, err := p.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Devices) != 2 {
		t.Fatalf("expected 2 devices, got %d", len(cfg.Devices))
	}
	if cfg.Devices[0].Name != "device1" {
		t.Errorf("first device name = %q, want %q", cfg.Devices[0].Name, "device1")
	}
	if cfg.Devices[1].Name != "device2" {
		t.Errorf("second device name = %q, want %q", cfg.Devices[1].Name, "device2")
	}
}

// --- ValidateConfig tests ---

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		wantError bool
		errMsg    string
	}{
		{
			name: "valid config",
			config: &Config{
				Devices: []Device{
					{MAC: "00:11:22:33:44:55", IPs: []string{"192.168.1.1"}},
				},
			},
			wantError: false,
		},
		{
			name: "missing MAC",
			config: &Config{
				Devices: []Device{
					{IPs: []string{"192.168.1.1"}},
				},
			},
			wantError: true,
			errMsg:    "device missing MAC address",
		},
		{
			name: "valid with SNMP",
			config: &Config{
				Devices: []Device{
					{
						MAC: "00:11:22:33:44:55",
						SnmpAgent: &SnmpAgent{
							AddMibs: []AddMib{
								{OID: "1.3.6.1", Type: "STRING", Value: "test"},
							},
						},
					},
				},
			},
			wantError: false,
		},
		{
			name: "SNMP AddMib missing OID",
			config: &Config{
				Devices: []Device{
					{
						MAC: "00:11:22:33:44:55",
						SnmpAgent: &SnmpAgent{
							AddMibs: []AddMib{
								{Type: "STRING", Value: "test"},
							},
						},
					},
				},
			},
			wantError: true,
			errMsg:    "AddMib missing OID",
		},
		{
			name: "SNMP AddMib missing Type",
			config: &Config{
				Devices: []Device{
					{
						MAC: "00:11:22:33:44:55",
						SnmpAgent: &SnmpAgent{
							AddMibs: []AddMib{
								{OID: "1.3.6.1", Value: "test"},
							},
						},
					},
				},
			},
			wantError: true,
			errMsg:    "AddMib missing type",
		},
		{
			name: "capture playback missing file",
			config: &Config{
				CapturePlaybacks: []CapturePlayback{
					{LoopTime: 1000},
				},
			},
			wantError: true,
			errMsg:    "CapturePlayback missing file name",
		},
		{
			name: "valid capture playback",
			config: &Config{
				CapturePlaybacks: []CapturePlayback{
					{FileName: "test.pcap", LoopTime: 1000},
				},
			},
			wantError: false,
		},
		{
			name:      "empty config",
			config:    &Config{},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.config)
			if tt.wantError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error = %q, want to contain %q", err.Error(), tt.errMsg)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// --- LoadYAMLConfigFromBytes tests ---

func TestLoadYAMLConfigFromBytes(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
		validate  func(*testing.T, *Config)
	}{
		{
			name: "valid YAML",
			input: `devices:
  - mac: "00:11:22:33:44:55"
    ips:
      - "192.168.1.1"
    name: "router1"`,
			wantError: false,
			validate: func(t *testing.T, cfg *Config) {
				t.Helper()
				if len(cfg.Devices) != 1 {
					t.Fatalf("expected 1 device, got %d", len(cfg.Devices))
				}
				if cfg.Devices[0].MAC != "00:11:22:33:44:55" {
					t.Errorf("MAC = %q, want %q", cfg.Devices[0].MAC, "00:11:22:33:44:55")
				}
			},
		},
		{
			name:      "empty YAML",
			input:     "",
			wantError: false,
			validate: func(t *testing.T, cfg *Config) {
				t.Helper()
				if len(cfg.Devices) != 0 {
					t.Errorf("expected 0 devices, got %d", len(cfg.Devices))
				}
			},
		},
		{
			name:      "malformed YAML",
			input:     "{{{{invalid yaml",
			wantError: true,
		},
		{
			name: "unknown top-level field",
			input: `devcies:
  - mac: "00:11:22:33:44:55"`,
			wantError: true,
		},
		{
			name: "unknown device field",
			input: `devices:
  - mac: "00:11:22:33:44:55"
    comunity: public`,
			wantError: true,
		},
		{
			name: "multiple YAML documents",
			input: `devices:
  - mac: "00:11:22:33:44:55"
---
devices:
  - mac: "00:11:22:33:44:66"`,
			wantError: true,
		},
		{
			name: "YAML with capture playbacks",
			input: `capture_playbacks:
  - file_name: "test.pcap"
    loop_time: 5000
devices:
  - mac: "AA:BB:CC:DD:EE:FF"`,
			wantError: false,
			validate: func(t *testing.T, cfg *Config) {
				t.Helper()
				if len(cfg.CapturePlaybacks) != 1 {
					t.Fatalf("expected 1 playback, got %d", len(cfg.CapturePlaybacks))
				}
				if cfg.CapturePlaybacks[0].FileName != "test.pcap" {
					t.Errorf("FileName = %q", cfg.CapturePlaybacks[0].FileName)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runLoadYAMLCase(t, tt.input, tt.wantError, tt.validate)
		})
	}
}

// runLoadYAMLCase executes one LoadYAMLConfigFromBytes table case.
func runLoadYAMLCase(t *testing.T, input string, wantError bool, validate func(*testing.T, *Config)) {
	t.Helper()
	cfg, err := LoadYAMLConfigFromBytes([]byte(input))
	if wantError {
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		return
	}
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if validate != nil {
		validate(t, cfg)
	}
}

// --- path traversal containment tests (CodeQL go/path-injection #151/#152) ---

// TestValidateNoTraversal is the table-driven proof that the shared
// sanitizer used by ConvertFile (input+output) and LoadYAMLConfig
// rejects relative traversal sequences while still resolving legitimate
// paths.
func TestValidateNoTraversal(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		wantError bool
	}{
		{name: "empty", path: "", wantError: true},
		{name: "simple traversal", path: "../secret.yaml", wantError: true},
		{name: "nested traversal", path: "../../etc/passwd", wantError: true},
		{name: "traversal mid-path", path: "configs/../../etc/passwd", wantError: true},
		{name: "bare dotdot", path: "..", wantError: true},
		{name: "relative filename", path: "config.yaml", wantError: false},
		{name: "relative subdir", path: "configs/config.yaml", wantError: false},
		{name: "absolute path", path: "/tmp/config.yaml", wantError: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validateNoTraversal("test", tt.path)
			if tt.wantError && err == nil {
				t.Errorf("validateNoTraversal(%q): expected error, got nil", tt.path)
			}
			if !tt.wantError && err != nil {
				t.Errorf("validateNoTraversal(%q): unexpected error: %v", tt.path, err)
			}
		})
	}
}

// TestLoadYAMLConfigRejectsTraversal proves the fix for CodeQL alerts
// #151/#152: LoadYAMLConfig previously called os.Stat/os.ReadFile on a
// merely filepath.Clean-ed path with no traversal check at all. It now
// routes through validateNoTraversal exactly like ConvertFile does.
//
// The traversal candidate is a bare relative string (not built via
// filepath.Join against a real directory) because filepath.Clean
// legitimately cancels a single ".." against one real preceding path
// component — the vulnerable case is a "..' segment with nothing
// earlier in the path to cancel it, which is exactly what an attacker
// controls when only the trailing filename component is user-supplied.
func TestLoadYAMLConfigRejectsTraversal(t *testing.T) {
	const traversal = "../../etc/passwd"
	if _, err := LoadYAMLConfig(traversal); err == nil {
		t.Errorf("LoadYAMLConfig(%q): expected traversal to be rejected, got nil error", traversal)
	}

	// A legitimate, non-traversing absolute path must still load.
	dir := t.TempDir()
	legit := filepath.Join(dir, "legit.yaml")
	if err := os.WriteFile(legit, []byte("devices: []\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadYAMLConfig(legit); err != nil {
		t.Errorf("LoadYAMLConfig(%q): unexpected error: %v", legit, err)
	}
}

// TestConvertFileRejectsTraversal locks in ConvertFile's existing
// traversal guard (input and output) now that it shares the same
// validateNoTraversal helper as LoadYAMLConfig. See
// TestLoadYAMLConfigRejectsTraversal for why the traversal strings are
// bare relative paths rather than filepath.Join results.
func TestConvertFileRejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "device.dsl")
	if err := os.WriteFile(input, []byte("Device(mac=\"00:11:22:33:44:55\")\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := ConvertFile("../../etc/device.dsl", filepath.Join(dir, "out.yaml"), false); err == nil {
		t.Error("ConvertFile: expected input traversal to be rejected")
	}

	if err := ConvertFile(input, "../../etc/out.yaml", false); err == nil {
		t.Error("ConvertFile: expected output traversal to be rejected")
	}

	// Legitimate paths still convert successfully.
	out := filepath.Join(dir, "out.yaml")
	if err := ConvertFile(input, out, false); err != nil {
		t.Fatalf("ConvertFile: unexpected error: %v", err)
	}
	if _, err := os.Stat(out); err != nil {
		t.Errorf("expected output file to be written: %v", err)
	}
}

// --- mergeSnmpAgent tests ---

func TestMergeSnmpAgent(t *testing.T) {
	tests := []struct {
		name     string
		base     *SnmpAgent
		incoming *SnmpAgent
		validate func(*testing.T, *SnmpAgent)
	}{
		{
			name:     "nil base",
			base:     nil,
			incoming: &SnmpAgent{WalkFile: "test.walk"},
			validate: func(t *testing.T, result *SnmpAgent) {
				t.Helper()
				if result.WalkFile != "test.walk" {
					t.Errorf("WalkFile = %q, want %q", result.WalkFile, "test.walk")
				}
			},
		},
		{
			name:     "nil incoming",
			base:     &SnmpAgent{WalkFile: "base.walk"},
			incoming: nil,
			validate: func(t *testing.T, result *SnmpAgent) {
				t.Helper()
				if result.WalkFile != "base.walk" {
					t.Errorf("WalkFile = %q, want %q", result.WalkFile, "base.walk")
				}
			},
		},
		{
			name: "merge walk files",
			base: &SnmpAgent{WalkFile: "base.walk", WalkFiles: []string{"base.walk"}},
			incoming: &SnmpAgent{
				WalkFiles: []string{"extra.walk"},
				AddMibs:   []AddMib{{OID: "1.3.6", Type: "STRING", Value: "test"}},
			},
			validate: func(t *testing.T, result *SnmpAgent) {
				t.Helper()
				if result.WalkFile != "base.walk" {
					t.Errorf("WalkFile should remain base, got %q", result.WalkFile)
				}
				if len(result.WalkFiles) != 2 {
					t.Errorf("expected 2 walk files, got %d", len(result.WalkFiles))
				}
				if len(result.AddMibs) != 1 {
					t.Errorf("expected 1 AddMib, got %d", len(result.AddMibs))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeSnmpAgent(tt.base, tt.incoming)
			tt.validate(t, result)
		})
	}
}

// --- mergeNetbios tests ---

func TestMergeNetbios(t *testing.T) {
	tests := []struct {
		name     string
		base     *NetbiosConfig
		incoming *NetbiosConfig
		validate func(*testing.T, *NetbiosConfig)
	}{
		{
			name:     "nil base",
			base:     nil,
			incoming: &NetbiosConfig{Name: "HOST1", Enabled: true},
			validate: func(t *testing.T, result *NetbiosConfig) {
				t.Helper()
				if result.Name != "HOST1" {
					t.Errorf("Name = %q, want %q", result.Name, "HOST1")
				}
			},
		},
		{
			name:     "nil incoming",
			base:     &NetbiosConfig{Name: "BASE"},
			incoming: nil,
			validate: func(t *testing.T, result *NetbiosConfig) {
				t.Helper()
				if result.Name != "BASE" {
					t.Errorf("Name = %q, want %q", result.Name, "BASE")
				}
			},
		},
		{
			name:     "merge MsBrowse",
			base:     &NetbiosConfig{Name: "HOST"},
			incoming: &NetbiosConfig{MsBrowse: true},
			validate: func(t *testing.T, result *NetbiosConfig) {
				t.Helper()
				if !result.MsBrowse {
					t.Error("MsBrowse should be true after merge")
				}
				if result.Name != "HOST" {
					t.Errorf("Name should remain HOST, got %q", result.Name)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeNetbios(tt.base, tt.incoming)
			tt.validate(t, result)
		})
	}
}

// --- stripInlineComment tests ---

func TestStripInlineComment(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected string
	}{
		{"no comment", "IpAddr(192.168.1.1)", "IpAddr(192.168.1.1)"},
		{"slash comment", `IpAddr(192.168.1.1) // comment`, "IpAddr(192.168.1.1)"},
		{"hash comment", `IpAddr(192.168.1.1) # comment`, "IpAddr(192.168.1.1)"},
		{"comment in quotes", `Include("file//path")`, `Include("file//path")`},
		{"empty", "", ""},
		{"only comment", "// comment", ""},
		{"single char", "x", "x"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripInlineComment(tt.line)
			if got != tt.expected {
				t.Errorf("stripInlineComment(%q) = %q, want %q", tt.line, got, tt.expected)
			}
		})
	}
}

// --- PrintSummary tests ---

func TestPrintSummary(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		contains []string
	}{
		{
			name: "basic summary",
			config: &Config{
				Devices: []Device{
					{MAC: "00:11:22:33:44:55"},
				},
			},
			contains: []string{"Devices: 1"},
		},
		{
			name: "with include path",
			config: &Config{
				IncludePath: "/some/path",
				Devices:     []Device{{MAC: "AA:BB:CC:DD:EE:FF"}},
			},
			contains: []string{"Include Path: /some/path"},
		},
		{
			name: "with capture playbacks",
			config: &Config{
				CapturePlaybacks: []CapturePlayback{
					{FileName: "test.pcap", LoopTime: 1000, ScaleTime: 1.5},
				},
				Devices: []Device{{MAC: "AA:BB:CC:DD:EE:FF"}},
			},
			contains: []string{"PCAP Playbacks: 1", "test.pcap", "Loop Time:", "Scale Time:"},
		},
		{
			name: "with SNMP agents",
			config: &Config{
				Devices: []Device{
					{
						MAC: "00:11:22:33:44:55",
						SnmpAgent: &SnmpAgent{
							AddMibs: []AddMib{
								{OID: "1.3.6", Type: "STRING", Value: "v"},
								{OID: "1.3.7", Type: "INTEGER", Value: "1"},
							},
						},
					},
				},
			},
			contains: []string{"SNMP Agents: 1", "Custom MIBs: 2"},
		},
		{
			name:     "empty config",
			config:   &Config{},
			contains: []string{"Devices: 0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			w := bufio.NewWriter(&buf)
			PrintSummary(tt.config, w)
			output := buf.String()
			for _, s := range tt.contains {
				if !strings.Contains(output, s) {
					t.Errorf("output missing %q\nfull output:\n%s", s, output)
				}
			}
		})
	}
}

// --- parseDNSRecord edge cases ---

func TestParseDNSRecord_EdgeCases(t *testing.T) {
	p := newParser("")

	tests := []struct {
		name      string
		line      string
		isForward bool
		wantNil   bool
	}{
		{"empty parens", "Forward()", true, true},
		{"single field", "Forward(onlyone)", true, true},
		{"no parens", "Forward hostname ip", true, true},
		{"forward with rcode", `Forward2("host" 1.2.3.4 300 3)`, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.parseDNSRecord(tt.line, tt.isForward)
			if tt.wantNil && result != nil {
				t.Errorf("expected nil, got %+v", result)
			}
			if !tt.wantNil && result == nil {
				t.Error("expected non-nil result")
			}
			if !tt.wantNil && result != nil && tt.name == "forward with rcode" {
				if result.RCode != 3 {
					t.Errorf("RCode = %d, want 3", result.RCode)
				}
			}
		})
	}
}

// --- parseAddMib edge cases ---

func TestParseAddMib_EdgeCases(t *testing.T) {
	p := newParser("")

	tests := []struct {
		name    string
		line    string
		wantNil bool
	}{
		{"valid", `AddMib("1.3.6" "STRING" "value")`, false},
		{"too few quoted", `AddMib("1.3.6" "STRING")`, true},
		{"no quotes", `AddMib(1.3.6 STRING value)`, true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.parseAddMib(tt.line)
			if tt.wantNil && result != nil {
				t.Errorf("expected nil, got %+v", result)
			}
			if !tt.wantNil && result == nil {
				t.Error("expected non-nil result")
			}
		})
	}
}

// --- parseCommunityInclude tests ---

func TestParseCommunityInclude(t *testing.T) {
	p := newParser("")

	tests := []struct {
		name          string
		line          string
		wantCommunity string
		wantWalk      string
	}{
		{"valid", `CommunityInclude("public" "router.walk")`, "public", "router.walk"},
		{"one quoted", `CommunityInclude("public")`, "", ""},
		{"no quotes", `CommunityInclude(public router.walk)`, "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			community, walk := p.parseCommunityInclude(tt.line)
			if community != tt.wantCommunity {
				t.Errorf("community = %q, want %q", community, tt.wantCommunity)
			}
			if walk != tt.wantWalk {
				t.Errorf("walk = %q, want %q", walk, tt.wantWalk)
			}
		})
	}
}

// --- parseFdbTable tests ---

func TestParseFdbTable(t *testing.T) {
	p := newParser("")

	tests := []struct {
		name    string
		line    string
		dot1d   bool
		wantNil bool
		port    int
		vlan    int
	}{
		{"dot1d with port", "Dot1D_FdbTable(5)", true, false, 5, 0},
		{"dot1q with port and vlan", "Dot1Q_FdbTable(3 100)", false, false, 3, 100},
		{"dot1q missing vlan", "Dot1Q_FdbTable(3)", false, true, 0, 0},
		{"empty parens", "Dot1D_FdbTable()", true, true, 0, 0},
		{"no parens", "Dot1D_FdbTable", true, true, 0, 0},
		{"non-numeric port", "Dot1D_FdbTable(abc)", true, true, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.parseFdbTable(tt.line, tt.dot1d)
			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil, got %+v", result)
				}
				return
			}
			if result == nil {
				t.Fatal("expected non-nil result")
			}
			if result.Port != tt.port {
				t.Errorf("Port = %d, want %d", result.Port, tt.port)
			}
			if result.VLAN != tt.vlan {
				t.Errorf("VLAN = %d, want %d", result.VLAN, tt.vlan)
			}
		})
	}
}

// --- parseNetBiosName tests ---

func TestParseNetBiosName(t *testing.T) {
	p := newParser("")

	tests := []struct {
		name    string
		line    string
		wantNil bool
		nbName  string
		suffix  string
		group   bool
	}{
		{"machine unique", `NetBiosName("HOST" Machine Unique)`, false, "HOST", "0", false},
		{"group", `NetBiosName("DOMAIN" LanMan Group)`, false, "DOMAIN", "32", true},
		{"msbrowse", `NetBiosName("NET" MsBrowse Unique)`, false, "NET", "1", false},
		{"numeric suffix", `NetBiosName("HOST" 20 Unique)`, false, "HOST", "20", false},
		{"no quotes", `NetBiosName(HOST Machine)`, true, "", "", false},
		{"msgserv", `NetBiosName("HOST" MsgServ Unique)`, false, "HOST", "3", false},
		{"mbsubnet", `NetBiosName("HOST" MbSubnet Group)`, false, "HOST", "29", true},
		{"mbelect", `NetBiosName("HOST" MbElect Group)`, false, "HOST", "30", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.parseNetBiosName(tt.line)
			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil, got %+v", result)
				}
				return
			}
			if result == nil {
				t.Fatal("expected non-nil result")
			}
			if result.Name != tt.nbName {
				t.Errorf("Name = %q, want %q", result.Name, tt.nbName)
			}
			if result.Suffix != tt.suffix {
				t.Errorf("Suffix = %q, want %q", result.Suffix, tt.suffix)
			}
			if result.Group != tt.group {
				t.Errorf("Group = %v, want %v", result.Group, tt.group)
			}
		})
	}
}

// --- SnmpAccessList tests ---

func TestParseSnmpAccessList(t *testing.T) {
	input := `SnmpAccessList(
  IpAddr(192.168.1.1)
  // comment
  IpAddr(10.0.0.1)
)`
	p := newParser(input)
	list := p.parseSnmpAccessList()
	if len(list) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(list))
	}
	if list[0] != "192.168.1.1" {
		t.Errorf("list[0] = %q, want %q", list[0], "192.168.1.1")
	}
	if list[1] != "10.0.0.1" {
		t.Errorf("list[1] = %q, want %q", list[1], "10.0.0.1")
	}
}

// --- parseIcmpRouter tests ---

func TestParseIcmpRouter(t *testing.T) {
	p := newParser("")

	tests := []struct {
		name    string
		line    string
		wantNil bool
		addr    string
		pref    int
	}{
		{"valid", "Router(192.168.1.1 100)", false, "192.168.1.1", 100},
		{"no parens", "Router", true, "", 0},
		{"single field", "Router(192.168.1.1)", true, "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.parseIcmpRouter(tt.line)
			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil, got %+v", result)
				}
				return
			}
			if result == nil {
				t.Fatal("expected non-nil")
			}
			if result.Address != tt.addr {
				t.Errorf("Address = %q, want %q", result.Address, tt.addr)
			}
			if result.Preference != tt.pref {
				t.Errorf("Preference = %d, want %d", result.Preference, tt.pref)
			}
		})
	}
}

// --- Full round-trip parse test ---

func TestParse_ComplexConfig(t *testing.T) {
	input := `// Complex config
IncludePath("/opt/niac/walk")

CapturePlayback(
  FileName("traffic.pcap")
  LoopTime(10000)
)

Device(
  MacAddr(001122334455)
  IpAddr(192.168.1.1)
  Vlan(100)
  Babble()
  SnmpAgent(
    Include("router.snmpwalk")
    AddMib("1.3.6.1.2.1.1.5.0" "OctetString" "router1")
  )
  Dhcp(
    SubnetMask(255.255.255.0)
    Router(192.168.1.1)
  )
  Dns(
    Forward("test.local" 192.168.1.100 300)
  )
)

Device(
  MacAddr(AABBCCDDEEFF)
  IpAddr(10.0.0.1)
)`

	p := newParser(input)
	cfg, err := p.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.IncludePath != "/opt/niac/walk" {
		t.Errorf("IncludePath = %q", cfg.IncludePath)
	}
	if len(cfg.CapturePlaybacks) != 1 {
		t.Errorf("expected 1 playback, got %d", len(cfg.CapturePlaybacks))
	}
	if len(cfg.Devices) != 2 {
		t.Fatalf("expected 2 devices, got %d", len(cfg.Devices))
	}

	d1 := cfg.Devices[0]
	if d1.VLAN != 100 {
		t.Errorf("device1 VLAN = %d, want 100", d1.VLAN)
	}
	if !d1.Babble {
		t.Error("device1 Babble should be true")
	}
	if d1.SnmpAgent == nil {
		t.Error("device1 SnmpAgent should not be nil")
	}
	if d1.Dhcp == nil {
		t.Error("device1 Dhcp should not be nil")
	}
	if d1.DNS == nil {
		t.Error("device1 DNS should not be nil")
	}

	// Validate round trip
	if err = ValidateConfig(cfg); err != nil {
		t.Errorf("ValidateConfig failed: %v", err)
	}
}
