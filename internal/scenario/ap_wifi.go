package scenario

import (
	"fmt"
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/converter"
	"github.com/MustardSeedNetworks/niac-go/internal/safeconv"
)

// Transmit power per band, in dBm. A 2.4 GHz radio is turned down relative to
// the 5 and 6 GHz ones because its cells are otherwise far larger than theirs,
// which is the first thing a site survey corrects.
const (
	apTxPower24GHzDBM = 14
	apTxPower5GHzDBM  = 17
	apTxPower6GHzDBM  = 17

	// apBSSIDPrefix is the locally administered, unicast OUI a simulated radio
	// answers on. The scenario invents these addresses, and claiming a real
	// vendor's OUI for an invented radio would put a lie in a tester's report.
	apBSSIDPrefix = 0x02
)

// accessPointWiFi authors the radios of one generated access point.
//
// The block is the AP's own account of its radios — SSID, BSSID, band, channel
// and power — and `internal/protocols/snmp` serves it as IEEE802dot11-MIB at
// each radio interface's ifIndex. Nothing else may state those objects: a
// generated AP that also carried them as `add_mibs` rows would answer two
// sources for one object, which is what #2163 found.
func accessPointWiFi(site string, macSuffix uint32) *converter.WifiConfig {
	plan := apRadioPlan()
	radios := make([]converter.WifiRadio, 0, len(plan))
	for index, radio := range plan {
		radios = append(radios, converter.WifiRadio{
			Interface:  fmt.Sprintf("Dot11Radio%d", index),
			SSID:       apSSID(site),
			BSSID:      apBSSID(macSuffix, index),
			Band:       radio.band,
			Channel:    radio.channel,
			TxPowerDBM: apTxPowerDBM(radio.band),
		})
	}
	return &converter.WifiConfig{Radios: radios}
}

// apSSID is the corporate network every radio of a site serves. The guest
// network the packs trunk on VLAN 230 is not here: a second SSID on one radio
// is a second BSS, and IEEE802dot11-MIB carries one dot11DesiredSSID per
// interface with no BSS table, so guest waits for the vendor MIB family that
// can express it.
func apSSID(site string) string {
	return strings.ToUpper(site) + "-CORP"
}

// apBSSID gives each radio of each access point its own address. The radio
// index is the byte below the device's identity suffix, which is unique per
// device, so no two radios in a pack — or across the fleet — collide.
func apBSSID(macSuffix uint32, radio int) string {
	return fmt.Sprintf("%02x:00:%02x:%02x:%02x:%02x",
		apBSSIDPrefix,
		radio+1,
		safeconv.ByteFromUint32(macSuffix>>macHighByteShift),
		safeconv.ByteFromUint32(macSuffix>>macMiddleByteShift),
		safeconv.ByteFromUint32(macSuffix),
	)
}

func apTxPowerDBM(band string) int {
	switch band {
	case apBand24GHz:
		return apTxPower24GHzDBM
	case apBand6GHz:
		return apTxPower6GHzDBM
	default:
		return apTxPower5GHzDBM
	}
}
