package scenario

import (
	"fmt"
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/converter"
	"github.com/MustardSeedNetworks/niac-go/internal/oui"
)

// Wireless stations are full simulated hosts on the corporate Wi-Fi network,
// added on top of the wired endpoints (owner, 2026-10-14). Each answers ARP,
// ping and its name service like any other endpoint, and the access point it
// is associated to reports it in its client table, which the site's
// controller aggregates.
//
// No switch learns a station's MAC. The packs' access points are lightweight
// ones joined to a controller, so client traffic is tunnelled to it rather
// than bridged onto the access port; the AP and controller tables are where
// an NMS finds a station, which is also where a real one would.
//
// Guest stations are not authored. A radio serves one SSID (see apSSID), and a
// station on the guest VLAN associated to the corporate SSID would contradict
// the radio that reports it.
const (
	// packWirelessClientsPerAP is one station per radio-bearing AP: the
	// densest pack (warehouse, 75 APs) then lands at 343 devices, inside the
	// presentation budget the owner raised for it.
	packWirelessClientsPerAP = 1
	maxWirelessClientsPerAP  = 8
	// maxSiteWirelessClients keeps the stations' static addresses below the
	// wireless network's DHCP pool, as maxSiteWorkstations does on the data
	// network.
	maxSiteWirelessClients = maxSiteWorkstations

	// wirelessClientLinkMbps is a 2x2 Wi-Fi 6 adapter on an 80 MHz channel.
	// A station's adapter, not the AP radio it talks to, sets the rate its
	// own interface reports.
	wirelessClientLinkMbps = 1_201

	wirelessClientSignalStrongestDBM = -45
	wirelessClientSignalSpreadDBM    = 25
	wirelessClientMinAssociated      = 600
	wirelessClientAssociatedSpread   = 28_200
	wirelessClientAssociatedStride   = 137
	// signalQualityOffsetDBM and signalQualityScale map a signal to the
	// percentage cDot11ClientSigQuality reports: -45 dBm reads 100 %, each
	// dB weaker costs two points.
	signalQualityOffsetDBM = 95
	signalQualityScale     = 2
	maxSignalQualityPct    = 100
)

// wirelessStation is one station of a site: the host it is, and the radio of
// the access point it is associated to.
type wirelessStation struct {
	kind      endpointKind
	name      string
	address   string
	macSuffix uint32
	ap        string
	radio     int
}

// wirelessKinds is what a vertical carries on its Wi-Fi: infusion pumps in a
// hospital, handhelds on a warehouse, plant or shop floor, and laptops
// everywhere else. Each is a kind the vertical's wired mix already has, so a
// station reads as the same product a tester has met on a wired port.
func wirelessKinds(profile string) []endpointKind {
	switch profile {
	case "hospital":
		return []endpointKind{appliance("infusion-pump", "PUMP")}
	case "warehouse", "manufacturing", "retail":
		return []endpointKind{appliance("rugged-handheld", "SCAN")}
	case "service-provider":
		return []endpointKind{personalComputer("windows-laptop", "LAP", "windows")}
	default:
		return []endpointKind{
			personalComputer("windows-laptop", "LAP", "windows"),
			personalComputer("macbook", "MBP", "darwin"),
		}
	}
}

// siteWirelessStations lays out a site's stations, access switch by access
// switch and AP by AP, spreading them over each AP's radios in plan order.
func siteWirelessStations(request Request, site Site) []wirelessStation {
	counts := request.Counts
	kinds := wirelessKinds(request.EndpointProfile)
	radios := len(apRadioPlan())
	perSwitch := counts.AccessPointsPerAccess * counts.WirelessClientsPerAP
	stations := make([]wirelessStation, 0, counts.AccessSwitches*perSwitch)
	for accessIndex := 1; accessIndex <= counts.AccessSwitches; accessIndex++ {
		for slot := 1; slot <= counts.AccessPointsPerAccess; slot++ {
			for client := 1; client <= counts.WirelessClientsPerAP; client++ {
				ordinal := len(stations) + 1
				onSwitch := (slot-1)*counts.WirelessClientsPerAP + client
				kind := kinds[(ordinal-1)%len(kinds)]
				stations = append(stations, wirelessStation{
					kind:      kind,
					name:      wirelessStationName(site, kind, accessIndex, onSwitch),
					address:   siteIP(site, vlanWiFiCorp, workstationHostOffset+ordinal),
					macSuffix: identitySuffix(&site, wirelessClientRole, ordinal),
					ap:        accessPointName(site, accessIndex, slot),
					radio:     (ordinal - 1) % radios,
				})
			}
		}
	}
	return stations
}

// wirelessClientRole keys the stations' identity codes apart from the wired
// endpoints', which a large site already numbers past what one byte can add.
const wirelessClientRole = "wireless-client"

// wirelessStationName follows wiredEndpointName, with a W before the number:
// it counts stations on that access switch's APs rather than switch ports, so
// it cannot collide with the wired name of the same slot. A personal computer
// keeps the compact form, which stays inside NetBIOS's 15 characters.
func wirelessStationName(site Site, kind endpointKind, accessIndex, onSwitch int) string {
	building, floor := location(accessIndex)
	if kind.personalComputer {
		return fmt.Sprintf("%s-%s-%d%dW%02d", site.Code, kind.prefix, building, floor, onSwitch)
	}
	return fmt.Sprintf("%s-%s-B%02d-F%02d-W%02d", site.Code, kind.prefix, building, floor, onSwitch)
}

func buildWirelessStations(request Request, site Site) []converter.Device {
	stations := siteWirelessStations(request, site)
	devices := make([]converter.Device, 0, len(stations))
	for _, station := range stations {
		nic := converter.Interface{
			Name: "wlan0", Type: "ieee80211", Network: siteNetworkName(site, "wifi-corp"),
			Address: station.address + "/24", MTU: standardMTU, Speed: wirelessClientLinkMbps,
			AdminStatus: "up", OperStatus: "up", Description: "Wireless client on " + apSSID(site.Code),
		}
		devices = append(devices, hostDevice(request, site, station.kind, hostPlacement{
			name: station.name, macSuffix: station.macSuffix, address: station.address,
			vlan: vlanWiFiCorp, nic: nic,
		}))
	}
	return devices
}

// associateWirelessStations writes every station into the client table of the
// radio it is associated to. The table carries the station's MAC, which the
// loader derives from the vendor's OUI, so it is resolved here the same way
// rather than restated.
func associateWirelessStations(request Request, devices []converter.Device) error {
	if request.Counts.WirelessClientsPerAP == 0 {
		return nil
	}
	registry, err := oui.LoadEmbedded()
	if err != nil {
		return fmt.Errorf("load IEEE OUI registry: %w", err)
	}
	byName := make(map[string]*converter.Device, len(devices))
	for index := range devices {
		byName[devices[index].Name] = &devices[index]
	}
	for _, site := range request.Sites {
		for _, station := range siteWirelessStations(request, site) {
			mac, allocateErr := registry.Allocate(profileByRole(station.kind.role).Vendor, station.macSuffix)
			if allocateErr != nil {
				return fmt.Errorf("station %s identity: %w", station.name, allocateErr)
			}
			radio := &byName[station.ap].Wifi.Radios[station.radio]
			radio.Clients = append(radio.Clients, wirelessClientEntry(station, mac.String()))
		}
	}
	return nil
}

// wirelessClientEntry is the station as its AP reports it. Signal and
// association time are seeded from the name, so they vary across a pack and
// never between two generations of it.
func wirelessClientEntry(station wirelessStation, mac string) converter.WifiClient {
	seed := 0
	for _, value := range []byte(station.name) {
		seed += int(value)
	}
	signal := wirelessClientSignalStrongestDBM - seed%wirelessClientSignalSpreadDBM
	return converter.WifiClient{
		MAC:       mac,
		IPAddress: station.address,
		AssociatedSeconds: wirelessClientMinAssociated +
			seed*wirelessClientAssociatedStride%wirelessClientAssociatedSpread,
		SignalDBM:        signal,
		SignalQualityPct: min(maxSignalQualityPct, signalQualityScale*(signal+signalQualityOffsetDBM)),
	}
}

func wirelessStationDNSRecords(request Request, site Site) []converter.DNSRecord {
	stations := siteWirelessStations(request, site)
	records := make([]converter.DNSRecord, 0, len(stations))
	for _, station := range stations {
		records = append(records, converter.DNSRecord{
			Name: strings.ToLower(station.name) + "." + request.Domain,
			IP:   station.address, TTL: dnsRecordTTL,
		})
	}
	return records
}
