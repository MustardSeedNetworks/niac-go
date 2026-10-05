package devicestate_test

import (
	"errors"
	"net/netip"
	"slices"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

const (
	stationMAC      = "02:c0:17:a4:03:6b"
	otherStationMAC = "02:c0:17:a4:03:6c"
)

func newStationTestStore(t *testing.T) *devicestate.Store {
	t.Helper()
	store := devicestate.NewStore(devicestate.Identity{Hostname: "MED-AP-01"})
	store.ReplaceNetwork(devicestate.Network{Interfaces: []devicestate.Interface{
		{Name: "Dot11Radio0", AdminUp: true, OperUp: true, CarrierUp: true},
		{Name: "Dot11Radio1", AdminUp: true, OperUp: true, CarrierUp: true},
	}})
	return store
}

func testStation(mac string) devicestate.Station {
	return devicestate.Station{
		MAC: mac, Radio: "Dot11Radio0", IPAddress: netip.MustParseAddr("10.20.220.101"),
		SignalDBM: -55, SignalQualityPct: 80,
		AssociatedAt: time.Date(2026, 10, 5, 9, 0, 0, 0, time.UTC),
	}
}

func stationMACs(stations []devicestate.Station) []string {
	result := make([]string, 0, len(stations))
	for _, station := range stations {
		result = append(result, station.MAC)
	}
	return result
}

// TestInstallStationsRecordsNoEvent: the authored clients were associated
// before the session started, so seeding them is not an association an NMS
// should hear about.
func TestInstallStationsRecordsNoEvent(t *testing.T) {
	store := newStationTestStore(t)
	before := len(store.Events())

	if err := store.InstallStations([]devicestate.Station{testStation("02:C0:17:A4:03:6B")}); err != nil {
		t.Fatalf("InstallStations() error = %v", err)
	}

	if got := stationMACs(store.Snapshot().Stations); !slices.Equal(got, []string{stationMAC}) {
		t.Errorf("stations = %v, want the station in canonical form", got)
	}
	if after := len(store.Events()); after != before {
		t.Errorf("seeding recorded %d events", after-before)
	}
}

// TestRoamMovesAStationAndRecordsBothHalves: the new AP records the
// association and the old one the roam, each targeting the station.
func TestRoamMovesAStationAndRecordsBothHalves(t *testing.T) {
	from, to := newStationTestStore(t), newStationTestStore(t)
	if err := from.InstallStations([]devicestate.Station{testStation(stationMAC)}); err != nil {
		t.Fatalf("InstallStations() error = %v", err)
	}

	roamed, err := from.RoamStation(stationMAC)
	if err != nil {
		t.Fatalf("RoamStation() error = %v", err)
	}
	if err = to.AssociateStation(roamed); err != nil {
		t.Fatalf("AssociateStation() error = %v", err)
	}

	if stations := from.Snapshot().Stations; len(stations) != 0 {
		t.Errorf("old AP still has %v", stationMACs(stations))
	}
	if got := stationMACs(to.Snapshot().Stations); !slices.Equal(got, []string{stationMAC}) {
		t.Errorf("new AP stations = %v", got)
	}
	for _, testCase := range []struct {
		store *devicestate.Store
		kind  devicestate.EventKind
	}{
		{from, devicestate.EventStationRoamed},
		{to, devicestate.EventStationAssociated},
	} {
		events := testCase.store.Events()
		last := events[len(events)-1]
		if last.Kind != testCase.kind || last.Target != stationMAC {
			t.Errorf("last event = %s %q, want %s %q", last.Kind, last.Target, testCase.kind, stationMAC)
		}
	}
}

func TestStationChangesThatDoNotMatchAreRefused(t *testing.T) {
	store := newStationTestStore(t)
	if err := store.InstallStations([]devicestate.Station{testStation(stationMAC)}); err != nil {
		t.Fatalf("InstallStations() error = %v", err)
	}
	version := store.Version()

	if err := store.AssociateStation(testStation(stationMAC)); !errors.Is(err, devicestate.ErrStationAssociated) {
		t.Errorf("AssociateStation(already here) error = %v, want ErrStationAssociated", err)
	}
	if _, err := store.RoamStation(otherStationMAC); !errors.Is(err, devicestate.ErrStationNotFound) {
		t.Errorf("RoamStation(not here) error = %v, want ErrStationNotFound", err)
	}
	invalid := testStation(otherStationMAC)
	invalid.IPAddress = netip.MustParseAddr("2001:db8::1")
	if err := store.AssociateStation(invalid); !errors.Is(err, devicestate.ErrStationInvalid) {
		t.Errorf("AssociateStation(IPv6) error = %v, want ErrStationInvalid", err)
	}
	if store.Version() != version {
		t.Errorf("a refused change moved the version from %d to %d", version, store.Version())
	}
}

// TestCheckpointRestoresWhoIsAssociated: who is associated is part of what a
// scenario is doing, as the faults are, so a checkpoint taken before a roam
// puts the station back.
func TestCheckpointRestoresWhoIsAssociated(t *testing.T) {
	store := newStationTestStore(t)
	if err := store.InstallStations([]devicestate.Station{testStation(stationMAC)}); err != nil {
		t.Fatalf("InstallStations() error = %v", err)
	}
	store.SaveCheckpoint("before-roam")
	if _, err := store.RoamStation(stationMAC); err != nil {
		t.Fatalf("RoamStation() error = %v", err)
	}

	if err := store.RestoreCheckpoint("before-roam"); err != nil {
		t.Fatalf("RestoreCheckpoint() error = %v", err)
	}

	if got := stationMACs(store.Snapshot().Stations); !slices.Equal(got, []string{stationMAC}) {
		t.Errorf("stations after restore = %v", got)
	}
}

func TestStateCarriesStationsThroughExportAndRestore(t *testing.T) {
	source := newStationTestStore(t)
	if err := source.InstallStations([]devicestate.Station{testStation(stationMAC)}); err != nil {
		t.Fatalf("InstallStations() error = %v", err)
	}
	moved := testStation(otherStationMAC)
	moved.Radio = "Dot11Radio1"
	if err := source.AssociateStation(moved); err != nil {
		t.Fatalf("AssociateStation() error = %v", err)
	}
	state := source.ExportState()

	target := newStationTestStore(t)
	if err := target.RestoreState(state); err != nil {
		t.Fatalf("RestoreState() error = %v", err)
	}

	if got, want := target.Snapshot().Stations, source.Snapshot().Stations; !slices.Equal(got, want) {
		t.Errorf("restored stations = %#v, want %#v", got, want)
	}
}

// TestRestoreStateRefusesStationsNoStoreProduces: a durable record is read
// back from disk, so a station the store could not have held -- off its own
// radios, twice, or not in canonical form -- is refused rather than served.
func TestRestoreStateRefusesStationsNoStoreProduces(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		mutate func(*devicestate.State)
	}{
		{"unknown radio", func(state *devicestate.State) { state.Stations[0].Radio = "Dot11Radio9" }},
		{"twice", func(state *devicestate.State) { state.Stations = append(state.Stations, state.Stations[0]) }},
		{"upper-case MAC", func(state *devicestate.State) { state.Stations[0].MAC = "02:C0:17:A4:03:6B" }},
		{"in a checkpoint", func(state *devicestate.State) { state.Checkpoints[0].Stations[0].Radio = "eth9" }},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			source := newStationTestStore(t)
			if err := source.InstallStations([]devicestate.Station{testStation(stationMAC)}); err != nil {
				t.Fatalf("InstallStations() error = %v", err)
			}
			source.SaveCheckpoint("saved")
			state := source.ExportState()
			testCase.mutate(&state)

			if err := newStationTestStore(t).RestoreState(state); !errors.Is(err, devicestate.ErrStateInvalid) {
				t.Errorf("RestoreState() error = %v, want ErrStateInvalid", err)
			}
		})
	}
}
