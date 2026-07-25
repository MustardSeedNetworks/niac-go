package snmp

import (
	"math"
	"strconv"
	"testing"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

func TestInterfaceAndBridgeCountersSharePacketTelemetry(t *testing.T) {
	device := createTestDevice()
	device.Type = "switch"
	device.SNMPConfig.WalkFile = "switch.walk"
	device.Interfaces = []config.Interface{
		{Name: "GigabitEthernet1/0/5", Address: "192.0.2.5/24"},
		{Name: "GigabitEthernet1/0/6", Address: "192.0.2.6/24"},
	}
	telemetry := NewProtocolTelemetry()
	agent := NewAgentWithCommunityAndTelemetry(device, "public", 0, telemetry)
	seedInterfaceIdentity(t, agent, "GigabitEthernet1/0/5", "10005", "5")
	seedInterfaceIdentity(t, agent, "GigabitEthernet1/0/6", "10006", "6")
	agent.refreshAuthoredInterfaceMIBs()
	agent.registerDot1dTpPortEntry(5)
	agent.registerDot1dTpPortEntry(6)

	telemetry.RecordInterfaceInbound("GigabitEthernet1/0/5", 128, false, false)
	telemetry.RecordInterfaceInbound("GigabitEthernet1/0/5", 64, true, true)
	telemetry.RecordInterfaceOutbound("GigabitEthernet1/0/5", 96, false, false)

	tests := []struct {
		oid  string
		want any
	}{
		{ifInOctets + ".10005", uint32(192)},
		{ifInUcastPkts + ".10005", uint32(1)},
		{ifInNUcastPkts + ".10005", uint32(1)},
		{ifInMulticastPkts + ".10005", uint32(0)},
		{ifInBroadcastPkts + ".10005", uint32(1)},
		{ifOutOctets + ".10005", uint32(96)},
		{ifHCInOctets + ".10005", uint64(192)},
		{ifHCInMulticastPkts + ".10005", uint64(0)},
		{ifHCInBroadcastPkts + ".10005", uint64(1)},
		{dot1dTpPortInFrames + ".5", uint32(2)},
		{dot1dTpPortOutFrames + ".5", uint32(1)},
		{ifInOctets + ".10006", uint32(0)},
		{dot1dTpPortInFrames + ".6", uint32(0)},
	}
	for _, test := range tests {
		t.Run(test.oid, func(t *testing.T) { assertMIBValue(t, agent, test.oid, test.want) })
	}
}

func TestInterfaceFaultCountersAreSNMPObservable(t *testing.T) {
	device := createTestDevice()
	device.Interfaces = []config.Interface{{
		Name: "GigabitEthernet1/0/5", Address: "192.0.2.5/24",
	}}
	telemetry := NewProtocolTelemetry()
	agent := NewAgentWithCommunityAndTelemetry(device, "public", 0, telemetry)

	telemetry.RecordInterfaceCounters("GigabitEthernet1/0/5", InterfaceCounterDelta{
		InOctets: 1_000, OutOctets: 800,
		InDiscards: 2, OutDiscards: 3,
		InErrors: 4, OutErrors: 5, FCSErrors: 6,
	})

	tests := []struct {
		oid  string
		want any
	}{
		{ifInOctets + ".1", uint32(1_000)},
		{ifHCInOctets + ".1", uint64(1_000)},
		{ifOutOctets + ".1", uint32(800)},
		{ifHCOutOctets + ".1", uint64(800)},
		{ifInDiscards + ".1", uint32(2)},
		{ifOutDiscards + ".1", uint32(3)},
		{ifInErrors + ".1", uint32(4)},
		{ifOutErrors + ".1", uint32(5)},
		{dot3StatsFCSErrors + ".1", uint32(6)},
	}
	for _, test := range tests {
		t.Run(test.oid, func(t *testing.T) { assertMIBValue(t, agent, test.oid, test.want) })
	}
}

func TestInterfaceCounter32WrapsWhileHighCapacityCounterContinues(t *testing.T) {
	device := createTestDevice()
	device.Interfaces = []config.Interface{{Name: "Gi0/1", Address: "192.0.2.5/24"}}
	telemetry := NewProtocolTelemetry()
	agent := NewAgentWithCommunityAndTelemetry(device, "public", 0, telemetry)
	seedInterfaceIdentity(t, agent, "Gi0/1", "1", "1")
	agent.refreshAuthoredInterfaceMIBs()

	telemetry.RecordInterfaceCounters("Gi0/1", InterfaceCounterDelta{InOctets: math.MaxUint32 + 5})

	assertMIBValue(t, agent, ifInOctets+".1", uint32(4))
	assertMIBValue(t, agent, ifHCInOctets+".1", uint64(math.MaxUint32+5))
}

func TestAuthoredInterfaceProblemsCorrelateAcrossMIBs(t *testing.T) {
	device := createTestDevice()
	device.SNMPConfig.WalkFile = "switch.walk"
	device.Interfaces = []config.Interface{{
		Name: "GigabitEthernet1/0/5", Address: "192.0.2.5/24", Speed: 10,
		Duplex: "half", AdminStatus: "up", OperStatus: "down", Description: "degraded uplink",
	}}
	agent := NewAgent(device, 0)
	seedInterfaceIdentity(t, agent, "GigabitEthernet1/0/5", "10005", "5")
	agent.refreshAuthoredInterfaceMIBs()

	tests := []struct {
		oid  string
		want any
	}{
		{ifSpeed + ".10005", uint32(10_000_000)},
		{ifHighSpeed + ".10005", uint32(10)},
		{ifAdminStatus + ".10005", interfaceStatusUp},
		{ifOperStatus + ".10005", interfaceStatusDown},
		{ifAlias + ".10005", "degraded uplink"},
		{dot3StatsDuplexStatus + ".10005", duplexHalf},
	}
	for _, test := range tests {
		t.Run(test.oid, func(t *testing.T) { assertMIBValue(t, agent, test.oid, test.want) })
	}
}

func seedInterfaceIdentity(t *testing.T, agent *Agent, name, ifIndex, bridgePort string) {
	t.Helper()
	index, parseErr := strconv.Atoi(ifIndex)
	if parseErr != nil {
		t.Fatal(parseErr)
	}
	if err := agent.SetOID(ifName+"."+ifIndex, &OIDValue{Type: gosnmp.OctetString, Value: name}); err != nil {
		t.Fatal(err)
	}
	if err := agent.SetOID(
		dot1dBasePortIfIndex+"."+bridgePort,
		&OIDValue{Type: gosnmp.Integer, Value: index},
	); err != nil {
		t.Fatal(err)
	}
}
