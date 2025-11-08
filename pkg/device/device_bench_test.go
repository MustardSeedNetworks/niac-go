package device

import (
	"net"
	"testing"

	"github.com/krisarmstrong/niac-go/pkg/config"
	"github.com/krisarmstrong/niac-go/pkg/errors"
	"github.com/krisarmstrong/niac-go/pkg/logging"
	"github.com/krisarmstrong/niac-go/pkg/protocols"
)

// BenchmarkDeviceCreation benchmarks creating a new simulated device
func BenchmarkDeviceCreation(b *testing.B) {
	mac, _ := net.ParseMAC("00:11:22:33:44:55")
	ip := net.ParseIP("192.168.1.1")

	cfg := &config.Config{
		Devices: []config.Device{
			{
				Name:        "test-device",
				Type:        "router",
				MACAddress:  mac,
				IPAddresses: []net.IP{ip},
				SNMPConfig: config.SNMPConfig{
					Community:   "public",
					SysName:     "test-device",
					SysDescr:    "Test Router",
					SysContact:  "admin@test.com",
					SysLocation: "Test Lab",
				},
			},
		},
	}

	debugConfig := logging.NewDebugConfig(0)
	stack := protocols.NewStack(nil, cfg, debugConfig)
	errorMgr := errors.NewStateManager()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = NewSimulator(cfg, stack, errorMgr, 0)
	}
}

// BenchmarkDeviceCreation_WithMultipleIPs benchmarks device creation with multiple IP addresses
func BenchmarkDeviceCreation_WithMultipleIPs(b *testing.B) {
	mac, _ := net.ParseMAC("00:11:22:33:44:55")
	ips := []net.IP{
		net.ParseIP("192.168.1.1"),
		net.ParseIP("192.168.1.2"),
		net.ParseIP("10.0.0.1"),
		net.ParseIP("2001:db8::1"),
	}

	cfg := &config.Config{
		Devices: []config.Device{
			{
				Name:        "test-device",
				Type:        "router",
				MACAddress:  mac,
				IPAddresses: ips,
				SNMPConfig: config.SNMPConfig{
					Community: "public",
					SysName:   "test-device",
				},
			},
		},
	}

	debugConfig := logging.NewDebugConfig(0)
	stack := protocols.NewStack(nil, cfg, debugConfig)
	errorMgr := errors.NewStateManager()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = NewSimulator(cfg, stack, errorMgr, 0)
	}
}

// BenchmarkDeviceCreation_MultipleDevices benchmarks creating multiple devices
func BenchmarkDeviceCreation_MultipleDevices(b *testing.B) {
	devices := make([]config.Device, 10)
	for i := 0; i < 10; i++ {
		mac, _ := net.ParseMAC("00:11:22:33:44:55")
		mac[5] = byte(i)
		ip := net.ParseIP("192.168.1.1")
		ip[15] = byte(i + 1)

		devices[i] = config.Device{
			Name:        "device-" + string(rune('0'+i)),
			Type:        "router",
			MACAddress:  mac,
			IPAddresses: []net.IP{ip},
			SNMPConfig: config.SNMPConfig{
				Community: "public",
				SysName:   "device-" + string(rune('0'+i)),
			},
		}
	}

	cfg := &config.Config{Devices: devices}
	debugConfig := logging.NewDebugConfig(0)
	stack := protocols.NewStack(nil, cfg, debugConfig)
	errorMgr := errors.NewStateManager()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = NewSimulator(cfg, stack, errorMgr, 0)
	}
}

// BenchmarkProtocolHandlerRegistration benchmarks registering protocol handlers
func BenchmarkProtocolHandlerRegistration(b *testing.B) {
	mac, _ := net.ParseMAC("00:11:22:33:44:55")
	ip := net.ParseIP("192.168.1.1")

	cfg := &config.Config{
		Devices: []config.Device{
			{
				Name:        "test-device",
				Type:        "router",
				MACAddress:  mac,
				IPAddresses: []net.IP{ip},
				SNMPConfig: config.SNMPConfig{
					Community: "public",
					SysName:   "test-device",
				},
			},
		},
	}

	debugConfig := logging.NewDebugConfig(0)
	stack := protocols.NewStack(nil, cfg, debugConfig)
	errorMgr := errors.NewStateManager()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		sim := NewSimulator(cfg, stack, errorMgr, 0)
		// Protocol handlers are registered during initialization
		_ = sim
	}
}

// BenchmarkDeviceStateLookup benchmarks looking up device state
func BenchmarkDeviceStateLookup(b *testing.B) {
	mac, _ := net.ParseMAC("00:11:22:33:44:55")
	ip := net.ParseIP("192.168.1.1")

	cfg := &config.Config{
		Devices: []config.Device{
			{
				Name:        "test-device",
				Type:        "router",
				MACAddress:  mac,
				IPAddresses: []net.IP{ip},
				SNMPConfig: config.SNMPConfig{
					Community: "public",
					SysName:   "test-device",
				},
			},
		},
	}

	debugConfig := logging.NewDebugConfig(0)
	stack := protocols.NewStack(nil, cfg, debugConfig)
	errorMgr := errors.NewStateManager()
	sim := NewSimulator(cfg, stack, errorMgr, 0)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		dev := sim.GetDevice("test-device")
		if dev != nil {
			_ = dev.State
		}
	}
}

// BenchmarkDeviceCounterIncrement benchmarks incrementing device counters
func BenchmarkDeviceCounterIncrement(b *testing.B) {
	mac, _ := net.ParseMAC("00:11:22:33:44:55")
	ip := net.ParseIP("192.168.1.1")

	cfg := &config.Config{
		Devices: []config.Device{
			{
				Name:        "test-device",
				Type:        "router",
				MACAddress:  mac,
				IPAddresses: []net.IP{ip},
				SNMPConfig: config.SNMPConfig{
					Community: "public",
					SysName:   "test-device",
				},
			},
		},
	}

	debugConfig := logging.NewDebugConfig(0)
	stack := protocols.NewStack(nil, cfg, debugConfig)
	errorMgr := errors.NewStateManager()
	sim := NewSimulator(cfg, stack, errorMgr, 0)
	dev := sim.GetDevice("test-device")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		dev.Counters.PacketsReceived++
		dev.Counters.PacketsSent++
	}
}

// BenchmarkDeviceWithLLDPConfig benchmarks device creation with LLDP configuration
func BenchmarkDeviceWithLLDPConfig(b *testing.B) {
	mac, _ := net.ParseMAC("00:11:22:33:44:55")
	ip := net.ParseIP("192.168.1.1")

	cfg := &config.Config{
		Devices: []config.Device{
			{
				Name:        "test-device",
				Type:        "switch",
				MACAddress:  mac,
				IPAddresses: []net.IP{ip},
				SNMPConfig: config.SNMPConfig{
					Community: "public",
					SysName:   "test-device",
				},
				LLDPConfig: &config.LLDPConfig{
					Enabled:           true,
					AdvertiseInterval: 30,
					TTL:               120,
					SystemDescription: "Test Switch",
					PortDescription:   "Port 1",
					ChassisIDType:     "mac",
				},
			},
		},
	}

	debugConfig := logging.NewDebugConfig(0)
	stack := protocols.NewStack(nil, cfg, debugConfig)
	errorMgr := errors.NewStateManager()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = NewSimulator(cfg, stack, errorMgr, 0)
	}
}

// BenchmarkDeviceWithDHCPConfig benchmarks device creation with DHCP server configuration
func BenchmarkDeviceWithDHCPConfig(b *testing.B) {
	mac, _ := net.ParseMAC("00:11:22:33:44:55")
	ip := net.ParseIP("192.168.1.1")
	router := net.ParseIP("192.168.1.1")
	dns := net.ParseIP("8.8.8.8")
	subnetMask := net.IPv4Mask(255, 255, 255, 0)

	cfg := &config.Config{
		Devices: []config.Device{
			{
				Name:        "test-device",
				Type:        "router",
				MACAddress:  mac,
				IPAddresses: []net.IP{ip},
				SNMPConfig: config.SNMPConfig{
					Community: "public",
					SysName:   "test-device",
				},
				DHCPConfig: &config.DHCPConfig{
					SubnetMask:       subnetMask,
					Router:           router,
					DomainNameServer: []net.IP{dns},
					ServerIdentifier: ip,
					DomainName:       "example.com",
				},
			},
		},
	}

	debugConfig := logging.NewDebugConfig(0)
	stack := protocols.NewStack(nil, cfg, debugConfig)
	errorMgr := errors.NewStateManager()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = NewSimulator(cfg, stack, errorMgr, 0)
	}
}

// BenchmarkDeviceWithFullConfig benchmarks device creation with complete configuration
func BenchmarkDeviceWithFullConfig(b *testing.B) {
	mac, _ := net.ParseMAC("00:11:22:33:44:55")
	ips := []net.IP{
		net.ParseIP("192.168.1.1"),
		net.ParseIP("2001:db8::1"),
	}
	router := net.ParseIP("192.168.1.1")
	dns := net.ParseIP("8.8.8.8")
	subnetMask := net.IPv4Mask(255, 255, 255, 0)

	cfg := &config.Config{
		Devices: []config.Device{
			{
				Name:        "test-device",
				Type:        "router",
				MACAddress:  mac,
				IPAddresses: ips,
				Interfaces: []config.Interface{
					{
						Name:        "eth0",
						Speed:       1000,
						Duplex:      "full",
						AdminStatus: "up",
						OperStatus:  "up",
						Description: "Management Interface",
					},
				},
				SNMPConfig: config.SNMPConfig{
					Community:   "public",
					SysName:     "test-device",
					SysDescr:    "Test Router",
					SysContact:  "admin@test.com",
					SysLocation: "Test Lab",
				},
				LLDPConfig: &config.LLDPConfig{
					Enabled:           true,
					AdvertiseInterval: 30,
					TTL:               120,
					SystemDescription: "Test Router",
					ChassisIDType:     "mac",
				},
				CDPConfig: &config.CDPConfig{
					Enabled:           true,
					AdvertiseInterval: 60,
					Holdtime:          180,
					Version:           2,
					SoftwareVersion:   "1.0.0",
					Platform:          "Test Platform",
				},
				DHCPConfig: &config.DHCPConfig{
					SubnetMask:       subnetMask,
					Router:           router,
					DomainNameServer: []net.IP{dns},
					ServerIdentifier: ips[0],
					DomainName:       "example.com",
				},
				ICMPConfig: &config.ICMPConfig{
					Enabled:   true,
					TTL:       64,
					RateLimit: 0,
				},
			},
		},
	}

	debugConfig := logging.NewDebugConfig(0)
	stack := protocols.NewStack(nil, cfg, debugConfig)
	errorMgr := errors.NewStateManager()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = NewSimulator(cfg, stack, errorMgr, 0)
	}
}
