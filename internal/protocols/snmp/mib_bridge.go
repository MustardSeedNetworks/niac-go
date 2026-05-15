package snmp

import (
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/krisarmstrong/niac-go/internal/config"
	"github.com/krisarmstrong/niac-go/internal/safeconv"
)

func (a *Agent) initializeBridgeMIB() {
	logger := slog.Default()
	device := a.device
	if device == nil {
		return
	}

	numPorts := len(device.TrunkPorts)
	if numPorts == 0 {
		numPorts = 1
	}

	macBytes := parseMACBytes(device.MACAddress.String())

	// Register dot1dBase group
	a.registerDot1dBaseGroup(numPorts, macBytes)

	// Register dot1dStp group if STP is enabled
	if device.STPConfig != nil && device.STPConfig.Enabled {
		a.registerDot1dStpGroup(device.STPConfig, numPorts, macBytes)
	}

	// Register dot1dTp group (transparent bridging)
	a.registerDot1dTpGroup(device, numPorts, macBytes)

	if a.debugLevel >= DebugLevelMinimum {
		logger.Info("Initialized BRIDGE-MIB", "device", device.Name, "ports", numPorts)
	}
}

// registerDot1dBaseGroup registers dot1dBase OIDs.
func (a *Agent) registerDot1dBaseGroup(numPorts int, macBytes []byte) {
	a.mib.Set(dot1dBaseBridgeAddress, &OIDValue{Type: gosnmp.OctetString, Value: macBytes})
	a.mib.Set(dot1dBaseNumPorts, &OIDValue{Type: gosnmp.Integer, Value: numPorts})
	a.mib.Set(dot1dBaseType, &OIDValue{Type: gosnmp.Integer, Value: BridgeTypeTransparent})

	for portIdx := 1; portIdx <= numPorts; portIdx++ {
		a.registerDot1dBasePortEntry(portIdx)
	}
}

// registerDot1dBasePortEntry registers a single dot1dBasePortTable entry.
func (a *Agent) registerDot1dBasePortEntry(portIdx int) {
	portStr := strconv.Itoa(portIdx)
	a.mib.Set(dot1dBasePort+"."+portStr, &OIDValue{Type: gosnmp.Integer, Value: portIdx})
	a.mib.Set(dot1dBasePortIfIndex+"."+portStr, &OIDValue{Type: gosnmp.Integer, Value: portIdx})
	a.mib.Set(
		dot1dBasePortCircuit+"."+portStr,
		&OIDValue{Type: gosnmp.ObjectIdentifier, Value: "0.0"},
	)
	a.mib.Set(
		dot1dBasePortDelayExceededDiscards+"."+portStr,
		&OIDValue{Type: gosnmp.Counter32, Value: uint32(0)},
	)
	a.mib.Set(
		dot1dBasePortMtuExceededDiscards+"."+portStr,
		&OIDValue{Type: gosnmp.Counter32, Value: uint32(0)},
	)
}

// registerDot1dStpGroup registers dot1dStp OIDs when STP is enabled.
func (a *Agent) registerDot1dStpGroup(stp *config.STPConfig, numPorts int, macBytes []byte) {
	priority := int(stp.BridgePriority)
	if priority == 0 {
		priority = 32768
	}

	bridgeID := a.buildBridgeID(priority, macBytes)

	// Register STP scalars
	a.registerDot1dStpScalars(stp, priority, bridgeID)

	// Register STP port table
	for portIdx := 1; portIdx <= numPorts; portIdx++ {
		a.registerDot1dStpPortEntry(portIdx, bridgeID)
	}
}

// buildBridgeID constructs an 8-byte bridge ID from priority and MAC.
func (a *Agent) buildBridgeID(priority int, macBytes []byte) []byte {
	bridgeID := make([]byte, BridgeIDLength)
	prio := safeconv.Uint32(priority)
	bridgeID[0] = safeconv.ByteFromUint32(prio >> BitShiftByte)
	bridgeID[1] = safeconv.ByteFromUint32(prio)
	copy(bridgeID[2:], macBytes)
	return bridgeID
}

// registerDot1dStpScalars registers dot1dStp scalar OIDs.
func (a *Agent) registerDot1dStpScalars(stp *config.STPConfig, priority int, bridgeID []byte) {
	a.mib.Set(dot1dStpProtocolSpecification, &OIDValue{Type: gosnmp.Integer, Value: STPProtocolIEEE})
	a.mib.Set(dot1dStpPriority, &OIDValue{Type: gosnmp.Integer, Value: priority})
	a.mib.SetDynamic(dot1dStpTimeSinceTopologyChange, func() *OIDValue {
		return &OIDValue{Type: gosnmp.TimeTicks, Value: uptimeTicks(time.Since(a.startTime))}
	})
	a.mib.Set(dot1dStpTopChanges, &OIDValue{Type: gosnmp.Counter32, Value: uint32(0)})
	a.mib.Set(dot1dStpDesignatedRoot, &OIDValue{Type: gosnmp.OctetString, Value: bridgeID})
	a.mib.Set(dot1dStpRootCost, &OIDValue{Type: gosnmp.Integer, Value: 0})
	a.mib.Set(dot1dStpRootPort, &OIDValue{Type: gosnmp.Integer, Value: 0})

	// STP timers (in hundredths of seconds)
	maxAge, helloTime, forwardDelay := a.getStpTimers(stp)
	a.mib.Set(dot1dStpMaxAge, &OIDValue{Type: gosnmp.Integer, Value: maxAge * HundredthsPerSecond})
	a.mib.Set(dot1dStpHelloTime, &OIDValue{Type: gosnmp.Integer, Value: helloTime * HundredthsPerSecond})
	a.mib.Set(dot1dStpHoldTime, &OIDValue{Type: gosnmp.Integer, Value: 1})
	a.mib.Set(dot1dStpForwardDelay, &OIDValue{Type: gosnmp.Integer, Value: forwardDelay * HundredthsPerSecond})
	a.mib.Set(dot1dStpBridgeMaxAge, &OIDValue{Type: gosnmp.Integer, Value: maxAge * HundredthsPerSecond})
	a.mib.Set(dot1dStpBridgeHelloTime, &OIDValue{Type: gosnmp.Integer, Value: helloTime * HundredthsPerSecond})
	a.mib.Set(
		dot1dStpBridgeForwardDelay,
		&OIDValue{Type: gosnmp.Integer, Value: forwardDelay * HundredthsPerSecond},
	)
}

// getStpTimers returns STP timer values with defaults.
func (a *Agent) getStpTimers(stp *config.STPConfig) (int, int, int) {
	maxAge := int(stp.MaxAge)
	if maxAge == 0 {
		maxAge = 20
	}
	helloTime := int(stp.HelloTime)
	if helloTime == 0 {
		helloTime = 2
	}
	forwardDelay := int(stp.ForwardDelay)
	if forwardDelay == 0 {
		forwardDelay = 15
	}
	return maxAge, helloTime, forwardDelay
}

// registerDot1dStpPortEntry registers a single dot1dStpPortTable entry.
func (a *Agent) registerDot1dStpPortEntry(portIdx int, bridgeID []byte) {
	portStr := strconv.Itoa(portIdx)
	a.mib.Set(dot1dStpPort+"."+portStr, &OIDValue{Type: gosnmp.Integer, Value: portIdx})
	a.mib.Set(dot1dStpPortPriority+"."+portStr, &OIDValue{Type: gosnmp.Integer, Value: STPPortPriorityDefault})
	a.mib.Set(dot1dStpPortState+"."+portStr, &OIDValue{Type: gosnmp.Integer, Value: STPPortStateForwarding})
	a.mib.Set(dot1dStpPortEnable+"."+portStr, &OIDValue{Type: gosnmp.Integer, Value: 1})
	a.mib.Set(dot1dStpPortPathCost+"."+portStr, &OIDValue{Type: gosnmp.Integer, Value: STPPortPathCostDefault})
	a.mib.Set(
		dot1dStpPortDesignatedRoot+"."+portStr,
		&OIDValue{Type: gosnmp.OctetString, Value: bridgeID},
	)
	a.mib.Set(dot1dStpPortDesignatedCost+"."+portStr, &OIDValue{Type: gosnmp.Integer, Value: 0})
	a.mib.Set(
		dot1dStpPortDesignatedBridge+"."+portStr,
		&OIDValue{Type: gosnmp.OctetString, Value: bridgeID},
	)
	a.mib.Set(
		dot1dStpPortDesignatedPort+"."+portStr,
		&OIDValue{Type: gosnmp.OctetString, Value: []byte{0x80, safeconv.Byte(portIdx)}},
	)
	a.mib.Set(
		dot1dStpPortForwardTransitions+"."+portStr,
		&OIDValue{Type: gosnmp.Counter32, Value: uint32(1)},
	)
}

// registerDot1dTpGroup registers dot1dTp (transparent bridging) OIDs.
func (a *Agent) registerDot1dTpGroup(device *config.Device, numPorts int, macBytes []byte) {
	a.mib.Set(dot1dTpLearnedEntryDiscards, &OIDValue{Type: gosnmp.Counter32, Value: uint32(0)})
	a.mib.Set(dot1dTpAgingTime, &OIDValue{Type: gosnmp.Integer, Value: FDBAgingTimeDefault})

	// Register FDB self entry
	macIndex := macToOIDIndex(device.MACAddress.String())
	a.mib.Set(dot1dTpFdbAddress+"."+macIndex, &OIDValue{Type: gosnmp.OctetString, Value: macBytes})
	a.mib.Set(dot1dTpFdbPort+"."+macIndex, &OIDValue{Type: gosnmp.Integer, Value: 0})
	a.mib.Set(dot1dTpFdbStatus+"."+macIndex, &OIDValue{Type: gosnmp.Integer, Value: FDBStatusSelf})

	// Register port table entries
	for portIdx := 1; portIdx <= numPorts; portIdx++ {
		a.registerDot1dTpPortEntry(portIdx)
	}
}

// registerDot1dTpPortEntry registers a single dot1dTpPortTable entry.
func (a *Agent) registerDot1dTpPortEntry(portIdx int) {
	portStr := strconv.Itoa(portIdx)
	startTime := a.startTime

	a.mib.Set(dot1dTpPort+"."+portStr, &OIDValue{Type: gosnmp.Integer, Value: portIdx})
	a.mib.Set(dot1dTpPortMaxInfo+"."+portStr, &OIDValue{Type: gosnmp.Integer, Value: DefaultMTU})
	a.mib.SetDynamic(dot1dTpPortInFrames+"."+portStr, func() *OIDValue {
		elapsed := time.Since(startTime).Seconds()
		frames := uint32(elapsed * 100 * float64(portIdx%10+1))
		return &OIDValue{Type: gosnmp.Counter32, Value: frames}
	})
	a.mib.SetDynamic(dot1dTpPortOutFrames+"."+portStr, func() *OIDValue {
		elapsed := time.Since(startTime).Seconds()
		frames := uint32(elapsed * 80 * float64(portIdx%10+1))
		return &OIDValue{Type: gosnmp.Counter32, Value: frames}
	})
	a.mib.Set(
		dot1dTpPortInDiscards+"."+portStr,
		&OIDValue{Type: gosnmp.Counter32, Value: uint32(0)},
	)
}

// macToOIDIndex converts a MAC address to OID index format (decimal octets separated by dots).
func macToOIDIndex(mac string) string {
	macBytes := parseMACBytes(mac)

	parts := make([]string, MACAddressOctets)

	for i, b := range macBytes {
		parts[i] = strconv.FormatUint(uint64(b), 10)
	}

	return strings.Join(parts, ".")
}
