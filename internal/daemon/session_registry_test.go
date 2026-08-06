package daemon

import (
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

func TestSessionRegistryAllowsDistinctTrunkVLANs(t *testing.T) {
	registry := newSessionRegistry()
	registry.replace("hospital", &Simulation{Binding: trunkBinding(200)})

	if err := registry.validateReplacement("warehouse", trunkBinding(201)); err != nil {
		t.Fatalf("validateReplacement() error = %v", err)
	}
}

func TestSessionRegistryRejectsDuplicateTrunkVLAN(t *testing.T) {
	registry := newSessionRegistry()
	registry.replace("hospital", &Simulation{Binding: trunkBinding(200)})

	err := registry.validateReplacement("warehouse", trunkBinding(200))
	if !errors.Is(err, ErrPhysicalVLANInUse) {
		t.Fatalf("validateReplacement() error = %v, want ErrPhysicalVLANInUse", err)
	}
}

func TestSessionRegistryAllowsAtomicReplacementWithinSession(t *testing.T) {
	registry := newSessionRegistry()
	previous := &Simulation{Binding: trunkBinding(200)}
	registry.replace("hospital", previous)

	if err := registry.validateReplacement("hospital", trunkBinding(201)); err != nil {
		t.Fatalf("validateReplacement() error = %v", err)
	}
	replacement := &Simulation{Binding: trunkBinding(201)}
	if got := registry.replace("hospital", replacement); got != previous {
		t.Fatalf("replace() previous = %p, want %p", got, previous)
	}
}

func TestSessionRegistryRejectsMixedOwnershipOnOneInterface(t *testing.T) {
	registry := newSessionRegistry()
	registry.replace("direct", &Simulation{Binding: fabric.Binding{
		Interface: "eth0", Mode: fabric.ModeDirect,
	}})

	err := registry.validateReplacement("hospital", trunkBinding(200))
	if !errors.Is(err, ErrInterfaceInUse) {
		t.Fatalf("validateReplacement() error = %v, want ErrInterfaceInUse", err)
	}
}

func TestSessionRegistryRemove(t *testing.T) {
	registry := newSessionRegistry()
	active := &Simulation{Binding: trunkBinding(200)}
	registry.replace("hospital", active)

	if got := registry.remove("hospital"); got != active {
		t.Fatalf("remove() = %p, want %p", got, active)
	}
	if got := registry.remove("hospital"); got != nil {
		t.Fatalf("second remove() = %p, want nil", got)
	}
}

func trunkBinding(vlan uint16) fabric.Binding {
	return fabric.Binding{Interface: "eth0", Mode: fabric.ModeTrunk, AccessVLAN: vlan}
}
