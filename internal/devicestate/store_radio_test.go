package devicestate_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func lastEventKind(t *testing.T, store *devicestate.Store) devicestate.EventKind {
	t.Helper()
	events := store.Events()
	if len(events) == 0 {
		t.Fatal("no events recorded")
	}
	return events[len(events)-1].Kind
}

// TestRadioTxPowerMovesAndReturns: a radio off its authored power is state
// with one event each way, and a repeat of either is no transition.
func TestRadioTxPowerMovesAndReturns(t *testing.T) {
	store := newStationTestStore(t)

	if err := store.SetRadioTxPower("Dot11Radio0", 8); err != nil {
		t.Fatalf("SetRadioTxPower() error = %v", err)
	}
	if dBm, moved := store.RadioTxPowerDBM("Dot11Radio0"); !moved || dBm != 8 {
		t.Fatalf("RadioTxPowerDBM() = %d, %v; want 8, true", dBm, moved)
	}
	if kind := lastEventKind(t, store); kind != devicestate.EventRadioUpdated {
		t.Errorf("event = %s, want %s", kind, devicestate.EventRadioUpdated)
	}
	version := store.Version()
	if err := store.SetRadioTxPower("Dot11Radio0", 8); err != nil || store.Version() != version {
		t.Errorf("repeating the power changed state: error %v, version %d -> %d", err, version, store.Version())
	}

	store.RestoreRadioTxPower("Dot11Radio0")
	if _, moved := store.RadioTxPowerDBM("Dot11Radio0"); moved {
		t.Error("the radio is still off its authored power after the restore")
	}
	version = store.Version()
	store.RestoreRadioTxPower("Dot11Radio0")
	if store.Version() != version {
		t.Error("restoring a radio already at its authored power changed state")
	}
}

func TestRadioTxPowerRefusesWhatNoRadioRuns(t *testing.T) {
	for _, testCase := range []struct {
		name  string
		iface string
		dBm   int
		want  error
	}{
		{"zero", "Dot11Radio0", 0, devicestate.ErrRadioTxPowerInvalid},
		{"above the ceiling", "Dot11Radio0", 31, devicestate.ErrRadioTxPowerInvalid},
		{"no such radio", "Dot11Radio9", 8, devicestate.ErrInterfaceNotFound},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			store := newStationTestStore(t)
			version := store.Version()

			if err := store.SetRadioTxPower(testCase.iface, testCase.dBm); !errors.Is(err, testCase.want) {
				t.Fatalf("SetRadioTxPower() error = %v, want %v", err, testCase.want)
			}
			if store.Version() != version {
				t.Error("a refused power changed state")
			}
		})
	}
}

// TestRadioTxPowerSurvivesCheckpointAndRestart: a checkpoint taken at full
// power returns the radio to it, and a durable record carries the drop.
func TestRadioTxPowerSurvivesCheckpointAndRestart(t *testing.T) {
	source := newStationTestStore(t)
	source.SaveCheckpoint("full-power")
	if err := source.SetRadioTxPower("Dot11Radio1", 5); err != nil {
		t.Fatal(err)
	}
	if err := source.SetRadioTxPower("Dot11Radio0", 8); err != nil {
		t.Fatal(err)
	}

	target := newStationTestStore(t)
	if err := target.RestoreState(source.ExportState()); err != nil {
		t.Fatalf("RestoreState() error = %v", err)
	}
	want := []devicestate.RadioTxPower{{Interface: "Dot11Radio0", DBM: 8}, {Interface: "Dot11Radio1", DBM: 5}}
	if got := target.ExportState().RadioTxPowers; !slices.Equal(got, want) {
		t.Errorf("restored powers = %+v, want %+v", got, want)
	}

	if err := target.RestoreCheckpoint("full-power"); err != nil {
		t.Fatal(err)
	}
	if got := target.ExportState().RadioTxPowers; len(got) != 0 {
		t.Errorf("powers after restoring the full-power checkpoint = %+v", got)
	}
}

func TestRestoreStateRefusesRadioTxPowersNoStoreSets(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		mutate func(*devicestate.State)
	}{
		{"out of range", func(state *devicestate.State) { state.RadioTxPowers[0].DBM = 31 }},
		{"unknown radio", func(state *devicestate.State) { state.RadioTxPowers[0].Interface = "Dot11Radio9" }},
		{"twice", func(state *devicestate.State) {
			state.RadioTxPowers = append(state.RadioTxPowers, state.RadioTxPowers[0])
		}},
		{"in a checkpoint", func(state *devicestate.State) { state.Checkpoints[0].RadioTxPowers[0].DBM = 0 }},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			source := newStationTestStore(t)
			if err := source.SetRadioTxPower("Dot11Radio0", 8); err != nil {
				t.Fatal(err)
			}
			source.SaveCheckpoint("dropped")
			state := source.ExportState()
			testCase.mutate(&state)

			if err := newStationTestStore(t).RestoreState(state); !errors.Is(err, devicestate.ErrStateInvalid) {
				t.Errorf("RestoreState() error = %v, want ErrStateInvalid", err)
			}
		})
	}
}
