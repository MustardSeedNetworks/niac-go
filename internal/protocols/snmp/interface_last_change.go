package snmp

import "github.com/gosnmp/gosnmp"

type interfaceChangeBinding struct {
	name      string
	baseline  *OIDValue
	installed *OIDValue
}

func (a *Agent) interfaceLastChangeTicks(name string) (uint32, bool) {
	changed := a.deviceState.InterfaceLastChange(name)
	if boot := a.deviceState.DeviceTelemetry().RebootedAt; !boot.IsZero() {
		if changed.IsZero() || changed.Before(boot) {
			return 0, true
		}
		return uptimeTicks(changed.Sub(boot)), true
	}
	if changed.IsZero() || changed.Before(a.startTime) {
		return 0, false
	}
	return uptimeTicks(a.uptimeBase + changed.Sub(a.startTime)), true
}

// Preserve the captured timestamp until this management lifetime changes the
// interface. Retaining the entry avoids sampling a prior overlay as baseline.
func (a *Agent) registerInterfaceLastChange(name, index string) {
	if a.interfaceChangeBindings == nil {
		a.interfaceChangeBindings = make(map[string]interfaceChangeBinding)
	}
	oid := ifLastChange + "." + index
	a.mib.mu.RLock()
	baseline := a.mib.entries[oid]
	a.mib.mu.RUnlock()
	if prior, exists := a.interfaceChangeBindings[oid]; exists && baseline == prior.installed {
		if prior.name == name {
			return
		}
		baseline = prior.baseline
	}
	if baseline == nil {
		baseline = &OIDValue{Type: gosnmp.TimeTicks, Value: uint32(0)}
	}
	installed := &OIDValue{Dynamic: func() *OIDValue {
		if ticks, changed := a.interfaceLastChangeTicks(name); changed {
			return &OIDValue{Type: gosnmp.TimeTicks, Value: ticks}
		}
		if baseline.Dynamic != nil {
			return baseline.Dynamic()
		}
		return baseline
	}}
	a.mib.Set(oid, installed)
	a.interfaceChangeBindings[oid] = interfaceChangeBinding{name: name, baseline: baseline, installed: installed}
}

func (a *Agent) restoreRemappedInterfaceChanges() {
	for oid, prior := range a.interfaceChangeBindings {
		if index, ok := a.ifIndexForInterface(prior.name); ok && oid == ifLastChange+"."+index {
			continue
		}
		a.mib.mu.Lock()
		if a.mib.entries[oid] == prior.installed {
			a.mib.entries[oid] = prior.baseline
		}
		a.mib.mu.Unlock()
		delete(a.interfaceChangeBindings, oid)
		a.registerDeviceActionOID(oid)
	}
}

func (a *Agent) changedInterfaceOIDs() map[string]struct{} {
	result := make(map[string]struct{})
	if a.deviceState == nil {
		return result
	}
	for _, iface := range a.deviceState.Snapshot().Network.Interfaces {
		if _, changed := a.interfaceLastChangeTicks(iface.Name); !changed {
			continue
		}
		if index, ok := a.ifIndexForInterface(iface.Name); ok {
			result[ifLastChange+"."+index] = struct{}{}
		}
	}
	return result
}
