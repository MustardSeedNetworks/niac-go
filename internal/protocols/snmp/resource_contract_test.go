package snmp

import (
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestResourceContractOnlySubstitutesArmedRows(t *testing.T) {
	agent, state := resourceTestAgent(t)
	for _, tc := range []struct {
		fault devicestate.DeviceFaultType
		oid   string
	}{
		{devicestate.FaultCPUPercent, hrProcessorLoadPrefix + "7"},
		{devicestate.FaultMemoryPercent, hrStorageUsedPrefix + "10"},
		{devicestate.FaultDiskPercent, hrStorageUsedPrefix + "20"},
	} {
		t.Run(string(tc.fault), func(t *testing.T) {
			if got := agent.WalkContract().Classify(tc.oid); got != BucketKept {
				t.Fatalf("healthy row classified %s", got)
			}
			if err := state.SetDeviceFault(tc.fault, 90); err != nil {
				t.Fatal(err)
			}
			contract := agent.WalkContract()
			if got := contract.Classify(tc.oid); got != BucketLive || contract.DropsFromWalk(tc.oid) {
				t.Fatalf("armed row classified %s, drop=%v", got, contract.DropsFromWalk(tc.oid))
			}
			assertUnchangedResourceRows(t, contract)
			state.ClearDeviceFaults()
			if got := agent.WalkContract().Classify(tc.oid); got != BucketKept {
				t.Fatalf("cleared row classified %s", got)
			}
		})
	}
}

func assertUnchangedResourceRows(t *testing.T, contract WalkContract) {
	t.Helper()
	for _, oid := range []string{hrStorageSizePrefix + "10", hrStorageUsedPrefix + "30", hrProcessorLoadPrefix + "99"} {
		if got := contract.Classify(oid); got != BucketKept {
			t.Fatalf("unaffected row %s classified %s", oid, got)
		}
	}
}
