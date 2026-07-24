package snmp

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

func TestMIBIIProtocolScalarConformance(t *testing.T) {
	agent := NewAgent(createTestDevice(), 0)

	required := make(map[string]gosnmp.Asn1BER, 49)
	for id := 1; id <= 26; id++ {
		required[fmt.Sprintf("%s.%d.0", icmpMIBRoot, id)] = gosnmp.Counter32
	}
	for _, id := range []int{1, 2, 3, 4} {
		required[fmt.Sprintf("%s.%d.0", tcpMIBRoot, id)] = gosnmp.Integer
	}
	for _, id := range []int{5, 6, 7, 8, 10, 11, 12, 14, 15} {
		required[fmt.Sprintf("%s.%d.0", tcpMIBRoot, id)] = gosnmp.Counter32
	}
	required[tcpMIBRoot+".9.0"] = gosnmp.Gauge32
	for id := 1; id <= 4; id++ {
		required[fmt.Sprintf("%s.%d.0", udpMIBRoot, id)] = gosnmp.Counter32
		required[fmt.Sprintf("%s.%d.0", egpMIBRoot, id)] = gosnmp.Counter32
	}
	required[egpMIBRoot+".6.0"] = gosnmp.Integer

	for oid, wantType := range required {
		value := agent.mib.Get(oid)
		if value == nil {
			t.Errorf("mandatory scalar %s is not registered", oid)
			continue
		}
		if value.Type != wantType {
			t.Errorf("%s type = %v, want %v", oid, value.Type, wantType)
		}
	}
}

func TestMIBIIConfiguredListenerTables(t *testing.T) {
	device := createTestDevice()
	device.HTTPConfig = &config.HTTPConfig{Enabled: true}
	device.FTPConfig = &config.FTPConfig{Enabled: true}
	device.NetBIOSConfig = &config.NetBIOSConfig{Enabled: true}
	device.IPerf3 = &config.IPerf3Config{Enabled: true, Port: 5202}
	device.SSHConfig = &config.SSHConfig{Enabled: true}
	device.DHCPConfig = &config.DHCPConfig{}
	device.DNSConfig = &config.DNSConfig{}
	agent := NewAgent(device, 0)

	for _, port := range []int{21, 22, 80, 139, 5202} {
		index := fmt.Sprintf("192.168.1.1.%d.0.0.0.0.0", port)
		for column := 1; column <= 5; column++ {
			oid := fmt.Sprintf("%s.13.1.%d.%s", tcpMIBRoot, column, index)
			if agent.mib.Get(oid) == nil {
				t.Errorf("configured TCP listener column missing: %s", oid)
			}
		}
	}
	for _, port := range []int{53, 67, 137, 138, 161} {
		index := fmt.Sprintf("192.168.1.1.%d", port)
		for column := 1; column <= 2; column++ {
			oid := fmt.Sprintf("%s.5.1.%d.%s", udpMIBRoot, column, index)
			if agent.mib.Get(oid) == nil {
				t.Errorf("configured UDP listener column missing: %s", oid)
			}
		}
	}
}

func TestMIBIIUnconfiguredTablesAreEmpty(t *testing.T) {
	agent := NewAgent(createTestDevice(), 0)

	for _, prefix := range []string{tcpMIBRoot + ".13.1", egpMIBRoot + ".5.1"} {
		for _, oid := range agent.mib.AllOIDs() {
			if strings.HasPrefix(oid, prefix+".") {
				t.Errorf("unconfigured table %s contains invented row %s", prefix, oid)
			}
		}
	}
	if agent.mib.Get(udpMIBRoot+".5.1.1.192.168.1.1.161") == nil {
		t.Error("SNMP agent UDP listener is not exposed")
	}
}

func TestMIBIIEGPNeighborColumnsRemainEmpty(t *testing.T) {
	agent := NewAgent(createTestDevice(), 0)

	for column := 1; column <= 15; column++ {
		prefix := fmt.Sprintf("%s.5.1.%d.", egpMIBRoot, column)
		for _, oid := range agent.mib.AllOIDs() {
			if strings.HasPrefix(oid, prefix) {
				t.Errorf("EGP neighbor column %d contains invented row %s", column, oid)
			}
		}
	}
}
