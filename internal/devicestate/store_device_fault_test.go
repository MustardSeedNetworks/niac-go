package devicestate_test

import (
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestStoreDeviceFaultsAreIndependentOfInterfaceFaults(t *testing.T) {
	store := faultStore()

	if err := store.SetDeviceFault(devicestate.FaultDHCPNoOffer, 1); err != nil {
		t.Fatalf("SetDeviceFault(dhcp_no_offer) error = %v", err)
	}
	if err := store.SetInterfaceFault("Gi0/1", devicestate.FaultFCS, 25); err != nil {
		t.Fatalf("SetInterfaceFault(fcs) error = %v", err)
	}

	snapshot := store.Snapshot()
	if len(snapshot.DeviceFaults) != 1 {
		t.Fatalf("device fault count = %d, want 1", len(snapshot.DeviceFaults))
	}
	if snapshot.DeviceFaults[0].Type != devicestate.FaultDHCPNoOffer {
		t.Fatalf("device fault = %#v", snapshot.DeviceFaults[0])
	}
	if len(snapshot.Faults) != 1 || snapshot.Faults[0].Type != devicestate.FaultFCS {
		t.Fatalf("interface faults = %#v", snapshot.Faults)
	}
	if !store.DeviceFaultActive(devicestate.FaultDHCPNoOffer) {
		t.Fatal("DeviceFaultActive(dhcp_no_offer) = false, want true")
	}
	if store.DeviceFaultActive(devicestate.FaultDNSTimeout) {
		t.Fatal("DeviceFaultActive(dns_timeout) = true, want false")
	}
}

// The two axes are separate types, so passing one axis's constant to the
// other's setter does not compile — that half of the confusion is
// unrepresentable rather than tested. What remains testable is a value that
// only looks like a fault type, which is what an operator-supplied label can
// produce.
func TestStoreDeviceFaultRefusesAnUnknownType(t *testing.T) {
	store := faultStore()

	err := store.SetDeviceFault(devicestate.DeviceFaultType("fcs_errors"), 25)
	if !errors.Is(err, devicestate.ErrDeviceFaultTypeInvalid) {
		t.Fatalf("SetDeviceFault(fcs_errors) error = %v, want ErrDeviceFaultTypeInvalid", err)
	}
	if snapshot := store.Snapshot(); len(snapshot.DeviceFaults) != 0 {
		t.Fatalf("snapshot recorded a refused fault: %#v", snapshot)
	}
}

func TestStoreDeviceFaultZeroClearsOnlyNamedFault(t *testing.T) {
	store := faultStore()
	for _, faultType := range []devicestate.DeviceFaultType{
		devicestate.FaultDHCPNoOffer, devicestate.FaultDNSNXDomain,
	} {
		if err := store.SetDeviceFault(faultType, 1); err != nil {
			t.Fatalf("SetDeviceFault(%s) error = %v", faultType, err)
		}
	}

	if err := store.SetDeviceFault(devicestate.FaultDHCPNoOffer, 0); err != nil {
		t.Fatalf("SetDeviceFault(clear) error = %v", err)
	}

	remaining := store.Snapshot().DeviceFaults
	if len(remaining) != 1 || remaining[0].Type != devicestate.FaultDNSNXDomain {
		t.Fatalf("remaining device faults = %#v", remaining)
	}
}

// Each fault type carries its own ceiling: a rate stops at 100, and latency
// is milliseconds, so a shared 0-100 clamp would cap it at a tenth of a second
// and silently rule out every delay a tester would notice.
func TestStoreDeviceFaultValueIsBoundedPerType(t *testing.T) {
	for _, definition := range devicestate.DeviceFaultDefinitions() {
		t.Run(string(definition.Type), func(t *testing.T) {
			store := faultStore()

			if err := store.SetDeviceFault(
				definition.Type, definition.MaxValue,
			); err != nil {
				t.Fatalf("SetDeviceFault(%d) error = %v", definition.MaxValue, err)
			}
			if got := store.DeviceFaultValue(definition.Type); got != definition.MaxValue {
				t.Fatalf("DeviceFaultValue() = %d, want %d", got, definition.MaxValue)
			}

			if err := store.SetDeviceFault(
				definition.Type, definition.MaxValue+1,
			); !errors.Is(err, devicestate.ErrFaultValueInvalid) {
				t.Fatalf("SetDeviceFault(%d) error = %v, want ErrFaultValueInvalid",
					definition.MaxValue+1, err)
			}
		})
	}
}

// Latency is the reason the ceiling became per-type; state the number the
// rest of the stack relies on rather than leaving it to the loop above.
func TestStoreLatencyFaultAcceptsMilliseconds(t *testing.T) {
	store := faultStore()

	if err := store.SetDeviceFault(devicestate.FaultLatency, 2500); err != nil {
		t.Fatalf("SetDeviceFault(2500) error = %v", err)
	}
	if got := store.DeviceFaultValue(devicestate.FaultLatency); got != 2500 {
		t.Fatalf("DeviceFaultValue() = %d, want 2500", got)
	}
	if got := store.DeviceFaultValue(devicestate.FaultDNSTimeout); got != 0 {
		t.Fatalf("unarmed DeviceFaultValue() = %d, want 0", got)
	}
}

func TestStoreClearDeviceFaultsClearsOnlyDeviceAxis(t *testing.T) {
	store := faultStore()
	if err := store.SetDeviceFault(devicestate.FaultDNSTimeout, 1); err != nil {
		t.Fatalf("SetDeviceFault() error = %v", err)
	}
	if err := store.SetInterfaceFault("Gi0/1", devicestate.FaultFCS, 25); err != nil {
		t.Fatalf("SetInterfaceFault() error = %v", err)
	}

	store.ClearDeviceFaults()

	snapshot := store.Snapshot()
	if len(snapshot.DeviceFaults) != 0 {
		t.Fatalf("device faults = %#v, want none", snapshot.DeviceFaults)
	}
	if len(snapshot.Faults) != 1 {
		t.Fatalf("interface faults = %#v, want the FCS fault kept", snapshot.Faults)
	}
}

// A service outage must not reach a syslog or trap consumer as an interface
// event: P2-3 keys on the event kind.
func TestStoreDeviceFaultRecordsItsOwnEventKind(t *testing.T) {
	store := faultStore()
	if err := store.SetDeviceFault(devicestate.FaultDHCPNoOffer, 1); err != nil {
		t.Fatalf("SetDeviceFault() error = %v", err)
	}
	if err := store.SetDeviceFault(devicestate.FaultDHCPNoOffer, 0); err != nil {
		t.Fatalf("SetDeviceFault(clear) error = %v", err)
	}

	events := store.Events()
	kinds := make([]devicestate.EventKind, 0, 2)
	for _, event := range events {
		if event.Kind == devicestate.EventDeviceFaultUpdated ||
			event.Kind == devicestate.EventDeviceFaultCleared {
			kinds = append(kinds, event.Kind)
			if event.Target != string(devicestate.FaultDHCPNoOffer) {
				t.Fatalf("event target = %q, want the fault type", event.Target)
			}
		}
		if event.Kind == devicestate.EventFaultUpdated ||
			event.Kind == devicestate.EventFaultCleared {
			t.Fatalf("device fault recorded an interface fault event: %#v", event)
		}
	}
	if len(kinds) != 2 {
		t.Fatalf("device fault event kinds = %v, want updated then cleared", kinds)
	}
}

// Labels are what an operator sends, so they must be unambiguous across the
// two catalogs even though the types are not.
func TestStoreFaultCatalogsShareNoLabel(t *testing.T) {
	deviceLabels := make(map[string]struct{})
	for _, definition := range devicestate.DeviceFaultDefinitions() {
		if definition.Label == "" {
			t.Fatalf("device fault %q has no label", definition.Type)
		}
		if _, ok := devicestate.ParseDeviceFaultLabel(definition.Label); !ok {
			t.Fatalf("device label %q does not parse back", definition.Label)
		}
		if _, clash := devicestate.ParseFaultLabel(definition.Label); clash {
			t.Fatalf("device label %q also parses as an interface fault", definition.Label)
		}
		deviceLabels[definition.Label] = struct{}{}
	}
	for _, definition := range devicestate.InterfaceFaultDefinitions() {
		if _, clash := deviceLabels[definition.Label]; clash {
			t.Fatalf("label %q is in both catalogs", definition.Label)
		}
	}
	for _, want := range []devicestate.DeviceFaultType{
		devicestate.FaultDHCPNoOffer, devicestate.FaultDNSNXDomain, devicestate.FaultDNSTimeout,
	} {
		if want.Label() == "" {
			t.Fatalf("device catalog is missing %q", want)
		}
	}
}

// P2-2 restores faults from a checkpoint; a device fault that a checkpoint
// does not carry would come back armed after a restore that should have
// cleared it.
func TestStoreCheckpointCarriesDeviceFaults(t *testing.T) {
	store := faultStore()

	store.SaveCheckpoint("healthy")
	if err := store.SetDeviceFault(devicestate.FaultDNSNXDomain, 1); err != nil {
		t.Fatalf("SetDeviceFault() error = %v", err)
	}
	store.SaveCheckpoint("degraded")

	if err := store.RestoreCheckpoint("healthy"); err != nil {
		t.Fatalf("RestoreCheckpoint(healthy) error = %v", err)
	}
	if got := store.Snapshot().DeviceFaults; len(got) != 0 {
		t.Fatalf("device faults after healthy restore = %#v, want none", got)
	}

	if err := store.RestoreCheckpoint("degraded"); err != nil {
		t.Fatalf("RestoreCheckpoint(degraded) error = %v", err)
	}
	got := store.Snapshot().DeviceFaults
	if len(got) != 1 || got[0].Type != devicestate.FaultDNSNXDomain || got[0].Value != 1 {
		t.Fatalf("device faults after degraded restore = %#v", got)
	}
	if !store.DeviceFaultActive(devicestate.FaultDNSNXDomain) {
		t.Fatal("restored device fault is not active")
	}
}
