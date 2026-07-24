package config

import (
	"net"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/MustardSeedNetworks/niac-go/internal/converter"
)

func TestMarshalConfigYAMLRoundTripsCompleteRoutedDevice(t *testing.T) {
	walk := filepath.Join(t.TempDir(), "router.walk")
	if err := os.WriteFile(walk, []byte(".1.3.6.1.2.1.1.1.0 = STRING: router\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	enabled := true
	cfg := &Config{
		IncludePath: filepath.Dir(walk),
		CapturePlayback: &CapturePlayback{
			FileName: "traffic.pcap", LoopTime: 1000, ScaleTime: 2,
		},
		Networks:    []Network{{Name: "access", Subnet: "10.254.200.0/24", VirtualVLAN: 200}},
		Attachments: []LogicalAttachment{{Name: "tester", Network: "access"}},
		DiscoveryProtocols: &DiscoveryProtocols{
			LLDP: &ProtocolConfig{Enabled: true, Interval: 30},
			CDP:  &ProtocolConfig{Enabled: true, Interval: 60},
			EDP:  &ProtocolConfig{Enabled: true, Interval: 30},
			FDP:  &ProtocolConfig{Enabled: true, Interval: 60},
		},
		Devices: []Device{completeRoundTripDevice(walk, &enabled)},
	}

	data, err := MarshalConfigYAML(cfg)
	if err != nil {
		t.Fatalf("MarshalConfigYAML() error = %v", err)
	}

	var authored converter.Config
	if err = yaml.Unmarshal(data, &authored); err != nil {
		t.Fatalf("unmarshal serialized DTO: %v\n%s", err, data)
	}
	assertCompleteAuthoredDTO(t, &authored, walk)

	roundTrip, err := LoadYAMLBytes(data)
	if err != nil {
		t.Fatalf("LoadYAMLBytes() error = %v\n%s", err, data)
	}
	assertCompleteRuntimeConfig(t, roundTrip, walk)
}

func TestMarshalConfigYAMLPreservesSegments(t *testing.T) {
	cfg := &Config{Segments: []Segment{
		{Tag: UntaggedTag, Devices: []Device{{
			Name: "native", Type: "router", MACAddress: mustMAC(t, "02:00:00:00:00:01"),
		}}},
		{Tag: 200, Devices: []Device{{
			Name: "site", Type: "switch", MACAddress: mustMAC(t, "02:00:00:00:00:02"),
		}}},
	}}

	data, err := MarshalConfigYAML(cfg)
	if err != nil {
		t.Fatalf("MarshalConfigYAML() error = %v", err)
	}
	var authored converter.Config
	if err = yaml.Unmarshal(data, &authored); err != nil {
		t.Fatalf("unmarshal serialized DTO: %v", err)
	}
	if len(authored.Segments) != 2 {
		t.Fatalf("segments = %#v", authored.Segments)
	}
	if authored.Segments[0].Tag != "untagged" || len(authored.Segments[0].Devices) != 1 {
		t.Fatalf("inline segment = %#v", authored.Segments[0])
	}
	if authored.Segments[1].Tag != "200" || len(authored.Segments[1].Devices) != 1 {
		t.Fatalf("resolved segment = %#v", authored.Segments[1])
	}
}

func completeRoundTripDevice(walk string, enabled *bool) Device {
	return Device{
		Name: "edge", Type: "router", MACAddress: mustParseMAC("02:00:00:00:00:01"),
		IPAddresses: []net.IP{net.ParseIP("10.254.200.1")}, MapToIP: net.ParseIP("192.0.2.10"),
		Babble: true, VLAN: 200, Properties: map[string]string{"vendor": "Cisco"},
		TTLConfig: &TTLConfig{TTL: 2, IP: net.ParseIP("10.240.0.1"), Mask: net.CIDRMask(24, 32)},
		Interfaces: []Interface{{
			Name: "Gi0/0", Network: "access", Address: "10.254.200.1/24", Speed: 1000,
			Duplex: "full", AdminStatus: "up", OperStatus: "up", VLANs: []int{200},
		}},
		Routes: []Route{{Destination: "10.240.0.0/16", Via: "Gi0/0", NextHop: "10.254.200.2"}},
		SNMPConfig: SNMPConfig{
			Enabled: enabled, Community: "NetAllyDemo", SysName: "edge.example",
			SysDescr: "Cisco IOS", SysContact: "noc@example.test", SysLocation: "Lab",
			WalkFile: walk, WalkFiles: []string{walk},
			AddMibs:           []AddMib{{OID: ".1.3.6.1.2.1.1.5.0", Type: "STRING", Value: "edge"}},
			CommunityIncludes: []CommunityInclude{{Community: "monitor", WalkFile: walk}},
			AccessList: []net.IP{
				net.ParseIP("10.254.200.50"),
			}, SnmpAddr: net.ParseIP("10.254.200.1"),
			Dot1DFdbTable: &FdbTableConfig{Port: 7, VLAN: 200},
			Dot1QFdbTable: &FdbTableConfig{Port: 8, VLAN: 201},
			Traps: &TrapConfig{
				Enabled: true, Receivers: []string{"10.254.200.50:162"}, Community: "traps",
				ColdStart: &TrapTriggerConfig{Enabled: true, OnStartup: true},
				LinkState: &LinkStateTrapConfig{Enabled: true, LinkDown: true, LinkUp: true},
			},
		},
		DHCPConfig: &DHCPConfig{
			SubnetMask: net.CIDRMask(24, 32), Router: net.ParseIP("10.254.200.1"),
			DomainNameServer: []net.IP{net.ParseIP("10.254.200.53")},
			ServerIdentifier: net.ParseIP(
				"10.254.200.2",
			), NextServerIP: net.ParseIP("10.254.200.3"),
			PoolStart: net.ParseIP("10.254.200.100"), PoolEnd: net.ParseIP("10.254.200.150"),
			NTPServers: []net.IP{
				net.ParseIP("10.254.200.123"),
			}, DomainSearch: []string{"example.test"},
			TFTPServerName: "boot.example.test", BootfileName: "pxelinux.0", VendorSpecific: []byte("vendor"),
			ClientLeases: []DHCPLease{{
				ClientIP: net.ParseIP(
					"10.254.200.20",
				), MACAddress: mustParseMAC("02:00:00:00:00:20"),
			}},
		},
		DNSConfig: &DNSConfig{
			ForwardRecords: []DNSRecord{
				{Name: "edge.example.test", IP: net.ParseIP("10.254.200.1"), TTL: 300},
			},
			ReverseRecords: []DNSRecord{
				{Name: "1.200.254.10.in-addr.arpa", IP: net.ParseIP("10.254.200.1"), TTL: 300},
			},
		},
		LLDPConfig: &LLDPConfig{
			Enabled:           true,
			AdvertiseInterval: 30,
			TTL:               120,
			SystemDescription: "edge",
		},
		CDPConfig: &CDPConfig{
			Enabled:           true,
			AdvertiseInterval: 60,
			Holdtime:          180,
			Version:           2,
			Platform:          "C9300",
		},
		EDPConfig: &EDPConfig{Enabled: true, AdvertiseInterval: 30, VersionString: "1"},
		FDPConfig: &FDPConfig{
			Enabled:           true,
			AdvertiseInterval: 60,
			Holdtime:          180,
			Platform:          "ICX",
		},
		STPConfig: &STPConfig{Enabled: true, BridgePriority: 4096, HelloTime: 2, Version: "rstp"},
		HTTPConfig: &HTTPConfig{
			Enabled:    true,
			ServerName: "edge",
			Endpoints:  []HTTPEndpoint{{Path: "/", Method: "GET", StatusCode: 200, Body: "ok"}},
		},
		FTPConfig: &FTPConfig{
			Enabled:       true,
			WelcomeBanner: "ready",
			Users:         []FTPUser{{Username: "demo", Password: "test", HomeDir: "/"}},
		},
		NetBIOSConfig: &NetBIOSConfig{
			Enabled: true, Name: "EDGE", Workgroup: "LAB", NodeType: "H", TTL: 300,
			Services: []string{"workstation"}, Names: []NetBIOSName{{Name: "EDGE", Suffix: 32}},
		},
		SNMPv3Config: &SNMPv3Config{Enabled: true, EngineID: "8000000001", Users: []SNMPv3User{{
			Username: "monitor", AuthProtocol: "sha256", AuthPassword: "authpass123",
			PrivProtocol: "aes", PrivPassword: "privpass123",
		}}},
		ICMPConfig: &ICMPConfig{
			Enabled:          true,
			TTL:              64,
			AddressMaskReply: net.ParseIP("255.255.255.0"),
		},
		ICMPv6Config: &ICMPv6Config{
			Enabled: true, HopLimit: 64, RateLimit: 100,
			RouterAdvertisement: &Icmpv6RouterAdvertisement{
				Period: 30, Lifetime: 1800, PrefixInfo: []Icmpv6PrefixInfo{{
					PrefixLength: 64, Onlink: 1, Auto: 1, Prefix: net.ParseIP("2001:db8:200::"),
				}},
			},
		},
		DHCPv6Config: &DHCPv6Config{
			Enabled: true,
			Pools: []DHCPv6Pool{{
				Network: "2001:db8:200::/64", RangeStart: "2001:db8:200::100", RangeEnd: "2001:db8:200::1ff",
			}},
			PreferredLifetime: 3600, ValidLifetime: 7200,
			DNSServers: []net.IP{
				net.ParseIP("2001:db8:200::53"),
			}, DomainList: []string{"example.test"},
		},
		OSFingerprintConfig: &OSFingerprintConfig{
			OSType:    "cisco-ios",
			TTL:       255,
			SSHBanner: "SSH-2.0-Cisco",
		},
		SSHConfig: &SSHConfig{
			Enabled: true, Username: "admin", PasswordEnv: "NIAC_TEST_SSH_PASSWORD",
		},
		SyslogConfig:    &SyslogConfig{Enabled: true, Receivers: []string{"192.0.2.50:514"}},
		IPerf3:          &IPerf3Config{Enabled: true, Port: 5201, MaxBandwidthMbps: 1000},
		ReflectorConfig: &ReflectorConfig{LatencyMs: 5, JitterMs: 1, DSCP: true},
		TrunkPorts: []TrunkPort{{
			Interface: "Gi0/1", VLANs: []int{200, 210}, NativeVLAN: 200,
			RemoteDevice: "dist", RemoteInterface: "Gi1/0/1", FDBOnly: true,
		}},
		PortChannels: []PortChannel{{ID: 1, Members: []string{"Gi0/1", "Gi0/2"}, Mode: "active"}},
	}
}

func assertCompleteAuthoredDTO(t *testing.T, cfg *converter.Config, walk string) {
	t.Helper()
	if len(cfg.Networks) != 1 || cfg.Networks[0].Subnet != "10.254.200.0/24" {
		t.Fatalf("networks = %#v", cfg.Networks)
	}
	if len(cfg.Attachments) != 1 || cfg.Attachments[0].Connect != "access" {
		t.Fatalf("attachments = %#v", cfg.Attachments)
	}
	if len(cfg.CapturePlaybacks) != 1 || cfg.CapturePlaybacks[0].ScaleTime != 2 ||
		cfg.DiscoveryProtocols.EDP.Interval != 30 {
		t.Fatalf(
			"global configuration lost: playback=%#v discovery=%#v",
			cfg.CapturePlaybacks,
			cfg.DiscoveryProtocols,
		)
	}
	assertAuthoredRoutedDevice(t, &cfg.Devices[0], walk)
}

func assertAuthoredRoutedDevice(t *testing.T, device *converter.Device, walk string) {
	t.Helper()
	if device.Interfaces[0].Network != "access" ||
		device.Interfaces[0].Address != "10.254.200.1/24" {
		t.Fatalf("interface = %#v", device.Interfaces[0])
	}
	if device.Routes[0].NextHop != "10.254.200.2" || device.Routes[0].Via != "Gi0/0" {
		t.Fatalf("route = %#v", device.Routes[0])
	}
	if device.SnmpAgent.Community != "NetAllyDemo" || device.SnmpAgent.WalkFile != walk ||
		!device.SnmpAgent.Traps.LinkState.LinkDown {
		t.Fatalf("SNMP = %#v", device.SnmpAgent)
	}
	if device.Dhcp.PoolStart != "10.254.200.100" || device.DNS.ForwardRecords[0].TTL != 300 {
		t.Fatalf("services lost: DHCP=%#v DNS=%#v", device.Dhcp, device.DNS)
	}
	assertAuthoredProtocols(t, device)
	if device.TrunkPorts[0].RemoteDevice != "dist" || !device.TrunkPorts[0].FDBOnly ||
		device.PortChannels[0].ID != 1 {
		t.Fatalf(
			"topology metadata lost: trunks=%#v channels=%#v",
			device.TrunkPorts,
			device.PortChannels,
		)
	}
}

func assertAuthoredProtocols(t *testing.T, device *converter.Device) {
	t.Helper()
	if device.Snmpv3.Users[0].AuthProtocol != "sha256" || device.HTTP.Endpoints[0].Path != "/" ||
		device.Ftp.Users[0].Username != "demo" || device.Icmp.TTL != 64 ||
		device.Icmpv6.RouterAdvertisement.PrefixInfo[0].Prefix != "2001:db8:200::" ||
		device.Dhcpv6.Pools[0].Network != "2001:db8:200::/64" || device.Netbios.Names[0].Suffix != "32" ||
		device.SSH.Username != "admin" || device.SSH.PasswordEnv != "NIAC_TEST_SSH_PASSWORD" {
		t.Fatalf("protocol surface lost: %#v", device)
	}
}

func assertCompleteRuntimeConfig(t *testing.T, cfg *Config, walk string) {
	t.Helper()
	device := cfg.Devices[0]
	if cfg.Networks[0].Subnet != "10.254.200.0/24" || cfg.Attachments[0].Network != "access" {
		t.Fatalf("routed config lost: %#v %#v", cfg.Networks, cfg.Attachments)
	}
	if device.Routes[0].NextHop != "10.254.200.2" || device.Interfaces[0].Network != "access" {
		t.Fatalf("routed device lost: %#v %#v", device.Routes, device.Interfaces)
	}
	if device.SNMPConfig.Community != "NetAllyDemo" || device.SNMPConfig.WalkFile != walk {
		t.Fatalf("SNMP identity lost: %#v", device.SNMPConfig)
	}
	if device.DHCPConfig.PoolStart.String() != "10.254.200.100" ||
		device.DNSConfig.ForwardRecords[0].Name != "edge.example.test" {
		t.Fatalf("services lost: DHCP=%#v DNS=%#v", device.DHCPConfig, device.DNSConfig)
	}
	if device.SNMPv3Config.Users[0].Username != "monitor" ||
		device.HTTPConfig.Endpoints[0].Body != "ok" {
		t.Fatalf("protocol round trip lost: %#v", device)
	}
	if device.SSHConfig == nil || device.SSHConfig.Username != "admin" ||
		device.SSHConfig.PasswordEnv != "NIAC_TEST_SSH_PASSWORD" {
		t.Fatalf("SSH round trip lost: %#v", device.SSHConfig)
	}
	if device.SyslogConfig == nil || len(device.SyslogConfig.Receivers) != 1 ||
		device.SyslogConfig.Receivers[0] != "192.0.2.50:514" {
		t.Fatalf("SYSLOG round trip lost: %#v", device.SyslogConfig)
	}
	if device.ICMPv6Config.RouterAdvertisement.PrefixInfo[0].Prefix.String() != "2001:db8:200::" ||
		device.DHCPv6Config.Pools[0].Network != "2001:db8:200::/64" || device.NetBIOSConfig.Names[0].Suffix != 32 {
		t.Fatalf("IPv6 or NetBIOS round trip lost: %#v", device)
	}
}

func TestLoadYAMLBytesValidatesManagementServices(t *testing.T) {
	tests := []struct {
		name    string
		service string
	}{
		{name: "ssh missing password reference", service: "ssh:\n      enabled: true\n      username: admin"},
		{name: "syslog invalid receiver", service: "syslog:\n      enabled: true\n      receivers: [not-an-address]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yaml := "devices:\n  - name: edge-1\n    type: router\n    mac: '02:00:00:00:00:01'\n    " + tt.service + "\n"
			if _, err := LoadYAMLBytes([]byte(yaml)); err == nil {
				t.Fatal("LoadYAMLBytes() accepted invalid management service")
			}
		})
	}
}

func TestLoadYAMLBytesValidatesSegmentManagementServices(t *testing.T) {
	yaml := `segments:
  - tag: untagged
    devices:
      - name: edge-1
        type: router
        mac: "02:00:00:00:00:01"
        ssh:
          enabled: true
          username: admin
`
	if _, err := LoadYAMLBytes([]byte(yaml)); err == nil {
		t.Fatal("LoadYAMLBytes() accepted invalid segment SSH service")
	}
}

func mustMAC(t *testing.T, value string) net.HardwareAddr {
	t.Helper()
	address, err := net.ParseMAC(value)
	if err != nil {
		t.Fatal(err)
	}
	return address
}

func mustParseMAC(value string) net.HardwareAddr {
	address, _ := net.ParseMAC(value)
	return address
}
