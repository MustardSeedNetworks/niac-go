package protocols

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

// ErrRoamTargetInvalid means a roam's access point cannot take the station.
var ErrRoamTargetInvalid = errors.New("access point cannot take this station")

// RoamStation reassociates a station from one access point to another serving
// the same SSID. The new AP takes it first and the old one then reports it
// roamed, the order a real reassociation is seen in. Everything that can
// refuse is checked before either side changes, so a refused roam leaves the
// station where it was.
func (s *Stack) RoamStation(station, from, to string) error {
	s.reloadMu.RLock()
	defer s.reloadMu.RUnlock()
	fromDevice, fromStore, err := s.interfaceFaultTarget(from)
	if err != nil {
		return err
	}
	toDevice, toStore, err := s.interfaceFaultTarget(to)
	if err != nil {
		return err
	}
	current, associated := fromStore.Station(station)
	if !associated {
		return fmt.Errorf("roam %s from %s: %w", station, fromDevice.Name, devicestate.ErrStationNotFound)
	}
	radio, found := roamRadio(fromDevice, toDevice, current.Radio)
	if !found {
		return fmt.Errorf("roam %s to %s: %w", station, toDevice.Name, ErrRoamTargetInvalid)
	}
	next := current
	next.Radio = radio
	next.AssociatedAt = time.Now()
	if err = toStore.AssociateStation(next); err != nil {
		return fmt.Errorf("roam %s to %s: %w", station, toDevice.Name, err)
	}
	if _, err = fromStore.RoamStation(station); err != nil {
		return fmt.Errorf("roam %s from %s: %w", station, fromDevice.Name, err)
	}
	return nil
}

// roamRadio is the radio of the new AP that serves the SSID the station is
// on, so a station on the corporate network does not land on the guest one.
func roamRadio(from, to *config.Device, fromRadio string) (string, bool) {
	if from.WiFiConfig == nil || to.WiFiConfig == nil {
		return "", false
	}
	index := slices.IndexFunc(from.WiFiConfig.Radios, func(radio config.WiFiRadio) bool {
		return radio.Interface == fromRadio
	})
	if index < 0 {
		return "", false
	}
	ssid := from.WiFiConfig.Radios[index].SSID
	for _, radio := range to.WiFiConfig.Radios {
		if radio.SSID == ssid {
			return radio.Interface, true
		}
	}
	return "", false
}
