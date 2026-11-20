package scenario

import "github.com/MustardSeedNetworks/niac-go/internal/converter"

const (
	entPhysicalChassis  = "3"
	entPhysicalNoParent = "0"
	entPhysicalNoRelPos = "-1"
	entPhysicalNotFRU   = "2"
)

// answersEntityMIB reports whether a device type answers ENTITY-MIB the way
// real network gear does. Hosts and appliances do not inventory a chassis, and
// access points already author theirs in apDiscoveryMIBs.
func answersEntityMIB(deviceType string) bool {
	switch deviceType {
	case "switch", "layer3-switch", "router", "firewall":
		return true
	}

	return false
}

// manufacturerName is the entPhysicalMfgName a vendor reports.
func manufacturerName(vendor string) string {
	switch vendor {
	case "cisco":
		return "Cisco Systems, Inc."
	case "palo alto":
		return "Palo Alto Networks"
	}

	return vendor
}

// chassisEntityMIBs authors the one entPhysicalTable row a switch, router or
// firewall always carries: its chassis. Without it an inventory collector sees
// a device with interfaces and no hardware, which no real device reports.
func chassisEntityMIBs(name, vendor, model, platform, software string, macSuffix uint32) []converter.AddMib {
	const chassis = "1.3.6.1.2.1.47.1.1.1.1."

	return []converter.AddMib{
		{OID: chassis + "2.1", Type: snmpTypeString, Value: platform},
		{OID: chassis + "4.1", Type: snmpTypeInteger, Value: entPhysicalNoParent},
		{OID: chassis + "5.1", Type: snmpTypeInteger, Value: entPhysicalChassis},
		{OID: chassis + "6.1", Type: snmpTypeInteger, Value: entPhysicalNoRelPos},
		{OID: chassis + "7.1", Type: snmpTypeString, Value: name},
		{OID: chassis + "10.1", Type: snmpTypeString, Value: software},
		{OID: chassis + "11.1", Type: snmpTypeString, Value: apSerialNumber(macSuffix)},
		{OID: chassis + "12.1", Type: snmpTypeString, Value: manufacturerName(vendor)},
		{OID: chassis + "13.1", Type: snmpTypeString, Value: model},
		{OID: chassis + "16.1", Type: snmpTypeInteger, Value: entPhysicalNotFRU},
	}
}
