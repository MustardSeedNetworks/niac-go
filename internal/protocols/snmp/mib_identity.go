package snmp

import (
	"bytes"
	"crypto/sha256"
	"net"
	"strconv"
	"strings"

	"github.com/gosnmp/gosnmp"
)

const (
	entPhysicalSerialNumber = "1.3.6.1.2.1.47.1.1.1.1.11"
	locallyAdministeredBit  = byte(0x02)
	unicastAddressMask      = byte(0xfe)
)

func (a *Agent) refreshAuthoredPhysicalIdentity() {
	if a.device == nil || len(a.device.MACAddress) == 0 {
		return
	}
	oids := a.mib.snapshotOIDs()
	mac := []byte(a.device.MACAddress)
	if len(mac) == MACAddressOctets {
		a.refreshPhysicalAddresses(oids, mac)
		a.mib.Set(dot1dBaseBridgeAddress, &OIDValue{Type: gosnmp.OctetString, Value: mac})
		a.refreshLLDPChassisIdentity(mac)
	}
	for _, oid := range oids {
		if strings.HasPrefix(oid, entPhysicalSerialNumber+".") {
			a.refreshPhysicalSerial(oid)
		}
	}
}

func (a *Agent) refreshPhysicalAddresses(oids []string, mac []byte) {
	identities := [][]byte{
		oidValueBytes(a.mib.Get(dot1dBaseBridgeAddress)),
		oidValueBytes(a.mib.Get(lldpLocChassisID)),
	}
	primary := a.primaryPhysicalAddress(oids)
	for _, oid := range oids {
		value := oidValueBytes(a.mib.Get(oid))
		if !strings.HasPrefix(oid, ifPhysAddress+".") || !hasPhysicalAddress(value) {
			continue
		}
		address := derivedPhysicalAddress(mac, oid)
		if oid == primary || bytes.Equal(value, identities[0]) || bytes.Equal(value, identities[1]) {
			address = mac
		}
		a.mib.Set(oid, &OIDValue{Type: gosnmp.OctetString, Value: address})
	}
}

func (a *Agent) primaryPhysicalAddress(oids []string) string {
	for _, iface := range a.device.Interfaces {
		index, ok := a.ifIndexForInterface(iface.Name)
		if !ok {
			continue
		}
		oid := ifPhysAddress + "." + index
		if hasPhysicalAddress(oidValueBytes(a.mib.Get(oid))) {
			return oid
		}
	}
	return a.lowestPhysicalAddress(oids)
}

func (a *Agent) lowestPhysicalAddress(oids []string) string {
	lowestIndex := int(^uint(0) >> 1)
	lowestOID := ""
	for _, oid := range oids {
		value := oidValueBytes(a.mib.Get(oid))
		if !strings.HasPrefix(oid, ifPhysAddress+".") || !hasPhysicalAddress(value) {
			continue
		}
		index, err := strconv.Atoi(strings.TrimPrefix(oid, ifPhysAddress+"."))
		if err == nil && index < lowestIndex {
			lowestIndex, lowestOID = index, oid
		}
	}
	return lowestOID
}

func hasPhysicalAddress(value []byte) bool {
	return len(value) == MACAddressOctets && !bytes.Equal(value, make([]byte, MACAddressOctets))
}

func oidValueBytes(value *OIDValue) []byte {
	if value == nil {
		return nil
	}
	switch typed := value.Value.(type) {
	case []byte:
		return typed
	case string:
		if parsed, err := net.ParseMAC(typed); err == nil {
			return parsed
		}
		return []byte(typed)
	default:
		return nil
	}
}

func derivedPhysicalAddress(mac []byte, oid string) []byte {
	sum := sha256.Sum256(append(append([]byte{}, mac...), oid...))
	derived := append([]byte{}, sum[:MACAddressOctets]...)
	derived[0] = (derived[0] | locallyAdministeredBit) & unicastAddressMask
	return derived
}

func (a *Agent) refreshLLDPChassisIdentity(mac []byte) {
	if a.mib.Get(lldpLocChassisID) == nil {
		return
	}
	a.mib.Set(lldpLocChassisIDSubtype, &OIDValue{Type: gosnmp.Integer, Value: ChassisIDSubtypeMAC})
	a.mib.Set(lldpLocChassisID, &OIDValue{Type: gosnmp.OctetString, Value: mac})
}

func (a *Agent) refreshPhysicalSerial(oid string) {
	if oidValueString(a.mib.Get(oid)) == "" {
		return
	}
	mac := strings.ToUpper(strings.ReplaceAll(a.device.MACAddress.String(), ":", ""))
	index := strings.TrimPrefix(oid, entPhysicalSerialNumber+".")
	serial := "NIAC-" + mac + "-" + index
	a.mib.Set(oid, &OIDValue{Type: gosnmp.OctetString, Value: serial})
}
