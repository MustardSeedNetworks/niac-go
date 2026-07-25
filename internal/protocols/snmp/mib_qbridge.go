package snmp

import (
	"strconv"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/safeconv"
)

const (
	qBridgeMIBObjects          = "1.3.6.1.2.1.17.7.1"
	dot1qVlanVersionNumber     = qBridgeMIBObjects + ".1.1.0"
	dot1qMaxVlanID             = qBridgeMIBObjects + ".1.2.0"
	dot1qMaxSupportedVLANs     = qBridgeMIBObjects + ".1.3.0"
	dot1qNumVLANs              = qBridgeMIBObjects + ".1.4.0"
	dot1qGVRPStatus            = qBridgeMIBObjects + ".1.5.0"
	dot1qFDBTable              = qBridgeMIBObjects + ".2.1"
	dot1qFDBDynamicCount       = qBridgeMIBObjects + ".2.1.1.2"
	dot1qTpFDBTable            = qBridgeMIBObjects + ".2.2"
	dot1qTpFDBAddress          = qBridgeMIBObjects + ".2.2.1.1"
	dot1qTpFDBPort             = qBridgeMIBObjects + ".2.2.1.2"
	dot1qTpFDBStatus           = qBridgeMIBObjects + ".2.2.1.3"
	dot1qVlanFDBID             = qBridgeMIBObjects + ".4.2.1.3"
	dot1qVlanStatus            = qBridgeMIBObjects + ".4.2.1.6"
	dot1qVlanCreationTime      = qBridgeMIBObjects + ".4.2.1.7"
	dot1qPVID                  = qBridgeMIBObjects + ".4.5.1.1"
	maxVLANID                  = 4094
	qBridgeVLANVersion         = 1
	qBridgeGVRPDisabled        = 2
	qBridgeVLANStatusPermanent = 2
)

func (a *Agent) initializeQBridgeMIB() {
	vlans := configuredVLANs(a.device)
	if len(vlans) == 0 {
		return
	}

	a.mib.Set(dot1qVlanVersionNumber, &OIDValue{
		Type: gosnmp.Integer, Value: qBridgeVLANVersion,
	})
	a.mib.Set(dot1qMaxVlanID, &OIDValue{Type: gosnmp.Integer, Value: maxVLANID})
	a.mib.Set(dot1qMaxSupportedVLANs, &OIDValue{
		Type: gosnmp.Gauge32, Value: uint32(maxVLANID),
	})
	a.mib.Set(dot1qNumVLANs, &OIDValue{
		Type: gosnmp.Gauge32, Value: safeconv.Uint32(len(vlans)),
	})
	a.mib.Set(dot1qGVRPStatus, &OIDValue{
		Type: gosnmp.Integer, Value: qBridgeGVRPDisabled,
	})

	for _, vlan := range vlans {
		index := "0." + strconv.Itoa(vlan)
		a.mib.Set(dot1qFDBDynamicCount+"."+strconv.Itoa(vlan), &OIDValue{
			Type: gosnmp.Counter32, Value: uint32(0),
		})
		a.mib.Set(dot1qVlanFDBID+"."+index, &OIDValue{
			Type: gosnmp.Gauge32, Value: safeconv.Uint32(vlan),
		})
		a.mib.Set(dot1qVlanStatus+"."+index, &OIDValue{
			Type: gosnmp.Integer, Value: qBridgeVLANStatusPermanent,
		})
		a.mib.Set(dot1qVlanCreationTime+"."+index, &OIDValue{
			Type: gosnmp.TimeTicks, Value: a.sysUpTimeTicks(),
		})
	}

	a.initializeQBridgePVIDs()
}

func (a *Agent) initializeQBridgePVIDs() {
	offset, hasOffset := a.basePortOffset()
	maxPort := 0
	for _, trunk := range a.device.TrunkPorts {
		vlan := forwardingVLAN(trunk)
		if vlan == 0 {
			continue
		}
		port, ok := a.bridgePortForInterface(trunk.Interface, offset, hasOffset)
		if !ok {
			continue
		}
		a.mib.Set(dot1qPVID+"."+strconv.Itoa(port), &OIDValue{
			Type: gosnmp.Gauge32, Value: safeconv.Uint32(vlan),
		})
		maxPort = max(maxPort, port)
	}
	a.raiseBaseNumPorts(maxPort)
}

func configuredVLANs(device *config.Device) []int {
	if device == nil {
		return nil
	}

	seen := make(map[int]struct{})
	vlans := make([]int, 0)
	for _, trunk := range device.TrunkPorts {
		candidates := append([]int(nil), trunk.VLANs...)
		candidates = append(candidates, forwardingVLAN(trunk))
		for _, vlan := range candidates {
			if vlan <= 0 || vlan > maxVLANID {
				continue
			}
			if _, exists := seen[vlan]; exists {
				continue
			}
			seen[vlan] = struct{}{}
			vlans = append(vlans, vlan)
		}
	}
	return vlans
}

func (a *Agent) addLearnedQBridgeFDBEntry(vlan int, mac []byte, bridgePort int) {
	index := strconv.Itoa(vlan) + "." + macBytesToOIDIndex(mac)
	portOID := dot1qTpFDBPort + "." + index
	if a.mib.Get(portOID) == nil {
		countOID := dot1qFDBDynamicCount + "." + strconv.Itoa(vlan)
		count := oidCounterValue(a.mib.Get(countOID))
		a.mib.Set(countOID, &OIDValue{
			Type: gosnmp.Counter32, Value: safeconv.Uint32FromUint64(count + 1),
		})
	}
	a.mib.Set(dot1qTpFDBAddress+"."+index, &OIDValue{
		Type: gosnmp.OctetString, Value: mac,
	})
	a.mib.Set(portOID, &OIDValue{Type: gosnmp.Integer, Value: bridgePort})
	a.mib.Set(dot1qTpFDBStatus+"."+index, &OIDValue{
		Type: gosnmp.Integer, Value: FDBStatusLearned,
	})
}
