package snmp

import (
	"fmt"
	"log/slog"
	"strconv"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/safeconv"
)

func (a *Agent) initializeLLDPLocalMIB() {
	logger := slog.Default()
	device := a.device

	lldp := device.LLDPConfig
	if lldp == nil {
		return
	}

	// lldpLocChassisIDSubtype (4 = macAddress)
	a.mib.Set(lldpLocChassisIDSubtype, &OIDValue{
		Type:  gosnmp.Integer,
		Value: ChassisIDSubtypeMAC,
	})

	// lldpLocChassisID (MAC address)
	macBytes := parseMACBytes(device.MACAddress.String())
	a.mib.Set(lldpLocChassisID, &OIDValue{
		Type:  gosnmp.OctetString,
		Value: macBytes,
	})

	// lldpLocSysName - use device name since LLDP config doesn't have separate system name
	sysName := device.Name
	a.mib.Set(lldpLocSysName, &OIDValue{
		Type:  gosnmp.OctetString,
		Value: sysName,
	})

	// lldpLocSysDesc
	sysDesc := lldp.SystemDescription
	if sysDesc == "" {
		sysDesc = fmt.Sprintf("%s %s", device.Type, device.Name)
	}

	a.mib.Set(lldpLocSysDesc, &OIDValue{
		Type:  gosnmp.OctetString,
		Value: sysDesc,
	})

	// lldpLocSysCapSupported (bit field: bridge=2, router=4)
	capabilities := safeconv.Uint32(getCapabilitiesBitfield(device.Type))
	capBytes := []byte{
		safeconv.ByteFromUint32(capabilities >> BitShiftByte),
		safeconv.ByteFromUint32(capabilities),
	}
	a.mib.Set(lldpLocSysCapSupported, &OIDValue{
		Type:  gosnmp.OctetString,
		Value: capBytes,
	})

	// lldpLocSysCapEnabled
	a.mib.Set(lldpLocSysCapEnabled, &OIDValue{
		Type:  gosnmp.OctetString,
		Value: capBytes,
	})

	// Local port entries
	for idx, trunk := range device.TrunkPorts {
		portNum := a.discoveryIfIndex(trunk.Interface, idx+1)
		a.createLLDPLocalPortEntry(portNum, trunk.Interface, lldp.PortDescription)
	}

	if a.debugLevel >= DebugLevelMinimum {
		logger.Info("Initialized LLDP local MIB", "device", device.Name)
	}
}

// createLLDPLocalPortEntry creates a local port entry in LLDP-MIB.
func (a *Agent) createLLDPLocalPortEntry(portNum int, ifName, portDesc string) {
	portNumStr := strconv.Itoa(portNum)
	baseOID := lldpLocPortTable + ".1"

	// lldpLocPortIdSubtype (5 = interfaceName)
	a.mib.Set(baseOID+".2."+portNumStr, &OIDValue{
		Type:  gosnmp.Integer,
		Value: PortIDSubtypeInterfaceName,
	})

	// lldpLocPortId
	a.mib.Set(baseOID+".3."+portNumStr, &OIDValue{
		Type:  gosnmp.OctetString,
		Value: ifName,
	})

	// lldpLocPortDesc
	desc := portDesc
	if desc == "" {
		desc = ifName
	}

	a.mib.Set(baseOID+".4."+portNumStr, &OIDValue{
		Type:  gosnmp.OctetString,
		Value: desc,
	})
}

// initializeLLDPRemoteMIB populates LLDP remote (neighbor) table.
func (a *Agent) initializeLLDPRemoteMIB() {
	logger := slog.Default()
	device := a.device
	if device == nil || len(device.TrunkPorts) == 0 {
		return
	}

	remIndex := 1

	for portIdx, trunk := range device.TrunkPorts {
		if trunk.RemoteDevice == "" || trunk.FDBOnly {
			continue
		}

		portNum := a.discoveryIfIndex(trunk.Interface, portIdx+1)
		timeMark := 0 // lldpRemTimeMark

		// Create remote entry
		a.createLLDPRemoteEntry(timeMark, portNum, remIndex, trunk)

		remIndex++
	}

	if a.debugLevel >= DebugLevelMinimum {
		logger.Info("Initialized LLDP remote MIB", "neighbors", remIndex-1, "device", device.Name)
	}
}

// createLLDPRemoteEntry creates a remote (neighbor) entry in LLDP-MIB.
func (a *Agent) createLLDPRemoteEntry(timeMark, portNum, remIndex int, trunk config.TrunkPort) {
	// lldpRemEntry index: timeMark.portNum.remIndex
	indexStr := fmt.Sprintf("%d.%d.%d", timeMark, portNum, remIndex)
	entryBase := lldpRemTable + ".1"

	// The fleet resolver replaces this temporary local identity with the
	// remote MAC after every device has been loaded.
	a.mib.Set(entryBase+".4."+indexStr, &OIDValue{
		Type:  gosnmp.Integer,
		Value: ChassisIDSubtypeLocal,
	})

	a.mib.Set(entryBase+".5."+indexStr, &OIDValue{
		Type:  gosnmp.OctetString,
		Value: trunk.RemoteDevice,
	})

	// lldpRemPortIdSubtype (5 = interfaceName)
	a.mib.Set(entryBase+".6."+indexStr, &OIDValue{
		Type:  gosnmp.Integer,
		Value: PortIDSubtypeInterfaceName,
	})

	// lldpRemPortId
	a.mib.Set(entryBase+".7."+indexStr, &OIDValue{
		Type:  gosnmp.OctetString,
		Value: trunk.RemoteInterface,
	})

	// lldpRemPortDesc
	a.mib.Set(entryBase+".8."+indexStr, &OIDValue{
		Type:  gosnmp.OctetString,
		Value: trunk.RemoteInterface,
	})

	// lldpRemSysName
	a.mib.Set(entryBase+".9."+indexStr, &OIDValue{
		Type:  gosnmp.OctetString,
		Value: trunk.RemoteDevice,
	})

	// lldpRemSysDesc (use device name as placeholder)
	a.mib.Set(entryBase+".10."+indexStr, &OIDValue{
		Type:  gosnmp.OctetString,
		Value: "LLDP neighbor: " + trunk.RemoteDevice,
	})
}

// initializeCDPMIB populates Cisco CDP MIB.
func (a *Agent) initializeCDPMIB() {
	logger := slog.Default()
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
		if trunk.RemoteDevice == "" || trunk.FDBOnly {
			continue
		}

		ifIndex := a.discoveryIfIndex(trunk.Interface, ifIdx+1)
		a.createCDPCacheEntry(ifIndex, deviceIndex, trunk, cdp)

		deviceIndex++
	}

	if a.debugLevel >= DebugLevelMinimum {
		logger.Info("Initialized CDP MIB", "neighbors", deviceIndex-1, "device", device.Name)
	}
}

func (a *Agent) discoveryIfIndex(interfaceName string, fallback int) int {
	index, ok := a.ifIndexForInterface(interfaceName)
	if !ok {
		return fallback
	}
	resolved, err := strconv.Atoi(index)
	if err != nil {
		return fallback
	}
	return resolved
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

// Helper functions
