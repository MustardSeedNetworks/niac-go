package protocols

import (
	"encoding/binary"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
	"github.com/MustardSeedNetworks/niac-go/internal/logging"
)

type recordingCapture struct {
	frame []byte
}

func (*recordingCapture) ReadPacket([]byte) ([]byte, error) { return nil, nil }
func (c *recordingCapture) SendPacket(frame []byte) error {
	c.frame = append([]byte(nil), frame...)
	return nil
}
func (*recordingCapture) SetFilter(string) error { return nil }
func (*recordingCapture) Filter() string         { return "" }

type recordingObserver struct {
	pkt *Packet
}

func (o *recordingObserver) OnPacket(direction string, pkt *Packet) {
	if direction == "tx" {
		o.pkt = pkt.Clone()
	}
}

func TestRoutedEgressIsAlwaysUntagged(t *testing.T) {
	tests := []struct {
		name           string
		mode           fabric.AttachmentMode
		authoredTagged bool
	}{
		{name: "access VLAN metadata", mode: fabric.ModeAccess},
		{name: "access authored tag", mode: fabric.ModeAccess, authoredTagged: true},
		{name: "direct VLAN metadata", mode: fabric.ModeDirect},
		{name: "direct authored tag", mode: fabric.ModeDirect, authoredTagged: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertRoutedEgressUntagged(t, tt.mode, tt.authoredTagged)
		})
	}
}

func assertRoutedEgressUntagged(t *testing.T, mode fabric.AttachmentMode, authoredTagged bool) {
	t.Helper()
	capture := &recordingCapture{}
	stack := newStack(capture, &config.Config{}, logging.NewDebugConfig(0))
	stack.ConfigureFabric(&fabric.Topology{Binding: fabric.CompiledBinding{
		Binding: fabric.Binding{Mode: mode, PolicyApproved: true}, WireTagged: false,
	}})
	observer := &recordingObserver{}
	stack.AddPacketObserver(observer)
	frame := untaggedFrame(t)
	if authoredTagged {
		frame = insertDot1Q(frame, 200)
	}

	stack.sendPacket(&Packet{Buffer: frame, Length: len(frame), VLAN: 200})

	assertUntaggedFrame(t, capture.frame)
	if observer.pkt == nil {
		t.Fatal("observer did not receive transmitted packet")
	}
	assertUntaggedFrame(t, observer.pkt.Buffer)
	if observer.pkt.VLAN > 0 {
		t.Fatalf("observer VLAN = %d, want untagged", observer.pkt.VLAN)
	}
	if got := stack.GetStats().PacketsSent; got != 1 {
		t.Fatalf("PacketsSent = %d, want 1", got)
	}
}

func TestLegacyEgressPreservesVLANTagging(t *testing.T) {
	capture := &recordingCapture{}
	stack := newStack(capture, &config.Config{}, logging.NewDebugConfig(0))
	observer := &recordingObserver{}
	stack.AddPacketObserver(observer)
	frame := untaggedFrame(t)

	stack.sendPacket(&Packet{Buffer: frame, Length: len(frame), VLAN: 200})

	assertTaggedFrame(t, capture.frame, 200)
	if observer.pkt == nil {
		t.Fatal("observer did not receive transmitted packet")
	}
	assertTaggedFrame(t, observer.pkt.Buffer, 200)
	if observer.pkt.VLAN != 200 {
		t.Fatalf("observer VLAN = %d, want 200", observer.pkt.VLAN)
	}
}

func TestRoutedEgressStripsStackedAndProviderTags(t *testing.T) {
	capture := &recordingCapture{}
	stack := newStack(capture, &config.Config{}, logging.NewDebugConfig(0))
	stack.ConfigureFabric(&fabric.Topology{Binding: fabric.CompiledBinding{
		Binding: fabric.Binding{Mode: fabric.ModeAccess, PolicyApproved: true}, WireTagged: false,
	}})
	frame := rawVLANTag(rawVLANTag(untaggedFrame(t), dot1adTPID, 200), legacyQinQTPID, 300)

	stack.sendPacket(&Packet{Buffer: frame, Length: len(frame), VLAN: 300})

	assertUntaggedFrame(t, capture.frame)
	if got := binary.BigEndian.Uint16(capture.frame[ethMACsLen:]); isVLANTPID(got) {
		t.Fatalf("wire EtherType = %#x, want all VLAN tags removed", got)
	}
}

func TestRoutedEgressRejectionIsObservable(t *testing.T) {
	stack := newStack(&recordingCapture{}, &config.Config{}, logging.NewDebugConfig(0))
	stack.ConfigureFabric(&fabric.Topology{Binding: fabric.CompiledBinding{
		Binding: fabric.Binding{Mode: fabric.ModeAccess, AccessVLAN: 200, PolicyApproved: true},
	}})
	observer := &recordingObserver{}
	stack.AddPacketObserver(observer)

	stack.sendPacket(&Packet{Buffer: []byte{0x01}, Length: 1})

	if observer.pkt == nil {
		t.Fatal("observer did not receive rejected packet")
	}
	trace := observer.pkt.FabricTrace()
	if trace.RouteDecision != "dropped" || trace.RejectionReason != "egress_rejected" {
		t.Fatalf("FabricTrace() = %#v", trace)
	}
	if got := stack.GetStats().FabricDrops; got != 1 {
		t.Fatalf("FabricDrops = %d, want 1", got)
	}
}

func TestRoutedSendQueueOverflowIsObservable(t *testing.T) {
	stack := newStack(&recordingCapture{}, &config.Config{}, logging.NewDebugConfig(0))
	stack.ConfigureFabric(&fabric.Topology{Binding: fabric.CompiledBinding{
		Binding: fabric.Binding{Mode: fabric.ModeAccess, AccessVLAN: 200, PolicyApproved: true},
	}})
	stack.sendQueue = make(chan *Packet, 1)
	stack.Send(NewPacket(64))
	observer := &recordingObserver{}
	stack.AddPacketObserver(observer)

	stack.Send(NewPacket(64))

	if observer.pkt == nil {
		t.Fatal("observer did not receive dropped packet")
	}
	trace := observer.pkt.FabricTrace()
	if trace.RouteDecision != "dropped" || trace.RejectionReason != "send_queue_full" {
		t.Fatalf("FabricTrace() = %#v", trace)
	}
	if got := stack.GetStats().FabricDrops; got != 1 {
		t.Fatalf("FabricDrops = %d, want 1", got)
	}
}

func assertUntaggedFrame(t *testing.T, frame []byte) {
	t.Helper()
	if len(frame) < ethHeaderLen {
		t.Fatalf("frame length = %d, want at least %d", len(frame), ethHeaderLen)
	}
	if got := binary.BigEndian.Uint16(frame[ethMACsLen:]); got == dot1qTPID {
		t.Fatalf("wire EtherType = %#x, want untagged", got)
	}
}

func assertTaggedFrame(t *testing.T, frame []byte, vlan int) {
	t.Helper()
	if len(frame) < ethHeaderLen+dot1qTagLen {
		t.Fatalf("frame length = %d, want VLAN header", len(frame))
	}
	if got := binary.BigEndian.Uint16(frame[ethMACsLen:]); got != dot1qTPID {
		t.Fatalf("wire EtherType = %#x, want 802.1Q", got)
	}
	if got := int(binary.BigEndian.Uint16(frame[ethMACsLen+2:]) & dot1qVLANIDMask); got != vlan {
		t.Fatalf("wire VLAN = %d, want %d", got, vlan)
	}
}
