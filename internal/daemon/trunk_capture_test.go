package daemon

import (
	"context"
	"encoding/binary"
	"errors"
	"slices"
	"testing"
	"time"

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
	drops := capture.drops.snapshot()
	if drops.Total != 2 {
		t.Fatalf("total drops = %d, want 2", drops.Total)
	}
	// The two drops have different causes and must not be conflated: one frame
	// carried no tag, the other a tag no session serves.
	if drops.Untagged != 1 {
		t.Errorf("untagged drops = %d, want 1", drops.Untagged)
	}
	if drops.Unapproved != 1 {
		t.Errorf("unapproved drops = %d, want 1", drops.Unapproved)
	}
	if got := drops.UnapprovedByVLAN[299]; got != 1 {
		t.Errorf("unapproved drops on VLAN 299 = %d, want 1", got)
	}
}

func TestTrunkDropsSeparateOverrunFromUnapproved(t *testing.T) {
	capture := newTrunkCapture(&fakeTrunkPhysical{})
	transport, err := capture.register(200)
	if err != nil {
		t.Fatal(err)
	}
	// Fill the session's ingress queue so the next frame for its own VLAN has
	// nowhere to go. That is a session falling behind, not stray trunk traffic.
	for range trunkIngressQueue {
		transport.rx <- []byte{}
	}
	if capture.dispatchFrame(taggedTestFrame(200)) {
		t.Fatal("dispatch into a full queue = true, want false")
	}
	drops := capture.drops.snapshot()
	if drops.Overrun != 1 {
		t.Fatalf("overrun drops = %d, want 1", drops.Overrun)
	}
	if drops.Unapproved != 0 {
		t.Errorf("unapproved drops = %d, want 0 — an overrun is not stray traffic", drops.Unapproved)
	}
	if got := drops.OverrunByVLAN[200]; got != 1 {
		t.Errorf("overrun drops on VLAN 200 = %d, want 1", got)
	}
}

func TestTrunkUnapprovedVLANTrackingIsBounded(t *testing.T) {
	capture := newTrunkCapture(&fakeTrunkPhysical{})
	// Whatever is on the wire must not be able to size our map.
	for vlan := 1; vlan <= maxTrackedUnapprovedVLANs*4; vlan++ {
		capture.dispatchFrame(taggedTestFrame(uint16(vlan)))
	}
	drops := capture.drops.snapshot()
	if len(drops.UnapprovedByVLAN) > maxTrackedUnapprovedVLANs {
		t.Errorf("tracked %d distinct VLANs, want at most %d",
			len(drops.UnapprovedByVLAN), maxTrackedUnapprovedVLANs)
	}
	// The total still counts every dropped frame; only the per-VLAN detail is capped.
	if want := uint64(maxTrackedUnapprovedVLANs * 4); drops.Unapproved != want {
		t.Errorf("unapproved total = %d, want %d", drops.Unapproved, want)
	}
}

func TestTrunkCaptureFailureWakesSessionsAndReportsUnhealthy(t *testing.T) {
	capture := newTrunkCapture(&fakeTrunkPhysical{})
	transport, err := capture.register(200)
	if err != nil {
		t.Fatal(err)
	}

	read := make(chan struct{})
	go func() {
		// A session blocked here would hang forever on a dead capture.
		_, _ = transport.ReadPacket(make([]byte, 2048))
		close(read)
	}()

	capture.fail(errors.New("pcap handle closed"))

	select {
	case <-read:
	case <-time.After(2 * time.Second):
		t.Fatal("ReadPacket still blocked after the capture failed")
	}

	health := capture.health("eth0")
	if health.Healthy {
		t.Error("health.Healthy = true after a capture failure, want false")
	}
	if health.Error != "pcap handle closed" {
		t.Errorf("health.Error = %q, want the capture error", health.Error)
	}
	if !slices.Contains(health.SessionVLANs, uint16(200)) {
		t.Errorf("health.SessionVLANs = %v, want it to name the affected session", health.SessionVLANs)
	}
}

func TestTrunkCaptureFailureStopsSendAndNewSessions(t *testing.T) {
	capture := newTrunkCapture(&fakeTrunkPhysical{})
	transport, err := capture.register(200)
	if err != nil {
		t.Fatal(err)
	}
	capture.fail(errors.New("interface went down"))

	// A degraded session must not appear to transmit.
	if sendErr := transport.SendPacket(taggedTestFrame(200)); !errors.Is(sendErr, ErrTrunkCaptureFailed) {
		t.Errorf("SendPacket error = %v, want ErrTrunkCaptureFailed", sendErr)
	}
	// Nor should a new session start on a dead trunk and report success.
	if _, regErr := capture.register(201); !errors.Is(regErr, ErrTrunkCaptureFailed) {
		t.Errorf("register error = %v, want ErrTrunkCaptureFailed", regErr)
	}
}

func TestTrunkCaptureFailureIsRecordedOnce(t *testing.T) {
	capture := newTrunkCapture(&fakeTrunkPhysical{})
	capture.fail(errors.New("first"))
	capture.fail(errors.New("second"))
	if got := capture.health("eth0").Error; got != "first" {
		t.Errorf("health.Error = %q, want the first failure to be kept", got)
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
