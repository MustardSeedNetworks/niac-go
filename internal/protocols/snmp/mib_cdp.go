package snmp

import (
	"fmt"

	"github.com/gosnmp/gosnmp"

	"github.com/krisarmstrong/niac-go/internal/config"
)

// CDP-MIB OID prefixes (Cisco Discovery Protocol)
// Reference: https://www.cisco.com/c/en/us/td/docs/net_mgmt/prime/network/3-8/reference/guide/CiscoPrimeNetworkMIBs/CiscoCdpMib.html
const (
	// ciscoCdpMIB (1.3.6.1.4.1.9.9.23).
	cdpMIBBase = "1.3.6.1.4.1.9.9.23"

	// cdpGlobal (1.3.6.1.4.1.9.9.23.1.1).
	cdpGlobal                = cdpMIBBase + ".1.1"
	cdpGlobalRun             = cdpGlobal + ".1.0" // Is CDP running (TruthValue)
	cdpGlobalMessageInterval = cdpGlobal + ".2.0" // CDP message interval (seconds)
	cdpGlobalHoldTime        = cdpGlobal + ".3.0" // CDP holdtime (seconds)
	cdpGlobalDeviceID        = cdpGlobal + ".6.0" // Local device ID

	// cdpCache (1.3.6.1.4.1.9.9.23.1.2).
	cdpCache = cdpMIBBase + ".1.2"

	// cdpCacheTable (1.3.6.1.4.1.9.9.23.1.2.1)
	// Index: cdpCacheIfIndex, cdpCacheDeviceIndex.
	cdpCacheTable = cdpCache + ".1"
)

// initializeCDPMIB populates Cisco CDP MIB.
func (a *Agent) initializeCDPMIB() {
	device := a.device

	cdp := device.CDPConfig
	if cdp == nil {
		return
	}

	// cdpGlobalRun (TruthValue: 1=true)
	a.mib.Set(cdpGlobalRun, &OIDValue{
		Type:  gosnmp.Integer,
		Value: 1,
	})

	// cdpGlobalMessageInterval
	interval := cdp.AdvertiseInterval
	if interval == 0 {
		interval = 60
	}

	a.mib.Set(cdpGlobalMessageInterval, &OIDValue{
		Type:  gosnmp.Integer,
		Value: interval,
	})

	// cdpGlobalHoldTime
	holdtime := cdp.Holdtime
	if holdtime == 0 {
		holdtime = 180
	}

	a.mib.Set(cdpGlobalHoldTime, &OIDValue{
		Type:  gosnmp.Integer,
		Value: holdtime,
	})

	// cdpGlobalDeviceID
	a.mib.Set(cdpGlobalDeviceID, &OIDValue{
		Type:  gosnmp.OctetString,
		Value: device.Name,
	})

	// CDP Cache entries (neighbors)
	deviceIndex := 1

	for ifIdx, trunk := range device.TrunkPorts {
		if trunk.RemoteDevice == "" {
			continue
		}

		ifIndex := ifIdx + 1
		a.createCDPCacheEntry(ifIndex, deviceIndex, trunk, cdp)

		deviceIndex++
	}

	if a.debugLevel >= DebugLevelMinimum {
		a.logger.Info("Initialized CDP MIB", "neighbors", deviceIndex-1, "device", device.Name)
	}
}

// createCDPCacheEntry creates a CDP cache (neighbor) entry.
func (a *Agent) createCDPCacheEntry(
	ifIndex, deviceIndex int,
	trunk config.TrunkPort,
	cdp *config.CDPConfig,
) {
	// cdpCacheEntry index: ifIndex.deviceIndex
	indexStr := fmt.Sprintf("%d.%d", ifIndex, deviceIndex)
	entryBase := cdpCacheTable + ".1"

	// cdpCacheAddressType (1 = IP)
	a.mib.Set(entryBase+".3."+indexStr, &OIDValue{
		Type:  gosnmp.Integer,
		Value: 1,
	})

	// cdpCacheAddress (placeholder - would be remote device IP)
	a.mib.Set(entryBase+".4."+indexStr, &OIDValue{
		Type:  gosnmp.OctetString,
		Value: []byte{10, 0, 0, 1}, // Placeholder IP
	})

	// cdpCacheVersion
	version := cdp.SoftwareVersion
	if version == "" {
		version = unknownPlaceholder
	}

	a.mib.Set(entryBase+".5."+indexStr, &OIDValue{
		Type:  gosnmp.OctetString,
		Value: version,
	})

	// cdpCacheDeviceId
	a.mib.Set(entryBase+".6."+indexStr, &OIDValue{
		Type:  gosnmp.OctetString,
		Value: trunk.RemoteDevice,
	})

	// cdpCacheDevicePort
	a.mib.Set(entryBase+".7."+indexStr, &OIDValue{
		Type:  gosnmp.OctetString,
		Value: trunk.RemoteInterface,
	})

	// cdpCachePlatform
	platform := cdp.Platform
	if platform == "" {
		platform = unknownPlaceholder
	}

	a.mib.Set(entryBase+".8."+indexStr, &OIDValue{
		Type:  gosnmp.OctetString,
		Value: platform,
	})

	// cdpCacheCapabilities (bit field)
	a.mib.Set(entryBase+".9."+indexStr, &OIDValue{
		Type:  gosnmp.OctetString,
		Value: []byte{0x00, 0x00, 0x00, 0x29}, // Router + Switch + IGMP
	})

	// cdpCacheVTPMgmtDomain
	a.mib.Set(entryBase+".10."+indexStr, &OIDValue{
		Type:  gosnmp.OctetString,
		Value: "",
	})

	// cdpCacheNativeVLAN
	nativeVLAN := trunk.NativeVLAN
	if nativeVLAN == 0 {
		nativeVLAN = 1
	}

	a.mib.Set(entryBase+".11."+indexStr, &OIDValue{
		Type:  gosnmp.Integer,
		Value: nativeVLAN,
	})

	// cdpCacheDuplex (1=unknown, 2=half, 3=full)
	a.mib.Set(entryBase+".12."+indexStr, &OIDValue{
		Type:  gosnmp.Integer,
		Value: DuplexFull,
	})
}
