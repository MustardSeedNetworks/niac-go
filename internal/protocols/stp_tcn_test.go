package protocols

import (
	"bytes"
	"encoding/binary"
	"net"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

func TestSTPAdvertisementsUseScenarioSegment(t *testing.T) {
	for _, kind := range []string{"configuration", "topology-change"} {
		t.Run(kind, func(t *testing.T) {
			assertSTPScenarioSegment(t, kind)
		})
	}
}

func assertSTPScenarioSegment(t *testing.T, kind string) {
	t.Helper()
	cfg := &config.Config{Segments: []config.Segment{
		{Tag: 200, Devices: []config.Device{segDevice("hospital", "02:00:00:00:02:00", "10.0.0.1")}},
		{Tag: 300, Devices: []config.Device{segDevice("warehouse", "02:00:00:00:03:00", "10.0.0.1")}},
	}}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	send := stack.stpHandler.SendConfigBPDU
	if kind == "topology-change" {
		send = stack.stpHandler.SendTopologyChange
	}
	for index := range cfg.Segments {
		segment := &cfg.Segments[index]
		device := &segment.Devices[0]
		if index == 1 {
			device.VLAN = 999
		}
		if err := send(device); err != nil {
			t.Fatal(err)
		}
		packet := <-stack.sendQueue
		frame, vlan, err := stack.finalizeEgressFrame(packet)
		if err != nil {
			t.Fatal(err)
		}
		if packet.Device != device || vlan != segment.Tag {
			t.Errorf("%s: VLAN=%d, want authored device on segment %d", kind, vlan, segment.Tag)
			continue
		}
		assertTaggedFrame(t, frame, segment.Tag)
	}
}

func TestTopologyChangeFrameCarriesExactSourceAndVLAN(t *testing.T) {
	_, stack := flatDiscoveryStack()
	device := &stack.config.Devices[0]
	device.VLAN = 220
	if err := stack.stpHandler.SendTopologyChange(device); err != nil {
		t.Fatal(err)
	}
	packet := <-stack.sendQueue
	if packet.Device != device || packet.VLAN != device.VLAN {
		t.Fatal("TCN lost authored identity or segment")
	}
	frame, vlan, err := stack.finalizeEgressFrame(packet)
	if err != nil || vlan != 220 {
		t.Fatalf("egress: VLAN=%d err=%v", vlan, err)
	}
	untagged, err := stripDot1Q(frame)
	if err != nil {
		t.Fatal(err)
	}
	destination, _ := net.ParseMAC(STPMulticastMAC)
	if !bytes.Equal(untagged[:6], destination) || !bytes.Equal(untagged[6:12], device.MACAddress) {
		t.Fatalf("wrong TCN Ethernet identity: %x", untagged[:12])
	}
	if binary.BigEndian.Uint16(untagged[12:14]) != 7 ||
		!bytes.Equal(untagged[14:21], []byte{0x42, 0x42, 0x03, 0, 0, 0, 0x80}) {
		t.Fatalf("not an LLC + short TCN BPDU: %x", untagged[12:21])
	}
	if len(untagged) != 60 || !bytes.Equal(untagged[21:], make([]byte, 39)) {
		t.Fatalf("invalid TCN padding: %x", untagged)
	}
}

func TestTopologyChangeStaysOnRoutedAttachment(t *testing.T) {
	for index, allowed := range []bool{true, false} {
		_, stack := routedDiscoveryStack()
		device := &stack.config.Devices[index]
		if err := stack.stpHandler.SendTopologyChange(device); err != nil {
			t.Fatal(err)
		}
		packet := <-stack.sendQueue
		_, _, err := stack.finalizeEgressFrame(packet)
		if (err == nil) != allowed {
			t.Fatalf("device %s egress allowed=%v, error=%v", device.Name, allowed, err)
		}
	}
}

func TestTopologyChangeRejectsInvalidSource(t *testing.T) {
	_, stack := flatDiscoveryStack()
	if err := stack.stpHandler.SendTopologyChange(&config.Device{}); err == nil {
		t.Fatal("accepted a TCN without source MAC")
	}
	if len(stack.sendQueue) != 0 {
		t.Fatal("invalid frame was queued")
	}
}
