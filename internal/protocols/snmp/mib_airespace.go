package snmp

import (
	"net"
	"slices"
	"strconv"
	"strings"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
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

// joinedAccessPoint is one AP that joined this controller, and the state its
// associations live in once the stack has bound it.
type joinedAccessPoint struct {
	device  *config.Device
	state   *devicestate.Store
	version uint64
}

// SynthesizeWirelessController registers the APs that joined this controller
// and their associated stations. The stack supplies the APs, because an agent
// knows only its own device. A controller whose capture carries the AIRESPACE
// tree keeps it: the real controller is the authority on what joined it.
func (a *Agent) SynthesizeWirelessController(accessPoints []*config.Device) {
	if a == nil || len(accessPoints) == 0 || a.walkOwnsAirespace() {
		return
	}
	a.joined = make([]joinedAccessPoint, 0, len(accessPoints))
	for _, accessPoint := range accessPoints {
		if len(accessPoint.MACAddress) == 0 || accessPoint.WiFiConfig == nil {
			continue
		}
		a.registerJoinedAccessPoint(accessPoint)
		a.joined = append(a.joined, joinedAccessPoint{device: accessPoint})
	}
	a.replaceMobileStations()
	a.mib.Reindex()
}

// BindJoinedAccessPointStates gives the controller the state of each AP that
// joined it, so a station roaming between them moves in the controller's table
// too. The stack binds them once every device has state; until then the
// controller reports the authored clients.
func (a *Agent) BindJoinedAccessPointStates(state func(*config.Device) *devicestate.Store) {
	if a == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	for index := range a.joined {
		a.joined[index].state = state(a.joined[index].device)
		a.joined[index].version = 0
	}
	a.syncJoinedStationsLocked()
}

// syncJoinedStations rebuilds the station table when any joined AP's state has
// moved on since it was last built. Each request asks, as it does of the
// controller's own state, so a roam is visible to the next walk.
func (a *Agent) syncJoinedStations() {
	if len(a.joined) == 0 {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.syncJoinedStationsLocked()
}

func (a *Agent) syncJoinedStationsLocked() {
	changed := false
	for index := range a.joined {
		if state := a.joined[index].state; state != nil {
			if version := state.Version(); version != a.joined[index].version {
				a.joined[index].version = version
				changed = true
			}
		}
	}
	if changed {
		a.replaceMobileStations()
	}
}

func (a *Agent) walkOwnsAirespace() bool {
	next, _ := a.mib.GetNext(airespaceRoot)

	return strings.HasPrefix(strings.TrimPrefix(next, "."), airespaceRoot+".")
}

func (a *Agent) registerJoinedAccessPoint(accessPoint *config.Device) {
	suffix := "." + octetIndex(accessPoint.MACAddress)

	a.mib.Set(bsnAPDot3MacAddress+suffix, macValue(accessPoint.MACAddress))
	a.mib.Set(bsnAPNumOfSlots+suffix, &OIDValue{Type: gosnmp.Integer, Value: len(accessPoint.WiFiConfig.Radios)})
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
}

// replaceMobileStations rebuilds the station table from who is on each joined
// AP now. The table is keyed by the station alone, so a roam moves its row's
// AP columns rather than adding a row.
func (a *Agent) replaceMobileStations() {
	entries := make(map[string]*OIDValue)
	for _, joined := range a.joined {
		stations := config.AuthoredStations(joined.device, a.startTime)
		if joined.state != nil {
			stations = joined.state.Snapshot().Stations
		}
		for _, station := range stations {
			addMobileStation(entries, joined.device, station)
		}
	}
	a.mib.ReplacePrefix(bsnMobileStationEntry, entries)
}

func addMobileStation(entries map[string]*OIDValue, accessPoint *config.Device, station devicestate.Station) {
	mac, err := net.ParseMAC(station.MAC)
	if err != nil {
		return
	}
	// The slot is the radio's position on the AP, counted from 0 as the
	// capture's stations report it.
	slot := slices.IndexFunc(accessPoint.WiFiConfig.Radios, func(radio config.WiFiRadio) bool {
		return radio.Interface == station.Radio
	})
	if slot < 0 {
		return
	}
	suffix := "." + octetIndex(mac)

	entries[bsnMobileStationMacAddress+suffix] = macValue(mac)
	entries[bsnMobileStationIPAddress+suffix] = &OIDValue{Type: gosnmp.IPAddress, Value: station.IPAddress.String()}
	entries[bsnMobileStationAPMacAddr+suffix] = macValue(accessPoint.MACAddress)
	entries[bsnMobileStationAPIfSlotID+suffix] = &OIDValue{Type: gosnmp.Integer, Value: slot}
	entries[bsnMobileStationSsid+suffix] = &OIDValue{
		Type: gosnmp.OctetString, Value: accessPoint.WiFiConfig.Radios[slot].SSID,
	}
	entries[bsnMobileStationStatus+suffix] = &OIDValue{Type: gosnmp.Integer, Value: bsnStationAssociated}
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
