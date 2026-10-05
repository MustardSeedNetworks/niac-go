package config

import (
	"net/netip"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

// AuthoredStations is every client the device's radios author, each
// associated for as long as it says it has been at now. They seed the
// device's state; where a station is afterwards is that state's to say.
func AuthoredStations(device *Device, now time.Time) []devicestate.Station {
	if device == nil || device.WiFiConfig == nil {
		return nil
	}
	var stations []devicestate.Station
	for _, radio := range device.WiFiConfig.Radios {
		for _, client := range radio.Clients {
			// Validation has already checked the address syntax.
			address, _ := netip.ParseAddr(client.IPAddress)
			stations = append(stations, devicestate.Station{
				MAC: client.MAC, Radio: radio.Interface, IPAddress: address,
				SignalDBM: client.SignalDBM, SignalQualityPct: client.SignalQualityPct,
				AssociatedAt: now.Add(-time.Duration(client.AssociatedSeconds) * time.Second),
			})
		}
	}
	return stations
}
