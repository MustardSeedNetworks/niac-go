package devicestate

import (
	"reflect"
	"testing"
	"time"
)

func TestDeviceActionsCommitOnceAndRecover(t *testing.T) {
	store := NewStore(Identity{Hostname: "switch-1"})
	now := time.Date(2026, time.September, 10, 12, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }
	for _, kind := range []DeviceActionType{ActionReboot, ActionSTPTopologyChange} {
		before := store.Version()
		id := "phase:" + string(kind)
		applied, err := store.ExecuteDeviceAction(kind, id)
		if err != nil || !applied || store.Version() != before+1 {
			t.Fatalf("apply %s: applied=%v err=%v version=%d", kind, applied, err, store.Version())
		}
		saved := store.ExportState()
		now = now.Add(time.Second)
		if applied, err = store.ExecuteDeviceAction(kind, id); err != nil || applied {
			t.Fatalf("repeated %s: applied=%v err=%v", kind, applied, err)
		}
		if !reflect.DeepEqual(store.ExportState(), saved) {
			t.Fatal("repeat changed telemetry, history or version")
		}
		recovered := NewStore(Identity{Hostname: "switch-1"})
		if err = recovered.RestoreState(saved); err != nil {
			t.Fatal(err)
		}
		if applied, err = recovered.ExecuteDeviceAction(kind, id); err != nil || applied {
			t.Fatalf("recovery replayed %s: applied=%v err=%v", kind, applied, err)
		}
		if !reflect.DeepEqual(recovered.ExportState(), saved) {
			t.Fatal("recovery changed committed state")
		}
	}
	telemetry := store.DeviceTelemetry()
	if telemetry.RebootedAt.IsZero() || telemetry.STPChanges != 1 ||
		telemetry.STPChangedAt.IsZero() {
		t.Fatalf("missing action telemetry: %+v", telemetry)
	}
}

func TestCheckpointRestoresTelemetryWithoutRearmingActions(t *testing.T) {
	store := NewStore(Identity{Hostname: "switch-1"})
	store.SaveCheckpoint("before")
	for _, kind := range []DeviceActionType{ActionReboot, ActionSTPTopologyChange} {
		if _, err := store.ExecuteDeviceAction(kind, string(kind)); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.RestoreCheckpoint("before"); err != nil {
		t.Fatal(err)
	}
	if got := store.DeviceTelemetry(); got != (DeviceTelemetry{}) {
		t.Fatalf("checkpoint did not restore telemetry: %+v", got)
	}
	before := store.ExportState()
	for _, kind := range []DeviceActionType{ActionReboot, ActionSTPTopologyChange} {
		if applied, err := store.ExecuteDeviceAction(kind, string(kind)); err != nil || applied {
			t.Fatalf("checkpoint rearmed %s: applied=%v err=%v", kind, applied, err)
		}
	}
	if !reflect.DeepEqual(store.ExportState(), before) {
		t.Fatal("replaying a consumed action changed restored state")
	}
}

func TestInvalidDeviceActionsDoNotChangeState(t *testing.T) {
	store := NewStore(Identity{Hostname: "switch-1"})
	for _, tc := range []struct {
		kind DeviceActionType
		id   string
	}{{"unknown", "phase:0"}, {ActionReboot, ""}} {
		before := store.ExportState()
		if _, err := store.ExecuteDeviceAction(tc.kind, tc.id); err == nil {
			t.Fatalf("accepted invalid action: %+v", tc)
		}
		if !reflect.DeepEqual(store.ExportState(), before) {
			t.Fatal("invalid action mutated state")
		}
	}
}
