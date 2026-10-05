package devicestate

import (
	"cmp"
	"errors"
	"maps"
	"net"
	"net/netip"
	"slices"
	"time"
)

// Errors returned when a station change does not match who is associated.
var (
	ErrStationInvalid    = errors.New("invalid wireless station")
	ErrStationAssociated = errors.New("station is already associated to this device")
	ErrStationNotFound   = errors.New("station is not associated to this device")
)

// Station is one wireless client associated to a radio of this device. Who is
// associated is state, not configuration: a roam moves a station between two
// access points without either one's configuration changing, so the authored
// clients only seed this set.
type Station struct {
	MAC              string // canonical lower-case colon form; the set's key
	Radio            string // the ieee80211 interface the station associated on
	IPAddress        netip.Addr
	SignalDBM        int
	SignalQualityPct int
	AssociatedAt     time.Time
	// ReturnRadio is the radio of the access point the station roamed from,
	// which a reset roams it back onto. An AP may serve one SSID on several
	// radios, and a cause degraded this one, so "any radio serving the SSID"
	// is not where the station came from.
	ReturnRadio string
}

// RoamCause is what made a station leave its access point. The cause happens
// to the radio the station was on, before the station reassociates elsewhere.
type RoamCause string

// The roam causes a timeline can author. An empty cause is a roam with no
// event behind it, the client's own choice.
const (
	// RoamCauseRadioDown takes the old radio's carrier down for the phase.
	RoamCauseRadioDown RoamCause = "radio_down"
	// RoamCauseTxPowerDrop lowers the old radio's transmit power for the
	// phase, so its clients hear it weaker and leave.
	RoamCauseTxPowerDrop RoamCause = "tx_power_drop"
)

// InstallStations seeds the authored associations. It records no event: the
// stations were associated before the simulation started, which is what the
// authored association time says.
func (s *Store) InstallStations(stations []Station) error {
	installed := make(map[string]Station, len(stations))
	for _, station := range stations {
		canonical, err := canonicalStation(station)
		if err != nil {
			return err
		}
		installed[canonical.MAC] = canonical
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stations = installed
	return nil
}

// AssociateStation adds one station. A station already associated here is
// refused rather than re-associated: a timeline that moves a station onto an
// AP it is already on has lost track of where the station is.
func (s *Store) AssociateStation(station Station) error {
	canonical, err := canonicalStation(station)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.stations[canonical.MAC]; exists {
		return ErrStationAssociated
	}
	if s.stations == nil {
		s.stations = make(map[string]Station)
	}
	s.stations[canonical.MAC] = canonical
	s.version++
	s.recordEvent(EventStationAssociated, canonical.MAC)
	return nil
}

// RoamStation removes a station that reassociated to another access point,
// returning what it was so the new AP can take it over.
func (s *Store) RoamStation(mac string) (Station, error) {
	canonical, err := canonicalMAC(mac)
	if err != nil {
		return Station{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	station, exists := s.stations[canonical]
	if !exists {
		return Station{}, ErrStationNotFound
	}
	delete(s.stations, canonical)
	s.version++
	s.recordEvent(EventStationRoamed, canonical)
	return station, nil
}

// Station returns one associated station.
func (s *Store) Station(mac string) (Station, bool) {
	canonical, err := canonicalMAC(mac)
	if err != nil {
		return Station{}, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	station, exists := s.stations[canonical]
	return station, exists
}

func canonicalStation(station Station) (Station, error) {
	mac, err := canonicalMAC(station.MAC)
	if err != nil || station.Radio == "" || !station.IPAddress.Is4() {
		return Station{}, ErrStationInvalid
	}
	station.MAC = mac
	return station, nil
}

func canonicalMAC(mac string) (string, error) {
	parsed, err := net.ParseMAC(mac)
	if err != nil || len(parsed) != stationMACLength {
		return "", ErrStationInvalid
	}
	return parsed.String(), nil
}

const stationMACLength = 6

func sortedStations(stations map[string]Station) []Station {
	return slices.SortedFunc(maps.Values(stations), func(left, right Station) int {
		return cmp.Compare(left.MAC, right.MAC)
	})
}

func importStations(stations []Station) map[string]Station {
	result := make(map[string]Station, len(stations))
	for _, station := range stations {
		result[station.MAC] = station
	}
	return result
}

// validStateStations accepts only stations a store could have produced: each
// in canonical form, once, on one of the device's own interfaces.
func validStateStations(stations []Station, interfaces []Interface) bool {
	seen := make(map[string]bool, len(stations))
	for _, station := range stations {
		canonical, err := canonicalStation(station)
		if err != nil || canonical.MAC != station.MAC || seen[station.MAC] ||
			!slices.ContainsFunc(interfaces, func(iface Interface) bool { return iface.Name == station.Radio }) {
			return false
		}
		seen[station.MAC] = true
	}
	return true
}
