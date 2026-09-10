package protocols

import (
	"slices"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestDuplicateOfferTargetsUseStaticAttachment(t *testing.T) {
	stack, _ := isolationRoutedDHCP(t)
	assertTargets := func() {
		t.Helper()
		seen := make(map[string]bool)
		for _, target := range stack.DeviceFaultTargets() {
			seen[target.Device] = slices.Contains(target.Faults, devicestate.FaultDuplicateDHCPOffer)
		}
		if !seen["a"] || seen["remote"] {
			t.Fatalf("duplicate-offer eligibility = %v, want local a only", seen)
		}
	}
	assertTargets()
	for _, device := range stack.fabric.attachmentDHCP {
		store := stack.deviceStates[device]
		for _, iface := range store.Snapshot().Network.Interfaces {
			if err := store.SetInterfaceFault(iface.Name, devicestate.FaultLinkDown, 1); err != nil {
				t.Fatal(err)
			}
		}
	}
	assertTargets()
}
