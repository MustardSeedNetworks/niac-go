package daemon

import (
	"context"
	"encoding/binary"
	"errors"
	"testing"

	"github.com/gopacket/gopacket"
)

type fakeTrunkPhysical struct {
	sent []byte
}

func (*fakeTrunkPhysical) StartCaptureContext(context.Context, func(gopacket.Packet)) error {
	return nil
}

func (f *fakeTrunkPhysical) SendPacket(frame []byte) error {
	f.sent = append([]byte(nil), frame...)
	return nil
}
func (*fakeTrunkPhysical) Close() {}

func TestTrunkCaptureDispatchesOneVLANToOneSession(t *testing.T) {
	capture := newTrunkCapture(&fakeTrunkPhysical{})
	hospital, err := capture.register(200)
	if err != nil {
		t.Fatalf("register(200): %v", err)
	}
	warehouse, err := capture.register(201)
	if err != nil {
		t.Fatalf("register(201): %v", err)
	}

	frame := taggedTestFrame(200)
	if !capture.dispatchFrame(frame) {
		t.Fatal("dispatchFrame() = false, want true")
	}
	buffer := make([]byte, len(frame))
	got, readErr := hospital.ReadPacket(buffer)
	if readErr != nil {
		t.Fatalf("ReadPacket(): %v", readErr)
	}
	if vlan, ok := frameVLAN(got); !ok || vlan != 200 {
		t.Fatalf("hospital frame VLAN = %d, ok = %t", vlan, ok)
	}
	select {
	case gotWarehouse := <-warehouse.rx:
		t.Fatalf("warehouse received hospital frame: %x", gotWarehouse)
	default:
	}
}

func TestTrunkCaptureDropsUntaggedAndUnassignedFrames(t *testing.T) {
	capture := newTrunkCapture(&fakeTrunkPhysical{})
	if capture.dispatchFrame(make([]byte, ethernetHeaderLength)) {
		t.Fatal("untagged dispatch = true, want false")
	}
	if capture.dispatchFrame(taggedTestFrame(299)) {
		t.Fatal("unassigned dispatch = true, want false")
	}
	if got := capture.drops.Load(); got != 2 {
		t.Fatalf("drops = %d, want 2", got)
	}
}

func TestTrunkSessionTransportEnforcesEgressVLAN(t *testing.T) {
	physical := &fakeTrunkPhysical{}
	capture := newTrunkCapture(physical)
	hospital, err := capture.register(200)
	if err != nil {
		t.Fatalf("register(200): %v", err)
	}
	if err = hospital.SendPacket(taggedTestFrame(201)); !errors.Is(err, ErrTrunkEgressVLAN) {
		t.Fatalf("SendPacket(wrong VLAN) error = %v", err)
	}
	frame := taggedTestFrame(200)
	if err = hospital.SendPacket(frame); err != nil {
		t.Fatalf("SendPacket(200): %v", err)
	}
	if vlan, ok := frameVLAN(physical.sent); !ok || vlan != 200 {
		t.Fatalf("physical frame VLAN = %d, ok = %t", vlan, ok)
	}
}

func TestTrunkReplaySenderRetagsCapturedFrames(t *testing.T) {
	physical := &fakeTrunkPhysical{}
	capture := newTrunkCapture(physical)
	transport, err := capture.register(200)
	if err != nil {
		t.Fatal(err)
	}
	sender := &trunkReplaySender{transport: transport, vlan: 200}
	untagged := make([]byte, ethernetHeaderLength)
	binary.BigEndian.PutUint16(untagged[12:14], 0x0800)
	if err = sender.SendPacket(untagged); err != nil {
		t.Fatal(err)
	}
	if vlan, ok := frameVLAN(physical.sent); !ok || vlan != 200 {
		t.Fatalf("untagged playback VLAN = %d, ok = %t", vlan, ok)
	}
	if err = sender.SendPacket(taggedTestFrame(999)); err != nil {
		t.Fatal(err)
	}
	if vlan, ok := frameVLAN(physical.sent); !ok || vlan != 200 {
		t.Fatalf("tagged playback VLAN = %d, ok = %t", vlan, ok)
	}
}

func TestTrunkCaptureRejectsDuplicateRegistration(t *testing.T) {
	capture := newTrunkCapture(&fakeTrunkPhysical{})
	if _, err := capture.register(200); err != nil {
		t.Fatalf("first register(200): %v", err)
	}
	if _, err := capture.register(200); !errors.Is(err, ErrTrunkVLANUnavailable) {
		t.Fatalf("second register(200) error = %v", err)
	}
	capture.unregister(200, capture.sessions[200])
	if _, err := capture.register(200); err != nil {
		t.Fatalf("register(200) after unregister: %v", err)
	}
}

func TestTrunkCaptureAtomicallyReplacesOneVLANTransport(t *testing.T) {
	capture := newTrunkCapture(&fakeTrunkPhysical{})
	previous, err := capture.register(200)
	if err != nil {
		t.Fatal(err)
	}
	replacement, displaced := capture.replace(200)
	if displaced != previous || capture.sessions[200] != replacement {
		t.Fatalf("replacement = %p, displaced = %p", replacement, displaced)
	}
	capture.unregister(200, previous)
	if capture.sessions[200] != replacement {
		t.Fatal("closing the displaced transport removed the replacement")
	}
}

func taggedTestFrame(vlan uint16) []byte {
	frame := make([]byte, ethernetHeaderLength+4)
	binary.BigEndian.PutUint16(frame[12:14], dot1QEtherType)
	binary.BigEndian.PutUint16(frame[14:16], vlan)
	binary.BigEndian.PutUint16(frame[16:18], 0x0800)
	return frame
}
