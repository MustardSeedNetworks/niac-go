package snmp

import (
	"strconv"
	"strings"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

const (
	hrProcessorLoadPrefix = "1.3.6.1.2.1.25.3.3.1.2."
	hrStorageTypePrefix   = "1.3.6.1.2.1.25.2.3.1.2."
	hrStorageSizePrefix   = "1.3.6.1.2.1.25.2.3.1.5."
	hrStorageUsedPrefix   = "1.3.6.1.2.1.25.2.3.1.6."
	hrStorageRAM          = "1.3.6.1.2.1.25.2.1.2"
	hrStorageFixedDisk    = "1.3.6.1.2.1.25.2.1.4"
	resourcePercentScale  = 100
)

type resourceFaultBinding struct {
	baseline  *OIDValue
	installed *OIDValue
	source    resourceFaultSource
}

type resourceFaultSource struct {
	cpu         bool
	storageType *OIDValue
	storageSize *OIDValue
}

// ResourceFaultObservable requires a registered numeric resource row. Merely
// enabling SNMP does not make a captured device expose CPU or storage telemetry.
func (a *Agent) ResourceFaultObservable(fault devicestate.DeviceFaultType) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.deviceState == nil {
		return false
	}
	for oid, binding := range a.resourceFaultBindings {
		kind, capacity := binding.source.current()
		if kind != fault || capacity == 0 {
			continue
		}
		value := a.mib.Get(oid)
		if value != nil && value.Type == gosnmp.Integer {
			return true
		}
	}
	return false
}

func (a *Agent) activeResourceOIDs() map[string]struct{} {
	a.mu.RLock()
	defer a.mu.RUnlock()
	active := make(map[string]struct{})
	if a.deviceState == nil {
		return active
	}
	for oid, binding := range a.resourceFaultBindings {
		fault, capacity := binding.source.current()
		if capacity == 0 || a.deviceState.DeviceFaultValue(fault) == 0 {
			continue
		}
		if value := a.mib.Get(oid); value != nil && value.Type == gosnmp.Integer {
			active[oid] = struct{}{}
		}
	}
	return active
}

// Retain the original entry, including any authored dynamic callback. Pointer
// identity distinguishes our wrapper from an override loaded since the last
// pass, without sampling an armed fault into its own baseline.
func (a *Agent) registerResourceFaults() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.resourceFaultBindings == nil {
		a.resourceFaultBindings = make(map[string]resourceFaultBinding)
	}
	for oid, current := range a.resourceEntries() {
		baseline := current
		if previous, exists := a.resourceFaultBindings[oid]; exists && current == previous.installed {
			baseline = previous.baseline
		}
		source := a.resourceSource(oid)
		installed := &OIDValue{Dynamic: func() *OIDValue {
			value := baseline
			if baseline.Dynamic != nil {
				value = baseline.Dynamic()
			}
			if value == nil || value.Type != gosnmp.Integer || a.deviceState == nil {
				return value
			}
			fault, capacity := source.current()
			if capacity == 0 {
				return value
			}
			percent := a.deviceState.DeviceFaultValue(fault)
			if percent == 0 {
				return value
			}
			return &OIDValue{Type: gosnmp.Integer, Value: int(capacity * int64(percent) / resourcePercentScale)}
		}}
		a.mib.Set(oid, installed)
		a.resourceFaultBindings[oid] = resourceFaultBinding{baseline: baseline, installed: installed, source: source}
	}
}

func (a *Agent) resourceEntries() map[string]*OIDValue {
	a.mib.mu.RLock()
	defer a.mib.mu.RUnlock()
	entries := make(map[string]*OIDValue)
	for oid, value := range a.mib.entries {
		if resourceIndex(oid, hrProcessorLoadPrefix) != "" || resourceIndex(oid, hrStorageUsedPrefix) != "" {
			entries[oid] = value
		}
	}
	return entries
}

// Dynamic MIB callbacks run under the MIB read lock. Capture raw entries here
// so evaluation never recursively acquires that lock behind a queued writer.
func (a *Agent) resourceSource(oid string) resourceFaultSource {
	if resourceIndex(oid, hrProcessorLoadPrefix) != "" {
		return resourceFaultSource{cpu: true}
	}
	index := resourceIndex(oid, hrStorageUsedPrefix)
	a.mib.mu.RLock()
	defer a.mib.mu.RUnlock()
	return resourceFaultSource{
		storageType: a.mib.entries[hrStorageTypePrefix+index],
		storageSize: a.mib.entries[hrStorageSizePrefix+index],
	}
}

func (source resourceFaultSource) current() (devicestate.DeviceFaultType, int64) {
	if source.cpu {
		return devicestate.FaultCPUPercent, resourcePercentScale
	}
	typeValue, size := source.storageType, source.storageSize
	if typeValue != nil && typeValue.Dynamic != nil {
		typeValue = typeValue.Dynamic()
	}
	if size != nil && size.Dynamic != nil {
		size = size.Dynamic()
	}
	if typeValue == nil || typeValue.Type != gosnmp.ObjectIdentifier || size == nil || size.Type != gosnmp.Integer {
		return "", 0
	}
	capacity, err := strconv.ParseInt(oidValueString(size), 10, 32)
	if err != nil || capacity <= 0 {
		return "", 0
	}
	switch strings.TrimPrefix(oidValueString(typeValue), ".") {
	case hrStorageRAM:
		return devicestate.FaultMemoryPercent, capacity
	case hrStorageFixedDisk:
		return devicestate.FaultDiskPercent, capacity
	default:
		return "", 0
	}
}

func resourceIndex(oid, prefix string) string {
	index, ok := strings.CutPrefix(oid, prefix)
	if !ok {
		return ""
	}
	value, err := strconv.ParseInt(index, 10, 32)
	if err != nil || value <= 0 {
		return ""
	}
	return index
}
