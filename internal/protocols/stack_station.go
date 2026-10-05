package protocols

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/behavior"
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
//
// A cause happens first, to the radio the station is on, because it is why
// the station left. A return roam undoes the cause on the radio the station
// came from and lands it there.
func (s *Stack) RoamStation(roam behavior.RoamAction) error {
	s.reloadMu.RLock()
	defer s.reloadMu.RUnlock()
	fromDevice, fromStore, err := s.interfaceFaultTarget(roam.From)
	if err != nil {
		return err
	}
	toDevice, toStore, err := s.interfaceFaultTarget(roam.To)
	if err != nil {
		return err
	}
	current, associated := fromStore.Station(roam.Station)
	if !associated {
		return fmt.Errorf("roam %s from %s: %w", roam.Station, fromDevice.Name, devicestate.ErrStationNotFound)
	}
	if _, already := toStore.Station(roam.Station); already {
		return fmt.Errorf("roam %s to %s: %w", roam.Station, toDevice.Name, devicestate.ErrStationAssociated)
	}
	radio, found := roamRadio(fromDevice, toDevice, current.Radio)
	if roam.Return {
		radio, found = current.ReturnRadio, current.ReturnRadio != "" && servesRadio(toDevice, radio)
	}
	if !found {
		return fmt.Errorf("roam %s to %s: %w", roam.Station, toDevice.Name, ErrRoamTargetInvalid)
	}
	now := time.Now()
	causeDevice, causeStore, causeRadio := fromDevice, fromStore, current.Radio
	if roam.Return {
		causeDevice, causeStore, causeRadio = toDevice, toStore, radio
	}
	if err = s.applyRoamCause(roam, causeDevice, causeStore, causeRadio, now); err != nil {
		return fmt.Errorf("roam %s cause %s on %s: %w", roam.Station, roam.Cause, causeDevice.Name, err)
	}
	next := current
	next.Radio, next.ReturnRadio = radio, current.Radio
	next.AssociatedAt = now
	if err = toStore.AssociateStation(next); err != nil {
		return fmt.Errorf("roam %s to %s: %w", roam.Station, toDevice.Name, err)
	}
	if _, err = fromStore.RoamStation(roam.Station); err != nil {
		return fmt.Errorf("roam %s from %s: %w", roam.Station, fromDevice.Name, err)
	}
	return nil
}

// applyRoamCause does to the radio what the cause names, or undoes it when the
// station returns to that radio.
func (s *Stack) applyRoamCause(
	roam behavior.RoamAction,
	device *config.Device,
	store *devicestate.Store,
	radio string,
	now time.Time,
) error {
	switch roam.Cause {
	case "":
		return nil
	case devicestate.RoamCauseRadioDown:
		down := carrierFaultValue
		if roam.Return {
			down = 0
		}
		return s.setInterfaceFaultNoLock(device.Name, radio, devicestate.FaultLinkDown, down, now)
	case devicestate.RoamCauseTxPowerDrop:
		if !s.snmpAgents[device].interfaceFaultObservable(radio) {
			return ErrFaultUnobservable
		}
		if roam.Return {
			store.RestoreRadioTxPower(radio)
			return nil
		}
		return store.SetRadioTxPower(radio, roam.TxPowerDBM)
	}
	return fmt.Errorf("unknown cause %q", roam.Cause)
}

// carrierFaultValue is what arms link_down: the fault is an outcome, and any
// non-zero value takes the carrier down.
const carrierFaultValue = 1

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

func servesRadio(device *config.Device, iface string) bool {
	return device.WiFiConfig != nil && slices.ContainsFunc(device.WiFiConfig.Radios, func(radio config.WiFiRadio) bool {
		return radio.Interface == iface
	})
}
