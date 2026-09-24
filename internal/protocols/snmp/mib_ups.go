package snmp

import (
	"strings"

	"github.com/gosnmp/gosnmp"
)

// APCUPSSysObjectID is the sysObjectID an APC Network Management Card reports
// from inside a Smart-UPS. A discovery tool reads it first and then walks RFC
// 1628, which is what makes the device a UPS on its map.
const APCUPSSysObjectID = "1.3.6.1.4.1.318.1.3.27"

// apcManufacturer is what an APC card reports in upsIdentManufacturer.
const apcManufacturer = "American Power Conversion Corp."

// UPS-MIB (RFC 1628). Only objects NIAC can state truthfully for a UPS on
// mains with a charged battery are served: the input, output and bypass line
// tables would need voltages and loads nobody authored.
const (
	upsMIBObjects = "1.3.6.1.2.1.33.1"

	upsIdentManufacturer         = upsMIBObjects + ".1.1.0"
	upsIdentModel                = upsMIBObjects + ".1.2.0"
	upsIdentAgentSoftwareVersion = upsMIBObjects + ".1.4.0"
	upsIdentName                 = upsMIBObjects + ".1.5.0"

	upsBatteryStatus             = upsMIBObjects + ".2.1.0"
	upsSecondsOnBattery          = upsMIBObjects + ".2.2.0"
	upsEstimatedMinutesRemaining = upsMIBObjects + ".2.3.0"
	upsEstimatedChargeRemaining  = upsMIBObjects + ".2.4.0"

	upsOutputSource  = upsMIBObjects + ".4.1.0"
	upsAlarmsPresent = upsMIBObjects + ".6.1.0"

	upsBatteryStatusNormal = 2
	upsOutputSourceNormal  = 3
	upsFullCharge          = 100
	// upsRuntimeMinutes is a 3 kVA unit's runtime at the light load an IDF
	// closet puts on it.
	upsRuntimeMinutes = 42
)

// initializeUPSMIB serves UPS-MIB on a device whose identity names a UPS agent.
// A walk-backed device waits for its walk, which carries the sysObjectID that
// decides and may carry the MIB itself.
func (a *Agent) initializeUPSMIB() {
	if a.hasWalkContent() {
		return
	}

	a.registerUPSMIB()
}

// refreshWalkedUPSMIB serves a UPS whose capture carried no UPS-MIB. One that
// did keeps it untouched.
func (a *Agent) refreshWalkedUPSMIB(walkOwnsUPS bool) {
	if walkOwnsUPS {
		return
	}

	a.registerUPSMIB()
}

// walkOwnsUPS reports whether a parsed capture carries UPS-MIB of its own.
func walkOwnsUPS(entries []WalkEntry) bool {
	for _, entry := range entries {
		if strings.HasPrefix(strings.TrimPrefix(entry.OID, "."), upsMIBObjects+".") {
			return true
		}
	}

	return false
}

func (a *Agent) registerUPSMIB() {
	if strings.TrimPrefix(oidValueString(a.mib.Get("1.3.6.1.2.1.1.2.0")), ".") != APCUPSSysObjectID {
		return
	}

	model := a.device.Properties["model"]
	if model == "" {
		model = oidValueString(a.mib.Get("1.3.6.1.2.1.1.1.0"))
	}

	a.mib.Set(upsIdentManufacturer, &OIDValue{Type: gosnmp.OctetString, Value: apcManufacturer})
	a.mib.Set(upsIdentModel, &OIDValue{Type: gosnmp.OctetString, Value: model})
	a.mib.Set(upsIdentAgentSoftwareVersion, &OIDValue{
		Type: gosnmp.OctetString, Value: a.device.Properties["software"],
	})
	a.mib.Set(upsIdentName, &OIDValue{
		Type: gosnmp.OctetString, Value: oidValueString(a.mib.Get("1.3.6.1.2.1.1.5.0")),
	})

	a.mib.Set(upsBatteryStatus, &OIDValue{Type: gosnmp.Integer, Value: upsBatteryStatusNormal})
	a.mib.Set(upsSecondsOnBattery, &OIDValue{Type: gosnmp.Integer, Value: 0})
	a.mib.Set(upsEstimatedMinutesRemaining, &OIDValue{Type: gosnmp.Integer, Value: upsRuntimeMinutes})
	a.mib.Set(upsEstimatedChargeRemaining, &OIDValue{Type: gosnmp.Integer, Value: upsFullCharge})

	a.mib.Set(upsOutputSource, &OIDValue{Type: gosnmp.Integer, Value: upsOutputSourceNormal})
	a.mib.Set(upsAlarmsPresent, &OIDValue{Type: gosnmp.Gauge32, Value: uint32(0)})
}
