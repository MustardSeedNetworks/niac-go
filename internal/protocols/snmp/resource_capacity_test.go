package snmp

import (
	"testing"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestResourceFaultUsesCurrentCapacity(t *testing.T) {
	agent, state := resourceTestAgent(t)
	capacity := 0
	agent.mib.SetDynamic(hrStorageSizePrefix+"10", func() *OIDValue {
		return &OIDValue{Type: gosnmp.Integer, Value: capacity}
	})
	agent.Reindex()
	if agent.ResourceFaultObservable(devicestate.FaultMemoryPercent) {
		t.Fatal("zero capacity advertised as observable")
	}
	if err := state.SetDeviceFault(devicestate.FaultMemoryPercent, 50); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		capacity int
		want     string
	}{
		{0, "201"}, {1000, "500"}, {2000, "1000"}, {0, "201"},
	} {
		capacity = tc.capacity
		assertResourceValue(t, agent, hrStorageUsedPrefix+"10", tc.want)
		if got := agent.ResourceFaultObservable(devicestate.FaultMemoryPercent); got != (capacity > 0) {
			t.Fatalf("capacity %d: observable=%v", capacity, got)
		}
		wantBucket := BucketKept
		if capacity > 0 {
			wantBucket = BucketLive
		}
		if got := agent.WalkContract().Classify(hrStorageUsedPrefix + "10"); got != wantBucket {
			t.Fatalf("capacity %d: bucket=%s, want %s", capacity, got, wantBucket)
		}
	}
}
