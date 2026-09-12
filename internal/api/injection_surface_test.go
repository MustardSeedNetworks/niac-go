package api

import (
	"slices"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

// The injection screen is server-driven: it renders whatever GET /api/v1/errors
// advertises. So "can an operator inject this problem by hand" is decided
// entirely here, and a problem the runtime supports but this surface omits is
// invisible to the operator no matter what the UI does.
func TestInjectionSurfaceOffersEveryFaultType(t *testing.T) {
	offered := make([]string, 0)
	for _, entry := range availableErrorTypes() {
		offered = append(offered, entry["type"])
	}
	for _, entry := range availableDeviceErrorTypes() {
		offered = append(offered, entry.Type)
	}

	for _, faultType := range devicestate.AuthorableFaultTypes() {
		label := faultLabel(faultType)
		if !slices.Contains(offered, label) {
			t.Errorf("%q (%s) is a runtime fault the injection screen does not offer", faultType, label)
		}
	}
}

// Every advertised entry needs a description, because the panel renders one and
// the catalog is restated here as a map keyed by type: a fault added to the
// runtime still appears in the list through the definitions loop, but with an
// empty description beside it.
func TestInjectionSurfaceDescribesEveryEntry(t *testing.T) {
	for _, entry := range availableErrorTypes() {
		if strings.TrimSpace(entry["description"]) == "" {
			t.Errorf("interface fault %q is offered with no description", entry["type"])
		}
	}
	for _, entry := range availableDeviceErrorTypes() {
		if strings.TrimSpace(entry.Description) == "" {
			t.Errorf("device fault %q is offered with no description", entry.Type)
		}
		if entry.ValueKind == "" {
			t.Errorf("device fault %q is offered with no value kind", entry.Type)
		}
	}
}

// Reboot and an STP topology change are two of the problems the NetAlly
// cross-reference names, and until this landed neither could be injected:
// Stack.ExecuteDeviceAction had exactly one production caller, the behavior
// timeline runner, so an operator could schedule them inside an authored
// timeline and never trigger one by hand — while every other problem in the
// catalog was one click away on the same screen.
func TestInjectionSurfaceOffersEveryDeviceAction(t *testing.T) {
	offered := make([]string, 0)
	for _, entry := range availableDeviceActions() {
		offered = append(offered, entry.Type)
		if strings.TrimSpace(entry.Description) == "" {
			t.Errorf("device action %q is offered with no description", entry.Type)
		}
	}

	for _, action := range devicestate.DeviceActionTypes() {
		if !slices.Contains(offered, string(action)) {
			t.Errorf("%q is a runtime device action the injection screen does not offer", action)
		}
	}
}

// faultLabel resolves the name the injection surface advertises a fault under.
// Most come from a catalog definition; the two payload-bearing faults that live
// outside the catalogs carry the package's own label constants, which this
// reads rather than restating.
func faultLabel(faultType string) string {
	switch faultType {
	case string(devicestate.FaultDuplicateIP):
		return duplicateIPLabel
	case string(devicestate.FaultBadMask):
		return badMaskLabel
	}
	for _, definition := range devicestate.InterfaceFaultDefinitions() {
		if string(definition.Type) == faultType {
			return definition.Label
		}
	}
	for _, definition := range devicestate.DeviceFaultDefinitions() {
		if string(definition.Type) == faultType {
			return definition.Label
		}
	}

	return devicestate.DeviceFaultType(faultType).Label()
}
