package snmp

import (
	"strings"

	"github.com/gosnmp/gosnmp"
)

func (a *Agent) registerUnmappedRebootTimestamps() {
	a.mib.mu.RLock()
	var oids []string
	for oid := range a.mib.entries {
		if strings.HasPrefix(oid, ifLastChange+".") {
			oids = append(oids, oid)
		}
	}
	a.mib.mu.RUnlock()
	for _, oid := range oids {
		if _, mapped := a.interfaceChangeBindings[oid]; !mapped {
			a.registerDeviceActionOID(oid)
		}
	}
}

func (a *Agent) activeDeviceActionOIDs() map[string]struct{} {
	a.mu.RLock()
	defer a.mu.RUnlock()
	result := make(map[string]struct{})
	if a.deviceState == nil {
		return result
	}
	telemetry := a.deviceState.DeviceTelemetry()
	for oid := range a.deviceActionBindings {
		kind := gosnmp.TimeTicks
		active := !telemetry.RebootedAt.IsZero()
		if oid == dot1dStpTopChanges {
			kind = gosnmp.Counter32
			active = telemetry.STPChanges > 0
		}
		if oid == dot1dStpTimeSinceTopologyChange {
			active = !telemetry.STPChangedAt.IsZero()
		}
		if _, valid := actionScalar(a.mib.Get(oid), kind); active && valid {
			result[oid] = struct{}{}
		}
	}
	return result
}
