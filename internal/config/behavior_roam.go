package config

import (
	"errors"
	"fmt"
	"maps"
	"net"
	"slices"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
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
		if err = validateBehaviorRoam(targets, roam, station); err != nil {
			return err
		}
	}
	return nil
}

func validateBehaviorRoam(targets map[string]behaviorTarget, roam BehaviorRoam, station net.HardwareAddr) error {
	if roam.From == roam.To {
		return fmt.Errorf("%w: station %s roams from %s to itself", ErrBehaviorRoamInvalid, station, roam.From)
	}
	for _, name := range []string{roam.From, roam.To} {
		if err := validateBehaviorDevice(targets, name); err != nil {
			return err
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
	if err := validateRoamCause(targets[roam.From].device, roam, ssid); err != nil {
		return fmt.Errorf("%w: station %s: %w", ErrBehaviorRoamInvalid, station, err)
	}
	return nil
}

// errRoamCauseUnobservable is a cause on an access point that serves no SNMP:
// a radio taken down there changes nothing a poller can see, and the fault
// path refuses it, so the timeline would stop at its first roam.
var errRoamCauseUnobservable = errors.New("the old access point serves no SNMP to report its radio")

func validateRoamCause(from Device, roam BehaviorRoam, ssid string) error {
	if roam.Cause != devicestate.RoamCauseTxPowerDrop && roam.TxPowerDBM != 0 {
		return fmt.Errorf("tx_power_dbm %d needs cause %s", roam.TxPowerDBM, devicestate.RoamCauseTxPowerDrop)
	}
	switch roam.Cause {
	case "":
		return nil
	case devicestate.RoamCauseRadioDown, devicestate.RoamCauseTxPowerDrop:
		if !SNMPv2Enabled(from.SNMPConfig) && !SNMPv3Enabled(from.SNMPv3Config) {
			return fmt.Errorf("cause %s on %s: %w", roam.Cause, from.Name, errRoamCauseUnobservable)
		}
	default:
		return fmt.Errorf("unknown cause %q", roam.Cause)
	}
	if roam.Cause == devicestate.RoamCauseTxPowerDrop {
		return validateTxPowerDrop(from, roam.TxPowerDBM, ssid)
	}
	return nil
}

// validateTxPowerDrop requires the power to be a drop on whichever of the old
// AP's radios the station is on. Which one that is decides at run time, so the
// power must be below every radio serving the station's SSID.
func validateTxPowerDrop(from Device, dBm int, ssid string) error {
	if dBm == 0 {
		return fmt.Errorf("cause %s needs tx_power_dbm", devicestate.RoamCauseTxPowerDrop)
	}
	for _, radio := range from.WiFiConfig.Radios {
		if radio.SSID == ssid && dBm >= radio.TxPowerDBM {
			return fmt.Errorf("tx_power_dbm %d is no drop from %s %s at %d dBm",
				dBm, from.Name, radio.Interface, radio.TxPowerDBM)
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
