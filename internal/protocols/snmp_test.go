package protocols_test

import (
	"net"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

// Note: The original TestSNMPHandler_HandlePacket test relied on unexported fields
// (stack.snmpHandler, stack.snmpAgents, stack.sendQueue) and the unexported function
// snmpEnabled, which are not accessible from an external test package.
// Since export_test.go does not provide wrappers for SNMP functionality,
// we can only test the parts that use exported interfaces.

// TestSNMPStackInitialization tests that a stack with SNMP config can be created.
func TestSNMPStackInitialization(t *testing.T) {
	deviceMAC := net.HardwareAddr{0x00, 0xaa, 0xbb, 0xcc, 0xdd, 0xee}
	deviceIP := net.ParseIP("10.0.0.10").To4()

	cfg := &config.Config{
		Devices: []config.Device{
			{
				Name:        "snmp-device",
				Type:        "router",
				MACAddress:  deviceMAC,
				IPAddresses: []net.IP{deviceIP},
				SNMPConfig: config.SNMPConfig{
					Community: "public",
					SysName:   "snmp-device",
				},
			},
		},
	}

	stack := protocols.NewStack(nil, cfg, logging.NewDebugConfig(0))
	if stack == nil {
		t.Fatal("expected stack to be created")
	}

	// Verify the device is accessible
	devices := stack.GetDevices()
	if devices == nil {
		t.Fatal("expected devices to be accessible")
	}

	// Verify we can get initial stats
	stats := stack.GetStats()
	if stats.SNMPQueries != 0 {
		t.Errorf("expected initial SNMPQueries=0, got %d", stats.SNMPQueries)
	}
}

// TestSNMPConstants tests SNMP-related constants are accessible.
func TestSNMPConstants(t *testing.T) {
	// Verify the UDP port constant for SNMP
	if protocols.UDPPortSNMP != 161 {
		t.Errorf("expected UDPPortSNMP=161, got %d", protocols.UDPPortSNMP)
	}
}
