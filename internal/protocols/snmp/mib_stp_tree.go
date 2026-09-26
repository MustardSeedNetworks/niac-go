package snmp

import (
	"strconv"
	"strings"

	"github.com/gosnmp/gosnmp"
)

// SpanningTreePosition is where the agent's bridge sits in its spanning tree.
type SpanningTreePosition struct {
	Root          []byte // the elected root's 8-octet bridge ID
	Cost          int    // root path cost
	RootInterface string // the port towards the root; empty on the root
	Upstream      []byte // bridge ID at the other end of RootInterface
}

// SynthesizeSpanningTree puts the elected tree into dot1dStp, where each bridge
// otherwise names itself root: the root, the cost to it and the root port, and
// on every port row the root and the designated bridge and cost. An agent with
// no synthesized dot1dStp group (STP off, or a capture walk owns BRIDGE-MIB) is
// left alone. Call before Reindex.
func (a *Agent) SynthesizeSpanningTree(position SpanningTreePosition) {
	if a == nil || a.device == nil || a.hasWalkContent() || a.mib.Get(dot1dStpDesignatedRoot) == nil {
		return
	}

	own := a.buildBridgeID(stpPriority(a.device.STPConfig.BridgePriority), parseMACBytes(a.device.MACAddress.String()))
	rootPort := 0
	if position.RootInterface != "" {
		offset, hasOffset := a.basePortOffset()
		if port, ok := a.bridgePortForInterface(position.RootInterface, offset, hasOffset); ok {
			rootPort = port
			a.raiseBaseNumPorts(port)
			if a.mib.Get(dot1dStpPort+"."+strconv.Itoa(port)) == nil {
				a.registerDot1dStpPortEntry(port, own)
			}
		}
	}

	a.mib.Set(dot1dStpDesignatedRoot, &OIDValue{Type: gosnmp.OctetString, Value: position.Root})
	a.mib.Set(dot1dStpRootCost, &OIDValue{Type: gosnmp.Integer, Value: position.Cost})
	a.mib.Set(dot1dStpRootPort, &OIDValue{Type: gosnmp.Integer, Value: rootPort})

	for _, port := range a.stpPorts() {
		suffix := "." + strconv.Itoa(port)
		bridge, cost := own, position.Cost
		if port == rootPort {
			bridge, cost = position.Upstream, position.Cost-STPPortPathCostDefault
		}
		a.mib.Set(dot1dStpPortDesignatedRoot+suffix, &OIDValue{Type: gosnmp.OctetString, Value: position.Root})
		a.mib.Set(dot1dStpPortDesignatedBridge+suffix, &OIDValue{Type: gosnmp.OctetString, Value: bridge})
		a.mib.Set(dot1dStpPortDesignatedCost+suffix, &OIDValue{Type: gosnmp.Integer, Value: cost})
	}
}

func (a *Agent) stpPorts() []int {
	prefix := dot1dStpPort + "."
	a.mib.mu.RLock()
	defer a.mib.mu.RUnlock()
	var ports []int
	for oid := range a.mib.entries {
		rest, found := strings.CutPrefix(oid, prefix)
		if !found {
			continue
		}
		if port, err := strconv.Atoi(rest); err == nil {
			ports = append(ports, port)
		}
	}
	return ports
}

func stpPriority(authored uint16) int {
	if authored == 0 {
		return STPPriorityDefault
	}
	return int(authored)
}
