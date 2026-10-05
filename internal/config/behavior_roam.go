package config

import (
	"errors"
	"fmt"
	"maps"
	"net"
	"slices"
)

// ErrBehaviorRoamInvalid indicates a roam that no access point pair can serve.
var ErrBehaviorRoamInvalid = errors.New("invalid behavior roam")

// validateBehaviorRoams checks what can be known before the session runs: both
// ends are access points, the station is an authored client, and both APs
// serve the SSID it is on. Whether the station is on `from` when the phase
// starts depends on the phases before it, so the runtime refuses that one.
func validateBehaviorRoams(targets map[string]behaviorTarget, roams []BehaviorRoam) error {
	seen := make(map[string]struct{}, len(roams))
	for _, roam := range roams {
		station, err := net.ParseMAC(roam.Station)
		if err != nil {
			return fmt.Errorf("%w: station %q", ErrBehaviorRoamInvalid, roam.Station)
		}
		if _, exists := seen[station.String()]; exists {
			return fmt.Errorf("%w: station %s roams twice in one phase", ErrBehaviorRoamInvalid, station)
		}
		seen[station.String()] = struct{}{}
		if roam.From == roam.To {
			return fmt.Errorf("%w: station %s roams from %s to itself", ErrBehaviorRoamInvalid, station, roam.From)
		}
		for _, name := range []string{roam.From, roam.To} {
			if deviceErr := validateBehaviorDevice(targets, name); deviceErr != nil {
				return deviceErr
			}
		}
		ssid, authored := authoredStationSSID(targets, station)
		if !authored {
			return fmt.Errorf("%w: station %s is not an authored wireless client", ErrBehaviorRoamInvalid, station)
		}
		for _, name := range []string{roam.From, roam.To} {
			if !servesSSID(targets[name].device, ssid) {
				return fmt.Errorf("%w: %s has no radio serving %q for station %s",
					ErrBehaviorRoamInvalid, name, ssid, station)
			}
		}
	}
	return nil
}

func authoredStationSSID(targets map[string]behaviorTarget, station net.HardwareAddr) (string, bool) {
	for _, name := range slices.Sorted(maps.Keys(targets)) {
		target := targets[name]
		if target.device.WiFiConfig == nil {
			continue
		}
		for _, radio := range target.device.WiFiConfig.Radios {
			for _, client := range radio.Clients {
				if mac, err := net.ParseMAC(client.MAC); err == nil && slices.Equal(mac, station) {
					return radio.SSID, true
				}
			}
		}
	}
	return "", false
}

func servesSSID(device Device, ssid string) bool {
	return device.WiFiConfig != nil && slices.ContainsFunc(device.WiFiConfig.Radios, func(radio WiFiRadio) bool {
		return radio.SSID == ssid
	})
}
