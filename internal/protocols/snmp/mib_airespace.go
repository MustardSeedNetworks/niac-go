package snmp

import (
	"net"
	"strconv"
	"strings"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// AIRESPACE-WIRELESS-MIB object prefixes. A lightweight AP is reported by the
// controller it joined, not by itself: an NMS walks the controller's AP table
// to find the APs and its mobile-station table to find who is on them. Every
// arc below is read off the controller captures in the walk corpus
// (walks/raw/cisco/cisco-controller-0*.walk, "Cisco Controller"), where an
// associated station's bsnMobileStationAPMacAddr is the bsnAPDot3MacAddress
// row key of the AP it is on.
//
// Only the columns an authored AP or client has something true to say are
// registered. The capture also answers the AP's serial, boot version, primary
// controller and forty more, and nothing in a scenario authors any of them.
const (
	airespaceRoot = "1.3.6.1.4.1.14179"

	// bsnMobileStationTable (…14179.2.1.4), indexed by the station's MAC as
	// six bare octets.
	bsnMobileStationEntry      = airespaceRoot + ".2.1.4.1"
	bsnMobileStationMacAddress = bsnMobileStationEntry + ".1"
	bsnMobileStationIPAddress  = bsnMobileStationEntry + ".2"
	bsnMobileStationAPMacAddr  = bsnMobileStationEntry + ".4"
	bsnMobileStationAPIfSlotID = bsnMobileStationEntry + ".5"
	bsnMobileStationSsid       = bsnMobileStationEntry + ".7"
	bsnMobileStationStatus     = bsnMobileStationEntry + ".9"

	// bsnAPTable (…14179.2.2.1), indexed by the AP's MAC as six bare octets.
	bsnAPEntry           = airespaceRoot + ".2.2.1.1"
	bsnAPDot3MacAddress  = bsnAPEntry + ".1"
	bsnAPNumOfSlots      = bsnAPEntry + ".2"
	bsnAPName            = bsnAPEntry + ".3"
	bsnAPLocation        = bsnAPEntry + ".4"
	bsnAPOperationStatus = bsnAPEntry + ".6"
	bsnAPModel           = bsnAPEntry + ".16"
	bsnAPIPAddress       = bsnAPEntry + ".19"

	bsnAPAssociated      = 1 // bsnAPOperationStatus associated(1)
	bsnStationAssociated = 3 // bsnMobileStationStatus associated(3)
)

// SynthesizeWirelessController registers the APs that joined this controller
// and their associated stations. The stack supplies the APs, because an agent
// knows only its own device. A controller whose capture carries the AIRESPACE
// tree keeps it: the real controller is the authority on what joined it.
func (a *Agent) SynthesizeWirelessController(accessPoints []*config.Device) {
	if a == nil || len(accessPoints) == 0 || a.walkOwnsAirespace() {
		return
	}
	for _, accessPoint := range accessPoints {
		a.registerJoinedAccessPoint(accessPoint)
	}
	a.mib.Reindex()
}

func (a *Agent) walkOwnsAirespace() bool {
	next, _ := a.mib.GetNext(airespaceRoot)

	return strings.HasPrefix(strings.TrimPrefix(next, "."), airespaceRoot+".")
}

func (a *Agent) registerJoinedAccessPoint(accessPoint *config.Device) {
	if len(accessPoint.MACAddress) == 0 || accessPoint.WiFiConfig == nil {
		return
	}
	suffix := "." + octetIndex(accessPoint.MACAddress)
	radios := accessPoint.WiFiConfig.Radios

	a.mib.Set(bsnAPDot3MacAddress+suffix, macValue(accessPoint.MACAddress))
	a.mib.Set(bsnAPNumOfSlots+suffix, &OIDValue{Type: gosnmp.Integer, Value: len(radios)})
	a.mib.Set(bsnAPName+suffix, &OIDValue{Type: gosnmp.OctetString, Value: accessPoint.Name})
	if location := accessPoint.SNMPConfig.SysLocation; location != "" {
		a.mib.Set(bsnAPLocation+suffix, &OIDValue{Type: gosnmp.OctetString, Value: location})
	}
	a.mib.Set(bsnAPOperationStatus+suffix, &OIDValue{Type: gosnmp.Integer, Value: bsnAPAssociated})
	if model := accessPoint.Properties["model"]; model != "" {
		a.mib.Set(bsnAPModel+suffix, &OIDValue{Type: gosnmp.OctetString, Value: model})
	}
	if address := firstDeviceIPv4(accessPoint); address != nil {
		a.mib.Set(bsnAPIPAddress+suffix, &OIDValue{Type: gosnmp.IPAddress, Value: address.String()})
	}

	// The slot is the radio's position on the AP, counted from 0 as the
	// capture's stations report it.
	for slot, radio := range radios {
		for _, client := range radio.Clients {
			a.registerMobileStation(client, accessPoint.MACAddress, slot, radio.SSID)
		}
	}
}

func (a *Agent) registerMobileStation(client config.WiFiClient, accessPoint net.HardwareAddr, slot int, ssid string) {
	station, err := net.ParseMAC(client.MAC)
	if err != nil {
		return
	}
	suffix := "." + octetIndex(station)
	// The table is keyed by the station alone, so a station authored on two
	// radios (a roam mid-flight) is reported where it was first authored.
	if a.mib.Get(bsnMobileStationMacAddress+suffix) != nil {
		return
	}

	a.mib.Set(bsnMobileStationMacAddress+suffix, macValue(station))
	if address := net.ParseIP(client.IPAddress).To4(); address != nil {
		a.mib.Set(bsnMobileStationIPAddress+suffix,
			&OIDValue{Type: gosnmp.IPAddress, Value: address.String()})
	}
	a.mib.Set(bsnMobileStationAPMacAddr+suffix, macValue(accessPoint))
	a.mib.Set(bsnMobileStationAPIfSlotID+suffix, &OIDValue{Type: gosnmp.Integer, Value: slot})
	a.mib.Set(bsnMobileStationSsid+suffix, &OIDValue{Type: gosnmp.OctetString, Value: ssid})
	a.mib.Set(bsnMobileStationStatus+suffix, &OIDValue{Type: gosnmp.Integer, Value: bsnStationAssociated})
}

// octetIndex encodes a MAC as six bare arcs. MacAddress is fixed-length, so the
// index carries no length prefix -- the capture's rows are keyed exactly so.
func octetIndex(address net.HardwareAddr) string {
	arcs := make([]string, 0, len(address))
	for _, octet := range address {
		arcs = append(arcs, strconv.Itoa(int(octet)))
	}

	return strings.Join(arcs, ".")
}

func firstDeviceIPv4(device *config.Device) net.IP {
	for _, address := range device.IPAddresses {
		if ipv4 := address.To4(); ipv4 != nil {
			return ipv4
		}
	}

	return nil
}
