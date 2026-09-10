package snmp

import (
	"strconv"
	"strings"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

const deviceUptimeOID = "1.3.6.1.2.1.1.3.0"

type deviceActionBinding struct {
	installed *OIDValue
}

// UptimeTicks returns the served management uptime, independently of USM clocks.
func (a *Agent) UptimeTicks() uint32 {
	value, _ := actionScalar(a.mib.Get(deviceUptimeOID), gosnmp.TimeTicks)
	return value
}

// DeviceActionObservable requires the original scalar rows, never invented STP data.
func (a *Agent) DeviceActionObservable(kind devicestate.DeviceActionType) bool {
	if a.deviceState == nil {
		return false
	}
	switch kind {
	case devicestate.ActionReboot:
		_, ok := actionScalar(a.mib.Get(deviceUptimeOID), gosnmp.TimeTicks)
		return ok
	case devicestate.ActionSTPTopologyChange:
		_, count := actionScalar(a.mib.Get(dot1dStpTopChanges), gosnmp.Counter32)
		_, elapsed := actionScalar(a.mib.Get(dot1dStpTimeSinceTopologyChange), gosnmp.TimeTicks)
		return count && elapsed
	default:
		return false
	}
}

func actionScalar(value *OIDValue, kind gosnmp.Asn1BER) (uint32, bool) {
	if value == nil || value.Type != kind {
		return 0, false
	}
	parsed, err := strconv.ParseUint(oidValueString(value), 10, 32)
	return uint32(parsed), err == nil
}

func (a *Agent) registerDeviceActions() {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, oid := range []string{deviceUptimeOID, dot1dStpTopChanges, dot1dStpTimeSinceTopologyChange} {
		a.registerDeviceActionOID(oid)
	}
}

func (a *Agent) registerDeviceActionOID(oid string) {
	if a.deviceActionBindings == nil {
		a.deviceActionBindings = make(map[string]deviceActionBinding)
	}
	a.mib.mu.RLock()
	baseline := a.mib.entries[oid]
	a.mib.mu.RUnlock()
	if baseline == nil {
		return
	}
	if previous, exists := a.deviceActionBindings[oid]; exists && baseline == previous.installed {
		return
	}
	installed := &OIDValue{Dynamic: func() *OIDValue { return a.projectDeviceAction(oid, baseline) }}
	a.mib.Set(oid, installed)
	a.deviceActionBindings[oid] = deviceActionBinding{installed: installed}
}

// Callbacks run under the MIB read lock; evaluate retained source entries directly.
func (a *Agent) projectDeviceAction(oid string, baseline *OIDValue) *OIDValue {
	value := baseline
	if baseline.Dynamic != nil {
		value = baseline.Dynamic()
	}
	if a.deviceState == nil {
		return value
	}
	telemetry := a.deviceState.DeviceTelemetry()
	switch oid {
	case deviceUptimeOID:
		if _, ok := actionScalar(value, gosnmp.TimeTicks); ok && !telemetry.RebootedAt.IsZero() {
			return &OIDValue{Type: gosnmp.TimeTicks, Value: uptimeTicks(time.Since(telemetry.RebootedAt))}
		}
	case dot1dStpTopChanges:
		if count, ok := actionScalar(value, gosnmp.Counter32); ok && telemetry.STPChanges > 0 {
			return &OIDValue{Type: gosnmp.Counter32, Value: count + telemetry.STPChanges}
		}
	case dot1dStpTimeSinceTopologyChange:
		if _, ok := actionScalar(value, gosnmp.TimeTicks); ok && !telemetry.STPChangedAt.IsZero() {
			return &OIDValue{Type: gosnmp.TimeTicks, Value: uptimeTicks(time.Since(telemetry.STPChangedAt))}
		}
	default:
		if strings.HasPrefix(oid, ifLastChange+".") && !telemetry.RebootedAt.IsZero() {
			if _, ok := actionScalar(value, gosnmp.TimeTicks); ok {
				return &OIDValue{Type: gosnmp.TimeTicks, Value: uint32(0)}
			}
		}
	}
	return value
}
