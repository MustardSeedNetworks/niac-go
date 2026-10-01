package api

import (
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/protocols"
)

// A pack that offers nothing to inject is a demo bed an operator cannot break
// on purpose, and the failure is silent: the screen lists every device and no
// fault type against any of them, which reads as "nothing is wrong" rather than
// "this build cannot do it". The lab container was in exactly that state on
// 2026-09-12 -- 0 of 159 devices offering a fault -- because it was running a
// build from six days and forty-six releases earlier. That was stale software
// rather than a defect in the packs, and this is the test that would have said
// so without a trip to the lab.
//
// The counts are deliberately not pinned: what matters is that each axis is
// non-empty, not that it holds a particular number that would have to be
// re-signed with every pack change.
func checkEveryPackOffersSomethingToInject(t *testing.T, packs []generatedPack) {
	t.Helper()
	for _, pack := range packs {
		t.Run(pack.id, func(t *testing.T) {
			if interfaceFaultOffers(pack.stack) == 0 {
				t.Errorf("no device offers an interface fault, of %d in the pack", len(pack.cfg.Devices))
			}
			if len(pack.stack.DeviceFaultTargets()) == 0 {
				t.Error("no device offers a service fault")
			}
			if len(pack.stack.DeviceActionTargets()) == 0 {
				t.Error("no device offers an action")
			}
		})
	}
}

func interfaceFaultOffers(stack *protocols.Stack) int {
	offering := 0
	for _, target := range stack.InterfaceFaultTargets() {
		if len(target.ErrorTypes) > 0 {
			offering++
		}
	}

	return offering
}
