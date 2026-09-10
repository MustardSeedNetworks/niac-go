package devicestate_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestResourceFaultBounds(t *testing.T) {
	for _, kind := range []devicestate.DeviceFaultType{"cpu_percent", "memory_percent", "disk_percent"} {
		t.Run(string(kind), func(t *testing.T) {
			store := faultStore()
			for _, value := range []int{1, 100, 0} {
				if err := store.SetDeviceFault(kind, value); err != nil {
					t.Fatalf("set %d: %v", value, err)
				}
				if store.DeviceFaultValue(kind) != value || store.DeviceFaultActive(kind) != (value > 0) {
					t.Fatalf("resource state differs from %d", value)
				}
			}
		})
	}
}

func TestResourceFaultInvalidBoundsPreserveState(t *testing.T) {
	for _, kind := range []devicestate.DeviceFaultType{"cpu_percent", "memory_percent", "disk_percent"} {
		t.Run(string(kind), func(t *testing.T) {
			store := faultStore()
			if err := store.SetDeviceFault(kind, 75); err != nil {
				t.Fatal(err)
			}
			before := store.Snapshot()
			for _, value := range []int{-1, 101} {
				if err := store.SetDeviceFault(kind, value); !errors.Is(err, devicestate.ErrFaultValueInvalid) {
					t.Fatalf("set %d: %v", value, err)
				}
			}
			if !reflect.DeepEqual(before, store.Snapshot()) {
				t.Fatal("invalid resource value mutated state")
			}
		})
	}
}

func TestResourceFaultCheckpointIndependence(t *testing.T) {
	store := faultStore()
	want := []devicestate.DeviceFault{
		{Type: "cpu_percent", Value: 91},
		{Type: "disk_percent", Value: 93},
		{Type: "memory_percent", Value: 92},
	}
	for _, fault := range want {
		if err := store.SetDeviceFault(fault.Type, fault.Value); err != nil {
			t.Fatal(err)
		}
	}
	store.SaveCheckpoint("resources")
	if err := store.SetDeviceFault("cpu_percent", 0); err != nil {
		t.Fatal(err)
	}
	if got := store.Snapshot().DeviceFaults; !reflect.DeepEqual(got, want[1:]) {
		t.Fatalf("clear CPU changed other resources: %v", got)
	}
	store.ClearDeviceFaults()
	if err := store.RestoreCheckpoint("resources"); err != nil {
		t.Fatal(err)
	}
	if got := store.Snapshot().DeviceFaults; !reflect.DeepEqual(got, want) {
		t.Fatalf("restored resources = %v, want %v", got, want)
	}
	store.ClearDeviceFaults()
	store.SaveCheckpoint("clear")
	if err := store.SetDeviceFault("cpu_percent", 100); err != nil {
		t.Fatal(err)
	}
	if err := store.RestoreCheckpoint("clear"); err != nil {
		t.Fatal(err)
	}
	if got := store.Snapshot().DeviceFaults; len(got) != 0 {
		t.Fatalf("clear checkpoint retained resources: %v", got)
	}
}
