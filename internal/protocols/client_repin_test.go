package protocols

import (
	"errors"
	"net"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

// A re-pin moves one tester on the running session (AP-6). A restart, or a
// ReloadConfig, would rebuild every DHCP handler, SNMP agent and placement and
// so unplug every other client along with the one being moved.

func repinTo(mac net.HardwareAddr, port string) config.AttachmentPin {
	return config.AttachmentPin{MAC: mac.String(), Device: placementAccess, Interface: port}
}

func TestRepinMovesOnlyThatClient(t *testing.T) {
	stack := placementStack(t)
	first, second := placementClient(1), placementClient(2)
	sendFrom(stack, first, "10.51.210.101")
	sendFrom(stack, second, "10.51.210.102")

	topology := compilePlacement(t, placementConfig(repinTo(first, placementPinnedPort)))
	if err := stack.RepinAttachedClient(topology, first); err != nil {
		t.Fatalf("RepinAttachedClient() = %v", err)
	}

	access := stack.agentFor(t, placementAccess)
	for mac, want := range map[string]string{
		first.String():  placementPinnedPort,
		second.String(): "GigabitEthernet1/0/44",
	} {
		hw, _ := net.ParseMAC(mac)
		if got, _ := fdbPortName(t, access, hw); got != want {
			t.Errorf("%s FDB port = %q, want %q", mac, got, want)
		}
	}
	if got := stack.fabric.placement.advertisedPort().Interface; got != placementPinnedPort {
		t.Errorf("LLDP names %q, want the moved client's new port %q", got, placementPinnedPort)
	}
	running, _ := stack.RuntimeFabricTopology()
	if pins := running.Attachments[0].Pins; len(pins) != 1 || pins[0].Interface != placementPinnedPort {
		t.Errorf("running pins = %#v, want the new pin", pins)
	}

	// The port it left is free again, for the next tester to arrive.
	third := placementClient(3)
	sendFrom(stack, third, "10.51.210.103")
	if got, _ := fdbPortName(t, access, third); got != "GigabitEthernet1/0/43" {
		t.Errorf("next client landed on %q, want the vacated GigabitEthernet1/0/43", got)
	}
}

func TestRepinRefusesAPortAnotherClientHolds(t *testing.T) {
	stack := placementStack(t)
	first, second := placementClient(1), placementClient(2)
	sendFrom(stack, first, "10.51.210.101")
	sendFrom(stack, second, "10.51.210.102")

	topology := compilePlacement(t, placementConfig(repinTo(first, "GigabitEthernet1/0/44")))
	err := stack.RepinAttachedClient(topology, first)

	if !errors.Is(err, ErrAttachmentPortOccupied) {
		t.Fatalf("RepinAttachedClient() = %v, want %v", err, ErrAttachmentPortOccupied)
	}
	access := stack.agentFor(t, placementAccess)
	if got, _ := fdbPortName(t, access, first); got != "GigabitEthernet1/0/43" {
		t.Errorf("refused client moved to %q", got)
	}
	if got, _ := fdbPortName(t, access, second); got != "GigabitEthernet1/0/44" {
		t.Errorf("the port's holder moved to %q", got)
	}
	running, _ := stack.RuntimeFabricTopology()
	if pins := running.Attachments[0].Pins; len(pins) != 0 {
		t.Errorf("a refused re-pin left pins %#v on the running session", pins)
	}
}

func TestRepinOfAClientNotYetSeenReservesItsPort(t *testing.T) {
	stack := placementStack(t)
	absent, other := placementClient(9), placementClient(1)

	topology := compilePlacement(t, placementConfig(repinTo(absent, "GigabitEthernet1/0/43")))
	if err := stack.RepinAttachedClient(topology, absent); err != nil {
		t.Fatalf("RepinAttachedClient() = %v", err)
	}
	sendFrom(stack, other, "10.51.210.101")
	sendFrom(stack, absent, "10.51.210.109")

	access := stack.agentFor(t, placementAccess)
	if got, _ := fdbPortName(t, access, absent); got != "GigabitEthernet1/0/43" {
		t.Errorf("pinned client landed on %q, want its pin", got)
	}
	if got, _ := fdbPortName(t, access, other); got != "GigabitEthernet1/0/44" {
		t.Errorf("unpinned client took %q; the pin reserves GigabitEthernet1/0/43", got)
	}
}

func TestRepinWithoutAPoolIsRefused(t *testing.T) {
	stack := observedClientStack(t)
	err := stack.RepinAttachedClient(&fabric.Topology{}, placementClient(1))
	if !errors.Is(err, errAttachmentPoolNotBound) {
		t.Fatalf("RepinAttachedClient() = %v, want %v", err, errAttachmentPoolNotBound)
	}
}
