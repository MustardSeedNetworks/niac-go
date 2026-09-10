package protocols

import (
	"fmt"
	"net/netip"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
)

func TestInterfaceConflictTargetsExcludeSelfAndOtherSegments(t *testing.T) {
	first := &config.Device{Name: "first", VLAN: 200}
	peer := &config.Device{Name: "peer", VLAN: 200}
	other := &config.Device{Name: "other", VLAN: 300}
	states := make(map[*config.Device]devicestate.Snapshot)
	for index, device := range []*config.Device{first, peer, other} {
		states[device] = devicestate.Snapshot{Network: devicestate.Network{Interfaces: []devicestate.Interface{
			{
				Name:    "eth0",
				AdminUp: true,
				OperUp:  true,
				Address: netip.MustParsePrefix(fmt.Sprintf("192.0.2.%d/24", index+1)),
			},
		}}}
	}
	stack := &Stack{}
	got := stack.interfaceConflictTargets(states)
	if !got[first]["eth0"] || !got[peer]["eth0"] || len(got[other]) != 0 {
		t.Fatalf("incorrect segment eligibility: %v", got)
	}
	state := states[peer]
	state.Network.Interfaces[0].OperUp = false
	states[peer] = state
	if got = stack.interfaceConflictTargets(states); len(got[first]) != 0 || len(got[peer]) != 0 {
		t.Fatalf("down peer remained eligible: %v", got)
	}
	state.Network.Interfaces[0].OperUp = true
	states[peer] = state
	if got = stack.interfaceConflictTargets(states); !got[first]["eth0"] || !got[peer]["eth0"] {
		t.Fatalf("recovered peer eligibility was stale: %v", got)
	}
}

func BenchmarkInterfaceConflictTargets500Devices48Ports(b *testing.B) {
	states := make(map[*config.Device]devicestate.Snapshot)
	for index := range 500 {
		device := &config.Device{Name: fmt.Sprintf("switch-%d", index), VLAN: 200}
		interfaces := make([]devicestate.Interface, 48)
		for port := range interfaces {
			interfaces[port] = devicestate.Interface{Name: fmt.Sprintf("eth%d", port), AdminUp: true, OperUp: true}
		}
		interfaces[0].Address = netip.MustParsePrefix(fmt.Sprintf("10.0.%d.%d/16", index/250, index%250+1))
		states[device] = devicestate.Snapshot{Network: devicestate.Network{Interfaces: interfaces}}
	}
	stack := &Stack{}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if got := stack.interfaceConflictTargets(states); len(got) != 500 {
			b.Fatal("eligible devices missing")
		}
	}
}
