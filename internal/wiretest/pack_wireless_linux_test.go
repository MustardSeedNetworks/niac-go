//go:build linux && integration

package wiretest_test

import (
	"net"
	"testing"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// cDot11ClientIPAddress is CISCO-DOT11-ASSOCIATION-MIB's station address
// column, the one an NMS reads to put a wireless client on its map.
const oidDot11ClientIPAddress = ".1.3.6.1.4.1.9.9.273.1.2.1.1.16"

// W1: a pack's wireless stations are simulated hosts, not rows. A tester on
// the pack's attachment port walks an AP's association table, then reaches the
// station at the address the AP reported, through the core, and the station
// answers for itself with the name the pack gave it.
func TestPackWirelessStationAnswersAtTheAddressItsAPReports(t *testing.T) {
	authored, _ := startPack(t, "hospital")
	ap, station := firstAssociatedStation(t, authored)

	rows := walkColumn(t, dialDevice(t, authored, ap), oidDot11ClientIPAddress)
	reported := false
	for _, row := range rows {
		if address, ok := row.Value.([]byte); ok && net.IP(address).Equal(station.IPAddresses[0]) {
			reported = true
		}
	}
	if !reported {
		t.Fatalf("%s association table %v does not report %s at %s", ap, rows, station.Name, station.IPAddresses[0])
	}

	result, err := dialDevice(t, authored, station.Name).Get([]string{".1.3.6.1.2.1.1.5.0"})
	if err != nil {
		t.Fatalf("polling %s at %s: %v", station.Name, station.IPAddresses[0], err)
	}
	name, _ := result.Variables[0].Value.([]byte)
	if result.Variables[0].Type != gosnmp.OctetString || string(name) != station.Name {
		t.Errorf("station at %s answered sysName %q, want %q", station.IPAddresses[0], name, station.Name)
	}
	t.Logf("%s reports %s at %s; the station answered sysName %q", ap, station.Name, station.IPAddresses[0], name)
}

// firstAssociatedStation finds an AP's first client row and the device that
// row describes. The hospital's stations are infusion pumps, which carry an
// SNMP agent, so the station can be asked who it is.
func firstAssociatedStation(t *testing.T, authored *config.Config) (string, *config.Device) {
	t.Helper()
	byMAC := make(map[string]*config.Device, len(authored.Devices))
	for index := range authored.Devices {
		byMAC[authored.Devices[index].MACAddress.String()] = &authored.Devices[index]
	}
	for index := range authored.Devices {
		ap := &authored.Devices[index]
		if ap.WiFiConfig == nil {
			continue
		}
		for _, radio := range ap.WiFiConfig.Radios {
			if len(radio.Clients) == 0 {
				continue
			}
			mac, err := net.ParseMAC(radio.Clients[0].MAC)
			if err != nil {
				t.Fatalf("%s station MAC %q: %v", ap.Name, radio.Clients[0].MAC, err)
			}
			station, simulated := byMAC[mac.String()]
			if !simulated {
				t.Fatalf("%s reports station %s, which no device answers on", ap.Name, mac)
			}
			return ap.Name, station
		}
	}
	t.Fatal("the hospital pack associates no wireless station")

	return "", nil
}
