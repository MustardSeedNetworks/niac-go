package protocols

import (
	"errors"
	"strconv"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/devicestate"
	"github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp"
)

// A pool port is "notconnect" until a tester is plugged into it. Placing a
// client is plugging it in: the port comes up, so a manager walking the switch
// never sees a MAC learned on a port whose link is down (niac-go#2363).

const (
	ifOperStatusOID = ".1.3.6.1.2.1.2.2.1.8"
	ifLastChangeOID = ".1.3.6.1.2.1.2.2.1.9"
)

// poolPortStatus answers what a manager reads for one pool port: its
// ifOperStatus and ifLastChange, found by ifDescr.
func poolPortStatus(t *testing.T, stack *Stack, port string) (int, uint32) {
	t.Helper()
	agent := stack.agentFor(t, placementAccess).Get(placementCommunity)
	for index := 1; index <= len(placementAccessSwitch().Interfaces); index++ {
		suffix := "." + strconv.Itoa(index)
		descr, err := agent.HandleGet(ifDescrOID + suffix)
		if err != nil || descr == nil || descr.Value != port {
			continue
		}
		oper, err := agent.HandleGet(ifOperStatusOID + suffix)
		if err != nil || oper == nil {
			t.Fatalf("%s has no ifOperStatus: %v", port, err)
		}
		changed, err := agent.HandleGet(ifLastChangeOID + suffix)
		if err != nil || changed == nil {
			t.Fatalf("%s has no ifLastChange: %v", port, err)
		}
		return oper.Value.(int), changed.Value.(uint32)
	}
	t.Fatalf("no ifDescr %s on %s", port, placementAccess)
	return 0, 0
}

func shutPoolPort(t *testing.T, stack *Stack, port string) {
	t.Helper()
	state := stack.deviceStates[stackDevice(t, stack, placementAccess)]
	err := state.UpdateInterface(port, func(iface devicestate.Interface) (devicestate.Interface, error) {
		iface.AdminUp, iface.OperUp = false, false
		return iface, nil
	})
	if err != nil {
		t.Fatalf("shut %s: %v", port, err)
	}
}

func TestPlacingAClientBringsItsPoolPortUp(t *testing.T) {
	stack := placementStack(t)
	if oper, _ := poolPortStatus(t, stack, "GigabitEthernet1/0/43"); oper != snmp.IfStatusDown {
		t.Fatalf("spare pool port ifOperStatus = %d before any client, want down(2)", oper)
	}

	sendFrom(stack, placementClient(1), "10.51.210.101")

	oper, changed := poolPortStatus(t, stack, "GigabitEthernet1/0/43")
	if oper != snmp.IfStatusUp {
		t.Errorf("placed port ifOperStatus = %d, want up(1)", oper)
	}
	if changed == 0 {
		t.Error("placed port ifLastChange did not move")
	}
	if spare, _ := poolPortStatus(t, stack, "GigabitEthernet1/0/44"); spare != snmp.IfStatusDown {
		t.Errorf("unplaced pool port ifOperStatus = %d, want down(2)", spare)
	}
}

func TestStoppingTheSessionUnplugsEveryClient(t *testing.T) {
	stack := placementStack(t)
	if err := stack.Start(); err != nil {
		t.Fatalf("Start() = %v", err)
	}
	sendFrom(stack, placementClient(1), "10.51.210.101")
	sendFrom(stack, placementClient(2), "10.51.210.102")

	stack.Stop()

	for _, port := range []string{"GigabitEthernet1/0/43", "GigabitEthernet1/0/44"} {
		if oper, _ := poolPortStatus(t, stack, port); oper != snmp.IfStatusDown {
			t.Errorf("%s ifOperStatus = %d after the session stopped, want down(2)", port, oper)
		}
	}
}

func TestAShutPoolPortTakesNoClient(t *testing.T) {
	pinned := placementClient(9)
	stack := placementStack(t, repinTo(pinned, placementPinnedPort))
	shutPoolPort(t, stack, "GigabitEthernet1/0/43")
	shutPoolPort(t, stack, placementPinnedPort)
	access := stack.agentFor(t, placementAccess)

	first := placementClient(1)
	sendFrom(stack, first, "10.51.210.101")
	if got, _ := fdbPortName(t, access, first); got != "GigabitEthernet1/0/44" {
		t.Errorf("client landed on %q, want GigabitEthernet1/0/44 past the shut port", got)
	}
	if oper, _ := poolPortStatus(t, stack, "GigabitEthernet1/0/43"); oper != snmp.IfStatusDown {
		t.Errorf("shut port ifOperStatus = %d, want down(2)", oper)
	}

	sendFrom(stack, pinned, "10.51.210.109")
	if port, found := fdbPortName(t, access, pinned); found {
		t.Errorf("client pinned to a shut port was learned on %q", port)
	}
	if oper, _ := poolPortStatus(t, stack, placementPinnedPort); oper != snmp.IfStatusDown {
		t.Errorf("shut pinned port ifOperStatus = %d, want down(2)", oper)
	}
}

func TestAShutPoolPortIsNotAdvertised(t *testing.T) {
	stack := placementStack(t)
	access := stackDevice(t, stack, placementAccess)

	shutPoolPort(t, stack, "GigabitEthernet1/0/43")
	if port, _ := stack.fabric.placement.advertisedPort(
		stack.fabric.portAdminUp,
	); port.Interface != "GigabitEthernet1/0/44" {
		t.Errorf("advertised port = %q, want GigabitEthernet1/0/44 past the shut port", port.Interface)
	}

	// The pinned port is reserved, so with 43 and 44 shut no free port is left.
	pinned := placementStack(t, repinTo(placementClient(9), placementPinnedPort))
	shutPoolPort(t, pinned, "GigabitEthernet1/0/43")
	shutPoolPort(t, pinned, "GigabitEthernet1/0/44")
	if pinned.fabric.advertisesAtClient(stackDevice(t, pinned, placementAccess)) {
		t.Error("the pool switch advertises with every free pool port shut")
	}
	if !stack.fabric.advertisesAtClient(access) {
		t.Error("the pool switch went silent with a usable pool port left")
	}
}

func TestRepinMovesTheLinkWithTheClient(t *testing.T) {
	stack := placementStack(t)
	first := placementClient(1)
	sendFrom(stack, first, "10.51.210.101")

	topology := compilePlacement(t, placementConfig(repinTo(first, placementPinnedPort)))
	if err := stack.RepinAttachedClient(topology, first); err != nil {
		t.Fatalf("RepinAttachedClient() = %v", err)
	}

	if oper, _ := poolPortStatus(t, stack, "GigabitEthernet1/0/43"); oper != snmp.IfStatusDown {
		t.Errorf("vacated port ifOperStatus = %d, want down(2)", oper)
	}
	if oper, _ := poolPortStatus(t, stack, placementPinnedPort); oper != snmp.IfStatusUp {
		t.Errorf("new port ifOperStatus = %d, want up(1)", oper)
	}
}

func TestRepinRefusesAShutPort(t *testing.T) {
	stack := placementStack(t)
	first := placementClient(1)
	sendFrom(stack, first, "10.51.210.101")
	shutPoolPort(t, stack, placementPinnedPort)

	topology := compilePlacement(t, placementConfig(repinTo(first, placementPinnedPort)))
	err := stack.RepinAttachedClient(topology, first)

	if !errors.Is(err, ErrAttachmentPortShut) {
		t.Fatalf("RepinAttachedClient() = %v, want %v", err, ErrAttachmentPortShut)
	}
	if got, _ := fdbPortName(t, stack.agentFor(t, placementAccess), first); got != "GigabitEthernet1/0/43" {
		t.Errorf("refused client moved to %q", got)
	}
	if oper, _ := poolPortStatus(t, stack, "GigabitEthernet1/0/43"); oper != snmp.IfStatusUp {
		t.Errorf("refused client's port ifOperStatus = %d, want up(1)", oper)
	}
}
