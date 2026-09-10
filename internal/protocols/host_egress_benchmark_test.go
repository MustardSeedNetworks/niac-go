package protocols

import (
	"fmt"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestHealthyHostEgressDoesNotAllocatePerInterface(t *testing.T) {
	for _, count := range []int{1, 48, 512} {
		stack, packet := hostEgressFixture(t, "192.0.2.20")
		store := stack.deviceStates[packet.generatedHost]
		network := store.Snapshot().Network
		for index := 1; index < count; index++ {
			iface := network.Interfaces[0]
			iface.Name = fmt.Sprintf("eth%d", index)
			network.Interfaces = append(network.Interfaces, iface)
		}
		store.ReplaceNetwork(network)
		allocations := testing.AllocsPerRun(100, func() {
			prepared, err := stack.prepareHostEgress(packet)
			if err != nil || prepared != packet {
				t.Fatal("healthy egress changed packet")
			}
		})
		if allocations != 0 {
			t.Fatalf("%d interfaces: %v allocations, want0", count, allocations)
		}
	}
}

func TestPrefixFaultPresenceFollowsClear(t *testing.T) {
	stack, packet := hostEgressFixture(t, "192.0.2.20")
	store := stack.deviceStates[packet.generatedHost]
	if store.HasInterfacePrefixFaults() {
		t.Fatal("healthy store reports mask fault")
	}
	armHostMask(t, stack, packet)
	if !store.HasInterfacePrefixFaults() {
		t.Fatal("armed fault not reported")
	}
	if err := store.ClearInterfacePrefixFault("eth0", devicestate.FaultBadMask); err != nil {
		t.Fatal(err)
	}
	if store.HasInterfacePrefixFaults() {
		t.Fatal("cleared store reports mask fault")
	}
}
