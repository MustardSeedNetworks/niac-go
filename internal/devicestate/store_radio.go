package devicestate

import (
	"cmp"
	"errors"
	"maps"
	"slices"
)

// Transmit-power bounds in dBm. The floor is one milliwatt, which is the
// weakest level IEEE802dot11-MIB can report, and the ceiling is the strongest
// any regulatory domain permits an indoor AP, so a mistyped power is caught
// rather than replayed.
const (
	MinRadioTxPowerDBM = 1
	MaxRadioTxPowerDBM = 30
)

// ErrRadioTxPowerInvalid indicates a transmit power outside the radio's range.
var ErrRadioTxPowerInvalid = errors.New("radio transmit power must be between 1 and 30 dBm")

// RadioTxPower is a radio transmitting at other than its authored power. Only
// the departures are state: a radio with none transmits what was authored, so
// a scenario edit to the authored power is never shadowed by a stale copy.
type RadioTxPower struct {
	Interface string
	DBM       int
}

// SetRadioTxPower moves one radio off its authored transmit power.
func (s *Store) SetRadioTxPower(interfaceName string, dBm int) error {
	if dBm < MinRadioTxPowerDBM || dBm > MaxRadioTxPowerDBM {
		return ErrRadioTxPowerInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !interfaceExists(s.running.network.Interfaces, interfaceName) {
		return ErrInterfaceNotFound
	}
	if current, set := s.radioTxPower[interfaceName]; set && current == dBm {
		return nil
	}
	if s.radioTxPower == nil {
		s.radioTxPower = make(map[string]int)
	}
	s.radioTxPower[interfaceName] = dBm
	s.version++
	s.recordEvent(EventRadioUpdated, interfaceName)
	return nil
}

// RestoreRadioTxPower returns one radio to its authored transmit power.
func (s *Store) RestoreRadioTxPower(interfaceName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, set := s.radioTxPower[interfaceName]; !set {
		return
	}
	delete(s.radioTxPower, interfaceName)
	s.version++
	s.recordEvent(EventRadioUpdated, interfaceName)
}

// RadioTxPowerDBM returns the power a radio transmits at when it is not the
// authored one. It is cheap enough for an SNMP read: no snapshot is built.
func (s *Store) RadioTxPowerDBM(interfaceName string) (int, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	dBm, set := s.radioTxPower[interfaceName]
	return dBm, set
}

func sortedRadioTxPowers(powers map[string]int) []RadioTxPower {
	result := make([]RadioTxPower, 0, len(powers))
	for _, name := range slices.Sorted(maps.Keys(powers)) {
		result = append(result, RadioTxPower{Interface: name, DBM: powers[name]})
	}
	return result
}

func importRadioTxPowers(powers []RadioTxPower) map[string]int {
	result := make(map[string]int, len(powers))
	for _, power := range powers {
		result[power.Interface] = power.DBM
	}
	return result
}

// validStateRadioTxPowers accepts only powers a store could have set: each in
// range, once, on one of the device's own interfaces, in the sorted order an
// export writes.
func validStateRadioTxPowers(powers []RadioTxPower, interfaces []Interface) bool {
	for index, power := range powers {
		if power.DBM < MinRadioTxPowerDBM || power.DBM > MaxRadioTxPowerDBM ||
			!interfaceExists(interfaces, power.Interface) ||
			(index > 0 && cmp.Compare(powers[index-1].Interface, power.Interface) >= 0) {
			return false
		}
	}
	return true
}
