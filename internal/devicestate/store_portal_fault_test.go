package devicestate_test

import (
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestCaptivePortalFaultIsBinary(t *testing.T) {
	store := faultStore()
	kind := devicestate.DeviceFaultType("captive_portal")
	if err := store.SetDeviceFault(kind, 1); err != nil {
		t.Fatal(err)
	}
	for _, value := range []int{-1, 2, 100} {
		if err := store.SetDeviceFault(kind, value); !errors.Is(err, devicestate.ErrFaultValueInvalid) {
			t.Fatalf("portal value %d: %v", value, err)
		}
		if store.DeviceFaultValue(kind) != 1 {
			t.Fatal("rejected value changed armed portal")
		}
	}
	if err := store.SetDeviceFault(kind, 0); err != nil || store.DeviceFaultValue(kind) != 0 {
		t.Fatalf("clear portal: %v", err)
	}
}
