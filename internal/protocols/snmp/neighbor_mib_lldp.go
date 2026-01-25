package snmp

import (
	"fmt"
	"log/slog"
	"strconv"

	"github.com/gosnmp/gosnmp"

	"github.com/krisarmstrong/niac-go/internal/config"
)

// LLDP-MIB OID prefixes (IEEE 802.1AB)
// Reference: http://www.ieee802.org/1/files/public/MIBs/LLDP-MIB-200505060000Z.txt
const (
	// lldpMIB (1.0.8802.1.1.2).
	lldpMIBBase = "1.0.8802.1.1.2"

	// lldpObjects (1.0.8802.1.1.2.1).
	lldpObjects = lldpMIBBase + ".1"

	// lldpLocalSystemData (1.0.8802.1.1.2.1.3).
	lldpLocalSystemData     = lldpObjects + ".3"
	lldpLocChassisIDSubtype = lldpLocalSystemData + ".1.0"
	lldpLocChassisID        = lldpLocalSystemData + ".2.0"
	lldpLocSysName          = lldpLocalSystemData + ".3.0"
	lldpLocSysDesc          = lldpLocalSystemData + ".4.0"
	lldpLocSysCapSupported  = lldpLocalSystemData + ".5.0"
	lldpLocSysCapEnabled    = lldpLocalSystemData + ".6.0"

	// lldpLocPortTable (1.0.8802.1.1.2.1.3.7).
	lldpLocPortTable = lldpLocalSystemData + ".7"

	// lldpRemoteSystemsData (1.0.8802.1.1.2.1.4).
	lldpRemoteSystemsData = lldpObjects + ".4"

	// lldpRemTable (1.0.8802.1.1.2.1.4.1)
	// Index: lldpRemTimeMark, lldpRemLocalPortNum, lldpRemIndex.
	lldpRemTable = lldpRemoteSystemsData + ".1"
)

// initializeLLDPLocalMIB populates LLDP local system data.
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
	capabilities := getCapabilitiesBitfield(device.Type)
	a.mib.Set(lldpLocSysCapSupported, &OIDValue{
		Type:  gosnmp.OctetString,
		Value: []byte{byte(capabilities >> BitShiftByte), byte(capabilities)},
	})

	// lldpLocSysCapEnabled
	a.mib.Set(lldpLocSysCapEnabled, &OIDValue{
		Type:  gosnmp.OctetString,
		Value: []byte{byte(capabilities >> BitShiftByte), byte(capabilities)},
	})

	// Local port entries
	for idx, trunk := range device.TrunkPorts {
		portNum := idx + 1
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
		if trunk.RemoteDevice == "" {
			continue
		}

		portNum := portIdx + 1
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

	// lldpRemChassisIdSubtype (4 = macAddress)
	a.mib.Set(entryBase+".4."+indexStr, &OIDValue{
		Type:  gosnmp.Integer,
		Value: ChassisIDSubtypeMAC,
	})

	// lldpRemChassisId - Use remote device name as chassis ID for now
	// In a real implementation, we'd look up the remote device's MAC
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
