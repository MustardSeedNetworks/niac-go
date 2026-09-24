package snmp

import (
	"strings"
	"testing"
)

const upsMIBRoot = "1.3.6.1.2.1.33"

func riserUPS() *Agent {
	device := createTestDevice()
	device.Name = "HQ-UPS-01"
	device.Type = "iot"
	device.SNMPConfig.SysName = "HQ-UPS-01"
	device.Properties = map[string]string{
		"sysObjectID": APCUPSSysObjectID,
		"model":       "Smart-UPS SRT 3000",
		"software":    "AOS 3.0.3",
	}

	return NewAgent(device, 0)
}

// TestUPSAnswersTheUPSMIB is the discovery probe for power gear: a manager
// that finds an APC Network Management Card walks RFC 1628 next, and a device
// that answers nothing there reads as an unclassified host.
func TestUPSAnswersTheUPSMIB(t *testing.T) {
	ups := riserUPS()

	next, value := ups.mib.GetNext(upsMIBRoot)
	if value == nil || next != upsIdentManufacturer {
		t.Fatalf("GETNEXT %s = %s, want %s", upsMIBRoot, next, upsIdentManufacturer)
	}
}

// TestUPSReportsANormalOnlineState pins the objects monitoring polls. A UPS on
// mains with a full battery reports exactly this; anything else pages someone.
func TestUPSReportsANormalOnlineState(t *testing.T) {
	ups := riserUPS()

	if got := oidValueString(mustGet(t, ups, upsIdentManufacturer)); got != "American Power Conversion Corp." {
		t.Errorf("upsIdentManufacturer = %q", got)
	}
	if got := oidValueString(mustGet(t, ups, upsIdentModel)); got != "Smart-UPS SRT 3000" {
		t.Errorf("upsIdentModel = %q, want the authored model", got)
	}
	if got := oidValueString(mustGet(t, ups, upsIdentAgentSoftwareVersion)); got != "AOS 3.0.3" {
		t.Errorf("upsIdentAgentSoftwareVersion = %q, want the authored software", got)
	}
	if got := oidValueString(mustGet(t, ups, upsIdentName)); got != "HQ-UPS-01" {
		t.Errorf("upsIdentName = %q, want the authored sysName", got)
	}
	wantInt(t, mustGet(t, ups, upsBatteryStatus), upsBatteryStatusNormal, "upsBatteryStatus")
	wantInt(t, mustGet(t, ups, upsSecondsOnBattery), 0, "upsSecondsOnBattery")
	wantInt(t, mustGet(t, ups, upsEstimatedChargeRemaining), upsFullCharge, "upsEstimatedChargeRemaining")
	wantInt(t, mustGet(t, ups, upsOutputSource), upsOutputSourceNormal, "upsOutputSource")
	if got := mustGet(t, ups, upsAlarmsPresent).Value; got != uint32(0) {
		t.Errorf("upsAlarmsPresent = %v, want 0", got)
	}
}

// TestUPSMIBOnlyOnUPSAgents is the other half: a switch that answered inside
// .33 would be filed as power equipment by the same manager.
func TestUPSMIBOnlyOnUPSAgents(t *testing.T) {
	agent := NewAgent(createTestDevice(), 0)

	if next, _ := agent.mib.GetNext(upsMIBRoot); strings.HasPrefix(next, upsMIBRoot+".") {
		t.Fatalf("a router answered GETNEXT %s with %s", upsMIBRoot, next)
	}
}

// TestUPSMIBAfterAWalkThatIdentifiesAUPS covers a capture of an NMC that
// carried only its system group: the walk's own sysObjectID is what names it.
func TestUPSMIBAfterAWalkThatIdentifiesAUPS(t *testing.T) {
	walk := writeTestWalk(t, ""+
		".1.3.6.1.2.1.1.2.0 = OID: ."+APCUPSSysObjectID+"\r\n"+
		".1.3.6.1.2.1.1.5.0 = STRING: HQ-UPS-02\r\n")
	agent := walkBackedUPS(t, walk)

	sysName := oidValueString(mustGet(t, agent, "1.3.6.1.2.1.1.5.0"))
	if got := oidValueString(mustGet(t, agent, upsIdentName)); got != sysName {
		t.Errorf("upsIdentName = %q, want the served sysName %q", got, sysName)
	}
}

// TestUPSMIBNotSynthesizedOverACaptureThatHasIt: a real UPS is the authority on
// its own UPS-MIB.
func TestUPSMIBNotSynthesizedOverACaptureThatHasIt(t *testing.T) {
	walk := writeTestWalk(t, ""+
		".1.3.6.1.2.1.1.2.0 = OID: ."+APCUPSSysObjectID+"\r\n"+
		".1.3.6.1.2.1.33.1.1.5.0 = STRING: captured-name\r\n")
	agent := walkBackedUPS(t, walk)

	if got := oidValueString(mustGet(t, agent, upsIdentName)); got != "captured-name" {
		t.Errorf("upsIdentName = %q, want the capture's own value", got)
	}
	if value := agent.mib.Get(upsBatteryStatus); value != nil {
		t.Errorf("synthesized an object the capture did not carry: %s", oidValueString(value))
	}
}

func walkBackedUPS(t *testing.T, walk string) *Agent {
	t.Helper()
	device := createTestDevice()
	device.Type = "iot"
	device.SNMPConfig.WalkFile = walk

	agent := NewAgent(device, 0)
	if err := agent.LoadWalkFile(walk); err != nil {
		t.Fatalf("LoadWalkFile() error = %v", err)
	}

	return agent
}

// TestUPSModelFallsBackToSysDescr: an authored UPS that names no model reports
// the sysDescr it already announces, rather than an empty upsIdentModel.
func TestUPSModelFallsBackToSysDescr(t *testing.T) {
	device := createTestDevice()
	device.Type = "iot"
	device.SNMPConfig.SysDescr = "APC Smart-UPS 1500"
	device.Properties = map[string]string{"sysObjectID": APCUPSSysObjectID}

	if got := oidValueString(mustGet(t, NewAgent(device, 0), upsIdentModel)); got != "APC Smart-UPS 1500" {
		t.Errorf("upsIdentModel = %q, want the sysDescr", got)
	}
}
