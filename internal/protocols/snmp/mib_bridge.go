package snmp

import (
	"strconv"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/krisarmstrong/niac-go/internal/config"
)

// BRIDGE-MIB OID prefixes (dot1dBridge).
const (
	// dot1dBridge (1.3.6.1.2.1.17).
	dot1dBridge = "1.3.6.1.2.1.17"

	// dot1dBase (1.3.6.1.2.1.17.1).
	dot1dBase              = dot1dBridge + ".1"
	dot1dBaseBridgeAddress = dot1dBase + ".1.0" // Bridge MAC address
	dot1dBaseNumPorts      = dot1dBase + ".2.0" // Number of ports
	dot1dBaseType          = dot1dBase + ".3.0" // Bridge type (1=unknown,2=transparent-only,3=sourceroute-only,4=srt)

	// dot1dBasePortTable (1.3.6.1.2.1.17.1.4).
	dot1dBasePortTable                 = dot1dBase + ".4"
	dot1dBasePortEntry                 = dot1dBasePortTable + ".1"
	dot1dBasePort                      = dot1dBasePortEntry + ".1" // Port number (index)
	dot1dBasePortIfIndex               = dot1dBasePortEntry + ".2" // IF-MIB interface index
	dot1dBasePortCircuit               = dot1dBasePortEntry + ".3" // Circuit ID
	dot1dBasePortDelayExceededDiscards = dot1dBasePortEntry + ".4"
	dot1dBasePortMtuExceededDiscards   = dot1dBasePortEntry + ".5"

	// dot1dStp (1.3.6.1.2.1.17.2) - Spanning Tree.
	dot1dStp                        = dot1dBridge + ".2"
	dot1dStpProtocolSpecification   = dot1dStp + ".1.0" // STP spec (1=unknown,2=decLb100,3=ieee8021d)
	dot1dStpPriority                = dot1dStp + ".2.0" // Bridge priority
	dot1dStpTimeSinceTopologyChange = dot1dStp + ".3.0"
	dot1dStpTopChanges              = dot1dStp + ".4.0"
	dot1dStpDesignatedRoot          = dot1dStp + ".5.0"
	dot1dStpRootCost                = dot1dStp + ".6.0"
	dot1dStpRootPort                = dot1dStp + ".7.0"
	dot1dStpMaxAge                  = dot1dStp + ".8.0" // hundredths of second
	dot1dStpHelloTime               = dot1dStp + ".9.0"
	dot1dStpHoldTime                = dot1dStp + ".10.0"
	dot1dStpForwardDelay            = dot1dStp + ".11.0"
	dot1dStpBridgeMaxAge            = dot1dStp + ".12.0"
	dot1dStpBridgeHelloTime         = dot1dStp + ".13.0"
	dot1dStpBridgeForwardDelay      = dot1dStp + ".14.0"

	// dot1dStpPortTable (1.3.6.1.2.1.17.2.15).
	dot1dStpPortTable              = dot1dStp + ".15"
	dot1dStpPortEntry              = dot1dStpPortTable + ".1"
	dot1dStpPort                   = dot1dStpPortEntry + ".1" // Port number (index)
	dot1dStpPortPriority           = dot1dStpPortEntry + ".2"
	dot1dStpPortState              = dot1dStpPortEntry + ".3" // 1=disabled,2=blocking,3=listening,4=learning,5=forwarding,6=broken
	dot1dStpPortEnable             = dot1dStpPortEntry + ".4" // 1=enabled,2=disabled
	dot1dStpPortPathCost           = dot1dStpPortEntry + ".5"
	dot1dStpPortDesignatedRoot     = dot1dStpPortEntry + ".6"
	dot1dStpPortDesignatedCost     = dot1dStpPortEntry + ".7"
	dot1dStpPortDesignatedBridge   = dot1dStpPortEntry + ".8"
	dot1dStpPortDesignatedPort     = dot1dStpPortEntry + ".9"
	dot1dStpPortForwardTransitions = dot1dStpPortEntry + ".10"

	// dot1dTp (1.3.6.1.2.1.17.4) - Transparent Bridging.
	dot1dTp                     = dot1dBridge + ".4"
	dot1dTpLearnedEntryDiscards = dot1dTp + ".1.0"
	dot1dTpAgingTime            = dot1dTp + ".2.0"

	// dot1dTpFdbTable (1.3.6.1.2.1.17.4.3) - Forwarding Database.
	dot1dTpFdbTable   = dot1dTp + ".3"
	dot1dTpFdbEntry   = dot1dTpFdbTable + ".1"
	dot1dTpFdbAddress = dot1dTpFdbEntry + ".1" // MAC address (index)
	dot1dTpFdbPort    = dot1dTpFdbEntry + ".2" // Port learned on
	dot1dTpFdbStatus  = dot1dTpFdbEntry + ".3" // 1=other,2=invalid,3=learned,4=self,5=mgmt

	// dot1dTpPortTable (1.3.6.1.2.1.17.4.4).
	dot1dTpPortTable      = dot1dTp + ".4"
	dot1dTpPortEntry      = dot1dTpPortTable + ".1"
	dot1dTpPort           = dot1dTpPortEntry + ".1" // Port number (index)
	dot1dTpPortMaxInfo    = dot1dTpPortEntry + ".2" // Max frame size
	dot1dTpPortInFrames   = dot1dTpPortEntry + ".3"
	dot1dTpPortOutFrames  = dot1dTpPortEntry + ".4"
	dot1dTpPortInDiscards = dot1dTpPortEntry + ".5"
)

// initializeBridgeMIB populates BRIDGE-MIB (dot1dBridge) for switches.
func (a *Agent) initializeBridgeMIB() {
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
		a.logger.Info("Initialized BRIDGE-MIB", "device", device.Name, "ports", numPorts)
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
	bridgeID[0] = byte(priority >> BitShiftByte)
	bridgeID[1] = byte(priority)
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
		&OIDValue{Type: gosnmp.OctetString, Value: []byte{0x80, byte(portIdx)}},
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
