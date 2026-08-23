package daemon

import (
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

// D19: a real trunk port carries a native VLAN — untagged frames alongside
// tagged ones. NIAC refused it: validateReplacement required *both* sessions to
// be ModeTrunk, so an access/direct session and any other session on the same
// interface was rejected outright. You could have N tagged scenarios, or one
// untagged scenario, never both.
//
// fabric.Binding models only Mode + AccessVLAN, even though NIAC already models
// a native VLAN for the devices it simulates (config.TrunkPort.NativeVLAN).
// Untagged frames arriving on the trunk were counted and discarded
// (trunkDrops.recordUntagged); the sweep saw drops.untagged = 67 on CT304.
//
// Owner-approved shape: demux key 0 is the native session — consistent with
// config.UntaggedTag = 0 — at most one per interface, coexisting with N tagged.

func nativeBinding() fabric.Binding {
	return fabric.Binding{Interface: "eth0", Mode: fabric.ModeAccess}
}

func TestNativeSessionCoexistsWithTaggedSessions(t *testing.T) {
	registry := newSessionRegistry()
	registry.replace("tagged-200", &Simulation{Binding: trunkBinding(200)})
	registry.replace("tagged-201", &Simulation{Binding: trunkBinding(201)})

	if err := registry.validateReplacement("native", nativeBinding()); err != nil {
		t.Fatalf("native session alongside tagged sessions = %v, want accepted", err)
	}
}

func TestTaggedSessionCoexistsWithNativeSession(t *testing.T) {
	registry := newSessionRegistry()
	registry.replace("native", &Simulation{Binding: nativeBinding()})

	if err := registry.validateReplacement("tagged-200", trunkBinding(200)); err != nil {
		t.Fatalf("tagged session alongside a native session = %v, want accepted", err)
	}
}

func TestSecondNativeSessionIsRejected(t *testing.T) {
	registry := newSessionRegistry()
	registry.replace("native", &Simulation{Binding: nativeBinding()})

	err := registry.validateReplacement("native-2", nativeBinding())
	if !errors.Is(err, ErrInterfaceInUse) {
		t.Fatalf("second native session = %v, want ErrInterfaceInUse — a port has one native VLAN", err)
	}
}

func TestDuplicateTaggedVLANStillRejected(t *testing.T) {
	registry := newSessionRegistry()
	registry.replace("tagged-200", &Simulation{Binding: trunkBinding(200)})

	err := registry.validateReplacement("other", trunkBinding(200))
	if !errors.Is(err, ErrPhysicalVLANInUse) {
		t.Fatalf("duplicate VLAN = %v, want ErrPhysicalVLANInUse", err)
	}
}

func TestReplacingTheSameSessionIsAllowed(t *testing.T) {
	registry := newSessionRegistry()
	registry.replace("native", &Simulation{Binding: nativeBinding()})

	if err := registry.validateReplacement("native", nativeBinding()); err != nil {
		t.Fatalf("restarting the same native session = %v, want accepted", err)
	}
}

// TestUntaggedFramesReachTheNativeSession is the other half: the registry may
// accept a native session, but the capture demux keyed only on a VLAN tag and
// discarded untagged frames outright, so the session would receive nothing.
func TestUntaggedFramesReachTheNativeSession(t *testing.T) {
	capture := newTrunkCapture(&fakeTrunkPhysical{})

	transport, err := capture.register(nativeVLANKey)
	if err != nil {
		t.Fatalf("register native session: %v", err)
	}

	// A plain untagged Ethernet frame: dst, src, ethertype, payload.
	frame := []byte{
		0x01, 0x80, 0xc2, 0x00, 0x00, 0x00,
		0xd4, 0xc1, 0x9e, 0x23, 0x58, 0x5e,
		0x08, 0x00,
		0xde, 0xad, 0xbe, 0xef,
	}

	if !capture.dispatchFrame(frame) {
		t.Fatal("untagged frame was dropped; the native session should have received it")
	}

	select {
	case got := <-transport.rx:
		if len(got) != len(frame) {
			t.Errorf("delivered %d bytes, want %d", len(got), len(frame))
		}
	default:
		t.Error("native session queue is empty")
	}
}

// With no native session registered, untagged frames must still be dropped and
// counted rather than delivered somewhere arbitrary.
func TestUntaggedFramesStillDropWithoutANativeSession(t *testing.T) {
	capture := newTrunkCapture(&fakeTrunkPhysical{})

	frame := []byte{
		0x01, 0x80, 0xc2, 0x00, 0x00, 0x00,
		0xd4, 0xc1, 0x9e, 0x23, 0x58, 0x5e,
		0x08, 0x00,
	}

	if capture.dispatchFrame(frame) {
		t.Error("untagged frame was delivered with no native session registered")
	}
}
